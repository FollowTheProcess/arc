package lex

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/arc/internal/syntax/token"
)

// eof signifies we've reached the end of the input.
const eof = rune(-1)

// scanner is a traditional character level scanner/tokeniser, it reads
// src one utf8 rune at a time until a token is recognised. The token
// is then "emitted" into the accumulator.
type scanner struct {
	file        *source.File            // The file we're currently in
	src         []byte                  // Raw input text
	tokens      []token.Token           // Token accumulator, emitted tokens land here
	diagnostics []diagnostic.Diagnostic // Diagnostics gathered during scanning
	pos         int                     // The scanner's current (byte offset) position in src
	start       int                     // The start position of the current token
	base        int                     // The base offset from which to transform relative offsets to absolute
}

// newScanner returns a new [scanner] given a chunk of source text
// and a base offset. The scanner uses the base offset to turn relative
// offsets into absolute ones within the larger context of the file.
func newScanner(span source.Span) *scanner {
	return &scanner{
		src:  span.Content(),
		file: span.File,
		base: span.StartOffset,
	}
}

// atEOF reports whether the scanner is at the end of the input.
func (s *scanner) atEOF() bool {
	return s.pos >= len(s.src)
}

// absStart returns the start position of the current token,
// adjusted so it's the absolute position in the file rather than
// the relative position from the start of the source chunk being
// scanned.
func (s *scanner) absStart() int {
	return s.base + s.start
}

// absPos returns the scanner's current position in src,
// adjusted so it's the absolute position in the file rather than
// the relative position from the start of the source chunk being
// scanned.
func (s *scanner) absPos() int {
	return s.base + s.pos
}

// char returns the next utf8 rune in the input or [eof], along with
// its width.
//
// char is side-effect free, a bad utf8 sequence returns ([utf8.RuneError], 1).
func (s *scanner) char() (rune, int) {
	if s.atEOF() {
		return eof, 0
	}

	r, width := utf8.DecodeRune(s.src[s.pos:])
	if r == utf8.RuneError || r == 0 {
		return utf8.RuneError, 1 // Bad byte, ensure we always make progress
	}

	return r, width
}

// next returns the next utf8 rune in the input or [eof], and advances
// the scanner over that rune such that successive calls to next iterate
// through src one rune at a time.
func (s *scanner) next() rune {
	char, width := s.char()

	// Always advance
	s.pos += width

	return char
}

// peek returns the next utf8 rune in the input, [eof], or [utf8.RuneError] but unlike [scanner.next],
// does not advance the scanner. Successive calls to peek return the
// same rune over and over again.
func (s *scanner) peek() rune {
	char, _ := s.char()

	return char
}

// discard brings the start position up to current, effectively discarding
// the span of text the scanner has "collected" up to this point.
func (s *scanner) discard() {
	s.start = s.pos
}

// restStartsWith reports whether the remainder of the input begins
// with the provided run of characters.
func (s *scanner) restStartsWith(prefix string) bool {
	return len(s.src)-s.pos >= len(prefix) &&
		string(s.src[s.pos:s.pos+len(prefix)]) == prefix
}

// skip ignores any characters for which the predicate returns true, stopping at the
// first one that returns false such that after it returns, [scanner.next] returns the
// first 'false' char.
//
// The scanner start position is brought up to the current position before returning,
// effectively ignoring everything it's travelled over in the meantime.
func (s *scanner) skip(predicate func(r rune) bool) {
	for predicate(s.peek()) {
		s.next()
	}

	s.discard()
}

// take consumes the next rune if it's from the valid set, and
// returns whether it was accepted.
func (s *scanner) take(valid string) bool {
	if strings.ContainsRune(valid, s.peek()) {
		s.next() // Accept it

		return true
	}

	return false
}

// takeWhile consumes characters so long as the predicate returns true,
// stopping at the first one that returns false such that after it returns,
// [scanner.next] returns the first 'false' char.
func (s *scanner) takeWhile(predicate func(r rune) bool) {
	for predicate(s.peek()) {
		s.next()
	}
}

// takeUntil consumes characters until it hits any of the specified runes.
//
// It stops before it consumes the first specified rune such that after it
// returns, [scanner.next] returns the offending rune.
//
// It also implicitly stops on [utf8.RuneError] and [eof].
func (s *scanner) takeUntil(runes ...rune) {
	for {
		next := s.peek()
		if next == utf8.RuneError || next == eof {
			return
		}

		if slices.Contains(runes, next) {
			return
		}

		s.next()
	}
}

// takeExact consumes exactly the provided text if it is the very
// next thing the scanner encounters and returns whether it
// took it.
//
// If the next characters in src do not match, this is a no-op.
func (s *scanner) takeExact(match string) bool {
	if !s.restStartsWith(match) {
		return false
	}

	s.pos += len(match)

	return true
}

// emit adds a token to the accumulator.
func (s *scanner) emit(kind token.Kind) {
	tok := token.Token{
		Kind:  kind,
		Start: s.absStart(),
		End:   s.absPos(),
	}

	s.tokens = append(s.tokens, tok)
	s.discard() // Reset s.start
}

// error adds a error level diagnostic to the accumulator and
// emits a [token.Error] token.
//
// The attached span is the span of text the scanner is tacking
// (s.start -> s.pos).
func (s *scanner) error(msg string, options ...diagnostic.Option) {
	span := source.Span{
		File:        s.file,
		StartOffset: s.absStart(),
		EndOffset:   s.absPos(),
	}

	diag := diagnostic.Error(msg, span, options...)
	s.diagnostics = append(s.diagnostics, diag)
	s.emit(token.Error)
}

// errorf calls [scanner.error] with a formatted error message.
func (s *scanner) errorf(format string, a ...any) {
	s.error(fmt.Sprintf(format, a...))
}

// isLineSpace reports whether r in a non line-terminating whitespace
// character, imagine [unicode.IsSpace] but without '\n' or '\r'.
func isLineSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isAlpha reports whether r is an alpha character.
func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isUpperAlpha reports whether r is an upper-case ASCII alpha character.
func isUpperAlpha(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// isDigit reports whether r is a valid ASCII digit.
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isAlphaNumeric reports whether r is a valid alpha-numeric character.
func isAlphaNumeric(r rune) bool {
	return isAlpha(r) || isDigit(r)
}

// isIdent reports whether r is a valid identifier character.
func isIdent(r rune) bool {
	return isAlphaNumeric(r) || r == '_' || r == '-'
}

// isURL reports whether r is valid in a URL.
func isURL(r rune) bool {
	if r == eof || r == utf8.RuneError {
		return false
	}

	return isAlphaNumeric(r) || strings.ContainsRune("$-_.+!*'(),:/?#[]@&;=", r)
}
