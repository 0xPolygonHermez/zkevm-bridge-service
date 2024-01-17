package utils

type NetworkChainIdMapping struct {
	chainIDs map[uint]uint32
}

var networkChainIdMapping NetworkChainIdMapping

func InitChainIdManager(networks []uint, chainIds []uint) {
	var chainIDs = make(map[uint]uint32)
	for i, network := range networks {
		chainIDs[network] = uint32(chainIds[i])
	}
	networkChainIdMapping = NetworkChainIdMapping{
		chainIDs: chainIDs,
	}
}

func GetChainIdByNetworkId(networkId uint) uint32 {
	return networkChainIdMapping.chainIDs[networkId]
}
