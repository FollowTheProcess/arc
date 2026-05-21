// Package block implements the block-level pass of arc's .http parser.
//
// A .http file is fundamentally line-oriented at the top level: separators, request
// lines, headers, and directives are one line per construct.
//
// The block parser walks the source line by line and produces a stream
// of typed [Block] values, each carrying a span into the original source.
//
// Inline content within a block (interpolations, header values, URLs) is
// tokenised by the lex package into token.Token values. This pass only
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
	p := newParser(file)

	for span := range file.Lines() {
		p.step(span)
	}

	p.flush()

	return p.blocks, p.diags
}

// a parser holds the state of block parsing and accumulates parsed
// blocks and diagnostics.
type parser struct {
	file                *source.File            // The src file
	blocks              []Block                 // Parsed blocks
	diags               []diagnostic.Diagnostic // Diagnostics accumulated during parsing
	prev                Kind                    // Last non-synthetic block kind, needed for URL continuation lookahead
	scriptStart         int                     // Start offset of an open multi-line script, valid only when state == stateScript
	bodyStart           int                     // Start offset of a request body, valid only when state == stateRequestBody
	lastNonBlankBodyEnd int                     // Offset of the last non-empty byte in a request body
	state               state                   // The current parsing state
	prevState           state                   // State to return to when an open multi-line script closes
}

// newParser creates and returns a new [parser].
func newParser(file *source.File) *parser {
	return &parser{
		file:  file,
		state: stateInitial,
	}
}

// step parses a single span of content.
func (p *parser) step(span source.Span) {
	// Inside an open multi-line script, all lines belong to the script
	// until it's closed with '%}'. Once we see one, we batch up everything
	// into a single script span and hand it to the lexer.
	if p.state == stateScript {
		if bytes.Contains(span.Content(), []byte("%}")) {
			p.emit(Script, source.Span{
				File:        p.file,
				StartOffset: p.scriptStart,
				EndOffset:   span.EndOffset,
			})
			p.state = p.prevState
		}

		return
	}

	if p.state != stateRequestBody {
		kind, next := dispatch(span.Content(), p.state, p.prev)

		// A '< {%' (or '> {%') opener with no matching '%}' on the same line
		// starts a multi-line script.
		if kind == Script && isMultilineScriptOpen(span.Content()) {
			p.scriptStart = span.StartOffset
			p.prevState = p.state
			p.state = stateScript

			return
		}

		p.emit(kind, span)

		p.state = next

		return
	}

	kind, next := dispatch(span.Content(), p.state, p.prev)
	if kind == Body {
		// Still inside a body
		if len(bytes.TrimSpace(span.Content())) > 0 {
			if p.bodyStart == 0 {
				p.bodyStart = span.StartOffset
			}

			p.lastNonBlankBodyEnd = span.EndOffset
		}

		return
	}

	// First time seeing a non-body thing, time to pinch off the body
	if p.lastNonBlankBodyEnd > p.bodyStart {
		p.emit(Body, source.Span{
			File:        p.file,
			StartOffset: p.bodyStart,
			EndOffset:   p.lastNonBlankBodyEnd,
		})
	}

	// Clean up
	p.bodyStart, p.lastNonBlankBodyEnd = 0, 0

	p.emit(kind, span)
	p.state = next
}

