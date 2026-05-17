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

		// Not isAlpha, so let's grab the run of any invalid characters
		// as one diagnostic. Then the loop goes around again and will pick up
		// any more valid ident chars.
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

		if s.restStartsWith("{{") {
			scanInterp(s)

			continue
		}

		if isURL(s.peek()) {
			for !s.atEOF() && isURL(s.peek()) && !s.restStartsWith("{{") {
				s.next()
			}

			s.emit(token.Text)

			continue
		}

		// Same trick, slurp up all bad chars into one diagnostic
		first := s.peek()
		s.next()

		for !s.atEOF() && !isURL(s.peek()) && !isLineSpace(s.peek()) && !s.restStartsWith("{{") {
			s.next()
		}

		s.errorf("invalid character %q in URL", first)
	}

	return s.tokens, s.diagnostics
}

// InterpolatedText is the tokeniser for a run of text that may contain interpolation
// fragments ("{{ ... }}").
func InterpolatedText(span source.Span) ([]token.Token, []diagnostic.Diagnostic) {
	s := newScanner(span)
	scanInterpolatedText(s)

	return s.tokens, s.diagnostics
}

// Header is the inline tokeniser for a header line.
func Header(span source.Span) ([]token.Token, []diagnostic.Diagnostic) {
	s := newScanner(span)
	s.skip(isLineSpace)

	// name e.g. 'Content-Type', run of idents
	if !isAlpha(s.peek()) {
		// Invalid header, slurp up to ':' or EOF as one diagnostic
		s.takeUntil(':') // Implicitly includes eof
		s.error("invalid header name")
	} else {
		s.takeWhile(isIdent)
		s.emit(token.Ident)
	}

	s.skip(isLineSpace)

	// ':'
	if s.take(":") {
		s.emit(token.Colon)
	} else {
		if !s.atEOF() {
			s.next()
		}

		s.error("header line missing ':'")
	}

	// value e.g. 'application/json'
	s.skip(isLineSpace)
	scanInterpolatedText(s)

	return s.tokens, s.diagnostics
}

// Directive scans a directive line, e.g. a global or
// request variable or config such as @no-redirect.
//
// The caller (the block parser) is responsible for ensuring the line
// is directive-shaped: an optional comment prefix ('#' or '//') and
// whitespace, then '@'. Both bare ('@x = y') and comment-disguised
// ('# @x = y') forms are accepted; the comment prefix is skipped
// without emitting any tokens for it.
func Directive(span source.Span) ([]token.Token, []diagnostic.Diagnostic) {
	s := newScanner(span)

	// Optional comment prefix. When present it is emitted as a
	// Comment token. The assembler treats '# @x' and '@x'
	// identically regardless.
	if s.takeExact("//") || s.takeExact("#") {
		s.emit(token.Comment)
	}

	s.skip(isLineSpace)

	// '@'
	if !s.takeExact("@") {
		s.errorf("expected '@' got %q", span.Content())
	} else {
		s.emit(token.At)
	}

	// 'baseURL'
	if !isAlpha(s.peek()) {
		s.errorf("expected an ident following '@', got %q", s.peek())
	} else {
		s.takeWhile(isIdent)
		s.emit(token.Ident)
	}

	s.skip(isLineSpace)

	// The '=' is optional
	if s.take("=") {
		s.emit(token.Eq)
	}

	s.skip(isLineSpace)

	// Value
	if next := s.peek(); isDigit(next) || next == '+' || next == '-' || next == '.' {
		scanNumber(s)

		// scanNumber stops at the first character it doesn't recognise as
		// part of a number; anything left on the line is unexpected.
		s.skip(isLineSpace)

		if !s.atEOF() {
			for !s.atEOF() {
				s.next()
			}

			s.error("unexpected content after directive value")
		}

		return s.tokens, s.diagnostics
	}

	// Normal text value, possibly with interpolations
	scanInterpolatedText(s)

	return s.tokens, s.diagnostics
}

