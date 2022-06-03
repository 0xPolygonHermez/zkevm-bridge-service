package config

// DefaultValues is the default configuration
const DefaultValues = `
[Log]
Level = "debug"
Outputs = ["stdout"]

[Database]
Database = "postgres"
User = "test_user"
Password = "test_password"
Name = "test_db"
Host = "localhost"
Port = "5433"
MaxConns = 20

[Etherman]
L1URL = "http://localhost"
L2URLs = ["http://localhost"]
PrivateKeyPath = "./test/test.keystore"
PrivateKeyPassword = "testonly"

[Synchronizer]
SyncInterval = "0s"
SyncChunkSize = 100
ForceBatch = false

[BridgeController]
Store = "postgres"
Height = 32

[BridgeServer]
GRPCPort = "9090"
HTTPPort = "8080"

`
