package ast_test

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/ast"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/test"
)

var (
	update = flag.Bool("update", false, "Update golden files and snapshots")
	clean  = flag.Bool("clean", false, "Regenerate snapshots from scratch")
)

func TestDump(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	// A representative .http source from which the test nodes draw their spans,
	// so the dumped positions read like real parser output.
	const src = "@no-redirect\n@timeout = 30s\n"

	file := source.NewFile("test.http", []byte(src))

	span := func(start, end int) source.Span {
		return source.Span{File: file, StartOffset: start, EndOffset: end}
	}

	// `@no-redirect`, a flag directive with no value.
	noRedirect := ast.Directive{
		Ident: ast.Ident{Span: span(1, 12)},
		Span:  span(0, 12),
	}

	// `@timeout = 30s`, a directive with an ident and a value.
	timeout := ast.Directive{
		Ident: ast.Ident{Span: span(14, 21)},
		Value: ast.TextLiteral{Value: "30s", Span: span(24, 27)},
		Span:  span(13, 27),
	}

	tests := []struct {
		node ast.Node // Node under test
		name string   // Name of the test case
	}{
		{
			name: "nil",
			node: nil,
		},
		{
			name: "ident",
			node: ast.Ident{Span: span(14, 21)},
		},
		{
			name: "text literal",
			node: ast.TextLiteral{Value: "30s", Span: span(24, 27)},
		},
		{
			name: "flag directive",
			node: noRedirect,
		},
		{
			name: "directive with value",
			node: timeout,
		},
		{
			name: "empty file",
			node: ast.File{Span: span(0, 0)},
		},
		{
			name: "file with directives",
			node: ast.File{
				Statements: []ast.Statement{noRedirect, timeout},
				Span:       span(0, len(src)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := snapshot.New(
				t,
				snapshot.Clean(*clean),
				snapshot.Update(*update),
				snapshot.WithFormatter(snapshot.TextFormatter()),
			)

			snap.Snap(ast.Dump(tt.node))
		})
	}
}

func TestInspectVisitsEveryNodeDepthFirst(t *testing.T) {
	var got []string

	ast.Inspect(tree(), func(node ast.Node) bool {
		if node != nil {
			got = append(got, fmt.Sprintf("%T", node))
		}

		return true
	})

	want := "ast.File\nast.Directive\nast.Ident\nast.TextLiteral"
	test.Diff(t, strings.Join(got, "\n"), want)
}

func TestInspectStopsDescendingWhenCallbackReturnsFalse(t *testing.T) {
	var got []string

	ast.Inspect(tree(), func(node ast.Node) bool {
		if node == nil {
			return true
		}

		got = append(got, fmt.Sprintf("%T", node))

		// Don't descend into the directive's children.
		_, isDirective := node.(ast.Directive)

		return !isDirective
	})

	want := "ast.File\nast.Directive"
	test.Diff(t, strings.Join(got, "\n"), want)
}

// tree is a small AST used by the traversal tests: a single file containing
// one `@timeout = 30s` directive with both an ident and a value.
func tree() ast.File {
	// TODO: Add more to this when we have more nodes
	file := source.NewFile("test.http", []byte("@timeout = 30s\n"))
	span := func(start, end int) source.Span {
		return source.Span{File: file, StartOffset: start, EndOffset: end}
	}

	return ast.File{
		Statements: []ast.Statement{
			ast.Directive{
				Ident: ast.Ident{Span: span(1, 8)},
				Value: ast.TextLiteral{Value: "30s", Span: span(11, 14)},
				Span:  span(0, 14),
			},
		},
		Span: span(0, 14),
	}
}
