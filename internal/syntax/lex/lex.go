// Package lex implements the inline tokenisation pass of arc's .http parser.
//
// The block package decides what kind of block each line is and emits a
// stream of typed [block.Block] values. lex takes a block's content and
// breaks it into [token.Token] values: header names and values, request
// methods, URL paths, interpolations, separator names, and so on.
//
// Tokenisation is dispatched per block kind. Each kind that carries inline
// content has a dedicated tokeniser exported from lex.
//
// Each tokeniser takes the block src ([]byte) and a base offset (the start offset
// of the block) from which to calculate the absolute token positions.
//
// Like the block pass, lex is fault tolerant. Malformed input becomes a
// token of kind [token.Error] with an attached diagnostic and tokenisation
// continues, so a syntactically broken line still yields a structured
// token stream that downstream consumers can work with.
//
// This is the inline half of the two-pass model common to markdown
// parsers: the block pass carves the file into typed regions, and in inline
// tokeniser tokenises what's inside each one.
package lex

import (
	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/arc/internal/syntax/token"
)

// Tokeniser is an inline tokeniser.
type Tokeniser func(span source.Span) ([]token.Token, []diagnostic.Diagnostic)

// Separator is the tokeniser for a separator block.
//
// The '###' results in a [token.Separator] and the request
// name, if present, becomes a [token.Ident].
//
// It assumes the '###' has already been recognised as the next input,
// the caller is responsible for ensuring '###' are the next bytes in src.
func Separator(span source.Span) ([]token.Token, []diagnostic.Diagnostic) {
	s := newScanner(span)
	if !s.takeExact("###") {
		s.errorf("expected '###' got %q", span.Content())
	} else {
		s.emit(token.Separator)
	}

	for {
		s.skip(isLineSpace)

		if s.atEOF() {
			break
		}

		if isAlpha(s.peek()) {
			s.takeWhile(isIdent)
			s.emit(token.Ident)

			continue
		}

		// Not a valid identifier leader. Take the contiguous run of bad
		// characters and surface it as a single diagnostic; the first rune
		// of the run picks the message category. The loop then resumes and
		// can pick up any trailing valid identifier.
		first := s.peek()
		s.next()

		for !s.atEOF() && !isAlpha(s.peek()) && !isLineSpace(s.peek()) {
			s.next()
		}

		if isIdent(first) {
			s.errorf("identifier cannot start with %q", first)
		} else {
			s.errorf("invalid character %q in separator", first)
		}
	}

	return s.tokens, s.diagnostics
}

// RequestLine is the tokeniser for a request line.
//
// It turns 'GET https://example.com' into a [token.Ident]
// followed by a [token.Text].
//
// The method ident is any contiguous run of uppercase ASCII letters; whether
// it is a known HTTP method is validated by the AST assembler downstream, not
// here. The caller (block classifier) is responsible for ensuring the line
// begins with such a run.
func RequestLine(span source.Span) ([]token.Token, []diagnostic.Diagnostic) {
	s := newScanner(span)

	s.takeWhile(isUpperAlpha)
	s.emit(token.Ident)

	for {
		s.skip(isLineSpace)

		if s.atEOF() {
			break
		}

		if isURL(s.peek()) {
			s.takeWhile(isURL)
			s.emit(token.Text)

			continue
		}

		// Not a valid URL char. Take the contiguous run of bad characters,
		// stopping at the next URL char or whitespace, and surface it as a
		// single diagnostic. The loop then resumes and can pick up trailing
		// valid URL bytes.
		first := s.peek()
		s.next()

		for !s.atEOF() && !isURL(s.peek()) && !isLineSpace(s.peek()) {
			s.next()
		}

		s.errorf("invalid character %q in URL", first)
	}

	return s.tokens, s.diagnostics
}
