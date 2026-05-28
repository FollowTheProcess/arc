package ast

import "fmt"

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
func Walk(v Visitor, node Node) {
	// TODO: Add more cases when we have more nodes
	// Will self-reveal with a panic hopefully
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
	case Ident, *Ident, TextLiteral, Comment, NumberLiteral, nil:
		// Leaves, no children to walk.
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
	case *HTTPVersion:
		if n != nil {
			Walk(v, n.Version)
		}

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

// inspector adapts an ordinary function to the [Visitor] interface.
type inspector func(Node) bool

// Visit implements [Visitor] for inspector.
func (f inspector) Visit(node Node) Visitor {
	if f(node) {
		return f
	}

	return nil
}
