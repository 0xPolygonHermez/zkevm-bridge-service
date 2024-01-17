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
	appName = "x1-bridge"
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
			Usage:   "Run the x1 bridge, including API, synchronizer, claimtxman, kafka consumer, etc.",
			Action:  runAll,
			Flags:   flags,
		},
		{
			Name:    "runAPI",
			Aliases: []string{},
			Usage:   "Run the x1 bridge API server",
			Action:  runAPI,
			Flags:   flags,
		},
		{
			Name:    "runTask",
			Aliases: []string{},
			Usage:   "Run the x1 bridge tasks, including synchronizer, claimtxman, kafka consumer",
			Action:  runTask,
			Flags:   flags,
		},
		{
			Name:    "runPushTask",
			Aliases: []string{},
			Usage:   "Run the x1 bridge push tasks (monitor the block/batch number and push change event to FE)",
			Action:  runPushTask,
			Flags:   flags,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		os.Exit(1)
	}
}
