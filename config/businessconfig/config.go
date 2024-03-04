package businessconfig

type Config struct {
	StandardChainIds []uint64 `mapstructure:"StandardChainIds"`
	InnerChainIds    []uint64 `mapstructure:"InnerChainIds"`
}
