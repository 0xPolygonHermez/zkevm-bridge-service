package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

const (
	flagCfg              = "cfg"
	flagNetwork          = "network"
	flagL1GenBlockNumber = "block_number"
)

const (
	// App name
	appName = "hermez-bridge"
	// version represents the program based on the git tag
	version = "v0.1.0"
	// commit represents the program based on the git commit
	commit = "dev"
	// date represents the date of application was built
	date = ""
)

func main() {
	app := cli.NewApp()
	app.Name = appName
	app.Version = version
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:     flagCfg,
			Aliases:  []string{"c"},
			Usage:    "Configuration `FILE`",
			Required: false,
		},
		&cli.StringFlag{
			Name:     flagNetwork,
			Aliases:  []string{"n"},
			Usage:    "Network: mainnet, testnet, internaltestnet, local. By default it uses mainnet",
			Required: false,
		},
		&cli.Int64Flag{
			Name:     flagL1GenBlockNumber,
			Aliases:  []string{"g"},
			Usage:    "Initial L1 block number to start synchronizing`",
			Required: false,
		},
	}

	app.Commands = []*cli.Command{
		{
			Name:    "version",
			Aliases: []string{},
			Usage:   "Application version and build",
			Action:  versionCmd,
		},
		{
			Name:    "run",
			Aliases: []string{},
			Usage:   "Run the zkevm bridge",
			Action:  start,
			Flags:   flags,
		},
		{
			Name:    "mockserver",
			Aliases: []string{},
			Usage:   "Run the zkevm bridge mock server",
			Action:  runMockServer,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		os.Exit(1)
	}
}
