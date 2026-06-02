package ast

import (
	"fmt"
	"strings"
)

// Visitor is invoked by [Walk] for each node it encounters.
//
// If the visitor v returned by Visit is non-nil, [Walk] visits each of the
// children of node with v, followed by a call of v.Visit(nil).
type Visitor interface {
	// Visit is called by [Walk] for each node it encounters. The returned
	// visitor v controls traversal of node's children: if non-nil, Walk
	// visits them with v, otherwise it skips them.
	Visit(node Node) (v Visitor)
}

// Walk traverses an AST in depth-first order.
//
// It starts by calling v.Visit(node); node must not be nil. If the visitor v
// returned by v.Visit(node) is not nil, Walk is invoked recursively with v for
// each of the non-nil children of node, followed by a call of v.Visit(nil).
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

		if n.URL != nil {
			Walk(v, n.URL)
		}

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
	case Builtin:
		Walk(v, n.Name)
	case Selector:
		if n.Expr != nil {
			Walk(v, n.Expr)
		}

		Walk(v, n.Sel)
	case Call:
		if n.Fun != nil {
			Walk(v, n.Fun)
		}

		for _, expr := range n.Args {
			Walk(v, expr)
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
		fmt.Fprintf(d.buf, "%sFile %s\n", indent, n.Span())
	case Comment, *Comment:
		fmt.Fprintf(d.buf, "%sComment %s\n", indent, n.Span())
	case Directive:
		fmt.Fprintf(d.buf, "%sDirective %s\n", indent, n.Span())
	case Ident, *Ident:
		fmt.Fprintf(d.buf, "%sIdent %q %s\n", indent, n.Span().Text(), n.Span())
	case TextLiteral:
		fmt.Fprintf(d.buf, "%sTextLiteral %q %s\n", indent, n.Value, n.Span())
	case NumberLiteral:
		if len(n.Span().Content()) != 0 {
			fmt.Fprintf(d.buf, "%sNumberLiteral %q %s\n", indent, n.Span().Text(), n.Span())
		}
	case Request:
		fmt.Fprintf(d.buf, "%sRequest %s\n", indent, n.Span())
	case *HTTPVersion:
		fmt.Fprintf(d.buf, "%sHTTPVersion %s\n", indent, n.Span())
	case Header:
		fmt.Fprintf(d.buf, "%sHeader %s\n", indent, n.Span())
	case Template:
		fmt.Fprintf(d.buf, "%sTemplate %s\n", indent, n.Span())
	case Interp:
		fmt.Fprintf(d.buf, "%sInterp %s\n", indent, n.Span())
	case Builtin:
		fmt.Fprintf(d.buf, "%sBuiltin %s\n", indent, n.Span())
	case Selector:
		fmt.Fprintf(d.buf, "%sSelector %s\n", indent, n.Span())
	case Call:
		fmt.Fprintf(d.buf, "%sCall %s\n", indent, n.Span())
	default:
		fmt.Fprintf(d.buf, "%sast.Dump: UNHANDLED %T\n", indent, node)
	}

	return dumpVisitor{buf: d.buf, depth: d.depth + 1}
}
