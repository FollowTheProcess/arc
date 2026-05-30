package ast_test

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

	tests := []struct {
		node ast.Node // Node under test
		name string   // Name of the test case
	}{
		{
			name: "nil",
			node: nil,
		},
		{
			// Dump recurses, so dumping the fixture exercises every node type.
			name: "file",
			node: tree(t),
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

	ast.Inspect(tree(t), func(node ast.Node) bool {
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
		"ast.Template",
		"ast.TextLiteral",
		"ast.Interp",
		"ast.Ident",
	}, "\n")
	test.Diff(t, strings.Join(got, "\n"), want)
}

func TestInspectStopsDescendingWhenCallbackReturnsFalse(t *testing.T) {
	var got []string

	ast.Inspect(tree(t), func(node ast.Node) bool {
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
		"ast.Template",
		"ast.TextLiteral",
		"ast.Interp",
		"ast.Ident",
	}, "\n")
	test.Diff(t, strings.Join(got, "\n"), want)
}

// tree builds the AST for testdata/fixture.http, exercising every node type in
// the package. Node spans are derived from the fixture text by substring lookup
// so the fixture reads like real source rather than a table of byte offsets.
func tree(t *testing.T) ast.File {
	t.Helper()

	src, err := os.ReadFile(filepath.Join("testdata", "fixture.http"))
	test.Ok(t, err)

	file := source.NewFile("fixture.http", src)

	// span returns the span of the first occurrence of sub in the fixture.
	span := func(sub string) source.Span {
		i := bytes.Index(src, []byte(sub))
		test.True(t, i >= 0, test.Context("substring %q not found in fixture", sub))

		return source.Span{File: file, StartOffset: i, EndOffset: i + len(sub)}
	}

	// enclosing spans from the start of from to the end of to.
	enclosing := func(from, to source.Span) source.Span {
		return source.Span{File: file, StartOffset: from.StartOffset, EndOffset: to.EndOffset}
	}

	return ast.File{
		Statements: []ast.Statement{
			ast.Comment{Range: span("# config")},
			ast.Directive{
				Ident: ast.Ident{Range: span("timeout")},
				Value: ast.TextLiteral{Value: "30s", Range: span("30s")},
				Range: span("@timeout = 30s"),
			},
			ast.Request{
				Doc:    &ast.Comment{Range: span("# Fetch a user")},
				Name:   &ast.Ident{Range: span("get-user")},
				Method: ast.Ident{Range: span("GET")},
				URL:    ast.TextLiteral{Value: "https://example.com", Range: span("https://example.com")},
				HTTPVersion: &ast.HTTPVersion{
					Version: ast.NumberLiteral{Range: span("1.1")},
					Range:   span("HTTP/1.1"),
				},
				Headers: []ast.Header{
					{
						Name: ast.Ident{Range: span("Authorization")},
						Value: ast.Template{
							Parts: []ast.Expression{
								ast.TextLiteral{Value: "Bearer ", Range: span("Bearer ")},
								ast.Interp{
									Inner: ast.Ident{Range: span("token")},
									Range: span("{{ token }}"),
								},
							},
							Range: span("Bearer {{ token }}"),
						},
						Range: span("Authorization: Bearer {{ token }}"),
					},
				},
				Range: enclosing(span("### get-user"), span("{{ token }}")),
			},
		},
		Range: source.Span{File: file, StartOffset: 0, EndOffset: len(src)},
	}
}
