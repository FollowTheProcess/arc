package block_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/block"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/arc/internal/syntax/syntaxtest"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
)

func TestBlockClassifier(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	for _, group := range []string{"valid", "invalid"} {
		t.Run(group, func(t *testing.T) {
			root := filepath.Join("testdata", group)
			n := syntaxtest.WalkTxtarCases(t, root, runBlockParserTest)
			test.True(t, n > 0, test.Context("no .txtar files found under %s", root))
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
			name: "body",
			block: block.Block{
				Kind: block.Body,
				Span: source.Span{File: file, StartOffset: 30, EndOffset: 30},
			},
			want: "<Block::Body start=5:1, end=5:1>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.block.String()
			test.Diff(t, got, tt.want)
		})
	}
}

func FuzzBlockParser(f *testing.F) {
	// Add every txtar's src.http as fuzz corpus, walking the tree so seeds are
	// picked up regardless of how testdata is grouped into subdirectories.
	syntaxtest.SeedCorpus(f, "testdata", func(src string) {
		f.Add([]byte(src))
	})

	f.Fuzz(func(t *testing.T, content []byte) {
		file := source.NewFile("fuzz", content)

		blocks, diagnostics := block.Parse(file)

		// Every block span must point at the file we just parsed and
		// stay within its bounds. Body blocks must additionally cover at
		// least one byte (empty regions don't emit a Body block at all).
		prevStart := 0

		for i, blk := range blocks {
			test.Equal(t, blk.Span.File, file, test.Context("block %d points at the wrong file", i))

			valid := blk.Span.StartOffset >= 0 &&
				blk.Span.EndOffset >= blk.Span.StartOffset &&
				blk.Span.EndOffset <= len(content) &&
				blk.Span.StartOffset >= prevStart

			test.True(
				t,
				valid,
				test.Context(
					"block %d span out of order or out of bounds: prevStart=%d span={%d,%d} len=%d kind=%s",
					i, prevStart, blk.Span.StartOffset, blk.Span.EndOffset, len(content), blk.Kind,
				),
			)

			prevStart = blk.Span.StartOffset

			// Tokens must live inside their owning block.
			for j, tok := range blk.Tokens {
				tokValid := tok.Start >= blk.Span.StartOffset &&
					tok.End >= tok.Start &&
					tok.End <= blk.Span.EndOffset

				test.True(
					t,
					tokValid,
					test.Context(
						"block %d token %d out of bounds: block={%d,%d} token={%d,%d} kind=%s",
						i, j, blk.Span.StartOffset, blk.Span.EndOffset, tok.Start, tok.End, tok.Kind,
					),
				)
			}

			if blk.Kind == block.Body {
				test.True(
					t,
					blk.Span.EndOffset > blk.Span.StartOffset,
					test.Context(
						"block %d Body has empty span: {%d,%d}",
						i, blk.Span.StartOffset, blk.Span.EndOffset,
					),
				)
			}
		}

		// Every diagnostic span must point at the file and stay within bounds.
		for i, d := range diagnostics {
			test.Equal(t, d.Span.File, file, test.Context("diagnostic %d points at the wrong file", i))

			valid := d.Span.StartOffset >= 0 &&
				d.Span.EndOffset >= d.Span.StartOffset &&
				d.Span.EndOffset <= len(content)

			test.True(
				t,
				valid,
				test.Context(
					"diagnostic %d span out of bounds: span={%d,%d} len=%d msg=%q",
					i, d.Span.StartOffset, d.Span.EndOffset, len(content), d.Message,
				),
			)
		}
	})
}

func formatBlocks(file *source.File, blocks []block.Block) string {
	b := &strings.Builder{}
	for _, block := range blocks {
		fmt.Fprintf(b, "%s\t`%s`\n", block, block.Span.Content())

		for _, token := range block.Tokens {
			tokSpan := source.Span{File: file, StartOffset: token.Start, EndOffset: token.End}
			fmt.Fprintf(b, "\t%s\t`%s`\n", token, tokSpan.Content())
		}
	}

	return b.String()
}

// runBlockParserTest exercises the block parser against a single txtar archive,
// either updating the archive in place when -update is set or diffing the
// observed output against the recorded expectations.
func runBlockParserTest(t *testing.T, file string) {
	t.Helper()

	want, err := txtar.ParseFile(file)
	test.Ok(t, err)

	src, ok := want.Read("src.http")
	test.True(t, ok, test.Context("archive %s missing src.http", file))

	test.True(t, want.Has("want.txt"), test.Context("archive %q missing want.txt", file))
	test.True(t, want.Has("diagnostics.txt"), test.Context("archive %q missing diagnostics.txt", file))

	// Give tests unique names (helps with debugging individual tests)
	// but don't inject OS separators into the fixtures
	path := filepath.Join(file, "src.http")
	path = filepath.ToSlash(path)

	srcFile := source.NewFile(path, []byte(src))

	blocks, diagnostics := block.Parse(srcFile)

	got, err := txtar.New(
		txtar.WithComment(want.Comment()),
		txtar.WithFile("src.http", src),
		txtar.WithFile("want.txt", formatBlocks(srcFile, blocks)),
		txtar.WithFile("diagnostics.txt", syntaxtest.FormatDiagnostics(diagnostics)),
	)
	test.Ok(t, err)

	if syntaxtest.Updating() {
		test.Ok(t, txtar.DumpFile(file, got))

		return
	}

	test.Diff(t, got.String(), want.String())
}
