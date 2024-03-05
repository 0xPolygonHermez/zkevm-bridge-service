package utils

import (
	"github.com/0xPolygonHermez/zkevm-bridge-service/config/businessconfig"
	"github.com/0xPolygonHermez/zkevm-node/log"
)

var standardIdKeyMapper, innerIdKeyMapper map[uint64]uint64

func InnitOkInnerChainIdMapper(cfg businessconfig.Config) {
	standardIdKeyMapper = make(map[uint64]uint64, len(cfg.StandardChainIds))
	innerIdKeyMapper = make(map[uint64]uint64, len(cfg.StandardChainIds))
	if cfg.StandardChainIds == nil {
		log.Infof("inner chain id config is empty, skip init!")
		return
	}
	for i, chainId := range cfg.StandardChainIds {
		innerChainId := cfg.InnerChainIds[i]
		standardIdKeyMapper[chainId] = innerChainId
		innerIdKeyMapper[innerChainId] = chainId
	}
}

func GetStandardChainIdByInnerId(innerChainId uint64) uint64 {
	chainId, found := innerIdKeyMapper[innerChainId]
	if !found {
		return innerChainId
	}
	return chainId
}

func GetInnerChainIdByStandardId(chainId uint64) uint64 {
	innerChainId, found := standardIdKeyMapper[chainId]
	if !found {
		return chainId
	}
	return innerChainId
}
