package ast

import "go.followtheprocess.codes/arc/internal/syntax/source"

// File is the AST node representing a single .http file.
type File struct {
	Span       source.Span
	Statements []Statement
}

// Pos returns the span for the file.
func (f File) Pos() source.Span {
	return f.Span
}

// Directive is the AST node representing a directive e.g.
// `@var id = 123` or `@no-redirect`.
type Directive struct {
	Value Expression
	Ident Ident
	Span  source.Span
}

// Pos returns the span for the directive.
func (d Directive) Pos() source.Span {
	return d.Span
}

// Statement implementations.
func (f File) statementNode()      {}
func (d Directive) statementNode() {}