// Script is the inline tokeniser for a script.
//
//   - '<' begins a request script
//   - '>' denotes a response script
//
// Scripts may be inline like '< {% ... %}' or point to a path
// on disk holding the script like '< path/to/script.js'.
//
// This tokeniser handles all kinds.
func Script(span source.Span) ([]token.Token, []diagnostic.Diagnostic) {
	s := newScanner(span)

	switch {
	case s.take("<"):
		s.emit(token.LAngle)
	case s.take(">"):
		s.emit(token.RAngle)
	default:
		s.errorf("expected '<' or '>', got %q", s.peek())
	}

	s.skip(isLineSpace)

	if s.atEOF() {
		s.error("unexpected EOF: expected filepath or '{% ... %}' script block")

		return s.tokens, s.diagnostics
	}

	if !s.takeExact("{%") {
		scanInterpolatedText(s)

		return s.tokens, s.diagnostics
	}

	// Must be a '{%'
	s.emit(token.OpenScript)

	for {
		if s.atEOF() {
			s.error("unterminated script block")

			return s.tokens, s.diagnostics
		}

		if s.restStartsWith("%}") {
			// Capture script body bytes verbatim so the assembler can preserve
			// the script source even though the parser never evaluates it.
			if s.pos > s.start {
				s.emit(token.Text)
			}

			s.takeExact("%}")
			s.emit(token.CloseScript)

			break
		}

		s.next()
	}

	// A '{% ... %}' block is the whole script; anything after the
	// CloseScript is user error. Trailing whitespace is fine.
	s.skip(isLineSpace)

	if !s.atEOF() {
		for !s.atEOF() {
			s.next()
		}

		s.error("unexpected content after script close")
	}

	return s.tokens, s.diagnostics
}

// Body tokenises a request body.
//
//   - < body.json (literal, the file bytes are sent verbatim)
//   - <@ body.json (templated, run the file contents through interpolation)
//   - <@{encoding} body.json (templated, with explicit source encoding)
//   - inline (raw text with possible '{{ ... }}' interpolations)
//
// A leading '<' marks a file-ref body only when followed by whitespace, '@',
// a line terminator, or EOF. Otherwise the '<' is just the first character of
// an inline body (e.g. '<html>...</html>').
//
// The path of a file-ref body occupies the rest of its line. Anything past
// that line is unexpected content and gets diagnosed.
func Body(span source.Span) ([]token.Token, []diagnostic.Diagnostic) {
	s := newScanner(span)

	if !isFileRefOpener(s) {
		scanInterpolatedText(s)

		return s.tokens, s.diagnostics
	}

	s.take("<")
	s.emit(token.LAngle)

	// '@' is only significant after a '<'
	if s.take("@") {
		s.emit(token.At)
	}

	// Is there an encoding? e.g. '<@utf8'
	if isAlpha(s.peek()) {
		s.takeWhile(isIdent)
		s.emit(token.Ident)
	}

	s.skip(isLineSpace)

	if s.atEOF() || isLineTerminator(s.peek()) {
		s.error("expected filepath")
		// Fall through so any content on subsequent lines is still
		// flagged by the trailing-content check rather than silently
		// dropped.
	}

	// The path lives on this line only; the tokeniser must not cross the
	// newline so that any trailing content can be flagged separately.
	scanInterpolatedTextLine(s)

	// Step over the line terminator so the trailing-content diagnostic
	// points at the offending bytes, not the newline.
	s.skip(isLineTerminator)

	if !s.atEOF() {
		for !s.atEOF() {
			s.next()
		}

		s.error("unexpected content after file-ref body")
	}

	return s.tokens, s.diagnostics
}

// ResponseRedirect tokenises a response redirect line.
//
//   - '>> response.json' redirects to the file, but fails if it already exists
//   - '>>! response.json' redirects and overwrites the file
func ResponseRedirect(span source.Span) ([]token.Token, []diagnostic.Diagnostic) {
	s := newScanner(span)

	switch {
	case s.takeExact(">>!"):
		s.emit(token.ResponseRedirectForce)
	case s.takeExact(">>"):
		s.emit(token.ResponseRedirect)
	default:
		// Consume the remainder of the input so the diagnostic spans the
		// offending bytes rather than producing a zero-width error and
		// silently dropping the line.
		for !s.atEOF() {
			s.next()
		}

		s.errorf("expected '>>' or '>>!', got %q", span.Content())

		return s.tokens, s.diagnostics
	}

	s.skip(isLineSpace)

	if s.atEOF() {
		s.error("expected filepath following response redirect")

		return s.tokens, s.diagnostics
	}

	scanInterpolatedTextLine(s)

	// Step over the line terminator so the trailing-content diagnostic
	// points at the offending bytes, not the newline.
	s.skip(isLineTerminator)

	if !s.atEOF() {
		for !s.atEOF() {
			s.next()
		}

		s.error("unexpected content after response redirect")
	}

	return s.tokens, s.diagnostics
}

