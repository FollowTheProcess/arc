package ast

import (
	"fmt"
	"strings"
)

// Visitor is invoked by [Walk] for each node it encounters.
//
// If the visitor w returned by Visit is non-nil, [Walk] visits each of the
// children of node with w, followed by a call of w.Visit(nil).
type Visitor interface {
	// Visit is called by [Walk] for each node it encounters. The returned
	// visitor w controls traversal of node's children: if non-nil, Walk
	// visits them with w, otherwise it skips them.
	Visit(node Node) (w Visitor)
}

// Walk traverses an AST in depth-first order.
//
// It starts by calling v.Visit(node); node must not be nil. If the visitor w
// returned by v.Visit(node) is not nil, Walk is invoked recursively with w for
// each of the non-nil children of node, followed by a call of w.Visit(nil).
//
//nolint:gocognit // Self contained ast recursive walking
func Walk(v Visitor, node Node) {
	if v = v.Visit(node); v == nil {
		return
	}

	switch n := node.(type) {
	case File:
		for _, statement := range n.Statements {
			Walk(v, statement)
		}
	case Directive:
		Walk(v, n.Ident)

		if n.Value != nil {
			Walk(v, n.Value)
		}
	case Request:
		if n.Doc != nil {
			Walk(v, n.Doc)
		}

		if n.Name != nil {
			Walk(v, n.Name)
		}

		Walk(v, n.Method)
		Walk(v, n.URL)

		if n.HTTPVersion != nil {
			Walk(v, n.HTTPVersion)
		}

		for _, header := range n.Headers {
			Walk(v, header)
		}
	case *HTTPVersion:
		if n != nil {
			Walk(v, n.Version)
		}
	case Header:
		Walk(v, n.Name)

		if n.Value != nil {
			Walk(v, n.Value)
		}
	case Template:
		for _, part := range n.Parts {
			Walk(v, part)
		}
	case Interp:
		if n.Inner != nil {
			Walk(v, n.Inner)
		}
	case Ident, *Ident, TextLiteral, Comment, *Comment, NumberLiteral, nil:
		// Leaves, no children to walk.
	default:
		panic(fmt.Sprintf("ast.Walk: unexpected node type %T", n))
	}

	v.Visit(nil)
}

// Inspect traverses an AST in depth-first order.
//
// It starts by calling f(node); node must not be nil. If f returns true,
// Inspect invokes f recursively for each of the non-nil children of node,
// followed by a call of f(nil).
func Inspect(node Node, f func(Node) bool) {
	Walk(inspector(f), node)
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

// inspector adapts an ordinary function to the [Visitor] interface.
type inspector func(Node) bool

// Visit implements [Visitor] for inspector.
func (f inspector) Visit(node Node) Visitor {
	if f(node) {
		return f
	}

	return nil
}

// dumpVisitor is a [Visitor] that writes an indented text representation of
// each node it visits, used by [Dump].
type dumpVisitor struct {
	buf   *strings.Builder
	depth int
}

// Visit implements [Visitor], formatting node and returning a visitor one
// level deeper for its children.
func (d dumpVisitor) Visit(node Node) Visitor {
	if node == nil {
		return nil
	}

	indent := strings.Repeat("  ", d.depth)

	switch n := node.(type) {
	case File:
		fmt.Fprintf(d.buf, "%sFile %s\n", indent, n.Pos())
	case Comment, *Comment:
		fmt.Fprintf(d.buf, "%sComment %s\n", indent, n.Pos())
	case Directive:
		fmt.Fprintf(d.buf, "%sDirective %s\n", indent, n.Pos())
	case Ident, *Ident:
		fmt.Fprintf(d.buf, "%sIdent %q %s\n", indent, n.Pos().Text(), n.Pos())
	case TextLiteral:
		fmt.Fprintf(d.buf, "%sTextLiteral %q %s\n", indent, n.Value, n.Pos())
	case NumberLiteral:
		if len(n.Span.Content()) != 0 {
			fmt.Fprintf(d.buf, "%sNumberLiteral %q %s\n", indent, n.Pos().Text(), n.Pos())
		}
	case Request:
		fmt.Fprintf(d.buf, "%sRequest %s\n", indent, n.Pos())
	case *HTTPVersion:
		fmt.Fprintf(d.buf, "%sHTTPVersion %s\n", indent, n.Pos())
	case Header:
		fmt.Fprintf(d.buf, "%sHeader %s\n", indent, n.Pos())
	case Template:
		fmt.Fprintf(d.buf, "%sTemplate %s\n", indent, n.Pos())
	case Interp:
		fmt.Fprintf(d.buf, "%sInterp %s\n", indent, n.Pos())
	default:
		fmt.Fprintf(d.buf, "%sast.Dump: UNHANDLED %T\n", indent, node)
	}

	return dumpVisitor{buf: d.buf, depth: d.depth + 1}
}
