Use this tools, you can easily transfer asset between L1 and L2.

## Prerequisites
The tutorial for current version of the environment requires Foundryup, check out the links provided below:
- https://book.getfoundry.sh/getting-started/installation

## Bridge tools
```
cd x1-bridge-service
go mod tidy

# setup all components 
make build-docker
make run

# build bridge tools
cd ./test/scripts/bridge
go build

# summury, transfer asset between L1 and L2: ./bridge [0: L1->L2 OKB; 1: L1->L2 ETH; 2:L2->L1 OKB; 3: L2->L1 ETH]

# bridge OKB L1->L2, and query L2 OKB balance
./bridge 0
cast balance 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8123

# bridge ETH L1->L2, and query L2 WETH balance
./bridge 1
cast call 0x82109a709138A2953C720D3d775168717b668ba6 "balanceOf(address)" 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8123

# bridge OKB L2->L1, and query L1 OKB balance
./bridge 2
cast call 0x82109a709138A2953C720D3d775168717b668ba6 "balanceOf(address)" 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8545

# bridge ETH L2->L1, and query L1 ETH balance
./bridge 3
cast balance 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8545

```

## Query balance
### Query L2 OKB Balance
```
cast balance 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8123
``` 

### Query L2 WETH Balance
```
cast call 0x82109a709138A2953C720D3d775168717b668ba6 "balanceOf(address)" 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8123
```

### Query L1 OKB Balance
```
cast call 0x82109a709138A2953C720D3d775168717b668ba6 "balanceOf(address)" 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8545
```

### Query L1 ETH Balance
```
cast balance 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8545
```


## Modify
If you want to modify, you can modify the main.go:

```
bridgeAddr = "0x10B65c586f795aF3eCCEe594fE4E38E1F059F780"
okbAddress = "0x82109a709138A2953C720D3d775168717b668ba6"
ethAddress = "0x82109a709138A2953C720D3d775168717b668ba6"

accHexAddress    = "0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225"
accHexPrivateKey = "0xde3ca643a52f5543e84ba984c4419ff40dbabd0e483c31c1d09fee8168d68e38"

l1NetworkURL = "http://localhost:8545"
l2NetworkURL = "http://localhost:8123"

l1Network uint32 = 0
l2Network uint32 = 1

funds     = 100000000
bridgeURL = "http://localhost:8080"
mtHeight  = 32
```

Then run `go build` again.
