package ast

import "go.followtheprocess.codes/arc/internal/syntax/source"

type (
	// Ident is a named identifier expression.
	Ident struct {
		Name string // The name of the ident
		Span source.Span
	}

	// TextLiteral is an (optionally quoted) string of text.
	TextLiteral struct {
		Value string // The (unquoted) text value
		Span  source.Span
	}
)

// Pos implementations.
func (i Ident) Pos() source.Span       { return i.Span }
func (t TextLiteral) Pos() source.Span { return t.Span }

// Expression implementations.
func (i Ident) expressionNode()       {}
func (t TextLiteral) expressionNode() {}
