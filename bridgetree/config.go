package bridgetree

// Config is state config
type Config struct {
	// DefaultChainID is the common ChainID to all the sequencers
	DefaultChainID uint64
	// Store is the kind of storage in the bridge tree
	Store string
}
