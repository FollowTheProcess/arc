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
		Doc         *Comment // Optional doc comment attached to the request
		Name        *Ident   // Optional `### [name]` or `# @name`
		Method      Ident
		URL         Expression
		HTTPVersion *HTTPVersion
		Headers     []Header
		Span        source.Span // The entire request span
	}

	// HTTPVersion is the AST node representing a `HTTP/<version>` statement
	// following a URL.
	HTTPVersion struct {
		Version NumberLiteral
		Span    source.Span // Includes the HTTP/ prefix
	}

	// Header is the AST node representing a single HTTP header statement.
	Header struct {
		Value Expression
		Span  source.Span
		Name  Ident
	}
)

// Pos implementations.
func (f File) Pos() source.Span        { return f.Span }
func (c Comment) Pos() source.Span     { return c.Span }
func (d Directive) Pos() source.Span   { return d.Span }
func (r Request) Pos() source.Span     { return r.Span }
func (h HTTPVersion) Pos() source.Span { return h.Span }
func (h Header) Pos() source.Span      { return h.Span }

// Statement implementations.
func (f File) statementNode()        {}
func (c Comment) statementNode()     {}
func (d Directive) statementNode()   {}
func (r Request) statementNode()     {}
func (h HTTPVersion) statementNode() {}
func (h Header) statementNode()      {}
