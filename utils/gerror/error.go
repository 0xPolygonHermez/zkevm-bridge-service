package gerror

import "errors"

var (
	// ErrStorageNotFound is used when the object is not found in the storage
	ErrStorageNotFound = errors.New("not found in the Storage")
	// ErrStorageNotRegister is used when the object is not found in the synchronizer
	ErrStorageNotRegister = errors.New("not registered storage")
	// ErrNilDBTransaction indicates the db transaction has not been properly initialized
	ErrNilDBTransaction = errors.New("database transaction not properly initialized")
	// ErrRestServerHealth indicates the health check of rest server failed
	ErrRestServerHealth = errors.New("not ready for the rest server")
	// ErrDepositNotSynced is used when the deposit is not synchronized in nodes
	ErrDepositNotSynced = errors.New("not synchronized deposit")
	// ErrNetworkNotRegister is used when the networkID is not registered in the bridge
	ErrNetworkNotRegister = errors.New("not registered network")
)
