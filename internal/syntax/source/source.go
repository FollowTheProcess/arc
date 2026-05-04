// Package source provides data structures and mechanisms for inspecting
// source text, including position, offset and file information.
package source

import (
	"bytes"
	"slices"
	"strconv"
)

// Position is an arbitrary 1-indexed line and column position within
// some source text.
type Position struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

// String returns a string representation of a [Position].
func (p Position) String() string {
	return strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Col)
}

// File represents a single file of .http source text.
//
// Construct with [NewFile]; the zero value is not valid.
type File struct {
	Name        string `json:"name"` // Name of the file, or "-" for stdin.
	content     []byte // Raw file contents.
	lineOffsets []int  // Line offset table, built at construction time.
}

// NewFile constructs and returns a new [File].
//
// The name should be the file name or "-" may be used to represent
// stdin.
//
// Internally it builds a line offset table for fast position
// lookups.
func NewFile(name string, content []byte) *File {
	numLines := bytes.Count(content, []byte{'\n'})
	lineOffsets := make([]int, 1, numLines+1)

	for base := 0; base < len(content); {
		i := bytes.IndexByte(content[base:], '\n')
		if i < 0 {
			break
		}

		next := base + i + 1
		if next < len(content) {
			lineOffsets = append(lineOffsets, next)
		}

		base = next
	}

	return &File{
		Name:        name,
		content:     content,
		lineOffsets: lineOffsets,
	}
}

// PositionAt returns the [Position] in a [File] of a given byte offset.
//
// Offsets outside 0 <= offset <= len(content) are clamped to the nearest end so that
// callers always get a valid 1-indexed [Position].
//
// A negative offset returns the file start, and an offset past EOF returns the position
// one past the last byte (the same value returned for offset == len(content)).
func (f *File) PositionAt(offset int) Position {
	offset = min(max(offset, 0), len(f.content))

	index, found := slices.BinarySearch(f.lineOffsets, offset)
	if !found {
		// slices.BinarySearch returns the insertion point when not found;
		// step back to the line whose start is <= offset.
		index--
	}

	return Position{
		Line: index + 1,
		Col:  offset - f.lineOffsets[index] + 1,
	}
}

// Span represents a span of source text.
//
// StartOffset is inclusive and EndOffset is exclusive, like a Go slice; the
// span covers the bytes [StartOffset, EndOffset) of File. EndOffset must be
// >= StartOffset.
type Span struct {
	File        *File `json:"file,omitempty"` // The file in question.
	StartOffset int   `json:"startOffset"`    // Byte offset of the span start (inclusive).
	EndOffset   int   `json:"endOffset"`      // Byte offset of the span end (exclusive).
}

// String returns a string representation of a [Span].
func (s Span) String() string {
	if s.File == nil {
		return ""
	}

	return s.File.Name + ":" + s.Start().String()
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
//
// contextLines is clamped to a sensible range: negative values are treated
// as zero, and values larger than the file's line count are capped at it
// (effectively returning the whole file).
func (s Span) Snippet(contextLines int) []byte {
	contextLines = min(max(contextLines, 0), len(s.File.lineOffsets))

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
