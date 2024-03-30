package utils

const (
	MainNetworkId = 0
)

var rollupNetworkId uint

func InitRollupNetworkId(rollupNetWorkId uint) {
	rollupNetworkId = rollupNetWorkId
}

func GetMainNetworkId() uint {
	return MainNetworkId
}

func GetRollupNetworkId() uint {
	return rollupNetworkId
}
