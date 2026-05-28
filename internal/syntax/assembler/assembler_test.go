package assembler_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/assembler"
	"go.followtheprocess.codes/arc/internal/syntax/ast"
	"go.followtheprocess.codes/arc/internal/syntax/block"
	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/arc/internal/syntax/syntaxtest"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
)

func TestAssembler(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	for _, group := range []string{"valid", "invalid"} {
		t.Run(group, func(t *testing.T) {
			root := filepath.Join("testdata", group)
			n := syntaxtest.WalkTxtarCases(t, root, runAssemblerTest)
			test.True(t, n > 0, test.Context("no .txtar files found under %s", root))
		})
	}
}

// TestAssembleEmptyInput pins that the assembler handles a block stream with
// no blocks. An empty .http file parses to zero blocks, and block.Parse feeds
// Assemble directly in the pipeline, so this must not panic.
func TestAssembleEmptyInput(t *testing.T) {
	tests := []struct {
		name   string
		blocks []block.Block
	}{
		{name: "nil", blocks: nil},
		{name: "empty", blocks: []block.Block{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diagnostics := assembler.Assemble(tt.blocks)

			test.Equal(t, len(file.Statements), 0, test.Context("expected no statements"))
			test.Equal(t, len(diagnostics), 0, test.Context("expected no diagnostics"))
		})
	}
}

// FuzzAssembler asserts the universal invariants that must hold for the
// assembler's output regardless of input. It uses the real pipeline
// (content -> block.Parse -> Assemble) so the block streams it sees are
// realistic.
//
// Invariants checked here are mainly to do with the resulting ast.
func FuzzAssembler(f *testing.F) {
	syntaxtest.SeedCorpus(f, "testdata", func(src string) {
		f.Add([]byte(src))
	})

	// Some extra seeds not likely to be in the testdata/ corpus but
	// worth including anyway
	extra := [][]byte{
		nil,
		[]byte(""),
		[]byte("\n"),
		[]byte("   "),
		[]byte("@"),
	}

	for _, seed := range extra {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, content []byte) {
		file := source.NewFile("fuzz", content)
		blocks, _ := block.Parse(file)

		// Assemble shouldn't panic on a valid block stream
		// (tested implicitly, fuzz automatically fails on panics)
		fileNode, diagnostics := assembler.Assemble(blocks)

		// Every AST node should refer to the file above in it's span. The only exception
		// being empty input returns an empty File with no span
		assertNodeSpans(t, file, fileNode)

		// Every diagnostic should point at the same file (the one above)
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

// assertNodeSpans asserts every node in the AST rooted at root carries a
// well-formed span:
//
//   - The file should be the one we parsed
//   - Offsets should be in order (none going backwards or staying constant).
func assertNodeSpans(t *testing.T, file *source.File, root ast.Node) {
	t.Helper()

	ast.Inspect(root, func(node ast.Node) bool {
		// Inspect calls back with nil to mark the end of a node's children.
		if node == nil {
			return true
		}

		span := node.Pos()
		if span.File == nil {
			return true
		}

		test.Equal(t, span.File, file, test.Context("node %T points at the wrong file", node))

		valid := span.StartOffset >= 0 &&
			span.EndOffset >= span.StartOffset &&
			span.EndOffset <= file.Len()

		test.True(
			t,
			valid,
			test.Context(
				"node %T span out of order or out of bounds: span={%d,%d} len=%d",
				node, span.StartOffset, span.EndOffset, file.Len(),
			),
		)

		return true
	})
}

// runAssemblerTest exercises the assembler against a single txtar archive,
// either updating the archive in place when -update is set or diffing the
// observed output against the recorded expectations.
func runAssemblerTest(t *testing.T, file string) {
	t.Helper()

	want, err := txtar.ParseFile(file)
	test.Ok(t, err)

	src, ok := want.Read("src.http")
	test.True(t, ok, test.Context("archive %s missing src.http", file))

	test.True(t, want.Has("want.txt"), test.Context("archive %q missing want.txt", file))
	test.True(t, want.Has("diagnostics.txt"), test.Context("archive %q missing diagnostics.txt", file))

	path := filepath.Join(file, "src.http")
	path = filepath.ToSlash(path)
	srcFile := source.NewFile(path, []byte(src))

	var allDiags []diagnostic.Diagnostic

	blocks, blockDiagnostics := block.Parse(srcFile)
	allDiags = append(allDiags, blockDiagnostics...)

	fileNode, diagnostics := assembler.Assemble(blocks)
	allDiags = append(allDiags, diagnostics...)

	got, err := txtar.New(
		txtar.WithComment(want.Comment()),
		txtar.WithFile("src.http", src),
		txtar.WithFile("want.txt", ast.Dump(fileNode)),
		txtar.WithFile("diagnostics.txt", syntaxtest.FormatDiagnostics(allDiags)),
	)

	test.Ok(t, err)

	if syntaxtest.Updating() {
		test.Ok(t, txtar.DumpFile(file, got))

		return
	}

	test.Diff(t, got.String(), want.String())
}
