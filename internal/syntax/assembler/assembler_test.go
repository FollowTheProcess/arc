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

	srcFile := source.NewFile("src.http", []byte(src))

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
