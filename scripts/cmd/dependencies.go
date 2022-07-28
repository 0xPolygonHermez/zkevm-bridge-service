package main

import (
	"fmt"
	"path"
	"path/filepath"
	"runtime"

	"github.com/0xPolygonHermez/zkevm-node/scripts/cmd/dependencies"
	"github.com/urfave/cli/v2"
)

func updateDeps(ctx *cli.Context) error {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../../")
	fmt.Print(dir)

	cfg := &dependencies.Config{
		Images: &dependencies.ImagesConfig{
			Names:          []string{"hermeznetwork/hermez-node-zkevm:develop", "hermeznetwork/geth-zkevm-contracts", "hermeznetwork/hez-mock-prover"},
			TargetFilePath: filepath.Join(dir, "docker-compose.yml"),
		},
		PB: &dependencies.PBConfig{
			TargetDirPath: filepath.Join(dir, "proto/src"),
			SourceRepo:    "https://github.com/0xPolygonHermez/zkevm-comms-protocol.git",
		},
		TV: &dependencies.TVConfig{
			TargetDirPath: filepath.Join(dir, "test/vectors/src"),
			SourceRepo:    "https://github.com/0xPolygonHermez/zkevm-testvectors.git",
		},
	}

	return dependencies.NewManager(cfg).Run()
}
