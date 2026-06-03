package ast

import "go.followtheprocess.codes/arc/internal/syntax/source"

type (
	// File is the AST node representing a single .http file.
	File struct {
		Statements []Statement
		Range      source.Span
	}

	// Comment represents a single line comment.
	Comment struct {
		Range source.Span
	}

	// Directive is the AST node representing a directive e.g.
	// `@var id = 123` or `@no-redirect`. In the case of a flag
	// directive like `@no-redirect`, Value is nil.
	Directive struct {
		Value Expression
		Ident Ident
		Range source.Span
	}

	// Request is the AST node representing a http request.
	Request struct {
		Doc         *Comment // Optional doc comment attached to the request
		Name        *Ident   // Optional `### [name]` or `# @name`
		Method      Ident
		URL         Expression
		HTTPVersion *HTTPVersion
		Headers     []Header
		Body        Body        // Optional request body, nil if absent
		Range       source.Span // The entire request span
	}

	// HTTPVersion is the AST node representing a `HTTP/<version>` statement
	// following a URL.
	HTTPVersion struct {
		Version NumberLiteral
		Range   source.Span // Includes the HTTP/ prefix
	}

	// Header is the AST node representing a single HTTP header statement.
	Header struct {
		Value Expression
		Range source.Span
		Name  Ident
	}
)

// Span implementations.
func (f File) Span() source.Span        { return f.Range }
func (c Comment) Span() source.Span     { return c.Range }
func (d Directive) Span() source.Span   { return d.Range }
func (r Request) Span() source.Span     { return r.Range }
func (h HTTPVersion) Span() source.Span { return h.Range }
func (h Header) Span() source.Span      { return h.Range }

// Statement implementations.
func (f File) statementNode()        {}
func (c Comment) statementNode()     {}
func (d Directive) statementNode()   {}
func (r Request) statementNode()     {}
func (h HTTPVersion) statementNode() {}
func (h Header) statementNode()      {}
