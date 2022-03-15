package bridgectrl

// Config is state config
type Config struct {
	// Store is the kind of storage in the bridge tree
	Store string
	// Height is the depth of the merkle tree
	Height uint8
}
