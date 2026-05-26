package ast

import "go.followtheprocess.codes/arc/internal/syntax/source"

type (
	// File is the AST node representing a single .http file.
	File struct {
		Statements []Statement
		Span       source.Span
	}

	// Comment represents a single line comment.
	Comment struct {
		Span source.Span
	}

	// Directive is the AST node representing a directive e.g.
	// `@var id = 123` or `@no-redirect`.
	Directive struct {
		Value Expression
		Ident Ident
		Span  source.Span
	}
)

// Pos implementations.
func (f File) Pos() source.Span      { return f.Span }
func (c Comment) Pos() source.Span   { return c.Span }
func (d Directive) Pos() source.Span { return d.Span }

// Statement implementations.
func (f File) statementNode()      {}
func (c Comment) statementNode()   {}
func (d Directive) statementNode() {}
