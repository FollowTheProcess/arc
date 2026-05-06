package token

// Kind is the type for a token.
type Kind int

// Kind definitions.
//
//go:generate stringer -type Kind -linecomment
const (
	EOF           Kind = iota // EOF
	Error                     // Error
	Separator                 // Separator
	Text                      // Text
	Ident                     // Ident
	Dollar                    // Dollar
	Dot                       // Dot
	LParen                    // LParen
	RParen                    // RParen
	Comma                     // Comma
	Colon                     // Colon
	Eq                        // Eq
	StringLiteral             // StringLiteral
	Number                    // Number
	OpenInterp                // OpenInterp
	CloseInterp               // CloseInterp
	OpenScript                // OpenScript
	CloseScript               // CloseScript
)
