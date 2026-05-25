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
	writeNode(buf, node, 0)

	return buf.String()
}

// writeNode writes a given node to the buffer.
func writeNode(buf *strings.Builder, node Node, depth int) {
	indent := strings.Repeat("  ", depth)

	switch n := node.(type) {
	case nil:
		fmt.Fprintf(buf, "%s<nil>\n", indent)
	case File:
		fmt.Fprintf(buf, "%sFile %s\n", indent, n.Pos())

		for _, statement := range n.Statements {
			writeNode(buf, statement, depth+1)
		}
	case Directive:
		fmt.Fprintf(buf, "%sDirective %s\n", indent, n.Pos())
		writeNode(buf, n.Ident, depth+1)

		if n.Value != nil {
			writeNode(buf, n.Value, depth+1)
		}
	case Ident:
		fmt.Fprintf(buf, "%sIdent %q %s\n", indent, n.Name, n.Pos())
	case TextLiteral:
		fmt.Fprintf(buf, "%sTextLiteral %q %s\n", indent, n.Value, n.Pos())
	default:
		fmt.Fprintf(buf, "%s<UNHANDLED %T>\n", indent, node)
	}
}
