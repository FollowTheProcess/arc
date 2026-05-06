package source_test

import (
	"bufio"
	"bytes"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
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
		{
			name:         "CRLF is preserved in snippet bytes",
			src:          "[key]\r\nval",
			contextLines: 0,
			want:         "key\r\n",
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

func TestFileLines(t *testing.T) {
	tests := []struct {
		name string   // Name of the test case
		src  string   // File content
		want [][2]int // Each entry is the (start, end) byte offsets of a yielded line span
	}{
		{
			name: "empty",
			src:  "",
			want: nil,
		},
		{
			name: "single line no trailing newline",
			src:  "hello",
			want: [][2]int{{0, 5}},
		},
		{
			name: "single line trailing newline",
			src:  "hello\n",
			want: [][2]int{{0, 5}},
		},
		{
			name: "just a newline",
			src:  "\n",
			want: [][2]int{{0, 0}},
		},
		{
			name: "two lines no trailing newline",
			src:  "foo\nbar",
			want: [][2]int{{0, 3}, {4, 7}},
		},
		{
			name: "two lines trailing newline",
			src:  "foo\nbar\n",
			want: [][2]int{{0, 3}, {4, 7}},
		},
		{
			name: "three lines",
			src:  "one\ntwo\nthree",
			want: [][2]int{{0, 3}, {4, 7}, {8, 13}},
		},
		{
			name: "blank line in the middle",
			src:  "name\n\nvalue",
			want: [][2]int{{0, 4}, {5, 5}, {6, 11}},
		},
		{
			name: "leading blank line",
			src:  "\nstart",
			want: [][2]int{{0, 0}, {1, 6}},
		},
		{
			name: "trailing blank line before EOF",
			src:  "end\n\n",
			want: [][2]int{{0, 3}, {4, 4}},
		},
		{
			name: "only newlines",
			src:  "\n\n\n",
			want: [][2]int{{0, 0}, {1, 1}, {2, 2}},
		},
		{
			name: "CRLF treated as a single terminator",
			src:  "key\r\nval",
			want: [][2]int{{0, 3}, {5, 8}},
		},
		{
			name: "CRLF with trailing CRLF",
			src:  "key\r\nval\r\n",
			want: [][2]int{{0, 3}, {5, 8}},
		},
		{
			name: "mixed CRLF and LF",
			src:  "a\r\nb\nc",
			want: [][2]int{{0, 1}, {3, 4}, {5, 6}},
		},
		{
			name: "empty CRLF line",
			src:  "a\r\n\r\nb",
			want: [][2]int{{0, 1}, {3, 3}, {5, 6}},
		},
		{
			name: "lone CR mid-line is not a line break",
			src:  "a\rb",
			want: [][2]int{{0, 3}},
		},
		{
			name: "lone trailing CR is stripped at EOF",
			src:  "abc\r",
			want: [][2]int{{0, 3}},
		},
		{
			name: "multibyte runes use byte offsets",
			src:  "héllo\nwörld",
			want: [][2]int{{0, 6}, {7, 13}}, // h(1)+é(2)+llo(3); w(1)+ö(2)+rld(3)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := source.NewFile(tt.name, []byte(tt.src))

			var got [][2]int

			for span := range file.Lines() {
				test.Equal(
					t,
					span.File,
					file,
					test.Context("span at offset %d references the wrong file", span.StartOffset),
				)
				got = append(got, [2]int{span.StartOffset, span.EndOffset})
			}

			test.EqualFunc(t, got, tt.want, slices.Equal)
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

// FuzzLines checks that [source.File.Lines] doesn't panic on arbitrary input,
// and that the yielded spans are valid (in range, ordered etc.) and that the
// line content precisely matches [bufio.ScanLines] as a reference implementation.
func FuzzLines(f *testing.F) {
	seeds := [][]byte{
		nil,
		[]byte(""),
		[]byte("\n"),
		[]byte("a"),
		[]byte("a\nb"),
		[]byte("a\nb\n"),
		[]byte("\n\n\n"),
		[]byte("a\r\nb"),
		[]byte("a\r\nb\r\n"),
		[]byte("a\rb"),
		[]byte("abc\r"),
		[]byte("a\n\r\nb"),
		[]byte("héllo\nwörld"),
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, content []byte) {
		file := source.NewFile("fuzz", content)
		spans := slices.Collect(file.Lines())

		// Each span must live inside content and start
		// at or after the previous span's end.
		prevEnd := 0

		for i, span := range spans {
			test.Equal(t, span.File, file, test.Context("span %d points at the wrong file", i))

			valid := span.StartOffset >= prevEnd && // Must start at or after previous end
				span.EndOffset >= span.StartOffset && // End must be >= Start
				span.EndOffset <= len(content) // End must be within content

			test.True(
				t,
				valid,
				test.Context(
					"span %d offsets out of order or out of bounds: prevEnd=%d span={%d,%d} len=%d",
					i, prevEnd, span.StartOffset, span.EndOffset, len(content),
				),
			)

			prevEnd = span.EndOffset
		}

		// The yielded line bytes must match bufio.ScanLines exactly.
		got := make([]string, 0, len(spans))
		for _, span := range spans {
			got = append(got, string(content[span.StartOffset:span.EndOffset]))
		}

		scanner := bufio.NewScanner(bytes.NewReader(content))
		// Buffer big enough that fuzz inputs never blow the default MaxScanTokenSize cap.
		scanner.Buffer(make([]byte, 0, 1024), max(len(content)+1, 1024))

		var want []string
		for scanner.Scan() {
			want = append(want, scanner.Text())
		}

		test.Ok(t, scanner.Err())

		test.EqualFunc(
			t,
			got,
			want,
			slices.Equal,
			test.Context("Lines diverged from bufio.ScanLines for content %q: got=%q want=%q", content, got, want),
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

func BenchmarkLines(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "bench", "lines.txt"))
	test.Ok(b, err)

	file := source.NewFile("bench", data)

	var sink int

	for b.Loop() {
		for span := range file.Lines() {
			sink += span.EndOffset - span.StartOffset
		}
	}

	_ = sink
}

func BenchmarkPositionAt(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "bench", "lines.txt"))
	test.Ok(b, err)

	file := source.NewFile("bench", data)
	n := len(data)

	cases := []struct {
		name   string
		offset int
	}{
		{name: "start", offset: 0},
		{name: "middle", offset: n / 2},
		{name: "end", offset: n},
		{name: "random", offset: rand.IntN(n)},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			var sink source.Position
			for b.Loop() {
				sink = file.PositionAt(c.offset)
			}

			_ = sink
		})
	}
}

func BenchmarkSnippet(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "bench", "lines.txt"))
	test.Ok(b, err)

	file := source.NewFile("bench", data)
	mid := len(data) / 2
	span := source.Span{File: file, StartOffset: mid, EndOffset: mid + 10}

	var sink []byte
	for b.Loop() {
		sink = span.Snippet(2)
	}

	_ = sink
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
