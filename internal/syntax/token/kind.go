package token

// Kind is the type for a token.
type Kind int

// Kind definitions.
//
//go:generate stringer -type Kind -linecomment
const (
	EOF                   Kind = iota // EOF
	Error                             // Error
	Comment                           // Comment
	Separator                         // Separator
	Text                              // Text
	HTTPVersion                       // HTTPVersion
	Quote                             // Quote
	Ident                             // Ident
	Dollar                            // Dollar
	At                                // At
	Dot                               // Dot
	LParen                            // LParen
	RParen                            // RParen
	LAngle                            // LAngle
	RAngle                            // RAngle
	Comma                             // Comma
	Colon                             // Colon
	Eq                                // Eq
	Number                            // Number
	OpenInterp                        // OpenInterp
	CloseInterp                       // CloseInterp
	OpenScript                        // OpenScript
	CloseScript                       // CloseScript
	ResponseRedirect                  // ResponseRedirect
	ResponseRedirectForce             // ResponseRedirectForce
	ResponseReference                 // ResponseReference
)
