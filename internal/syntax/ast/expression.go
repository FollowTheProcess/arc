package ast

import "go.followtheprocess.codes/arc/internal/syntax/source"

type (
	// Ident is a named identifier expression.
	Ident struct {
		Span source.Span
	}

	// TextLiteral is an (optionally quoted) string of text.
	//
	// Value holds the interpreted string (unquoted, and eventually
	// unescaped), while Span covers the raw literal in source including
	// any surrounding quotes.
	TextLiteral struct {
		Value string // The interpreted (unquoted) text value
		Span  source.Span
	}
)

// Pos implementations.
func (i Ident) Pos() source.Span       { return i.Span }
func (t TextLiteral) Pos() source.Span { return t.Span }

// Expression implementations.
func (i Ident) expressionNode()       {}
func (t TextLiteral) expressionNode() {}
