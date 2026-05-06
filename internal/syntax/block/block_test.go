package block_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/block"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
)

var update = flag.Bool("update", false, "Update txtar golden files")

func TestBlockClassifierValid(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	pattern := filepath.Join("testdata", "valid", "*.txtar")
	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	test.True(t, len(files) > 0, test.Context("no test files matching pattern %s", pattern))

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			archive, err := txtar.ParseFile(file)
			test.Ok(t, err)

			src, ok := archive.Read("src.http")
			test.True(t, ok, test.Context("archive %s missing src.http", file))

			want, ok := archive.Read("want.txt")
			test.True(t, ok, test.Context("archive %s missing want.txt", file))

			srcFile := source.NewFile("src.http", []byte(src))

			blocks, diagnostics := block.Parse(srcFile)

			// Valid files should have no diagnostics
			test.Equal(t, len(diagnostics), 0, test.Context("unexpected diagnostics: %v", diagnostics))

			buf := &strings.Builder{}
			for _, block := range blocks {
				buf.WriteString(block.String())
				buf.WriteByte('\t')
				buf.WriteByte('"')
				buf.Write(block.Span.Content())
				buf.WriteByte('"')
				buf.WriteByte('\n')
			}

			got := buf.String()

			if *update {
				test.Ok(t, archive.Write("want.txt", got))
				test.Ok(t, txtar.DumpFile(file, archive))
			}

			test.Diff(t, got, want)
		})
	}
}

func TestBlockString(t *testing.T) {
	// 1: "### login\n"  bytes  0..9   ('\n' at 9)
	// 2: "GET /users\n" bytes 10..20  ('\n' at 20)
	// 3: "Host: x\n"    bytes 21..28  ('\n' at 28)
	// 4: "\n"            byte  29
	// 5: "body\n"       bytes 30..34  ('\n' at 34)
	file := source.NewFile("test.http", []byte("### login\nGET /users\nHost: x\n\nbody\n"))

	tests := []struct {
		name  string      // Name of the test case
		want  string      // Expected String() value
		block block.Block // Block under test
	}{
		{
			name: "separator",
			block: block.Block{
				Kind: block.Separator,
				Span: source.Span{File: file, StartOffset: 0, EndOffset: 9},
			},
			want: "<Block::Separator start=1:1 end=1:10>",
		},
		{
			name: "request line",
			block: block.Block{
				Kind: block.RequestLine,
				Span: source.Span{File: file, StartOffset: 10, EndOffset: 20},
			},
			want: "<Block::RequestLine start=2:1 end=2:11>",
		},
		{
			name: "header",
			block: block.Block{
				Kind: block.Header,
				Span: source.Span{File: file, StartOffset: 21, EndOffset: 28},
			},
			want: "<Block::Header start=3:1 end=3:8>",
		},
		{
			name: "body open",
			block: block.Block{
				Kind: block.BodyOpen,
				Span: source.Span{File: file, StartOffset: 30, EndOffset: 30},
			},
			want: "<Block::BodyOpen start=5:1 end=5:1>",
		},
		{
			name: "body content",
			block: block.Block{
				Kind: block.BodyContent,
				Span: source.Span{File: file, StartOffset: 30, EndOffset: 34},
			},
			want: "<Block::BodyContent start=5:1 end=5:5>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.block.String()
			test.Equal(t, got, tt.want)
		})
	}
}
