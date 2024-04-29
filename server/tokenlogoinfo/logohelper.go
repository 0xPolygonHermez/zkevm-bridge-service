package tokenlogoinfo

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

func FillLogoInfos(ctx context.Context, redisStorage redisstorage.RedisStorage, transactionMap map[string][]*pb.Transaction) {
	noCacheTokenMap := make(map[uint32][]string)
	for k, v := range transactionMap {
		logoInfo, err := redisStorage.GetTokenLogoInfo(ctx, k)
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				log.Errorf("get token logo info failed, so use rpc to fetch, chainId: %v, token: %v, error: %v", v[0].FromChainId, v[0].BridgeToken, err)
			}
			log.Infof("token need to use rpc to get logo, token: %v, chainId: %v", v[0].BridgeToken, v[0].FromChainId)
			noCacheTokenMap[v[0].FromChainId] = append(noCacheTokenMap[v[0].FromChainId], v[0].BridgeToken)
			continue
		}
		for _, tx := range v {
			fillOneTxLogoInfo(tx, *logoInfo)
		}
	}
	if len(noCacheTokenMap) == 0 {
		return
	}
	logoParams := buildQueryLogoParams(noCacheTokenMap)
	tokenLogoMap, err := GetClient().GetTokenLogoInfos(logoParams)
	if err != nil {
		log.Errorf("get token logo infos by rpc failed, so skip these tokens")
		return
	}
	if tokenLogoMap == nil {
		log.Infof("get token logo infos, but result is empty, so skip these tokens")
		return
	}
	for k, v := range tokenLogoMap {
		for _, tx := range transactionMap[k] {
			fillOneTxLogoInfo(tx, v)
		}
		err = redisStorage.SetTokenLogoInfo(ctx, k, v)
		if err != nil {
			log.Errorf("failed to set logo info cache for token: %v", v.TokenContractAddress)
		}
	}
}

func buildQueryLogoParams(noCacheTokenMap map[uint32][]string) []*QueryLogoParam {
	var logoParams []*QueryLogoParam
	for k, v := range noCacheTokenMap {
		for _, addr := range v {
			logoParams = append(logoParams, &QueryLogoParam{
				ChainId:              k,
				TokenContractAddress: addr,
			})
		}
	}
	return logoParams
}

func fillOneTxLogoInfo(tx *pb.Transaction, logoInfo TokenLogoInfo) {
	tx.LogoInfo = &pb.TokenLogoInfo{
		Symbol:     logoInfo.TokenSymbol,
		TokenName:  logoInfo.TokenName,
		LogoOssUrl: logoInfo.LogoOssUrl,
		Decimal:    logoInfo.Unit,
	}
}
