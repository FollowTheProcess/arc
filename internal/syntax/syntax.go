// Package syntax provides syntax level primitives such as source
// positions, spans and diagnostics.
package syntax

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
)

// Severity is a severity of a diagnostic.
type Severity int

// Severity values, in increasing order of urgency.
//
//go:generate stringer -type Severity -linecomment
const (
	SeverityInvalid Severity = iota // invalid
	SeverityWarning                 // warning
	SeverityError                   // error
)

// Diagnostic is a single source level diagnostic.
type Diagnostic struct {
	Message  string   // Message text
	Span     Span     // The span of source text for which this diagnostic applies.
	Severity Severity // Severity is the severity of the diagnostic.
}

// String returns a string representation of a [Diagnostic].
func (d Diagnostic) String() string {
	if d.Severity == SeverityInvalid || d.Message == "" || d.Span.File == nil {
		return ""
	}

	return fmt.Sprintf("[%s] %s: %s", d.Severity, d.Span, d.Message)
}

// Position is a 1-indexed line and column position within a [SourceFile].
type Position struct {
	Line int
	Col  int
}

// String returns a string representation of a [Position].
func (p Position) String() string {
	return strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Col)
}

// SourceFile represents a single file of .http source text.
//
// Construct with [NewSourceFile]; the zero value is not valid.
type SourceFile struct {
	name        string // Name of the file, or "-" for stdin.
	content     []byte // Raw file contents.
	lineOffsets []int  // Line offset table, built at construction time.
}

// NewSourceFile constructs a [SourceFile] and builds its line offset table.
//
// The first entry of the table is always 0 (line 1 starts at byte 0). For
// every '\n' that is not the final byte of content, the offset of the byte
// following the '\n' is appended. A trailing '\n' does not introduce an
// extra empty line.
func NewSourceFile(name string, content []byte) *SourceFile {
	numLines := bytes.Count(content, []byte("\n"))
	lineOffsets := make([]int, 1, numLines+1)
	lineOffsets[0] = 0

	s := &SourceFile{
		name:        name,
		content:     content,
		lineOffsets: lineOffsets,
	}

	for i, b := range content {
		nextLineStart := i + 1
		if b == '\n' && nextLineStart < len(content) {
			s.lineOffsets = append(s.lineOffsets, nextLineStart)
		}
	}

	return s
}

// Name returns the name of the source file, or "-" for stdin.
func (s *SourceFile) Name() string { return s.name }

// PositionAt returns the [Position] in a [SourceFile] of a given byte offset.
//
// offset must be 0 <= offset <= len(content). If not, PositionAt panics.
func (s *SourceFile) PositionAt(offset int) Position {
	if offset < 0 || offset > len(s.content) {
		panic(fmt.Sprintf(
			"syntax: PositionAt offset %d out of range [0, %d] for file %q",
			offset, len(s.content), s.name,
		))
	}

	index, found := slices.BinarySearch(s.lineOffsets, offset)
	if !found {
		// slices.BinarySearch returns the insertion point when not found;
		// step back to the line whose start is <= offset.
		index--
	}

	return Position{
		Line: index + 1,
		Col:  offset - s.lineOffsets[index] + 1,
	}
}

// Span represents a span of source text.
//
// StartOffset is inclusive and EndOffset is exclusive, like a Go slice; the
// span covers the bytes [StartOffset, EndOffset) of File. EndOffset must be
// >= StartOffset.
type Span struct {
	File        *SourceFile // The file in question.
	StartOffset int         // Byte offset of the span start (inclusive).
	EndOffset   int         // Byte offset of the span end (exclusive).
}

// String returns a string representation of a [Span].
func (s Span) String() string {
	if s.File == nil {
		return ""
	}

	return s.File.name + ":" + s.Start().String()
}

// Start returns the line and column of the span start.
func (s Span) Start() Position {
	return s.File.PositionAt(s.StartOffset)
}

// End returns the line and column of the span end. The end is exclusive
// (one past the last byte of the span).
func (s Span) End() Position {
	return s.File.PositionAt(s.EndOffset)
}

// Snippet returns the source bytes covering the span plus contextLines of
// surrounding context on each side. The returned bytes include any line
// terminators between the lines.
func (s Span) Snippet(contextLines int) []byte {
	startLine := s.Start().Line
	endLine := s.End().Line

	firstIndex := max(0, startLine-1-contextLines)
	lastIndex := min(endLine-1+contextLines, len(s.File.lineOffsets)-1)

	startOffset := s.File.lineOffsets[firstIndex]

	// The last context line ends either at the start of the next line in the
	// table, or at end-of-file if there is no next line.
	endOffset := len(s.File.content)
	if lastIndex+1 < len(s.File.lineOffsets) {
		endOffset = s.File.lineOffsets[lastIndex+1]
	}

	return s.File.content[startOffset:endOffset]
}
