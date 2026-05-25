package ast

import "go.followtheprocess.codes/arc/internal/syntax/source"

// Ident is a named identifier expression.
type Ident struct {
	Name string // The name of the ident
	Span source.Span
}

// Pos returns the span for the ident.
func (i Ident) Pos() source.Span {
	return i.Span
}

// TextLiteral is an, optionally quoted, string of text.
type TextLiteral struct {
	Value string // The (unquoted) text value
	Span  source.Span
}

// Pos returns the span for the text.
func (t TextLiteral) Pos() source.Span {
	return t.Span
}

// Expression implementations.
func (i Ident) expressionNode()       {}
func (t TextLiteral) expressionNode() {}
