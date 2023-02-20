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

The current version of the environment requires `go`, `docker` and `docker-compose` to be previously installed, check the links bellow to understand how to install them:

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
| 0x8A791620dd6260079BF849Dc5567aDC3F2FdC318 | Proof of Efficiency |
| 0x60627AC8Ba44F4438186B4bCD5F1cb5E794e19fe | L1 Bridge |
| 0xd0a3d58d135e2ee795dFB26ec150D339394254B9 | L2 Bridge |
| 0x5FbDB2315678afecb367f032d93F642f64180aa3 | Matic token |
| 0xa513E6E4b8f2a923D98304ec87F64353C4D5C853 | GlobalExitRootManager |
