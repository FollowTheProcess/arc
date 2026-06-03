package assembler

import (
	"fmt"
	"strings"

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
		Range: p.block.Span,
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
	node.Value = p.parseValue()

	return node
}

// parseIdent parses an ident into an [ast.Ident].
func (p *parser) parseIdent() ast.Ident {
	return ast.Ident{
		Range: p.span(),
	}
}

// parseValue parses a directive, header, or request value: a quoted text value,
// or a run of literal text and interpolations (an [ast.Template], or a bare part
// if there's only one).
func (p *parser) parseValue() ast.Expression {
	// Only directive text values are quoted.
	if p.current.Is(token.Quote) {
		return p.parseQuotedText()
	}

	parts := p.parseTemplateParts()

	// If there's only 1 part, it's a literal, just emit that
	switch len(parts) {
	case 0:
		return nil
	case 1:
		return parts[0]
	default:
		return ast.Template{
			Parts: parts,
			Range: source.Span{
				File:        p.block.Span.File,
				StartOffset: parts[0].Span().StartOffset,
				EndOffset:   parts[len(parts)-1].Span().EndOffset,
			},
		}
	}
}

// parseQuotedText parses a double-quoted text value, e.g. `"rfc1123"` or
// `"Bearer {{ token }}"`. A value with no interpolations collapses to a single
// [ast.TextLiteral]; otherwise it's an [ast.Template]. Either way the node spans
// the surrounding quotes.
//
// It assumes p.current is on the opening quote and returns with p.current on the
// closing quote.
func (p *parser) parseQuotedText() ast.Expression {
	open := p.current

	// An empty quoted value "" has no inner parts; emit an empty literal
	// spanning the quotes.
	if p.next.Is(token.Quote) {
		p.advance()

		return ast.TextLiteral{
			Value:  "",
			Quoted: true,
			Range: source.Span{
				File:        p.block.Span.File,
				StartOffset: open.Start,
				EndOffset:   p.current.End,
			},
		}
	}

	p.advance() // Consume the opening quote

	parts := p.parseTemplateParts()

	end := p.current.End
	if p.expect(token.Quote) {
		end = p.current.End
	}

	return p.quotedValue(parts, source.Span{
		File:        p.block.Span.File,
		StartOffset: open.Start,
		EndOffset:   end,
	})
}

// parseTemplateParts collects a contiguous run of template parts (literal text,
// interpolations, and numbers) starting at p.current, returning with p.current
// on the last part consumed.
func (p *parser) parseTemplateParts() []ast.Expression {
	var parts []ast.Expression

	for {
		switch p.current.Kind {
		case token.Text:
			parts = append(parts, p.parseTextLiteral())
		case token.OpenInterp:
			parts = append(parts, p.parseInterp())
		case token.Number:
			parts = append(parts, p.parseNumberLiteral())
		case token.Error:
			// Nothing, the lexer has already reported
			return parts
		default:
			p.errorf(p.current, "parseValue: unexpected token %s", p.current.Kind)

			return parts
		}

		// A continuous run of template tokens
		if p.next.Is(token.Text, token.OpenInterp) && p.next.Start == p.current.End {
			p.advance()

			continue // go round again
		}

		return parts
	}
}

// quotedValue assembles the parts of a quoted value into a node spanning the
// quotes. A value with no interpolations collapses to a single [ast.TextLiteral],
// otherwise it's an [ast.Template].
func (p *parser) quotedValue(parts []ast.Expression, span source.Span) ast.Expression {
	var value strings.Builder

	for _, part := range parts {
		lit, ok := part.(ast.TextLiteral)
		if !ok {
			// An interpolation, so this is a genuine template.
			return ast.Template{Parts: parts, Range: span}
		}

		value.WriteString(lit.Value)
	}

	return ast.TextLiteral{Value: value.String(), Range: span, Quoted: true}
}

// parseInterp parses an interpolation e.g. `{{ <inner> }}`.
func (p *parser) parseInterp() ast.Interp {
	start := p.current.Start

	p.advance() // Consume the OpenInterp

	expr := p.parseInterpExpr()

	p.expect(token.CloseInterp)

	return ast.Interp{
		Range: source.Span{
			File:        p.block.Span.File,
			StartOffset: start,
			EndOffset:   p.current.End,
		},
		Inner: expr,
	}
}

// parseInterpExpr parses the inner expression inside a '{{ ... }}'.
func (p *parser) parseInterpExpr() ast.Expression {
	left := p.parsePrimaryExpr()
	if left == nil {
		// parsePrimaryExpr already diagnosed; nothing to attach trailers to.
		return nil
	}

	for {
		switch p.next.Kind {
		case token.Dot:
			left = p.parseSelector(left)
		case token.LParen:
			left = p.parseCall(left)
		default:
			return left
		}
	}
}

// parseSelector parses a selector expression (left.sel) into an [ast.Selector].
func (p *parser) parseSelector(left ast.Expression) ast.Selector {
	p.advance() // Consume the '.'

	var sel ast.Ident
	if p.expect(token.Ident) {
		sel = p.parseIdent()
	}

	return ast.Selector{
		Expr: left,
		Sel:  sel,
		Range: source.Span{
			File:        p.block.Span.File,
			StartOffset: left.Span().StartOffset,
			EndOffset:   p.current.End,
		},
	}
}

// parseCall parses a call expression (left(args...)) into an [ast.Call].
//
// It is called with p.current on left's final token and p.next on the '(',
// returning with p.current on the closing ')' (or the last token consumed if
// the call is unterminated).
func (p *parser) parseCall(left ast.Expression) ast.Call {
	p.advance() // Consume the '('

	args := p.parseArgs()

	p.expect(token.RParen)

	return ast.Call{
		Fun:  left,
		Args: args,
		Range: source.Span{
			File:        p.block.Span.File,
			StartOffset: left.Span().StartOffset,
			EndOffset:   p.current.End,
		},
	}
}

