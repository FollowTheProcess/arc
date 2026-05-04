package diagnostic

// Severity is a severity of a diagnostic.
type Severity uint8

// Severity values, in increasing order of urgency.
//
//go:generate stringer -type Severity -linecomment
const (
	SeverityInvalid Severity = iota // invalid
	SeverityWarning                 // warning
	SeverityError                   // error
)

// MarshalText implements [encoding.TextMarshaler] for [Severity].
func (s Severity) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}
