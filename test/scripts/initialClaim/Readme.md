# InitialClaim script
This script allows to create the claim tx and include it in a forcedBatch. This is require when the L2 network is empty and there are no funds in L2.
Typically this action is used to include the claim tx to fill the bridge autoclaim wallet with ethers in L2 in order to allow the service send the claim txs for the users.

## Parameters
At the beginning of the script there are the next constant variables that need to be reviewed.
```
    l2BridgeAddr = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
	zkevmAddr    = "0x8dAF17A20c9DBA35f005b6324F493785D239719d"

	accHexAddress    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	accHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	l1NetworkURL       = "http://localhost:8545"
	l2NetworkURL       = "http://localhost:8123"
	bridgeURL          = "http://localhost:8080"
```
`l2BridgeAddr` is the bridge address smart contract in L2
`zkevmAddr` is the polygonZkEvm address in L1
`accHexAddress` is the wallet address used to send the claim in L2 and to send the forcedBatch in L1
`accHexPrivateKey` is the wallet private key used to send the claim in L2 and to send the forcedBatch in L1
`l1NetworkURL` is the url of the L1 rpc
`l2NetworkURL` is the url of the L2 rpc
`bridgeURL` is the url of the bridge service