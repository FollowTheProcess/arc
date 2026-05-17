// Package diagnostic provides a syntax level diagnostic toolkit for
// high quality error reporting.
package diagnostic

import (
	"fmt"

	"go.followtheprocess.codes/arc/internal/syntax/source"
)

// TODO: Some sort of elegant way of attaching labels and fixes
//
// Functional options maybe

// Diagnostic is a single source level diagnostic.
type Diagnostic struct {
	Message  string      `json:"message,omitempty"` // Message text
	Span     source.Span `json:"span,omitzero"`     // The span of source text for which this diagnostic applies.
	Labels   []Label     `json:"labels,omitempty"`  // Supplementary spans e.g. "defined here".
	Fixes    []Fix       `json:"fixes,omitempty"`   // Suggested edits.
	Severity Severity    `json:"severity,omitzero"` // Severity is the severity of the diagnostic.
}

// String returns a string representation of a [Diagnostic].
func (d Diagnostic) String() string {
	if d.Severity == SeverityInvalid || d.Message == "" || d.Span.File == nil {
		return ""
	}

	return fmt.Sprintf("[%s] %s: %s", d.Severity, d.Span, d.Message)
}

// Label is a supplementary span adding further context to a [Diagnostic]
// by highlighting another region of source related to the primary finding.
type Label struct {
	Message string      `json:"message,omitempty"` // Label text to display, e.g. "previously defined here".
	Span    source.Span `json:"span,omitzero"`     // The region of source to which this label refers.
}

// Fix represents one or more suggested edits to a source span resulting from a [Diagnostic].
//
// The application of the fix should solve the original finding. A [Fix] is atomic, either
// all of it's edits apply together or none do. For example, in cases where one logical
// change touches two non-contiguous spans such as renaming a global variable, and then
// updating all it's call sites.
type Fix struct {
	Message string `json:"message,omitempty"` // Fix message.
	Edits   []Edit `json:"edits,omitempty"`   // List of suggested edits.
}

// Edit represents a single suggested edit resulting from a [Diagnostic].
type Edit struct {
	Replacement string      `json:"replacement"`   // Use this instead
	Span        source.Span `json:"span,omitzero"` // Range to replace
}
