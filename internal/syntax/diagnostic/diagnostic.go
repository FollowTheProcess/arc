// Package diagnostic provides a syntax level diagnostic toolkit for
// high quality error reporting.
package diagnostic

import (
	"fmt"

	"go.followtheprocess.codes/arc/internal/syntax/source"
)

// Severity is a severity of a diagnostic.
type Severity int

// Severity values, in increasing order of urgency.
//
//go:generate stringer -type Severity -linecomment
const (
	SeverityInvalid Severity = iota // invalid
	SeverityWarning                 // warning
	SeverityError                   // error
)

// Diagnostic is a single source level diagnostic.
type Diagnostic struct {
	Message  string      // Message text
	Span     source.Span // The span of source text for which this diagnostic applies.
	Severity Severity    // Severity is the severity of the diagnostic.
}

// String returns a string representation of a [Diagnostic].
func (d Diagnostic) String() string {
	if d.Severity == SeverityInvalid || d.Message == "" || d.Span.File == nil {
		return ""
	}

	return fmt.Sprintf("[%s] %s: %s", d.Severity, d.Span, d.Message)
}
