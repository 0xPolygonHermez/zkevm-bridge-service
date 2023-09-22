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

[ClaimTxManager]
Enabled = false
FrequencyToMonitorTxs = "1s"
PrivateKey = {Path = "./test/test.keystore", Password = "testonly"}
RetryInterval = "1s"
RetryNumber = 10
AuthorizedClaimMessageAddresses = []

[Etherman]
L1URL = "http://localhost:8545"
L2URLs = [""]

[Synchronizer]
SyncInterval = "2s"
SyncChunkSize = 100

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
`
