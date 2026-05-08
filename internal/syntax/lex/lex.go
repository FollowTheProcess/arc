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
		s.errorf("expected '###' got %s", s.src[s.pos:])
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

		// Not a valid identifier leader. Emit a precise per-rune diagnostic
		// and let the loop pick up any trailing valid identifier.
		r := s.next()
		if isIdent(r) {
			s.errorf("identifier cannot start with %q", r)
		} else {
			s.errorf("invalid character %q in separator", r)
		}
	}

	return s.tokens, s.diagnostics
}
