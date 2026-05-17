package lex_test

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/lex"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/arc/internal/syntax/token"
	"go.followtheprocess.codes/test"
	"go.followtheprocess.codes/txtar"
)

var update = flag.Bool("update", false, "Update txtar golden files")

func TestTokenisers(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	tests := []struct {
		tokeniser lex.Tokeniser // The tokeniser to test
		name      string        // Name of the test group (and testdata/{valid,invalid}/{name} sub directory)
	}{
		{name: "separator", tokeniser: lex.Separator},
		{name: "request-line", tokeniser: lex.RequestLine},
		{name: "interps", tokeniser: lex.InterpolatedText},
		{name: "header", tokeniser: lex.Header},
		{name: "directive", tokeniser: lex.Directive},
		{name: "script", tokeniser: lex.Script},
		{name: "body", tokeniser: lex.Body},
		{name: "response-redirect", tokeniser: lex.ResponseRedirect},
		{name: "response-reference", tokeniser: lex.ResponseReference},
	}

	for _, kind := range []string{"valid", "invalid"} {
		t.Run(kind, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					root := filepath.Join("testdata", kind, tt.name)
					n := walkTxtarCases(t, root, func(t *testing.T, file string) {
						t.Helper()
						runLexCase(t, tt.tokeniser, file)
					})
					test.True(t, n > 0, test.Context("no .txtar files found under %s", root))
				})
			}
		})
	}
}

func FuzzSeparator(f *testing.F) {
	// Unfortunately there's no f.Run so we can't do the same table
	// driven approach here
	fuzzTokeniser(f, "separator", lex.Separator)
}

func FuzzRequestLine(f *testing.F) {
	fuzzTokeniser(f, "request-line", lex.RequestLine)
}

func FuzzInterpolatedText(f *testing.F) {
	fuzzTokeniser(f, "interps", lex.InterpolatedText)
}

func FuzzHeader(f *testing.F) {
	fuzzTokeniser(f, "header", lex.Header)
}

func FuzzDirective(f *testing.F) {
	fuzzTokeniser(f, "directive", lex.Directive)
}

func FuzzScript(f *testing.F) {
	fuzzTokeniser(f, "script", lex.Script)
}

func FuzzBody(f *testing.F) {
	fuzzTokeniser(f, "body", lex.Body)
}

func FuzzResponseRedirect(f *testing.F) {
	fuzzTokeniser(f, "response-redirect", lex.ResponseRedirect)
}

