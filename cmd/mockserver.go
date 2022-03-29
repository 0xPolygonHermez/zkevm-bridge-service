package main

import (
	"os"
	"os/signal"

	"github.com/hermeznetwork/hermez-bridge/server"
	"github.com/urfave/cli/v2"
)

func runMockServer(ctx *cli.Context) error {
	err := server.RunMockServer()
	if err != nil {
		return err
	}

	// Wait for an in interrupt.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	return nil
}
