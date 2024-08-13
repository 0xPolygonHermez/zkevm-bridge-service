# Test Real network using e2e Test
This suite is for test the bridge service running in a real network.
The included tests are: 
- ERC20 L1->L2 expecting auto-claim
- ERC20 L2->L1 
- BridgeMessage L1->L2
- BridgeMessage L2->L1

# Build docker
First you need to build the docker that include the tests.
`make build-docker-e2e-real_network-ERC20`
or
`make  build-docker-e2e-real_network-MSG`


## Create a config file
Check the config example  `test/config/bridge_network_e2e/cardona.toml`

## Execute docker using the config file
- Create the config file in `/tmp/test.toml`
- Launch tests:
 ```
 $ docker run  --volume "./tmp/:/config/" --env BRIDGE_TEST_CONFIG_FILE=/config/test.toml bridge-e2e-realnetwork-erc20  
 $docker run  --volume "./tmp/:/config/" --env BRIDGE_TEST_CONFIG_FILE=/config/test.toml bridge-e2e-realnetwork-erc20  
 ```