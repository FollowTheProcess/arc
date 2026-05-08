// Package block implements the block-level pass of arc's .http parser.
//
// A .http file is fundamentally line-oriented at the top level: separators, request
// lines, headers, and directives are one line per construct.
//
// The block parser walks the source line by line and produces a stream
// of typed [Block] values, each carrying a span into the original source.
//
// Inline content within a block (interpolations, header values, URLs) is
// tokenised by the lex package into token.Token values; this pass only
// decides what each line is and dispatches to the right inline tokeniser.
//
// The parser carries a small amount of context. Lines are not always
// distinguishable by leading characters alone: "Key: value" is only a
// header within a request, and "@x = ..." is a global directive at file
// scope but a local variable inside a request prelude. Body regions are
// bracketed by synthetic open and close blocks so downstream consumers
// never have to detect body boundaries themselves.
//
// The pass is fault tolerant. A malformed line becomes its own block
// kind with an attached diagnostic and parsing continues.
//
// This is the block half of the two-pass model common to markdown
// parsers: the block pass carves the file into typed regions, and an inline
// tokeniser tokenises what's inside each one.
package block

import (
	"bytes"
	"fmt"

	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/lex"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/arc/internal/syntax/token"
)

// Block is a typed block of http source.
type Block struct {
	Span   source.Span   // Attached span
	Tokens []token.Token // Inline tokenised content, populated for blocks that have it
	Kind   Kind          // The kind of block this is.
}

// String returns a string representation of a [Block].
func (b Block) String() string {
	if b.Span.File == nil {
		return ""
	}

	return fmt.Sprintf("<Block::%s start=%s, end=%s>", b.Kind, b.Span.Start(), b.Span.End())
}

// Parse walks the file line by line, recognising regions of content as typed [Block]
// values. When a block is recognised, it's content is dispatched to an inline
// tokeniser for further parsing now we know it's type.
//
// An invalid line returns a block of type [Error], emits a [diagnostic.Diagnostic]
// and moves on, creating a naturally resilient parsing step.
func Parse(file *source.File) ([]Block, []diagnostic.Diagnostic) {
	p := &parser{file: file, state: stateInitial}
	defer p.flush()

	for span := range file.Lines() {
		p.step(span)
	}

	return p.blocks, p.diags
}

// a parser holds the state of block parsing and accumulates parsed
// blocks and diagnostics.
type parser struct {
	file   *source.File            // The src file
	blocks []Block                 // Parsed blocks
	diags  []diagnostic.Diagnostic // Diagnostics accumulated during parsing
	state  state                   // The current parsing state
	prev   Kind                    // Last non-synthetic block kind; needed for URL continuation lookahead
}

// step parses a single span of content.
func (p *parser) step(span source.Span) {
	kind, next := dispatch(span.Content(), p.state, p.prev)
	p.emit(kind, span)
	p.state = next
}

// dispatch decides what kind of block a line is given the current
// context and the previously-emitted block kind. It returns the kind
// for this line and the state the parser should move to afterwards.
func dispatch(line []byte, state state, prev Kind) (Kind, state) {
	if len(line) == 0 {
		switch state {
		case stateRequestHeaders:
			return HeaderBodySeparator, stateRequestBody
		case stateRequestBody:
			// Blank lines inside a body are still the body
			return BodyContent, stateRequestBody
		default:
			return Blank, state
		}
	}

	if bytes.HasPrefix(line, []byte("###")) {
		return Separator, stateRequestPrelude
	}

	// TODO(@FollowTheProcess): All the other stuff... probably

	return Error, state
}

// flush concludes parsing.
func (p *parser) flush() {
	// Add a BodyClose if we're in stateRequestBody
}

// emit appends a block to the accumulator.
func (p *parser) emit(kind Kind, span source.Span) {
	tokens, diagnostics := p.tokenise(kind, span)
	block := Block{
		Span:   span,
		Tokens: tokens,
		Kind:   kind,
	}

	p.blocks = append(p.blocks, block)
	p.diags = append(p.diags, diagnostics...)
}

// tokenise dispatches a block to it's dedicated inline tokeniser.
func (p *parser) tokenise(kind Kind, span source.Span) ([]token.Token, []diagnostic.Diagnostic) {
	switch kind {
	case Separator:
		return lex.Separator(span)
	default:
		return nil, []diagnostic.Diagnostic{
			{
				Message:  fmt.Sprintf("unhandled block kind: %s", kind),
				Span:     span,
				Severity: diagnostic.SeverityError,
			},
		}
	}
}
