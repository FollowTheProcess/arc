// Package syntax provides syntax level primitives such as source
// positions, spans and diagnostics.
package syntax

// TODO(@FollowTheProcess): Syntax gaps:
// - URL continuation, indented newlines after the request line with url content on them
// - HTTPVersion on the request line, do we need a HTTPVersion token?
// - Multi-arg directives like `@prompt id "The user's ID"` and `@secret password "Super secret shh"`
// - Add a state in the block dispatch for <>, >>, or > {% %} after a body
