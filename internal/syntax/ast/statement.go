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
	// `@var id = 123` or `@no-redirect`. In the case of a flag
	// directive like `@no-redirect`, Value is nil.
	Directive struct {
		Value Expression
		Ident Ident
		Span  source.Span
	}

	// Request is the AST node representing a http request.
	Request struct {
		Doc    *Comment // Optional doc comment attached to the request
		Name   *Ident   // Optional `### [name]` or `# @name`
		Method Ident
		URL    Expression
		Span   source.Span // The entire request span

		// TODO: HTTPVersion, Headers, BodyExpr etc.
	}
)

// Pos implementations.
func (f File) Pos() source.Span      { return f.Span }
func (c Comment) Pos() source.Span   { return c.Span }
func (d Directive) Pos() source.Span { return d.Span }
func (r Request) Pos() source.Span   { return r.Span }

// Statement implementations.
func (f File) statementNode()      {}
func (c Comment) statementNode()   {}
func (d Directive) statementNode() {}
func (r Request) statementNode()   {}
