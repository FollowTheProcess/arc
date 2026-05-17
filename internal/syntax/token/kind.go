package token

// Kind is the type for a token.
type Kind int

// Kind definitions.
//
//go:generate stringer -type Kind -linecomment
const (
	Error                 Kind = iota // Error
	Separator                         // Separator
	Text                              // Text
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
	Comment                           // Comment
)
