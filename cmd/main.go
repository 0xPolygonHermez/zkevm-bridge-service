package main

import (
	"fmt"
	"os"

	zkevmbridgeservice "github.com/0xPolygonHermez/zkevm-bridge-service"
	"github.com/urfave/cli/v2"
)

const (
	flagCfg     = "cfg"
	flagNetwork = "network"
)

const (
	// App name
	appName = "zkevm-bridge"
)

func main() {
	app := cli.NewApp()
	app.Name = appName
	app.Version = zkevmbridgeservice.Version
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
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		os.Exit(1)
	}
}
