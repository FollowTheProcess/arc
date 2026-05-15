// Package syntax provides syntax level primitives such as source
// positions, spans and diagnostics.
package syntax

// TODO(@FollowTheProcess): Quoted strings
//
// Currently we treat "a string" as Text including the quotes, need to do a bit
// of research as to what other http tools do, whether you can have bare strings
// or they *have* to be quoted, maybe just enforcing quotes is easier?
//
// Enforcing quotes means we can do variable expression type things without the
// interp blocks e.g. `@something = other.something` if we want that?
//
// A good compromise might be accepting a continuous run of characters as Text
// without quotes so like `https://thisis/all/astring` but as soon as it contains
// whitespace it needs to be wrapped in quotes?