// parseArgs parses a run of call argument expressions, stopping before the
// closing ')'.
func (p *parser) parseArgs() []ast.Expression {
	// Empty argument list, e.g. `$random.int()`
	if p.next.Is(token.RParen) {
		return nil
	}

	var args []ast.Expression

	for {
		// Each iteration must begin a fresh argument. A separator or premature
		// end here means a stray/trailing comma or an unterminated call, e.g.
		// `(1, 100,)` or `(,)`.
		if p.next.Is(token.RParen, token.Comma, token.CloseInterp, token.EOF) {
			p.errorf(p.next, "expected an argument, found %s", p.next.Kind)

			break
		}

		if p.next.Is(token.Error) {
			break // Already diagnosed by the lexer
		}

		p.advance()

		if arg := p.parseInterpExpr(); arg != nil {
			args = append(args, arg)
		}

		if !p.next.Is(token.Comma) {
			break
		}

		// Consume the ','
		p.advance()
	}

	return args
}

// parsePrimaryExpr parses a "primary" expression i.e. the left
// associative side of most interp expressions so the `$env` in a builtin
// or the ident of a variable etc.
func (p *parser) parsePrimaryExpr() ast.Expression {
	switch p.current.Kind {
	case token.Ident:
		return p.parseIdent()
	case token.Dollar:
		return p.parseBuiltin()
	case token.Number:
		return p.parseNumberLiteral()
	case token.Quote:
		return p.parseQuotedText()
	case token.Error:
		// Nothing, lexer has already reported
		return nil
	default:
		p.errorf(p.current, "parseInterp: unexpected token %s", p.current.Kind)

		return nil
	}
}

// parseBuiltin parses a `$` rooted builtin expression like `$env` or
// `$random`.
func (p *parser) parseBuiltin() ast.Builtin {
	start := p.current.Start // The '$'

	p.expect(token.Ident)

	return ast.Builtin{
		Name: p.parseIdent(),
		Range: source.Span{
			File:        p.block.Span.File,
			StartOffset: start,
			EndOffset:   p.current.End,
		},
	}
}

// parseTextLiteral parses a TextLiteral.
func (p *parser) parseTextLiteral() ast.TextLiteral {
	return ast.TextLiteral{
		Value: p.text(),
		Range: p.span(),
	}
}

// parseNumberLiteral parses a NumberLiteral.
func (p *parser) parseNumberLiteral() ast.NumberLiteral {
	return ast.NumberLiteral{
		Range: p.span(),
	}
}

// parseSeparator parses a request separator with an optional name.
//
// If the name is found it is returned as an ident, otherwise nil.
func (p *parser) parseSeparator() *ast.Ident {
	if p.next.Is(token.Ident) {
		p.advance()

		return &ast.Ident{
			Range: p.span(),
		}
	}

	return nil
}

// parseRequestLine parses a request's METHOD, URL, [HTTPVersion] line.
//
// It assumes the current token is the METHOD ident.
func (p *parser) parseRequestLine() (method ast.Ident, url ast.Expression, version *ast.HTTPVersion) {
	method = p.parseIdent()

	if p.expect(token.Text, token.OpenInterp) {
		url = p.parseValue()
	}

	// Optional HTTP/<version>
	if p.next.Is(token.EOF) {
		return method, url, nil
	}

	if !p.next.Is(token.HTTPVersion) {
		// A third field that isn't a version
		p.advance()
		p.errorf(p.current, "expected HTTP version (e.g. HTTP/1.2), found %q", p.text())

		return method, url, nil
	}

	p.advance()
	start := p.current.Start

	version = &ast.HTTPVersion{
		Range: p.span(),
	}

	if !p.expect(token.Number) {
		// No version number, bad
		return method, url, version
	}

	version = &ast.HTTPVersion{
		Version: ast.NumberLiteral{
			Range: p.span(),
		},
		Range: source.Span{
			File:        p.span().File,
			StartOffset: start,
			EndOffset:   p.span().EndOffset,
		},
	}

	return method, url, version
}

// parseHeader parses a header line into an [ast.Header].
func (p *parser) parseHeader() ast.Header {
	// Block pass means we can assume ident
	name := p.parseIdent()

	p.expect(token.Colon)
	p.advance() // Discard the colon

	value := p.parseValue()

	return ast.Header{
		Value: value,
		Range: p.block.Span,
		Name:  name,
	}
}

// parseBody parses a request body into one of the [ast.Body] nodes.
func (p *parser) parseBody() ast.Body {
	switch p.current.Kind {
	case token.Text, token.OpenInterp:
		// Inline body: a run of literal text and interpolations. The run
		// can open with an interp, e.g. a body of just `{{ payload }}`.
		parts := p.parseTemplateParts()
		if len(parts) == 0 {
			return nil // parseTemplateParts already diagnosed
		}

		return ast.BodyInline{
			Content: ast.Template{
				Parts: parts,
				Range: source.Span{
					File:        p.block.Span.File,
					StartOffset: parts[0].Span().StartOffset,
					EndOffset:   parts[len(parts)-1].Span().EndOffset,
				},
			},
			Range: p.block.Span,
		}
	case token.Error:
		// Nothing, lexer has already reported
		return nil
	default:
		p.errorf(p.current, "parseBody: unexpected token %s", p.current.Kind)

		return nil
	}
}
