// Package ast provides the abstract syntax tree for .http files.
package ast

import (
	"fmt"
	"strings"

	"go.followtheprocess.codes/arc/internal/syntax/source"
)

// Node is the interface implemented by all AST nodes.
type Node interface {
	// Pos returns the [source.Span] containing the node in the original
	// source text.
	Pos() source.Span
}

// Statement is the interface implemented by all statement AST nodes.
type Statement interface {
	Node

	statementNode()
}

// Expression is the interface implemented by all expression AST nodes.
type Expression interface {
	Node

	expressionNode()
}

// Dump returns a text representation of an ast [Node].
//
// It is primarily used for debugging and inspecting the ast.
func Dump(node Node) string {
	buf := &strings.Builder{}

	if node == nil {
		buf.WriteString("<nil>\n")

		return buf.String()
	}

	// Traversal lives in [Walk]; the visitor only formats each node it's
	// handed and tracks indentation depth.
	Walk(dumpVisitor{buf: buf}, node)

	return buf.String()
}

// dumpVisitor is a [Visitor] that writes an indented text representation of
// each node it visits, used by [Dump].
type dumpVisitor struct {
	buf   *strings.Builder
	depth int
}

// Visit implements [Visitor], formatting node and returning a visitor one
// level deeper for its children.
//
//nolint:ireturn // Visitor.Visit must return a Visitor to satisfy the interface.
func (d dumpVisitor) Visit(node Node) Visitor {
	if node == nil {
		return nil
	}

	indent := strings.Repeat("  ", d.depth)

	switch n := node.(type) {
	case File:
		fmt.Fprintf(d.buf, "%sFile %s\n", indent, n.Pos())
	case Comment:
		fmt.Fprintf(d.buf, "%sComment %s\n", indent, n.Pos())
	case Directive:
		fmt.Fprintf(d.buf, "%sDirective %s\n", indent, n.Pos())
	case Ident:
		fmt.Fprintf(d.buf, "%sIdent %q %s\n", indent, n.Pos().Text(), n.Pos())
	case TextLiteral:
		fmt.Fprintf(d.buf, "%sTextLiteral %q %s\n", indent, n.Value, n.Pos())
	default:
		fmt.Fprintf(d.buf, "%sast.Dump: UNHANDLED %T\n", indent, node)
	}

	return dumpVisitor{buf: d.buf, depth: d.depth + 1}
}
