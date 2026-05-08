package block_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/block"
	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
)

var update = flag.Bool("update", false, "Update txtar golden files")

func TestBlockClassifier(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	t.Run("valid", func(t *testing.T) {
		pattern := filepath.Join("testdata", "valid", "*.txtar")
		files, err := filepath.Glob(pattern)
		test.Ok(t, err)

		test.True(t, len(files) > 0, test.Context("no test files matching pattern %s", pattern))

		for _, file := range files {
			name := filepath.Base(file)
			t.Run(name, func(t *testing.T) {
				want, err := txtar.ParseFile(file)
				test.Ok(t, err)

				src, ok := want.Read("src.http")
				test.True(t, ok, test.Context("archive %s missing src.http", file))

				test.True(t, want.Has("want.txt"), test.Context("archive %q missing want.txt", file))
				test.True(t, want.Has("diagnostics.txt"), test.Context("archive %q missing diagnostics.txt", file))

				srcFile := source.NewFile("src.http", []byte(src))

				blocks, diagnostics := block.Parse(srcFile)

				got, err := txtar.New(
					txtar.WithFile("src.http", src),
					txtar.WithFile("want.txt", formatBlocks(srcFile, blocks)),
					txtar.WithFile("diagnostics.txt", formatDiagnostics(diagnostics)),
				)

				test.Ok(t, err)

				if *update {
					test.Ok(t, txtar.DumpFile(file, got))

					return
				}

				test.Diff(t, got.String(), want.String())
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		pattern := filepath.Join("testdata", "invalid", "*.txtar")
		files, err := filepath.Glob(pattern)
		test.Ok(t, err)

		test.True(t, len(files) > 0, test.Context("no test files matching pattern %s", pattern))

		for _, file := range files {
			name := filepath.Base(file)
			t.Run(name, func(t *testing.T) {
				want, err := txtar.ParseFile(file)
				test.Ok(t, err)

				src, ok := want.Read("src.http")
				test.True(t, ok, test.Context("archive %s missing src.http", file))

				test.True(t, want.Has("want.txt"), test.Context("archive %q missing want.txt", file))
				test.True(t, want.Has("diagnostics.txt"), test.Context("archive %q missing diagnostics.txt", file))

				srcFile := source.NewFile("src.http", []byte(src))

				blocks, diagnostics := block.Parse(srcFile)

				got, err := txtar.New(
					txtar.WithFile("src.http", src),
					txtar.WithFile("want.txt", formatBlocks(srcFile, blocks)),
					txtar.WithFile("diagnostics.txt", formatDiagnostics(diagnostics)),
				)

				test.Ok(t, err)

				if *update {
					test.Ok(t, txtar.DumpFile(file, got))

					return
				}

				test.Diff(t, got.String(), want.String())
			})
		}
	})
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
			want: "<Block::Separator start=1:1, end=1:10>",
		},
		{
			name: "request line",
			block: block.Block{
				Kind: block.RequestLine,
				Span: source.Span{File: file, StartOffset: 10, EndOffset: 20},
			},
			want: "<Block::RequestLine start=2:1, end=2:11>",
		},
		{
			name: "header",
			block: block.Block{
				Kind: block.Header,
				Span: source.Span{File: file, StartOffset: 21, EndOffset: 28},
			},
			want: "<Block::Header start=3:1, end=3:8>",
		},
		{
			name: "body open",
			block: block.Block{
				Kind: block.BodyOpen,
				Span: source.Span{File: file, StartOffset: 30, EndOffset: 30},
			},
			want: "<Block::BodyOpen start=5:1, end=5:1>",
		},
		{
			name: "body content",
			block: block.Block{
				Kind: block.BodyContent,
				Span: source.Span{File: file, StartOffset: 30, EndOffset: 34},
			},
			want: "<Block::BodyContent start=5:1, end=5:5>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.block.String()
			test.Diff(t, got, tt.want)
		})
	}
}

func formatBlocks(file *source.File, blocks []block.Block) string {
	b := &strings.Builder{}
	for _, block := range blocks {
		fmt.Fprintf(b, "%s\t%q\n", block, block.Span.Content())

		for _, token := range block.Tokens {
			tokSpan := source.Span{File: file, StartOffset: token.Start, EndOffset: token.End}
			fmt.Fprintf(b, "\t%s\t%q\n", token, tokSpan.Content())
		}
	}

	return b.String()
}

func formatDiagnostics(diagnostics []diagnostic.Diagnostic) string {
	var b strings.Builder
	for _, diagnostic := range diagnostics {
		b.WriteString(diagnostic.String())
		b.WriteByte('\n')
	}

	return b.String()
}
