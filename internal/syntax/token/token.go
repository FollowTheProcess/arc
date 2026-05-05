// Package token defines the set of lexical tokens for inline
// parsing of .http files.
//
// arc's parser is a block parser, the fundamental unit of a .http
// file is a block. The tokens provided here are for inline parsing
// of particular sections.
//
// The approach is analogous to parsing markdown files, a paragraph
// is a block, and within a paragraph can be inline tokens like '*bold*',
// '[link](url)' etc.
package token

import (
	"fmt"
	"slices"
)

// Kind is the type for a token.
type Kind int

// Kind definitions.
//
//go:generate stringer -type Kind -linecomment
const (
	EOF           Kind = iota // EOF
	Error                     // Error
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

// Token is a lexical token in a .http file.
type Token struct {
	Kind  Kind // The kind of token this is
	Start int  // Byte offset from the start of the file to the start of this token
	End   int  // Byte offset from the start of the file to the end of this token
}

// String implement [fmt.Stringer] for a [Token].
func (t Token) String() string {
	return fmt.Sprintf("<Token::%s start=%d, end=%d>", t.Kind, t.Start, t.End)
}

// Is reports whether the token is any of the provided [Kind]s.
func (t Token) Is(kinds ...Kind) bool {
	return slices.Contains(kinds, t.Kind)
}
