// Package syntax provides syntax level primitives such as source
// positions, spans, diagnostics and contains the .http parsing pipeline.
package syntax

// TODO(@FollowTheProcess): Syntax gaps:
// - URL continuation, indented newlines after the request line with url content on them
// - HTTPVersion on the request line, do we need a HTTPVersion token?
