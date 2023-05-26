package main

import (
	"os"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "zkevm-bridge-scripts"
	app.Commands = []*cli.Command{
		{
			Name:   "updatedeps",
			Usage:  "Updates external dependencies like images, test vectors or proto files",
			Action: updateDeps,
			Flags:  []cli.Flag{},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
