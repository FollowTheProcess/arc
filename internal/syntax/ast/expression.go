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

	// Builtin is a `$` prefixed builtin root like `$env` or the
	// `$random` in `$random.int`.
	Builtin struct {
		Name  Ident
		Range source.Span
	}

	// Selector is a dotted accessor Expr.Sel, e.g. `response.body` or
	// `$random.uuid`.
	Selector struct {
		Expr  Expression
		Sel   Ident
		Range source.Span
	}

	// Call is a call trailer Fun(Args...), e.g. `$random.int(1, 100)`.
	Call struct {
		Fun   Expression
		Args  []Expression
		Range source.Span
	}
)

// Span implementations.
func (i Ident) Span() source.Span         { return i.Range }
func (t TextLiteral) Span() source.Span   { return t.Range }
func (n NumberLiteral) Span() source.Span { return n.Range }
func (t Template) Span() source.Span      { return t.Range }
func (i Interp) Span() source.Span        { return i.Range }
func (b Builtin) Span() source.Span       { return b.Range }
func (s Selector) Span() source.Span      { return s.Range }
func (c Call) Span() source.Span          { return c.Range }

// Expression implementations.
func (i Ident) expressionNode()         {}
func (t TextLiteral) expressionNode()   {}
func (n NumberLiteral) expressionNode() {}
func (t Template) expressionNode()      {}
func (i Interp) expressionNode()        {}
func (b Builtin) expressionNode()       {}
func (s Selector) expressionNode()      {}
func (c Call) expressionNode()          {}
