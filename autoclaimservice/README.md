# AutoClaim service
This service allows to claim deposits whose destination is the network specified by the bridgeAddress and RPC network.

## Configuration file:
This is an example of the configuration file.
```
[Log]
Level = "debug"
Outputs = ["stdout"]

[AutoClaim]
AuthorizedClaimMessageAddresses = []
AutoClaimInterval = "10m"
MaxNumberOfClaimsPerGroup = 10
BridgeURL = "http://localhost:8080"

[BlockchainManager]
PrivateKey = {Path = "./test.keystore.autoclaim", Password = "testonly"}
L2RPC = "http://localhost:8123"
PolygonBridgeAddress = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
ClaimCompressorAddress = "0x2279B7A0a67DB372996a5FaB50D91eAA73d2eBe6"
```
- The log seccion allows to modify the log level. By default: debug mode
- The autoclaim seccion allows to configure the next parameters:
  - `AuthorizedClaimMessageAddresses` These are the addresses that can use the autoclaim feature for bridge messages. By default: none
  - `AutoClaimInterval` This is the param that controls the interval to check new bridges. By default: 10m
  - `MaxNumberOfClaimsPerGroup` This param allow to control the maximum numbre of claims that can be grouped in a single tx. By default: 10
  - `BridgeURL` This is the bridge service URL to get the bridges that can be claimed. By default: localhost:8080
- The BlockchainManager seccion allows to modify network parameters.
  - `PrivateKey` is the wallet used to send the claim txs. By default: 0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266
  - `L2RPC` is the URL of the L2 node to send claim txs. By default: localhost:8123
  - `PolygonBridgeAddress` is the L2 bridge address. By default: 0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E
  - `ClaimCompressorAddress` is the compressor smc address. By default: 0x2279B7A0a67DB372996a5FaB50D91eAA73d2eBe6

***Note: If ClaimCompressorAddress is not defined or MaxNumberOfClaimsPerGroup is 0, then the claim compressor feature is disabled and claim txs will be sent one by one.

## How to run:
This is a command example of how to run this service:
```
go run main.go run --cfg ./config/config.local.toml
```