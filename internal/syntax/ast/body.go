package ast

import "go.followtheprocess.codes/arc/internal/syntax/source"

type (
	// BodyInline is an inline specified body in the source file, it
	// can be templates and contain interps at any place.
	BodyInline struct {
		Content Template
		Range   source.Span
	}

	// BodyFile is the body node emitted when the body content is
	// referenced via a filepath e.g. `< body.json`.
	//
	// The content of the body file can be run through interpolation
	// with `<@ body.json`, this is stored in the Templated field.
	BodyFile struct {
		Encoding  string   // "" default, "utf8", "latin1" etc.
		Path      Template // The filepath (can be interpolated)
		Range     source.Span
		Templated bool // <@ (true) vs < (false)
	}

	// BodyMultipart represents a multipart body with a boundary.
	BodyMultipart struct {
		Boundary string
		Parts    []MultipartPart
		Range    source.Span
	}

	// MultipartPart is a single chunk of a [BodyMultipart].
	MultipartPart struct {
		Headers []Header
		Body    Body // A part can be any valid body, file, inline, interpolated etc.
		Range   source.Span
	}

	// BodyForm represents a url form encoded body.
	//
	// Fields are split out at parse time, rather than treating the body as
	// inline text, so that evaluation can tell structural '&' and '=' (written
	// in the source) apart from ones arriving inside interpolated values; the
	// latter must be percent-encoded, which is impossible to do correctly
	// after the parts have been joined.
	BodyForm struct {
		Fields []FormField
		Range  source.Span
	}

	// FormField is a single url form encoded key, value pair.
	FormField struct {
		Key   Expression // Normally TextLiteral but can be Template
		Value Expression
		Range source.Span
	}
)

// Span implementations.
func (bi BodyInline) Span() source.Span    { return bi.Range }
func (bf BodyFile) Span() source.Span      { return bf.Range }
func (bm BodyMultipart) Span() source.Span { return bm.Range }
func (bf BodyForm) Span() source.Span      { return bf.Range }
func (m MultipartPart) Span() source.Span  { return m.Range }
func (f FormField) Span() source.Span      { return f.Range }

// Expression implementations.
func (bi BodyInline) expressionNode()    {}
func (bf BodyFile) expressionNode()      {}
func (bm BodyMultipart) expressionNode() {}
func (bf BodyForm) expressionNode()      {}

// Body implementations, only these things can be bodies.
func (bi BodyInline) bodyNode()    {}
func (bf BodyFile) bodyNode()      {}
func (bm BodyMultipart) bodyNode() {}
func (bf BodyForm) bodyNode()      {}
