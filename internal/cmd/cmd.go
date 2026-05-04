// Package cmd defines the arc CLI.
package cmd

import (
	"context"
	"os"

	"go.followtheprocess.codes/arc/internal/arc"
	"go.followtheprocess.codes/cli"
)

//nolint:gochecknoglobals // These have to be here
var (
	version = "dev"
	commit  = ""
	date    = ""
)

// Build builds and returns the entire arc CLI.
func Build() (*cli.Command, error) {
	var debug bool

	return cli.New(
		"arc",
		cli.Short("A command line API Client and testing tool, driven by .http files"),
		cli.Version(version),
		cli.Commit(commit),
		cli.BuildDate(date),
		cli.Flag(&debug, "debug", 'd', "Enable debug logs"),
		cli.Run(func(ctx context.Context, cmd *cli.Command) error {
			app := arc.New(debug, version, os.Stdin, os.Stdout, os.Stderr)

			return app.Hello(ctx)
		}),
	)
}