// isFileRefOpener reports whether the scanner is positioned at a '<' that
// introduces a file-ref body (as opposed to an inline body that happens to
// begin with '<').
//
// The byte immediately following the '<' is the disambiguator: whitespace,
// '@', a line terminator, or EOF mean file-ref; anything else means inline
// content. A bare '<' at EOF is reported as a malformed file-ref by the main
// flow rather than silently treated as inline content.
func isFileRefOpener(s *scanner) bool {
	if s.peek() != '<' {
		return false
	}

	if s.pos+1 >= len(s.src) {
		return true
	}

	switch s.src[s.pos+1] {
	case ' ', '\t', '@', '\n', '\r':
		return true
	}

	return false
}

// isLineTerminator reports whether r is '\n' or '\r'.
func isLineTerminator(r rune) bool {
	return r == '\n' || r == '\r'
}

// scanInterpolatedTextLine is [scanInterpolatedText] but stops at the first
// line terminator. Used where the text must not cross a newline boundary
// (e.g. a file-ref body's path).
func scanInterpolatedTextLine(s *scanner) {
	for !s.atEOF() && !isLineTerminator(s.peek()) {
		if s.restStartsWith("{{") {
			scanInterp(s)

			continue
		}

		for !s.atEOF() && !s.restStartsWith("{{") && !isLineTerminator(s.peek()) {
			s.next()
		}

		if s.pos > s.start {
			s.emit(token.Text)
		}
	}
}

// scanInterpolatedText scans a chunk of text that may or may not
// contain "{{ ... }}" blocks.
func scanInterpolatedText(s *scanner) {
	for !s.atEOF() {
		if s.restStartsWith("{{") {
			scanInterp(s)

			continue
		}

		for !s.atEOF() && !s.restStartsWith("{{") {
			s.next()
		}

		if s.pos > s.start {
			s.emit(token.Text)
		}
	}
}

// scanInterp handles the actual "{{ ... }}" block.
func scanInterp(s *scanner) {
	if !s.takeExact("{{") {
		s.errorf("expected '{{' got: %q", s.src[s.pos:])
	} else {
		s.emit(token.OpenInterp)
	}

	s.skip(isLineSpace)

	if s.atEOF() {
		s.error("unexpected EOF in interpolation")

		return
	}

	if s.restStartsWith("}}") {
		s.error("empty interpolation")
		s.takeExact("}}")
		s.emit(token.CloseInterp)

		return
	}

	switch {
	case isAlpha(s.peek()):
		s.takeWhile(isIdent)
		s.emit(token.Ident)
	default:
		// TODO(@FollowTheProcess): Do all the other things that can appear inside {{ ... }}
		// Take the contiguous run of unrecognised characters so the rest of
		// the interp can resume from a known boundary (whitespace, '}}', or EOF).
		bad := s.peek()
		s.next()

		for !s.atEOF() && !isLineSpace(s.peek()) && !s.restStartsWith("}}") {
			s.next()
		}

		s.errorf("unexpected character in interpolation: %q", bad)
	}

	s.skip(isLineSpace)

	if !s.takeExact("}}") {
		s.error("unterminated interpolation")
	} else {
		s.emit(token.CloseInterp)
	}
}

// scanNumber scans a number literal, either integer or float.
func scanNumber(s *scanner) {
	// TODO(@FollowTheProcess): Hex and imaginary from Rob Pikes slides
	// I doubt we'll *need* them in most .http files but it's easy to
	// support so why not?
	// https://go.dev/talks/2011/lex.slide#35
	s.take("+-") // Optional leading sign

	beforeInt := s.pos
	s.takeWhile(isDigit)
	sawDigit := s.pos > beforeInt

	// Floats
	if s.take(".") {
		beforeFrac := s.pos
		s.takeWhile(isDigit)
		sawDigit = sawDigit || s.pos > beforeFrac
	}

	// A sign or dot with no digits is not a number. Slurp the rest of the
	// number-ish run so the user sees one focused error rather than partial
	// recovery.
	if !sawDigit {
		for !s.atEOF() {
			r := s.peek()
			if !isAlphaNumeric(r) && r != '.' && r != '+' && r != '-' {
				break
			}

			s.next()
		}

		s.error("number must have digits")

		return
	}

	// Powers
	if s.take("eE") {
		s.take("+-")
		s.takeWhile(isDigit)
	}

	// Bad trailing characters, eat the whole thing as a diagnostic
	if bad := s.peek(); isAlphaNumeric(bad) || bad == '.' {
		s.emit(token.Number)

		for !s.atEOF() && (isAlphaNumeric(s.peek()) || s.peek() == '.') {
			s.next()
		}

		s.errorf("unexpected %q in number", bad)

		return
	}

	s.emit(token.Number)
}
