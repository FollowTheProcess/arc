// Package assembler implements the second stage of arc's .http parsing pipeline.
//
// It consumes a stream of [block.Block] values, and emits the parsed file as
// an abstract syntax tree node ([ast.File]).
package assembler

import (
	"fmt"
	"mime"
	"strings"

	"go.followtheprocess.codes/arc/internal/syntax/ast"
	"go.followtheprocess.codes/arc/internal/syntax/block"
	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/source"
)

// Assemble consumes a stream of [block.Block] values and emits a parsed
// [ast.File] node as well as any diagnostics.
func Assemble(blocks []block.Block) (ast.File, []diagnostic.Diagnostic) {
	if len(blocks) == 0 {
		// An empty .http file parses to zero blocks, zero blocks
		// means zero file I guess!
		return ast.File{}, nil
	}

	a := &assembler{
		blocks: blocks,
		file: ast.File{Range: source.Span{
			File:        blocks[0].Span.File,
			StartOffset: blocks[0].Span.StartOffset,
			EndOffset:   blocks[len(blocks)-1].Span.EndOffset,
		}},
	}

	for a.pos < len(a.blocks) {
		before := a.pos
		a.step()

		if a.pos == before {
			// This is useful during development to guarantee we always make
			// progress
			a.error("internal: assembler made no progress", a.blocks[a.pos])
			a.advance()
		}
	}

	return a.file, a.diagnostics
}

// assembler holds the parsing state and implements the parsing methods.
type assembler struct {
	file        ast.File                // Current file we're building
	blocks      []block.Block           // Incoming block stream to parse
	diagnostics []diagnostic.Diagnostic // Accumulator for diagnostics
	pos         int                     // Current position in the stream (index into blocks)
}

// current returns the current block.
func (a *assembler) current() block.Block {
	return a.blocks[a.pos]
}

// advance advances the assembler in the block stream.
func (a *assembler) advance() {
	a.pos++
}

// step consumes a block from the stream and does some parsing work.
func (a *assembler) step() {
	switch current := a.current(); current.Kind {
	case block.Comment:
		a.parseComment(current)
	case block.Directive:
		a.parseDirective(current)
	case block.Separator:
		a.assembleRequest()
	case block.Error:
		// make progress, but don't report. The block parser has already
		// emitted a diagnostic for this
		a.advance()
	default:
		a.error("unhandled block type", current)
		a.advance() // make progress
	}
}

// error appends an error level diagnostic to the assembler.
func (a *assembler) error(msg string, b block.Block, options ...diagnostic.Option) {
	diag := diagnostic.Error(msg, b.Span, options...)
	a.diagnostics = append(a.diagnostics, diag)
}

// parseDirective parses a directive block into an [ast.Directive].
func (a *assembler) parseDirective(b block.Block) {
	p := newParser(b)
	a.file.Statements = append(a.file.Statements, p.parseDirective())
	a.diagnostics = append(a.diagnostics, p.diagnostics...)
	a.advance()
}

// parseComment parses a comment block into an [ast.Comment].
//
// Comments are parsed into ast nodes so that comments above requests may be
// used as their "docstring". Similar to how doc comments are attached
// to ast nodes in Go.
func (a *assembler) parseComment(b block.Block) {
	// A comment block is not tokenised, it's content is just
	// the entire block span, so we don't need a parser to emit
	// the ast node.
	comment := ast.Comment{
		Range: b.Span,
	}
	a.file.Statements = append(a.file.Statements, comment)
	a.advance()
}

func (a *assembler) assembleRequest() {
	end := a.findRequestEnd()
	req, diags := assembleRequest(a.blocks[a.pos:end])
	a.file.Statements = append(a.file.Statements, req)
	a.diagnostics = append(a.diagnostics, diags...)

	// Skip to the end of the request
	a.pos = end
}

// findRequestEnd returns the index one past the last block belonging to the
// request that starts at blocks[start]. A request runs until the next
// Separator or the end of the stream.
func (a *assembler) findRequestEnd() int {
	for i := a.pos + 1; i < len(a.blocks); i++ {
		if a.blocks[i].Kind == block.Separator {
			return i
		}
	}

	return len(a.blocks)
}

// assembleRequest builds a single [ast.Request] from the continuous run
// of blocks that make it up. blocks[0] is the Separator.
func assembleRequest(blocks []block.Block) (ast.Request, []diagnostic.Diagnostic) {
	req := ast.Request{Range: source.Span{
		File:        blocks[0].Span.File,
		StartOffset: blocks[0].Span.StartOffset,
		EndOffset:   blocks[len(blocks)-1].Span.EndOffset,
	}}

	var diags []diagnostic.Diagnostic

	for _, b := range blocks {
		p := newParser(b)

		switch b.Kind {
		case block.Separator:
			// Only meaningful token is the name (ident)
			req.Name = p.parseSeparator()
		case block.RequestLine:
			req.Method, req.URL, req.HTTPVersion = p.parseRequestLine()
		case block.Header:
			req.Headers = append(req.Headers, p.parseHeader())
		case block.Body:
			req.Body = p.parseBody(contentType(req.Headers))
		case block.Blank, block.Comment:
			// Nothing
		default:
			d := diagnostic.Error(fmt.Sprintf("unexpected block type: %s", b.Kind), b.Span)
			diags = append(diags, d)
		}

		diags = append(diags, p.diagnostics...)
	}

	return req, diags
}

// contentType returns the value of the Content-Type header (if present)
// or "" if there is no Content-Type header or it's value is dynamic.
func contentType(headers []ast.Header) (string, map[string]string) {
	for _, header := range headers {
		if !strings.EqualFold(header.Name.Range.Text(), "Content-Type") {
			continue
		}

		literal, ok := header.Value.(ast.TextLiteral)
		if !ok {
			return "", nil // Dynamic value, we can't know this at parse time
		}

		mediaType, params, err := mime.ParseMediaType(literal.Value)
		if err != nil {
			return "", nil
		}

		return mediaType, params
	}

	return "", nil
}
