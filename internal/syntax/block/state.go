package block

// state is an internal representation of the block classifier state.
//
// Syntactic constructs in .http files can't always be determined based
// on the first few characters in a line, so a small amount of "where are we"
// state is needed.
type state uint8

// state value definitions.
//
//go:generate stringer -type state -linecomment
const (
	// The top-level, starting state. No semantic meaning.
	stateInitial state = iota // Initial

	// After the '###' separator but before the `<METHOD> <url>` line.
	stateRequestPrelude // RequestPrelude

	// Inside a '{% ... %}' script block.
	stateScript // Script

	// After the `<METHOD> <url>` line, now parsing header lines.
	stateRequestHeaders // RequestHeaders

	// Now parsing the request body.
	stateRequestBody // RequestBody

	// Parsing any post body items like response redirects or handler scripts.
	stateRequestPostBody // RequestPostBody
)
