package config

// DefaultValues is the default configuration
const DefaultValues = `
[Log]
Level = "debug"
Outputs = ["stdout"]

[SyncDB]
Database = "postgres"
User = "test_user"
Password = "test_password"
Name = "test_db"
Host = "zkevm-bridge-db"
Port = "5432"
MaxConns = 20

[Etherman]
L1URL = "http://localhost:8545"
L2URLs = ["http://localhost:8123"]
PrivateKeyPath = "./test/test.keystore"
PrivateKeyPassword = "testonly"

[Synchronizer]
SyncInterval = "2s"
SyncChunkSize = 100
GrpcURL = "localhost:61090"

[BridgeController]
Store = "postgres"
Height = 32

[BridgeServer]
GRPCPort = "9090"
HTTPPort = "8080"
DefaultPageLimit = 25
CacheSize = 100000
MaxPageLimit = 100
BridgeVersion = "v1"
    [BridgeServer.DB]
    Database = "postgres"
    User = "test_user"
    Password = "test_password"
    Name = "test_db"
    Host = "zkevm-bridge-db"
    Port = "5432"
    MaxConns = 20

[NetworkConfig]
GenBlockNumber = 1
PoEAddr = "0x8A791620dd6260079BF849Dc5567aDC3F2FdC318"
BridgeAddr = "0x60627AC8Ba44F4438186B4bCD5F1cb5E794e19fe"
GlobalExitRootManAddr = "0xa513E6E4b8f2a923D98304ec87F64353C4D5C853"
MaticAddr = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
L2BridgeAddrs = ["0xd0a3d58d135e2ee795dFB26ec150D339394254B9"]
L1ChainID = 1337
`
