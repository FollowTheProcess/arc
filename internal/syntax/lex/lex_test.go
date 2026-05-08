package lex_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/lex"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/arc/internal/syntax/token"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
)

var update = flag.Bool("update", false, "Update txtar golden files")

func TestSeparator(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	t.Run("valid", func(t *testing.T) {
		pattern := filepath.Join("testdata", "separator", "valid", "*.txtar")
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

				test.True(t, want.Has("tokens.txt"), test.Context("archive %q missing tokens.txt", file))
				test.True(t, want.Has("diagnostics.txt"), test.Context("archive %q missing diagnostics.txt", file))

				span := lineSpan(src)

				tokens, diagnostics := lex.Separator(span)

				got, err := txtar.New(
					txtar.WithFile("src.http", src),
					txtar.WithFile("tokens.txt", formatTokens(tokens)),
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
		pattern := filepath.Join("testdata", "separator", "invalid", "*.txtar")
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

				test.True(t, want.Has("tokens.txt"), test.Context("archive %q missing tokens.txt", file))
				test.True(t, want.Has("diagnostics.txt"), test.Context("archive %q missing diagnostics.txt", file))

				span := lineSpan(src)

				tokens, diagnostics := lex.Separator(span)

				got, err := txtar.New(
					txtar.WithFile("src.http", src),
					txtar.WithFile("tokens.txt", formatTokens(tokens)),
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

// lineSpan builds a [source.Span] from a single line of test input,
// mirroring what the block parser hands the inline tokeniser:
// a span over the line bytes excluding the trailing terminator.
//
// txtar sections retain their trailing newline; trim it off the span
// bounds (but keep it in the file) so [source.Span.Content] returns
// what [source.File.Lines] would have yielded.
func lineSpan(src string) source.Span {
	file := source.NewFile("src.http", []byte(src))
	end := len(strings.TrimRight(src, "\r\n"))

	return source.Span{
		File:        file,
		StartOffset: 0,
		EndOffset:   end,
	}
}

func formatTokens(tokens []token.Token) string {
	var b strings.Builder
	for _, token := range tokens {
		b.WriteString(token.String())
		b.WriteByte('\n')
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
