package nacos

type Config struct {
	NacosUrls          string `mapstructure:"NacosUrls"`
	NamespaceId        string `mapstructure:"NamespaceId"`
	ApplicationName    string `mapstructure:"ApplicationName"`
	ExternalListenAddr string `mapstructure:"ExternalListenAddr"`
}
