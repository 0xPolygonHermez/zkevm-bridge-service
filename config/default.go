package config

// DefaultValues is the default configuration
const DefaultValues = `
[Log]
Level = "debug"
Outputs = ["stdout"]

[Database]
User = "test_user"
Password = "test_password"
Name = "test_db"
Host = "localhost"
Port = "5432"

[Etherman]
URL = "http://localhost"
PrivateKeyPath = "./test/test.keystore"
PrivateKeyPassword = "testonly"

[Synchronizer]
SyncInterval = "0s"
SyncChunkSize = 100

`
