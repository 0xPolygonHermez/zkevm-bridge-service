package utils

import (
	"encoding/hex"
	"math/big"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-node/encoding"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/constraints"
)

func PbToEthermanDeposit(pbDeposit *pb.Deposit) *etherman.Deposit {
	if pbDeposit == nil {
		return nil
	}
	amount, _ := new(big.Int).SetString(pbDeposit.Amount, encoding.Base10)
	return &etherman.Deposit{
		LeafType:           uint8(pbDeposit.LeafType),
		OriginalNetwork:    uint(pbDeposit.OrigNet),
		OriginalAddress:    common.HexToAddress(pbDeposit.OrigAddr),
		Amount:             amount,
		DestinationNetwork: uint(pbDeposit.DestNet),
		DestinationAddress: common.HexToAddress(pbDeposit.DestAddr),
		DepositCount:       uint(pbDeposit.DepositCnt),
		BlockNumber:        pbDeposit.BlockNum,
		NetworkID:          uint(pbDeposit.NetworkId),
		TxHash:             common.HexToHash(pbDeposit.TxHash),
		Metadata:           common.FromHex(pbDeposit.Metadata),
		ReadyForClaim:      pbDeposit.ReadyForClaim,
	}
}

func EthermanDepositToPbTransaction(deposit *etherman.Deposit) *pb.Transaction {
	if deposit == nil {
		return nil
	}

	return &pb.Transaction{
		FromChain:        uint32(deposit.NetworkID),
		ToChain:          uint32(deposit.DestinationNetwork),
		BridgeToken:      deposit.OriginalAddress.Hex(),
		TokenAmount:      deposit.Amount.String(),
		Time:             uint64(deposit.Time.UnixMilli()),
		TxHash:           deposit.TxHash.String(),
		FromChainId:      GetChainIdByNetworkId(deposit.NetworkID),
		ToChainId:        GetChainIdByNetworkId(deposit.DestinationNetwork),
		Id:               deposit.Id,
		Index:            uint64(deposit.DepositCount),
		Metadata:         "0x" + hex.EncodeToString(deposit.Metadata),
		DestAddr:         deposit.DestinationAddress.Hex(),
		LeafType:         uint32(deposit.LeafType),
		BlockNumber:      deposit.BlockNumber,
		DestContractAddr: deposit.DestContractAddress.Hex(),
		OriginalNetwork:  uint32(deposit.OriginalNetwork),
	}
}

// GenerateTraceID generates a random trace ID.
func GenerateTraceID() string {
	return generateRandomString(traceIDLen)
}

func Min[T constraints.Ordered](x, y T) T {
	if x < y {
		return x
	}
	return y
}
