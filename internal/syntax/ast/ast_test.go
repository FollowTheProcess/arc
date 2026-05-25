package ast_test

import (
	"flag"
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/ast"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/snapshot"
)

var (
	update = flag.Bool("update", false, "Update golden files and snapshots")
	clean  = flag.Bool("clean", false, "Regenerate snapshots from scratch")
)

func TestDump(t *testing.T) {
	// A representative .http source from which the test nodes draw their spans,
	// so the dumped positions read like real parser output.
	const src = "@no-redirect\n@timeout = 30s\n"

	file := source.NewFile("test.http", []byte(src))

	span := func(start, end int) source.Span {
		return source.Span{File: file, StartOffset: start, EndOffset: end}
	}

	// `@no-redirect`, a flag directive with no value.
	noRedirect := ast.Directive{
		Ident: ast.Ident{Name: "no-redirect", Span: span(1, 12)},
		Span:  span(0, 12),
	}

	// `@timeout = 30s`, a directive with an ident and a value.
	timeout := ast.Directive{
		Ident: ast.Ident{Name: "timeout", Span: span(14, 21)},
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
			node: ast.Ident{Name: "timeout", Span: span(14, 21)},
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