func FuzzResponseReference(f *testing.F) {
	fuzzTokeniser(f, "response-reference", lex.ResponseReference)
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

// runLexCase exercises tokeniser against a single txtar archive, either
// updating the archive in place when -update is set or diffing the observed
// output against the recorded expectations.
func runLexCase(t *testing.T, tokeniser lex.Tokeniser, file string) {
	t.Helper()

	want, err := txtar.ParseFile(file)
	test.Ok(t, err)

	src, ok := want.Read("src.http")
	test.True(t, ok, test.Context("archive %s missing src.http", file))

	test.True(t, want.Has("tokens.txt"), test.Context("archive %q missing tokens.txt", file))
	test.True(t, want.Has("diagnostics.txt"), test.Context("archive %q missing diagnostics.txt", file))

	span := lineSpan(src)

	tokens, diagnostics := tokeniser(span)

	got, err := txtar.New(
		txtar.WithComment(want.Comment()),
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
}

// fuzzTokeniser is a fuzz test harness that asserts a set of universal
// invariants that must hold for any [lex.Tokeniser].
func fuzzTokeniser(f *testing.F, name string, tokeniser lex.Tokeniser) {
	f.Helper()

	// Walk the tokeniser's testdata tree so corpus seeds are picked up
	// regardless of how cases are grouped into subdirectories.
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
		test.True(f, ok, test.Context("archive %s missing 'src.http'", path))

		f.Add(src)

		seeded++

		return nil
	}

	for _, kind := range []string{"valid", "invalid"} {
		test.Ok(f, filepath.WalkDir(filepath.Join("testdata", kind, name), walk))
	}

	test.True(f, seeded > 0, test.Context("no .txtar files found for tokeniser %s", name))

	f.Fuzz(func(t *testing.T, src string) {
		span := lineSpan(src)
		content := span.Content()
		tokens, diagnostics := tokeniser(span)

		assertTokenOrder(t, span, content, tokens)
		assertDiagnosticsMatchErrors(t, content, tokens, diagnostics)
		assertDiagnosticBounds(t, span, tokens, diagnostics)
		assertNoSilentDataLoss(t, span, content, tokens, diagnostics)
	})
}

// assertTokenOrder checks tokens are ordered, non-overlapping, and within
// the span bounds.
func assertTokenOrder(t *testing.T, span source.Span, content []byte, tokens []token.Token) {
	t.Helper()

	prevEnd := span.StartOffset
	for i, tok := range tokens {
		valid := tok.Start >= span.StartOffset &&
			tok.End >= tok.Start &&
			tok.End <= span.EndOffset &&
			tok.Start >= prevEnd

		test.True(
			t,
			valid,
			test.Context(
				"token %d out of order or out of bounds: span={%d,%d} prevEnd=%d token={%d,%d} kind=%s content=%q",
				i, span.StartOffset, span.EndOffset, prevEnd, tok.Start, tok.End, tok.Kind, content,
			),
		)

		prevEnd = tok.End
	}
}

// assertDiagnosticsMatchErrors checks the diagnostic count matches the
// number of Error tokens emitted.
func assertDiagnosticsMatchErrors(
	t *testing.T,
	content []byte,
	tokens []token.Token,
	diagnostics []diagnostic.Diagnostic,
) {
	t.Helper()

	errorTokens := 0

	for _, tok := range tokens {
		if tok.Kind == token.Error {
			errorTokens++
		}
	}

	test.Equal(
		t,
		len(diagnostics),
		errorTokens,
		test.Context("diagnostics %d != error tokens %d for content %q", len(diagnostics), errorTokens, content),
	)
}

// assertDiagnosticBounds checks every diagnostic span lies within the
// tokeniser's input span.
func assertDiagnosticBounds(t *testing.T, span source.Span, _ []token.Token, diagnostics []diagnostic.Diagnostic) {
	t.Helper()

	for i, d := range diagnostics {
		valid := d.Span.StartOffset >= span.StartOffset &&
			d.Span.EndOffset >= d.Span.StartOffset &&
			d.Span.EndOffset <= span.EndOffset

		test.True(
			t,
			valid,
			test.Context(
				"diagnostic %d span out of bounds: span={%d,%d} diag={%d,%d} msg=%q",
				i, span.StartOffset, span.EndOffset, d.Span.StartOffset, d.Span.EndOffset, d.Message,
			),
		)
	}
}

// assertNoSilentDataLoss checks every non-whitespace rune in the span
// content is covered by either an emitted token range or a diagnostic
// span. Whitespace ([unicode.IsSpace]) may be skipped silently.
func assertNoSilentDataLoss(
	t *testing.T,
	span source.Span,
	content []byte,
	tokens []token.Token,
	diagnostics []diagnostic.Diagnostic,
) {
	t.Helper()

	covered := make([]bool, len(content))
	mark := func(start, end int) {
		lo := max(start-span.StartOffset, 0)
		hi := min(end-span.StartOffset, len(content))

		for i := lo; i < hi; i++ {
			covered[i] = true
		}
	}

	for _, tok := range tokens {
		mark(tok.Start, tok.End)
	}

	for _, d := range diagnostics {
		mark(d.Span.StartOffset, d.Span.EndOffset)
	}

	for i := 0; i < len(content); {
		r, w := utf8.DecodeRune(content[i:])
		if w == 0 {
			w = 1
		}

		if !covered[i] && !unicode.IsSpace(r) {
			test.True(
				t,
				false,
				test.Context(
					"rune %q at offset %d covered by neither token nor diagnostic, content=%q",
					r, span.StartOffset+i, content,
				),
			)
		}

		i += w
	}
}

// lineSpan builds a [source.Span] from a single line of test input,
// mirroring what the block parser hands the inline tokeniser:
// a span over the line bytes excluding the trailing terminator.
//
// txtar sections retain their trailing newline, trim it off the span
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
