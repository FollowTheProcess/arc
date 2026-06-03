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
		Templated bool // <@ vs <
	}
)

// Span implementations.
func (bi BodyInline) Span() source.Span { return bi.Range }
func (bf BodyFile) Span() source.Span   { return bf.Range }

// Expression implementations.
func (bi BodyInline) expressionNode() {}
func (bf BodyFile) expressionNode()   {}

// Body implementations, only these things can be bodies.
func (bi BodyInline) bodyNode() {}
func (bf BodyFile) bodyNode()   {}
