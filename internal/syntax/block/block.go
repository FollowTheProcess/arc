// Package block implements the block-level pass of arc's .http parser.
//
// A .http file is fundamentally line-oriented at the top level: separators, request
// lines, headers, and directives are one line per construct.
//
// The block classifier walks the source line by line and produces a stream
// of typed [Block] values, each carrying a span into the original source.
//
// Inline content within a block (interpolations, header values, URLs) is
// tokenised by the lex package into token.Token values; this pass only
// decides what each line is and dispatches to the right inline tokeniser.
//
// The classifier carries a small amount of context. Lines are not always
// distinguishable by leading characters alone: "Key: value" is only a
// header within a request, and "@x = ..." is a global directive at file
// scope but a local variable inside a request prelude. Body regions are
// bracketed by synthetic open and close blocks so downstream consumers
// never have to detect body boundaries themselves.
//
// The pass is fault tolerant. A malformed line becomes its own block
// kind with an attached diagnostic and classification continues.
//
// This is the block half of the two-pass model common to markdown
// parsers, a block pass carves the file into typed regions, then an
// inline pass tokenises sub-line content.
package block

import (
	"fmt"

	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/arc/internal/syntax/token"
)

// Block is a typed block of http source.
type Block struct {
	Span    source.Span   // Attached span
	Content []byte        // Block contents
	Tokens  []token.Token // Inline tokenised content, populated for blocks that have it
	Kind    Kind          // The kind of block this is.
}

// String returns a string representation of a [Block].
func (b Block) String() string {
	if b.Span.File == nil {
		return ""
	}

	return fmt.Sprintf("<Block::%s start=%s end=%s>", b.Kind, b.Span.Start(), b.Span.End())
}

// Parse walks the file line by line, recognising regions of content as typed [Block]
// values. When a block is recognised, it's content is dispatched to an inline
// tokeniser for further parsing now we know it's type.
//
// An invalid line returns a block of type [Error], emits a [diagnostic.Diagnostic]
// and moves on, creating a naturally resilient parsing step.
func Parse(file *source.File) ([]Block, []diagnostic.Diagnostic) {
	return nil, nil
}
