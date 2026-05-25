// Package assembler implements the second stage of arc's .http parsing pipeline.
//
// It consumes a stream of [block.Block] values, and emits the parsed file as
// an abstract syntax tree node ([ast.File]).
package assembler

import (
	"go.followtheprocess.codes/arc/internal/syntax/ast"
	"go.followtheprocess.codes/arc/internal/syntax/block"
	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
)

// Assemble consumes a stream of [block.Block] values and emits a parsed
// [ast.File] node as well as any diagnostics.
func Assemble(blocks []block.Block) (ast.File, []diagnostic.Diagnostic) {
	a := &assembler{
		blocks: blocks,
		file:   ast.File{},
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

// peek returns the next block in the list, but does not advance
// the assembler.
//
// Repeated calls to peek return the same block over and over again.
//
// If we're at the end of the stream, it returns a zero block.
func (a *assembler) peek() (block.Block, bool) {
	if next := a.pos + 1; next < len(a.blocks) {
		return a.blocks[next], true
	}

	return block.Block{}, false
}

// advance advances the assembler in the block stream.
func (a *assembler) advance() {
	// TODO: Not sure if we have to check this is in bounds here?
	// or rely on the top level check in Assemble
	a.pos++
}

// step consumes a block from the stream and does some parsing work.
func (a *assembler) step() {
	switch current := a.current(); current.Kind {
	case block.Directive:
		a.parseDirective(current)
	case block.Error:
		a.error("error block", current)
		a.advance() // make progress
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
