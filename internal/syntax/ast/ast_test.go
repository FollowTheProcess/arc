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

	want := strings.Join([]string{
		"ast.File",
		"ast.Comment",
		"ast.Directive",
		"ast.Ident",
		"ast.TextLiteral",
		"ast.Request",
		"*ast.Comment",
		"*ast.Ident",
		"ast.Ident",
		"ast.TextLiteral",
		"*ast.HTTPVersion",
		"ast.NumberLiteral",
		"ast.Header",
		"ast.Ident",
		"ast.TextLiteral",
	}, "\n")
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

	want := strings.Join([]string{
		"ast.File",
		"ast.Comment",
		"ast.Directive",
		"ast.Request",
		"*ast.Comment",
		"*ast.Ident",
		"ast.Ident",
		"ast.TextLiteral",
		"*ast.HTTPVersion",
		"ast.NumberLiteral",
		"ast.Header",
		"ast.Ident",
		"ast.TextLiteral",
	}, "\n")
	test.Diff(t, strings.Join(got, "\n"), want)
}

// tree is the AST used by the traversal tests.
func tree() ast.File {
	const src = `# config
@timeout = 30s
### get-user
# Fetch a user
GET https://example.com HTTP/1.1
Accept: application/json
`

	file := source.NewFile("test.http", []byte(src))
	span := func(start, end int) source.Span {
		return source.Span{File: file, StartOffset: start, EndOffset: end}
	}

	return ast.File{
		Statements: []ast.Statement{
			ast.Comment{Span: span(0, 8)},
			ast.Directive{
				Ident: ast.Ident{Span: span(10, 17)},
				Value: ast.TextLiteral{Value: "30s", Span: span(20, 23)},
				Span:  span(9, 23),
			},
			ast.Request{
				Doc:    &ast.Comment{Span: span(37, 51)},
				Name:   &ast.Ident{Span: span(28, 36)},
				Method: ast.Ident{Span: span(52, 55)},
				URL:    ast.TextLiteral{Value: "https://example.com", Span: span(56, 75)},
				HTTPVersion: &ast.HTTPVersion{
					Version: ast.NumberLiteral{Span: span(81, 84)},
					Span:    span(76, 84),
				},
				Headers: []ast.Header{
					{
						Name:  ast.Ident{Span: span(85, 91)},
						Value: ast.TextLiteral{Value: "application/json", Span: span(93, 109)},
						Span:  span(85, 109),
					},
				},
				Span: span(24, 109),
			},
		},
		Span: span(0, len(src)),
	}
}
