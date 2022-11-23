# zkEVM Bridge service

This repo implements a backend service written in Go, that enables clients, like the [web UI](https://github.com/0xPolygonHermez/zkevm-bridge-ui),
to interact with the [bridge smart contract](https://github.com/0xPolygonHermez/zkevm-contracts) by providing Merkleproofs.

## Architecture

<p align="center">
  <img src="./docs/architecture.drawio.png"/>
</p>

## Running the bridge service

- [Running localy](docs/running_local.md)

## Disclaimer

This code has not yet been audited, and should not be used in any production systems.
