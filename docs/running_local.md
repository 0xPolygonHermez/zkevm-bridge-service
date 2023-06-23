> WARNING: This documentation is outdated, it will be updated soon

# Steps to run environment locally

## Overview

This documentation will help you running the following components:

- zkEVM Node Databases
- zkEVM Bridge Database
- L1 Network
- Prover
- zkEVM Node
- zkEVM Bridge Service

## Requirements

The current version of the environment requires `go`, `docker` and `docker-compose` to be previously installed, check the links below to understand how to install them:

- <https://go.dev/doc/install>
- <https://www.docker.com/get-started>
- <https://docs.docker.com/compose/install/>

The `zkevm-bridge-service` docker image must be built at least once and every time a change is made to the code.
If you haven't build the `zkevm-bridge-service` image yet, you must run:

```bash
make build-docker
```

## Controlling the environment

> All the data is stored inside of each docker container, this means once you remove the container, the data will be lost.

To run the environment:

```bash
make run
```

To stop the environment:

```bash
make stop
```

To run e2e and edge tests:

```bash
make test-full
make test-edge
```

## Accessing the environment

- zkEVM Bridge Database 
  - `Type:` Postgres DB
  - `User:` test_user
  - `Password:` test_password
  - `Database:` test_db
  - `Host:` localhost
  - `Port:` 5435
  - `Url:` <postgres://test_user:test_password@localhost:5435/test_db>
- zkEVM Bridge Service
  - `Type:` Web
  - `Host:` localhost
  - `Port:` 8080
  - `Url:` <http://localhost:8080>

## SC Addresses

| Address | Description |
|---|---|
| 0x610178dA211FEF7D417bC0e6FeD39F05609AD788 | Proof of Efficiency |
| 0xff0EE8ea08cEf5cb4322777F5CC3E8A584B8A4A0 | L1 Bridge |
| 0xff0EE8ea08cEf5cb4322777F5CC3E8A584B8A4A0 | L2 Bridge |
| 0x5FbDB2315678afecb367f032d93F642f64180aa3 | Matic token |
| 0x2279B7A0a67DB372996a5FaB50D91eAA73d2eBe6 | GlobalExitRootManager |


## Fund account on L2 with ETH

If you need account with funds you can use the [deposit script](https://github.com/0xPolygonHermez/zkevm-bridge-service/blob/develop/test/scripts/deposit/main.go)
to fund an account.
For a list with accounts that already have ETH check out [node's docs](https://github.com/0xPolygonHermez/zkevm-node/blob/develop/docs/running_local.md#accounts).

You can exchange the `l1AccHexAddress` and `l1AccHexPrivateKey` and once executing the script with
```
go run test/scripts/deposit/main.go
```
the account that you've specified under `l1AccHexAddress` would have been funded on L2.
