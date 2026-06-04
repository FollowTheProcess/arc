package ast

import (
	"fmt"
	"strconv"
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

		if n.Body != nil {
			Walk(v, n.Body)
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
	case BodyInline:
		Walk(v, n.Content)
	case BodyFile:
		Walk(v, n.Path)
	case BodyForm:
		for _, field := range n.Fields {
			Walk(v, field)
		}
	case BodyMultipart:
		for _, part := range n.Parts {
			Walk(v, part)
		}
	case MultipartPart:
		for _, header := range n.Headers {
			Walk(v, header)
		}

		if n.Body != nil {
			Walk(v, n.Body)
		}
	case FormField:
		if n.Key != nil {
			Walk(v, n.Key)
		}

		if n.Value != nil {
			Walk(v, n.Value)
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

// dumpEscaper escapes line-structure characters in [Dump] literal values so
// each node renders on a single line.
//
//nolint:gochecknoglobals // Immutable and effectively constant, fine as a global
var dumpEscaper = strings.NewReplacer("\n", `\n`, "\r", `\r`, "\t", `\t`)

// dumpText renders a literal value for [Dump] output: backtick delimited with
// only line-structure characters escaped, so interior quotes (common in JSON
// bodies) read cleanly. Anything that still can't be backquoted after that
// (backticks, other control characters) falls back to [strconv.Quote].
func dumpText(s string) string {
	escaped := dumpEscaper.Replace(s)
	if !strconv.CanBackquote(escaped) {
		return strconv.Quote(s)
	}

	return "`" + escaped + "`"
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
		fmt.Fprintf(d.buf, "%sTextLiteral %s %s\n", indent, dumpText(n.Value), n.Span())
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
	case BodyInline:
		fmt.Fprintf(d.buf, "%sBodyInline %s\n", indent, n.Span())
	case BodyFile:
		fmt.Fprintf(d.buf, "%sBodyFile", indent)

		if n.Templated {
			d.buf.WriteString(" templated")
		}

		if n.Encoding != "" {
			fmt.Fprintf(d.buf, " encoding=%q", n.Encoding)
		}

		fmt.Fprintf(d.buf, " %s\n", n.Span())
	case BodyForm:
		fmt.Fprintf(d.buf, "%sBodyForm %s\n", indent, n.Span())
	case BodyMultipart:
		fmt.Fprintf(d.buf, "%sBodyMultipart", indent)

		if n.Boundary != "" {
			fmt.Fprintf(d.buf, " boundary=%q", n.Boundary)
		}

		fmt.Fprintf(d.buf, " %s\n", n.Span())
	case FormField:
		fmt.Fprintf(d.buf, "%sFormField %s\n", indent, n.Span())
	case MultipartPart:
		fmt.Fprintf(d.buf, "%sMultipartPart %s\n", indent, n.Span())
	default:
		fmt.Fprintf(d.buf, "%sast.Dump: UNHANDLED %T\n", indent, node)
	}

	return dumpVisitor{buf: d.buf, depth: d.depth + 1}
}
