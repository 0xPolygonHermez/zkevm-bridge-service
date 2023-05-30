package main

import (
	"os"

	zkevmbridgeservice "github.com/0xPolygonHermez/zkevm-bridge-service"
	"github.com/urfave/cli/v2"
)

func versionCmd(*cli.Context) error {
	zkevmbridgeservice.PrintVersion(os.Stdout)
	return nil
}
