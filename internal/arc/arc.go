// Package arc implements the functionality of the arc tool, the CLI in package cmd
// dispatches to the exported members of this package.
package arc

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"go.followtheprocess.codes/log"
)

// Arc represents the arc tool.
type Arc struct {
	stdin   io.Reader   // Program inputs here (e.g. prompts)
	stdout  io.Writer   // Normal program output
	stderr  io.Writer   // Logs and errors
	logger  *log.Logger // The logger, threaded through the entire app
	version string      // The arc version, e.g. for User-Agent
}

// New returns a new [Arc].
func New(debug bool, version string, stdin io.Reader, stdout, stderr io.Writer) Arc {
	level := log.LevelInfo
	if debug {
		level = log.LevelDebug
	}

	logger := log.New(stderr, log.WithLevel(level), log.Prefix("arc"))

	return Arc{
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		logger:  logger,
		version: version,
	}
}

// Hello is a placeholder method for wiring up the CLI.
func (a Arc) Hello(ctx context.Context) error {
	fmt.Fprintln(a.stdout, "Hello from Arc! There's not much here right now, check back later ⌛")
	a.logger.Debug("This is a debug log", slog.String("version", a.version))

	return nil
}
