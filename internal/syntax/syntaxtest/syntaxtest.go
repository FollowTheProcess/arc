// Package syntaxtest provides shared helpers for the golden-file and snapshot
// tests across arc's syntax pipeline stages (lex, block, assembler, ...).
//
// It owns the -update and -clean flags so every stage shares one regeneration
// mechanism, and centralises the txtar case walker, the fuzz corpus seeder, and
// the diagnostic formatter that the stages would otherwise duplicate. The
// per-stage output formatters (blocks, tokens, AST) stay local to their tests
// since they are specific to what each stage produces.
package syntaxtest

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
)

//nolint:gochecknoglobals // Global test flags
var (
	update = flag.Bool("update", false, "Update golden files and snapshots")
	clean  = flag.Bool("clean", false, "Regenerate snapshots from scratch")
)

// Updating reports whether the -update flag was set, signalling that golden
// files and snapshots should be rewritten rather than diffed.
func Updating() bool {
	return *update
}

// Cleaning reports whether the -clean flag was set, signalling that snapshots
// should be regenerated from scratch.
func Cleaning() bool {
	return *clean
}

// WalkTxtarCases recursively walks root, nesting a subtest per directory and
// invoking fn for every .txtar file. This mirrors the testdata directory layout
// in the test name hierarchy so individual cases, whole directories, or the
// full group can be selected via -run. Returns the total number of .txtar files
// processed across the tree.
func WalkTxtarCases(t *testing.T, root string, fn func(t *testing.T, path string)) int {
	t.Helper()

	entries, err := os.ReadDir(root)
	test.Ok(t, err)

	total := 0

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())

		if entry.IsDir() {
			var sub int

			t.Run(entry.Name(), func(t *testing.T) {
				sub = WalkTxtarCases(t, path, fn)
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

// SeedCorpus walks root for .txtar archives and calls add with the src.http
// contents of each, letting fuzz tests seed their corpus from the golden
// fixtures regardless of how cases are grouped into subdirectories. The caller
// supplies add so it can choose the seed type ([]byte or string). Returns the
// number of archives seeded.
func SeedCorpus(tb testing.TB, root string, add func(src string)) int {
	tb.Helper()

	seeded := 0

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
		test.True(tb, ok, test.Context("archive %s missing 'src.http'", path))

		add(src)

		seeded++

		return nil
	}

	test.Ok(tb, filepath.WalkDir(root, walk))

	return seeded
}

// FormatDiagnostics renders diagnostics one per line via their String method,
// for embedding in a golden file's diagnostics.txt section.
func FormatDiagnostics(diagnostics []diagnostic.Diagnostic) string {
	var b strings.Builder
	for _, d := range diagnostics {
		b.WriteString(d.String())
		b.WriteByte('\n')
	}

	return b.String()
}
