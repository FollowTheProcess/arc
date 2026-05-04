package source_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/test"
)

// TestSpan tests the Span position calc logic by inserting start ('[')
// and end (']') markers in src and asserting the positions are correct.
//
// '[]' represents a point span (zero width) at the position of '['.
func TestSpan(t *testing.T) {
	tests := []struct {
		name  string          // Name of the test case
		src   string          // Source text under test, with the span bracketed by [ ]
		start source.Position // Expected position of the span start
		end   source.Position // Expected position of the span end
	}{
		{
			name:  "empty",
			src:   "[]",
			start: source.Position{Line: 1, Col: 1},
			end:   source.Position{Line: 1, Col: 1},
		},
		{
			name:  "point mid line 1",
			src:   "ab[]c\ndef\nghi",
			start: source.Position{Line: 1, Col: 3},
			end:   source.Position{Line: 1, Col: 3},
		},
		{
			name:  "point start of line 2",
			src:   "abc\n[]def\nghi",
			start: source.Position{Line: 2, Col: 1},
			end:   source.Position{Line: 2, Col: 1},
		},
		{
			name:  "point mid line 2",
			src:   "abc\nd[]ef\nghi",
			start: source.Position{Line: 2, Col: 2},
			end:   source.Position{Line: 2, Col: 2},
		},
		{
			name:  "point end of line 2",
			src:   "abc\ndef[]\nghi",
			start: source.Position{Line: 2, Col: 4},
			end:   source.Position{Line: 2, Col: 4},
		},
		{
			name:  "point start of last line",
			src:   "abc\ndef\n[]ghi",
			start: source.Position{Line: 3, Col: 1},
			end:   source.Position{Line: 3, Col: 1},
		},
		{
			name:  "point end of last line",
			src:   "abc\ndef\nghi[]",
			start: source.Position{Line: 3, Col: 4},
			end:   source.Position{Line: 3, Col: 4},
		},
		{
			name:  "point single line",
			src:   "ab[]c",
			start: source.Position{Line: 1, Col: 3},
			end:   source.Position{Line: 1, Col: 3},
		},
		{
			name:  "single line span",
			src:   "abc\nd[ef]\nghi",
			start: source.Position{Line: 2, Col: 2},
			end:   source.Position{Line: 2, Col: 4},
		},
		{
			name:  "two line span",
			src:   "ab[c\nde]f",
			start: source.Position{Line: 1, Col: 3},
			end:   source.Position{Line: 2, Col: 3},
		},
		{
			name:  "three line span",
			src:   "[abc\ndef\nghi]",
			start: source.Position{Line: 1, Col: 1},
			end:   source.Position{Line: 3, Col: 4},
		},
		{
			name:  "span ending at EOF",
			src:   "abc\nde[f]",
			start: source.Position{Line: 2, Col: 3},
			end:   source.Position{Line: 2, Col: 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, span := srcSpan(t, tt.src)

			test.Equal(t, span.Start(), tt.start, test.Context("wrong start of span"))
			test.Equal(t, span.End(), tt.end, test.Context("wrong end of span"))
		})
	}
}

func TestSnippet(t *testing.T) {
	tests := []struct {
		name         string // Name of the test case
		src          string // Source text under test, with the span bracketed
		want         string // Expected snippet
		contextLines int    // Number of context lines to include either side
	}{
		{
			name:         "empty",
			src:          "[]",
			contextLines: 1,
			want:         "",
		},
		{
			name:         "single line span no context",
			src:          "abc\nd[ef]\nghi",
			contextLines: 0,
			want:         "def\n",
		},
		{
			name:         "single line span with context",
			src:          "abc\nd[ef]\nghi",
			contextLines: 1,
			want:         "abc\ndef\nghi",
		},
		{
			name:         "multi line span no context",
			src:          "x\n[abc\ndef]\ny",
			contextLines: 0,
			want:         "abc\ndef\n",
		},
		{
			name:         "multi line span with context",
			src:          "x\n[abc\ndef]\ny",
			contextLines: 1,
			want:         "x\nabc\ndef\ny",
		},
		{
			name:         "context truncated at start of file",
			src:          "[abc]\ndef",
			contextLines: 5,
			want:         "abc\ndef",
		},
		{
			name:         "context truncated at end of file",
			src:          "abc\n[def]",
			contextLines: 5,
			want:         "abc\ndef",
		},
		{
			name:         "target last line no trailing newline",
			src:          "abc\nd[ef]",
			contextLines: 0,
			want:         "def",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, span := srcSpan(t, tt.src)

			got := span.Snippet(tt.contextLines)
			test.Diff(t, string(got), tt.want)
		})
	}
}

