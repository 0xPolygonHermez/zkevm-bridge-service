# zkEVM Bridge service

This repo implements a backend service written in Go, that enables clients, like the [web UI](https://github.com/0xPolygonHermez/zkevm-bridge-ui),
to interact with the [bridge smart contract](https://github.com/0xPolygonHermez/zkevm-contracts) by providing Merkleproofs.

## Architecture

<p align="center">
  <img src="./docs/architecture.drawio.png"/>
</p>

## Running the bridge service

- [Running locally](docs/running_local.md)

## Running e2e test for real networks
There are a test for ERC20 L1->L2 and L2->L1. This test is for be run externally. 
For this reason you can build  execution docker: 
```
make build-docker-e2e-real_network
```

- To run you need to pass a configuration file  as `test/config/bridge_network_e2e/cardona.toml`
- Example of usage: 

```
#!/bin/bash
make build-docker-e2e-real_network
mkdir tmp
cat <<EOF > ./tmp/test.toml
TestAddrPrivate= "${{ SECRET_PRIVATE_ADDR }}"
[ConnectionConfig]
L1NodeURL="${{ SECRET_L1URL }}"
L2NodeURL="${{ L2URL }}"
BridgeURL="${{ BRIDGEURL }}"
L1BridgeAddr="${{ BRIDGE_ADDR_L1 }}"
L2BridgeAddr="${{ BRIDGE_ADDR_L2 }}"
EOF
docker run  --volume "./tmp/:/config/" --env BRIDGE_TEST_CONFIG_FILE=/config/test.toml bridge-e2e-realnetwork-erc20
```

## Contact

For more discussions, please head to the [R&D Discord](https://discord.gg/0xPolygonRnD)
