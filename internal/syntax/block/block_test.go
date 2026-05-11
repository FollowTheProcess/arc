package block_test

import (
	"flag"
	"fmt"
	"io/fs"
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

	for _, group := range []string{"valid", "invalid"} {
		t.Run(group, func(t *testing.T) {
			root := filepath.Join("testdata", group)
			n := walkTxtarCases(t, root, runBlockParserTest)
			test.True(t, n > 0, test.Context("no .txtar files found under %s", root))
		})
	}
}

// walkTxtarCases recursively walks root, nesting a subtest per directory and
// invoking fn for every .txtar file. This mirrors the testdata directory layout
// in the test name hierarchy so individual cases, whole directories, or the
// full group can be selected via -run. Returns the total number of .txtar files
// processed across the tree.
func walkTxtarCases(t *testing.T, root string, fn func(t *testing.T, path string)) int {
	t.Helper()

	entries, err := os.ReadDir(root)
	test.Ok(t, err)

	total := 0

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())

		if entry.IsDir() {
			var sub int

			t.Run(entry.Name(), func(t *testing.T) {
				sub = walkTxtarCases(t, path, fn)
			})

			total += sub

			continue
		}

		if filepath.Ext(entry.Name()) != ".txtar" {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			fn(t, path)
		})

		total++
	}

	return total
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

	srcFile := source.NewFile("src.http", []byte(src))

	blocks, diagnostics := block.Parse(srcFile)

	got, err := txtar.New(
		txtar.WithComment(want.Comment()),
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

func FuzzBlockParser(f *testing.F) {
	// Add every txtar's src.http as fuzz corpus, walking the tree so seeds are
	// picked up regardless of how testdata is grouped into subdirectories.
	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(d.Name()) != ".txtar" {
			return nil
		}

		archive, err := txtar.ParseFile(path)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		src, ok := archive.Read("src.http")
		test.True(f, ok, test.Context("archive %s missing 'src.http'", path))

		f.Add([]byte(src))

		return nil
	}

	test.Ok(f, filepath.WalkDir("testdata", walk))

	f.Fuzz(func(t *testing.T, content []byte) {
		file := source.NewFile("fuzz", content)

		blocks, diagnostics := block.Parse(file)

		// Every block span must point at the file we just parsed and
		// stay within its bounds. BodyOpen and BodyClose markers must be
		// paired - every BodyOpen is followed by a matching BodyClose with
		// no intervening BodyOpen, and a BodyClose is never seen first.
		prevStart := 0
		inBody := false

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

			switch blk.Kind {
			case block.BodyOpen:
				test.False(t, inBody, test.Context("block %d: BodyOpen while already in body", i))
				inBody = true
			case block.BodyClose:
				test.True(t, inBody, test.Context("block %d: BodyClose without matching BodyOpen", i))
				inBody = false
			default:
				// Other block kinds don't participate in body framing.
			}
		}

		test.True(t, !inBody, test.Context("BodyOpen at end of input without matching BodyClose"))

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