func TestPositionString(t *testing.T) {
	tests := []struct {
		want      string
		line, col int
	}{
		{line: 0, col: 0, want: "0:0"},
		{line: 1, col: 1, want: "1:1"},
		{line: 12, col: 37, want: "12:37"},
		{line: 99999, col: 99999, want: "99999:99999"},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			pos := source.Position{Line: tt.line, Col: tt.col}
			got := pos.String()

			test.Equal(t, got, tt.want)
		})
	}
}

func TestSpanString(t *testing.T) {
	tests := []struct {
		name string
		want string
		span source.Span
	}{
		{
			name: "nil file",
			span: source.Span{},
			want: "",
		},
		{
			name: "start of file",
			span: source.Span{
				File:        source.NewFile("test.http", []byte("hello\nthere")),
				StartOffset: 0,
				EndOffset:   0,
			},
			want: "test.http:1:1",
		},
		{
			name: "start of line 2",
			span: source.Span{
				File:        source.NewFile("test.http", []byte("hello\nthere")),
				StartOffset: 6, // 't' in "there"
				EndOffset:   6,
			},
			want: "test.http:2:1",
		},
		{
			name: "mid line 2",
			span: source.Span{
				File:        source.NewFile("test.http", []byte("hello\nthere")),
				StartOffset: 8,
				EndOffset:   8, // Doesn't matter, String() returns start
			},
			want: "test.http:2:3",
		},
		{
			name: "EOF position",
			span: source.Span{
				File:        source.NewFile("test.http", []byte("hello")),
				StartOffset: 5, // one past 'o'
				EndOffset:   5,
			},
			want: "test.http:1:6",
		},
		{
			name: "multi-line span renders start only",
			span: source.Span{
				File:        source.NewFile("test.http", []byte("hello\nthere\nworld")),
				StartOffset: 1,  // 'e' in "hello"
				EndOffset:   14, // 'r' in "world"
			},
			want: "test.http:1:2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.span.String()
			test.Equal(t, got, tt.want)
		})
	}
}

func TestPositionAtClamp(t *testing.T) {
	file := source.NewFile("test.http", []byte("abc\ndef\nghi")) // lineOffsets = [0, 4, 8]

	tests := []struct {
		name   string          // Name of the test case
		want   source.Position // Expected position after clamping
		offset int             // Byte offset to look up
	}{
		{
			name:   "negative clamps to file start",
			offset: -1,
			want:   source.Position{Line: 1, Col: 1},
		},
		{
			name:   "large negative clamps to file start",
			offset: -9999,
			want:   source.Position{Line: 1, Col: 1},
		},
		{
			name:   "in range start",
			offset: 0,
			want:   source.Position{Line: 1, Col: 1},
		},
		{
			name:   "in range last byte",
			offset: 10,
			want:   source.Position{Line: 3, Col: 3},
		},
		{
			name:   "at EOF",
			offset: 11,
			want:   source.Position{Line: 3, Col: 4},
		},
		{
			name:   "past EOF clamps to EOF",
			offset: 12,
			want:   source.Position{Line: 3, Col: 4},
		},
		{
			name:   "far past EOF clamps to EOF",
			offset: 9999,
			want:   source.Position{Line: 3, Col: 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := file.PositionAt(tt.offset)
			test.Equal(t, got, tt.want)
		})
	}
}

