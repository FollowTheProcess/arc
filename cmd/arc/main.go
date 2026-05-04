// Package arc is a command line API Client and testing tool, driven by plaintext .http files.
package main

import (
	"context"
	"os"
	"os/signal"

	"go.followtheprocess.codes/arc/internal/cmd"
	"go.followtheprocess.codes/msg"
)

func main() {
	if err := run(); err != nil {
		msg.Err(err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()

	cli, err := cmd.Build()
	if err != nil {
		return err
	}

	return cli.Execute(ctx)
}
