package ast

import "go.followtheprocess.codes/arc/internal/syntax/source"

type (
	// Ident is a named identifier expression.
	Ident struct {
		Range source.Span
	}

	// TextLiteral is an (optionally quoted) string of text.
	//
	// Value holds the interpreted string (unquoted, and eventually
	// unescaped), while Span covers the raw literal in source including
	// any surrounding quotes.
	TextLiteral struct {
		Value string // The interpreted (unquoted) text value
		Range source.Span
	}

	// NumberLiteral is a number expression, can be an integer
	// or a decimal.
	NumberLiteral struct {
		Range source.Span
	}

	// Template is a sequence of literal and interpolation expressions
	// e.g. `Bearer {{ token }}` or a URL containing interps.
	Template struct {
		Parts []Expression
		Range source.Span
	}

	// Interp is a single `{{ ... }}` with an inner expression.
	Interp struct {
		Inner Expression
		Range source.Span
	}
)

// Span implementations.
func (i Ident) Span() source.Span         { return i.Range }
func (t TextLiteral) Span() source.Span   { return t.Range }
func (n NumberLiteral) Span() source.Span { return n.Range }
func (t Template) Span() source.Span      { return t.Range }
func (i Interp) Span() source.Span        { return i.Range }

// Expression implementations.
func (i Ident) expressionNode()         {}
func (t TextLiteral) expressionNode()   {}
func (n NumberLiteral) expressionNode() {}
func (t Template) expressionNode()      {}
func (i Interp) expressionNode()        {}