// FuzzPositionAt checks that [source.File.PositionAt] doesn't panic for arbitrary
// offset and content and the resulting [source.Position] is always a valid
// 1-indexed position:
//
//   - Line should always increase as we go through the file
//   - Col should reset to 1 after every '\n' (other than the final one)
func FuzzPositionAt(f *testing.F) {
	seeds := []struct {
		content []byte
		offset  int
	}{
		{content: []byte(""), offset: 0},
		{content: []byte("a"), offset: 0},
		{content: []byte("a\nb\nc"), offset: 3},
		{content: []byte("\n\n\n"), offset: 1},
		{content: []byte("abc\ndef\nghi"), offset: -1},
		{content: []byte("abc\ndef\nghi"), offset: 9999},
	}

	for _, s := range seeds {
		f.Add(s.content, s.offset)
	}

	f.Fuzz(func(t *testing.T, content []byte, offset int) {
		file := source.NewFile("fuzz", content)

		pos := file.PositionAt(offset)
		test.True(
			t,
			pos.Line >= 1 && pos.Col >= 1,
			test.Context("not 1-indexed: offset=%d pos=%s len=%d", offset, pos, len(content)),
		)

		var prev source.Position

		for i := range len(content) + 1 {
			p := file.PositionAt(i)

			posIsValid := p.Line >= 1 && p.Col >= 1
			lineAlwaysIncreases := p.Line >= prev.Line
			atNonTrailingNewline := i > 0 && i < len(content) && content[i-1] == '\n'
			colResetsAfterNewline := !atNonTrailingNewline || p.Col == 1

			test.True(
				t,
				posIsValid,
				test.Context("not 1-indexed: offset=%d pos=%s len=%d", offset, p, len(content)),
			)

			test.True(
				t,
				lineAlwaysIncreases,
				test.Context("line went backwards at offset %d: %d -> %d", i, prev, p),
			)

			test.True(
				t,
				colResetsAfterNewline,
				test.Context("col not reset after newline at offset %d: %s", i, p),
			)

			prev = p
		}
	})
}

// FuzzSnippet checks that [source.Span.Snippet] doesn't panic for any
// combination of content, span offsets, and contextLines, and that the
// returned bytes are always a subset of the file content.
func FuzzSnippet(f *testing.F) {
	seeds := []struct {
		content      []byte
		startOffset  int
		endOffset    int
		contextLines int
	}{
		{content: []byte(""), startOffset: 0, endOffset: 0, contextLines: 0},
		{content: []byte("abc"), startOffset: 0, endOffset: 3, contextLines: 0},
		{content: []byte("abc\ndef\nghi"), startOffset: 4, endOffset: 7, contextLines: 1},
		{content: []byte("a"), startOffset: 0, endOffset: 0, contextLines: -1},
		{content: []byte("a\nb"), startOffset: 0, endOffset: 1, contextLines: -100},
		{content: []byte("abc"), startOffset: 0, endOffset: 0, contextLines: 9999},
	}

	for _, s := range seeds {
		f.Add(s.content, s.startOffset, s.endOffset, s.contextLines)
	}

	f.Fuzz(func(t *testing.T, content []byte, startOffset, endOffset, contextLines int) {
		// Span requires EndOffset >= StartOffset; normalise so the fuzzer
		// doesn't waste its time on documented violations.
		if startOffset > endOffset {
			startOffset, endOffset = endOffset, startOffset
		}

		file := source.NewFile("fuzz", content)
		span := source.Span{
			File:        file,
			StartOffset: startOffset,
			EndOffset:   endOffset,
		}

		snippet := span.Snippet(contextLines)

		test.True(
			t,
			bytes.Contains(content, snippet),
			test.Context(
				"snippet is not a substring of content: snippet=%q content=%q contextLines=%d",
				snippet, content, contextLines,
			),
		)
	})
}

func BenchmarkNewFile(b *testing.B) {
	file := filepath.Join("testdata", "bench", "lines.txt")
	data, err := os.ReadFile(file)
	test.Ok(b, err)

	for b.Loop() {
		source.NewFile("bench", data)
	}
}

// srcSpan parses a source string in which the desired span is bracketed by
// '[' and ']'. The brackets are stripped, the line offset table is built, and
// a Span pointing at the bytes that were between the brackets is returned.
//
// '[]' means a zero-width point span.
func srcSpan(t *testing.T, src string) (*source.File, source.Span) {
	t.Helper()

	start := strings.IndexByte(src, '[')
	test.True(t, start >= 0, test.Context("no '[' in src"))

	rel := strings.IndexByte(src[start+1:], ']')
	test.True(t, rel >= 0, test.Context("no ']' after '[' in src"))
	end := start + 1 + rel

	content := src[:start] + src[start+1:end] + src[end+1:]
	file := source.NewFile("test.http", []byte(content))

	return file, source.Span{
		File:        file,
		StartOffset: start,
		EndOffset:   end - 1, // -1: '[' has been stripped, shifting ']' left by one
	}
}
