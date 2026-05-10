// Package source provides data structures and mechanisms for inspecting
// source text, including position, offset and file information.
package source

import (
	"bytes"
	"iter"
	"slices"
	"strconv"
)

// Position is an arbitrary 1-indexed line and column position within
// some source text.
type Position struct {
	Line int `json:"line,omitempty"`
	Col  int `json:"col,omitempty"`
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

// Len returns the number of bytes in the file's content.
func (f *File) Len() int {
	return len(f.content)
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

// Lines returns an iterator over the lines in a file, yielding each
// as a [Span] covering the line bytes excluding the trailing terminator.
//
// Line splitting matches [bufio.ScanLines], a line is terminated by '\n'
// or '\r\n', and any trailing '\r' is stripped (whether or not it was
// followed by '\n'). A lone '\r' that is not at the end of a line is
// preserved.
//
// The final line is yielded even if it does not have a trailing terminator.
func (f *File) Lines() iter.Seq[Span] {
	return func(yield func(Span) bool) {
		// An empty file has no lines
		if len(f.content) == 0 {
			return
		}

		for i, lineStart := range f.lineOffsets {
			var lineEnd int

			if i+1 < len(f.lineOffsets) {
				// Non-last line: lineOffsets[i+1] always points to the
				// byte right after a '\n'.
				lineEnd = f.lineOffsets[i+1] - 1
			} else {
				lineEnd = len(f.content)
				if lineEnd > lineStart && f.content[lineEnd-1] == '\n' {
					lineEnd--
				}
			}

			// Strip a trailing '\r', whether it was the leading half of
			// a '\r\n' pair or a lone '\r' at EOF. Matches the dropCR
			// step in bufio.ScanLines.
			if lineEnd > lineStart && f.content[lineEnd-1] == '\r' {
				lineEnd--
			}

			span := Span{
				File:        f,
				StartOffset: lineStart,
				EndOffset:   lineEnd,
			}

			if !yield(span) {
				return
			}
		}
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
//
// A zero-width span renders as "name:line:col". A non-zero-width span on a
// single line collapses to "name:line:start-end"; one that crosses lines
// renders as "name:line:col-line:col". The end column is exclusive, matching
// the underlying byte offset semantics.
func (s Span) String() string {
	if s.File == nil {
		return ""
	}

	start := s.Start()
	if s.EndOffset == s.StartOffset {
		return s.File.Name + ":" + start.String()
	}

	end := s.End()
	if end.Line == start.Line {
		return s.File.Name + ":" + start.String() + "-" + strconv.Itoa(end.Col)
	}

	return s.File.Name + ":" + start.String() + "-" + end.String()
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

// Content returns the chunk of raw source text in the span.
//
// Spans with offsets outside 0 <= offset <= len(content) are clamped to the nearest
// end so that callers always get a valid slice of content.
func (s Span) Content() []byte {
	if s.File == nil {
		return nil
	}

	n := len(s.File.content)
	start := min(max(s.StartOffset, 0), n)
	end := min(max(s.EndOffset, start), n)

	return s.File.content[start:end]
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
