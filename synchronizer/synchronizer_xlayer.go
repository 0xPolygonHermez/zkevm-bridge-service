package synchronizer

import (
	"context"
	"math/big"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/estimatetime"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/pushtask"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/jackc/pgx/v4"
)

func (s *ClientSynchronizer) beforeProcessDeposit(deposit *etherman.Deposit) {
	// If the deposit is USDC LxLy message, extract the user address from the metadata
	if deposit.LeafType == uint8(utils.LeafTypeMessage) && utils.IsUSDCContractAddress(deposit.OriginalAddress) {
		deposit.DestContractAddress = deposit.DestinationAddress
		deposit.DestinationAddress, _ = utils.DecodeUSDCBridgeMetadata(deposit.Metadata)
	}
}

func (s *ClientSynchronizer) afterProcessDeposit(deposit *etherman.Deposit, depositID uint64, dbTx pgx.Tx) error {
	// Add the deposit to Redis for L1
	if deposit.NetworkID == 0 {
		err := s.redisStorage.AddBlockDeposit(context.Background(), deposit)
		if err != nil {
			log.Errorf("networkID: %d, failed to add block deposit to Redis, BlockNumber: %d, Deposit: %+v, err: %s", s.networkID, deposit.BlockNumber, deposit, err)
			rollbackErr := s.storage.Rollback(s.ctx, dbTx)
			if rollbackErr != nil {
				log.Errorf("networkID: %d, error rolling back state to store block. BlockNumber: %v, rollbackErr: %v, err: %s",
					s.networkID, deposit.BlockNumber, rollbackErr, err.Error())
				return rollbackErr
			}
			return err
		}
	}

	// Notify FE about a new deposit
	go func() {
		if s.messagePushProducer == nil {
			log.Errorf("kafka push producer is nil, so can't push tx status change msg!")
			return
		}
		if deposit.LeafType != uint8(utils.LeafTypeAsset) {
			if !utils.IsUSDCContractAddress(deposit.OriginalAddress) {
				log.Infof("transaction is not asset, so skip push update change, hash: %v", deposit.TxHash)
				return
			}
		}
		err := s.messagePushProducer.PushTransactionUpdate(&pb.Transaction{
			FromChain:    uint32(deposit.NetworkID),
			ToChain:      uint32(deposit.DestinationNetwork),
			BridgeToken:  deposit.OriginalAddress.Hex(),
			TokenAmount:  deposit.Amount.String(),
			EstimateTime: s.getEstimateTimeForDepositCreated(deposit.NetworkID),
			Time:         uint64(deposit.Time.UnixMilli()),
			TxHash:       deposit.TxHash.String(),
			Id:           depositID,
			Index:        uint64(deposit.DepositCount),
			Status:       uint32(pb.TransactionStatus_TX_CREATED),
			BlockNumber:  deposit.BlockNumber,
			DestAddr:     deposit.DestinationAddress.Hex(),
			FromChainId:  utils.GetChainIdByNetworkId(deposit.NetworkID),
			ToChainId:    utils.GetChainIdByNetworkId(deposit.DestinationNetwork),
			GlobalIndex:  s.getGlobalIndex(deposit).String(),
			LeafType:     uint32(deposit.LeafType),
		})
		if err != nil {
			log.Errorf("PushTransactionUpdate error: %v", err)
		}
	}()
	return nil
}

func (s *ClientSynchronizer) getEstimateTimeForDepositCreated(networkId uint) uint32 {
	if networkId == 0 {
		return estimatetime.GetDefaultCalculator().Get(networkId)
	}
	return uint32(pushtask.GetAvgCommitDuration(s.ctx, s.redisStorage))
}

func (s *ClientSynchronizer) afterProcessClaim(claim *etherman.Claim) error {
	// Notify FE that the tx has been claimed
	go func() {
		if s.messagePushProducer == nil {
			log.Errorf("kafka push producer is nil, so can't push tx status change msg!")
			return
		}

		originNetwork := uint(0)
		if !claim.MainnetFlag {
			originNetwork = uint(claim.RollupIndex + 1)
		}

		// Retrieve deposit transaction info
		deposit, err := s.storage.GetDeposit(s.ctx, claim.Index, originNetwork, nil)
		if err != nil {
			log.Errorf("push message: GetDeposit error: %v", err)
			return
		}
		if deposit.LeafType != uint8(utils.LeafTypeAsset) {
			if !utils.IsUSDCContractAddress(deposit.OriginalAddress) {
				log.Infof("transaction is not asset, so skip push update change, hash: %v", deposit.TxHash)
				return
			}
		}
		err = s.messagePushProducer.PushTransactionUpdate(&pb.Transaction{
			FromChain:   uint32(deposit.NetworkID),
			ToChain:     uint32(deposit.DestinationNetwork),
			TxHash:      deposit.TxHash.String(),
			Index:       uint64(deposit.DepositCount),
			Status:      uint32(pb.TransactionStatus_TX_CLAIMED),
			ClaimTxHash: claim.TxHash.Hex(),
			ClaimTime:   uint64(claim.Time.UnixMilli()),
			DestAddr:    deposit.DestinationAddress.Hex(),
			GlobalIndex: s.getGlobalIndex(deposit).String(),
		})
		if err != nil {
			log.Errorf("PushTransactionUpdate error: %v", err)
		}
	}()
	return nil
}

func (s *ClientSynchronizer) getGlobalIndex(deposit *etherman.Deposit) *big.Int {
	isMainnet := deposit.NetworkID == 0
	rollupIndex := s.rollupID - 1
	return etherman.GenerateGlobalIndex(isMainnet, rollupIndex, deposit.DepositCount)
}
