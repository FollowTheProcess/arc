package assembler

import (
	"fmt"

	"go.followtheprocess.codes/arc/internal/syntax/ast"
	"go.followtheprocess.codes/arc/internal/syntax/block"
	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/arc/internal/syntax/token"
)

// parser is a small, block-scoped parser whose job is to parse
// the inline tokenised content for a single block into abstract
// syntax tree nodes.
type parser struct {
	diagnostics []diagnostic.Diagnostic // Diagnostics gathered during parsing
	block       block.Block             // The block under inspection
	current     token.Token             // The current token under inspection
	next        token.Token             // The next token in the stream
	pos         int                     // Current position as an index into blk.Tokens
}

// newParser creates a new parser for a given block.
func newParser(b block.Block) *parser {
	p := &parser{block: b}

	// Read 2 tokens so current and next are both set
	p.advance()
	p.advance()

	return p
}

// atEOF reports whether the parser is at the end of the block's
// token stream.
//
// Technically EOF means "end-of-file", but the fundamental unit here is the
// block so we repurpose it, the meaning is the same: No more meaningful tokens.
func (p *parser) atEOF() bool {
	return p.pos >= len(p.block.Tokens)
}

// advance advances the parser by a single token.
func (p *parser) advance() {
	p.current = p.next

	if p.atEOF() {
		// Emit a synthetic EOF token, makes a lot of other stuff easier
		p.next = token.Token{Kind: token.EOF, Start: p.current.End, End: p.current.End}

		return
	}

	p.next = p.block.Tokens[p.pos]
	p.pos++
}

// expect asserts that the next token is one of the given kinds, emitting a diagnostic and
// returning false if not.
//
// The parser is advanced only if the next token is one of these kinds such that after returning true,
// p.current will be one of the given kinds.
//
// If the next token is [token.Error], expect returns silently (without publishing a diagnostic),
// to avoid doubling up the error already emitted by the lexer.
func (p *parser) expect(kinds ...token.Kind) bool {
	if p.next.Is(token.Error) {
		return false // already diagnosed
	}

	switch len(kinds) {
	case 0:
		return false
	case 1:
		if !p.next.Is(kinds[0]) {
			p.errorf(p.next, "expected %s, got %s", kinds[0], p.next.Kind)

			return false
		}
	default:
		if !p.next.Is(kinds...) {
			p.errorf(p.next, "expected one of %v, got %s", kinds, p.next.Kind)

			return false
		}
	}

	p.advance()

	return true
}

// error appends an error level diagnostic to the parser referring to
// the given token.
func (p *parser) error(tok token.Token, msg string, options ...diagnostic.Option) {
	span := source.Span{
		File:        p.block.Span.File,
		StartOffset: tok.Start,
		EndOffset:   tok.End,
	}

	diag := diagnostic.Error(msg, span, options...)
	p.diagnostics = append(p.diagnostics, diag)
}

// errorf append a formatter error level diagnostic to the parser referring
// to the given token.
//
// To pass any [diagnostic.Option] to the diagnostic, use [parser.error].
func (p *parser) errorf(tok token.Token, format string, a ...any) {
	p.error(tok, fmt.Sprintf(format, a...))
}

// span returns a [source.Span] for the parser's current token.
func (p *parser) span() source.Span {
	return source.Span{
		File:        p.block.Span.File,
		StartOffset: p.current.Start,
		EndOffset:   p.current.End,
	}
}

// text returns the chunk of source text described by the p.current token
// as a string.
func (p *parser) text() string {
	return string(p.span().Content())
}

// parseDirective parses a directive block's tokens into an [ast.Directive].
func (p *parser) parseDirective() ast.Directive {
	node := ast.Directive{
		Span: p.block.Span,
	}

	// Optional leading comment
	if p.current.Is(token.Comment) {
		p.advance()
	}

	// Block pass guarantees current is now '@'

	if !p.expect(token.Ident) {
		return node
	}

	node.Ident = p.parseIdent()

	// TODO: Prompt and secret directives can have another ident
	// `@prompt ident ["description"]`
	//   ^^^^^^ ^^^^^
	//    one    two
	//
	// Not sure if it's better to have an ast.Prompt or handle this with
	// a *Ident on ast.Directive which is nil unless it's a secret or prompt
	//
	// That sounds more extensible tbf so maybe that one

	// Optional '='
	if p.next.Is(token.Eq) {
		p.advance()
	}

	// Flag directives like '@no-redirect' have no value
	if p.next.Is(token.EOF) {
		return node
	}

	p.advance()

	// Value (expression)
	node.Value = p.parseExpression(token.LowestPrecedence)

	return node
}

// parseIdent parses an ident into an [ast.Ident].
func (p *parser) parseIdent() ast.Ident {
	return ast.Ident{
		Span: p.span(),
	}
}

// parseExpression parses an arbitrary [ast.Expression].
func (p *parser) parseExpression(precedence int) ast.Expression {
	// TODO: Precedence, interps, and all that fun stuff
	switch p.current.Kind {
	case token.Text:
		return p.parseTextLiteral()
	case token.Quote:
		return p.parseQuotedText()
	case token.Ident:
		return p.parseIdent()
	default:
		p.errorf(p.current, "parseExpression: unexpected token %s", p.current.Kind)

		return nil
	}
}

// parseTextLiteral parses a TextLiteral.
func (p *parser) parseTextLiteral() ast.TextLiteral {
	return ast.TextLiteral{
		Value: p.text(),
		Span:  p.span(),
	}
}

// parseQuotedText parses a quoted string of text as a TextLiteral.
//
// The resulting span covers the entire literal including the surrounding
// quotes, while Value holds only the unquoted text. An empty literal ("")
// is valid and yields an empty Value.
func (p *parser) parseQuotedText() ast.TextLiteral {
	open := p.current // Opening quote
	end := open.End

	var value string

	if p.next.Is(token.Text) {
		p.advance()
		value = p.text()
		end = p.current.End
	}

	if p.expect(token.Quote) {
		end = p.current.End
	}

	return ast.TextLiteral{
		Value: value,
		Span: source.Span{
			File:        p.block.Span.File,
			StartOffset: open.Start,
			EndOffset:   end,
		},
	}
}

// parseSeparator parses a request separator with an optional name.
//
// If the name is found it is returned as an ident, otherwise nil.
func (p *parser) parseSeparator() *ast.Ident {
	if p.next.Is(token.Ident) {
		p.advance()

		return &ast.Ident{
			Span: p.span(),
		}
	}

	return nil
}

// parseRequestLine parses a request's METHOD, URL, [HTTPVersion] line.
//
// It assumes the current token is the METHOD ident.
func (p *parser) parseRequestLine() (ast.Ident, ast.Expression) {
	// TODO: HTTPVersion
	method := p.parseIdent()

	if p.expect(token.Text, token.OpenInterp) {
		url := p.parseExpression(token.LowestPrecedence)

		return method, url
	}

	return method, nil
}