// flush concludes parsing, it inserts any necessary closing blocks
// before the parser returns so blocks are not left unterminated.
func (p *parser) flush() {
	switch p.state {
	case stateRequestBody:
		// Started parsing the body but hit EOF before we were done, needs
		// a trigger to emit the span
		end := p.lastNonBlankBodyEnd
		if end > p.bodyStart {
			p.emit(Body, source.Span{
				File:        p.file,
				StartOffset: p.bodyStart,
				EndOffset:   end,
			})
		}
	case stateScript:
		span := source.Span{
			File:        p.file,
			StartOffset: p.scriptStart,
			EndOffset:   p.file.Len(),
		}
		// Same thing with a script, was opened with a '{%', but hit EOF
		// before seeing '%}', this is the trigger to emit what we have
		p.emit(Script, span)
		p.error("unterminated script, expected '%}' before EOF", span)

	default:
		// Nothing
	}
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

// error appends an error level diagnostic to the parser.
func (p *parser) error(msg string, span source.Span, options ...diagnostic.Option) {
	diag := diagnostic.Error(msg, span, options...)
	p.diags = append(p.diags, diag)
}

// tokenise dispatches a block to it's dedicated inline tokeniser.
func (p *parser) tokenise(kind Kind, span source.Span) ([]token.Token, []diagnostic.Diagnostic) {
	switch kind {
	case Blank, Comment:
		// Markers / lines with no inline content to tokenise.
		return nil, nil
	case Separator:
		return lex.Separator(span)
	case RequestLine:
		return lex.RequestLine(span)
	case Header:
		return lex.Header(span)
	case Directive:
		return lex.Directive(span)
	case Script:
		return lex.Script(span)
	case Body:
		return lex.Body(span)
	case ResponseRedirect:
		return lex.ResponseRedirect(span)
	case ResponseReference:
		return lex.ResponseReference(span)
	case Error:
		// A lexer error token
		p.error(fmt.Sprintf("unexpected line in this context: %s", p.state), span)

		// p.error appends to p.diags, no need to return anything
		return nil, nil
	default:
		// A missing implementation
		p.error(fmt.Sprintf("unhandled block kind: %s", kind), span)

		return nil, nil
	}
}

// dispatch decides what kind of block a line is given the current
// context and the previously-emitted block kind. It returns the kind
// for this line and the state the parser should move to afterwards.
func dispatch(line []byte, state state, prev Kind) (Kind, state) {
	// In a request body, only ###/<>/>>/> at column 0 break the body.
	// '>>' must be checked before '>' so the redirect doesn't get
	// swallowed by the response-script case.
	if state == stateRequestBody {
		switch {
		case lineStartsWith(line, "###"):
			return Separator, stateRequestPrelude
		case lineStartsWith(line, "<>"):
			return ResponseReference, stateRequestPostBody
		case lineStartsWith(line, ">>"), lineStartsWith(line, ">>!"):
			return ResponseRedirect, stateRequestPostBody
		case lineStartsWith(line, ">"):
			return Script, stateRequestPostBody
		default:
			return Body, stateRequestBody
		}
	}

	// Blank lines need to be treated specially depending on the current state.
	if len(line) == 0 {
		if state == stateRequestHeaders {
			// A blank after a run of headers marks the transition to the body.
			// No marker block is needed, the state transition is enough.
			return Blank, stateRequestBody
		}

		// Otherwise normal blank
		return Blank, state
	}

	if lineStartsWith(line, "###") {
		return Separator, stateRequestPrelude
	}

	if isDirective(line, state) {
		return Directive, state
	}

	switch {
	case lineStartsWith(line, "#"), lineStartsWith(line, "//"):
		return Comment, state // State unchanged by a comment
	case isMethodPrefix(line):
		return RequestLine, stateRequestHeaders
	case state == stateRequestHeaders:
		return Header, state
	case state == stateRequestPrelude && lineStartsWith(line, "<"):
		return Script, state
	default:
		return Error, state
	}
}

// isDirective reports whether line is a directive in the current state.
//
// The bare form ('@x = y') is the global directive, valid at file scope
// or in a request prelude.
//
// The comment-disguised form ('# @x = y' or '// @x = y') is the
// JetBrains/REST Client convention for request-scoped variables; it is
// only meaningful inside a request. At file scope a '#' or '//' line is
// a comment regardless of what follows the marker.
func isDirective(line []byte, state state) bool {
	bare := lineStartsWith(line, "@")
	disguised := lineStartsWith(line, "# @") || lineStartsWith(line, "// @")

	switch state {
	case stateInitial:
		return bare
	case stateRequestPrelude:
		return bare || disguised
	case stateRequestHeaders:
		return disguised
	default:
		return false
	}
}

// lineStartsWith reports whether line begins with prefix.
//
// Exists for pure readability as lineStartsWith(line, "###") is nicer
// than the bytes.HasPrefix equivalent, particularly in switch cases
// where multiple "startsWith" conditions are present in a branch.
//
// The compiler inlines this anyway and []byte(prefix) will not allocate.
func lineStartsWith(line []byte, prefix string) bool {
	return bytes.HasPrefix(line, []byte(prefix))
}

// isMethodPrefix reports whether the line begins like a request line: a
// non-empty run of uppercase ASCII letters followed by either a line-space
// character or the end of the line.
//
// This is a recognition check, not a validation: bare prefixes like "GETS"
// or unknown idents like "FROBNICATE" return true here so the assembler can
// later report a precise "unknown HTTP method" diagnostic. Only a complete
// uppercase ident at the start of the line counts, "/path" or "get" do not.
func isMethodPrefix(line []byte) bool {
	n := 0
	for n < len(line) && line[n] >= 'A' && line[n] <= 'Z' {
		n++
	}

	if n == 0 {
		return false
	}

	if n == len(line) {
		return true
	}

	return line[n] == ' ' || line[n] == '\t'
}

// isMultilineScriptOpen reports whether a script-opener line begins a
// multi-line '{% ...' block that continues onto subsequent lines before
// closing with '%}'.
//
// The path form ('< path/to/script.js') and the inline form with the close
// on the same line ('< {% ... %}') are both single-line and classify as a
// single [Script] block.
//
// Only an open '{%' with no matching '%}' returns true.
func isMultilineScriptOpen(line []byte) bool {
	_, after, found := bytes.Cut(line, []byte("{%"))
	if !found {
		return false
	}

	return !bytes.Contains(after, []byte("%}"))
}
