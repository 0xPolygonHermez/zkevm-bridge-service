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
PoEAddr = "0x2279B7A0a67DB372996a5FaB50D91eAA73d2eBe6"
BridgeAddr = "0x0165878A594ca255338adfa4d48449f69242Eb8F"
GlobalExitRootManAddr = "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
MaticAddr = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
L2BridgeAddrs = ["0x9d98deabc42dd696deb9e40b4f1cab7ddbf55988"]
L1ChainID = 1337
`
