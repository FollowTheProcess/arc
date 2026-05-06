package block

// Kind is the kind of a [Block].
type Kind int

// Block kind definitions.
//
//go:generate stringer -type Kind -linecomment
const (
	Blank                Kind = iota // Blank
	Error                            // Error
	Separator                        // Separator
	Comment                          // Comment
	Directive                        // Directive
	LocalVariable                    // LocalVariable
	RequestLine                      // RequestLine
	URLContinuation                  // URLContinuation
	Header                           // Header
	HeaderBodySeparator              // HeaderBodySeparator
	BodyOpen                         // BodyOpen
	BodyClose                        // BodyClose
	BodyContent                      // BodyContent
	FileBody                         // FileBody
	PreRequestScriptOpen             // PreRequestScriptOpen
	ResponseScriptOpen               // ResponseScriptOpen
	ScriptContent                    // ScriptContent
	ScriptClose                      // ScriptClose
	ResponseRedirect                 // ResponseRedirect
	ResponseReference                // ResponseReference
)
