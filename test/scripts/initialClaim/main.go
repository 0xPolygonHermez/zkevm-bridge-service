package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/0xPolygon/zkevm-bridge-service/etherman"
	"github.com/0xPolygon/zkevm-bridge-service/etherman/smartcontracts/polygonrollupmanager"
	"github.com/0xPolygon/zkevm-bridge-service/etherman/smartcontracts/polygonzkevm"
	"github.com/0xPolygon/zkevm-bridge-service/hex"
	"github.com/0xPolygon/zkevm-bridge-service/log"
	clientUtils "github.com/0xPolygon/zkevm-bridge-service/test/client"
	"github.com/0xPolygon/zkevm-bridge-service/utils"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	l2BridgeAddr      = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
	zkevmAddr         = "0x8dAF17A20c9DBA35f005b6324F493785D239719d"
	rollupManagerAddr = "0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"

	accHexAddress    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	accHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	l1NetworkURL     = "http://localhost:8545"
	l2NetworkURL     = "http://localhost:8123"
	bridgeURL        = "http://localhost:8080"

	forkID = 4

	l2GasLimit = 1000000

	mtHeight      = 32
	miningTimeout = 180
)

const (
	// FORKID_DRAGONFRUIT is the fork id 5
	FORKID_DRAGONFRUIT = 5

	ether155V = 27

	maxEffectivePercentage uint8 = 255
)

func main() {
	ctx := context.Background()
	c, err := utils.NewClient(ctx, l2NetworkURL, common.HexToAddress(l2BridgeAddr))
	if err != nil {
		log.Fatal("Error: ", err)
	}
	auth, err := c.GetSigner(ctx, accHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// Get Claim data
	cfg := clientUtils.Config{
		L1NodeURL:    l2NetworkURL,
		L2NodeURL:    l2NetworkURL,
		BridgeURL:    bridgeURL,
		L2BridgeAddr: common.HexToAddress(l2BridgeAddr),
	}
	client, err := clientUtils.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	deposits, _, err := client.GetBridges(accHexAddress, 0, 10) //nolint
	if err != nil {
		log.Fatal("Error: ", err)
	}
	bridgeData := deposits[0]
	proof, err := client.GetMerkleProof(deposits[0].NetworkId, deposits[0].DepositCnt)
	if err != nil {
		log.Fatal("error: ", err)
	}
	log.Debug("bridge: ", bridgeData)
	log.Debug("mainnetExitRoot: ", proof.MainExitRoot)
	log.Debug("rollupExitRoot: ", proof.RollupExitRoot)

	var smtProof, smtRollupProof [mtHeight][32]byte
	for i := 0; i < len(proof.MerkleProof); i++ {
		log.Debug("smtProof: ", proof.MerkleProof[i])
		smtProof[i] = common.HexToHash(proof.MerkleProof[i])
		log.Debug("smtRollupProof: ", proof.MerkleProof[i])
		smtRollupProof[i] = common.HexToHash(proof.RollupMerkleProof[i])
	}
	globalExitRoot := &etherman.GlobalExitRoot{
		ExitRoots: []common.Hash{common.HexToHash(proof.MainExitRoot), common.HexToHash(proof.RollupExitRoot)},
	}
	log.Info("Sending claim tx...")
	a, _ := big.NewInt(0).SetString(bridgeData.Amount, 0)
	metadata, err := hex.DecodeHex(bridgeData.Metadata)
	if err != nil {
		log.Fatal("error converting metadata to bytes. Error: ", err)
	}
	e := etherman.Deposit{
		LeafType:           uint8(bridgeData.LeafType), // nolint:gosec
		OriginalNetwork:    bridgeData.OrigNet,
		OriginalAddress:    common.HexToAddress(bridgeData.OrigAddr),
		Amount:             a,
		DestinationNetwork: bridgeData.DestNet,
		DestinationAddress: common.HexToAddress(bridgeData.DestAddr),
		DepositCount:       bridgeData.DepositCnt,
		BlockNumber:        bridgeData.BlockNum,
		NetworkID:          bridgeData.NetworkId,
		TxHash:             common.HexToHash(bridgeData.TxHash),
		Metadata:           metadata,
		ReadyForClaim:      bridgeData.ReadyForClaim,
	}
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(l1NetworkURL)
	if err != nil {
		log.Fatalf("error connecting to %s: %+v", l1NetworkURL, err)
	}
	polygonRollupManagerAddress := common.HexToAddress(rollupManagerAddr)
	polygonRollupManager, err := polygonrollupmanager.NewPolygonrollupmanager(polygonRollupManagerAddress, ethClient)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	tx, err := c.BuildSendClaim(ctx, &e, smtProof, smtRollupProof, globalExitRoot, 0, 0, l2GasLimit, auth)
	if err != nil {
		log.Fatal("error: ", err)
	}
	log.Info("L2 tx.Nonce: ", tx.Nonce())
	log.Info("L2 tx.GasPrice: ", tx.GasPrice())
	log.Info("L2 tx.Gas: ", tx.Gas())
	log.Info("L2 tx.Hash: ", tx.Hash())
	b, err := tx.MarshalBinary()
	if err != nil {
		log.Fatal("error: ", err)
	}
	encoded := hex.EncodeToHex(b)
	log.Info("tx encoded: ", encoded)
	byt, err := EncodeTransaction(tx, maxEffectivePercentage, forkID)
	if err != nil {
		log.Fatal("error: ", err)
	}
	log.Info("forcedBatch content: ", hex.EncodeToHex(byt))

	log.Info("Using address: ", auth.From)

	chainID, err := ethClient.ChainID(ctx)
	if err != nil {
		log.Fatal("error getting l1 chainID: ", err)
	}
	auth, err = GetAuth(accHexPrivateKey, chainID.Uint64())
	if err != nil {
		log.Fatal("error: ", err)
	}
	// Create smc client
	zkevmAddress := common.HexToAddress(zkevmAddr)
	zkevm, err := polygonzkevm.NewPolygonzkevm(zkevmAddress, ethClient)
	if err != nil {
		log.Fatal("error: ", err)
	}
	num, err := zkevm.LastForceBatch(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal("error getting lastForBatch number. Error : ", err)
	}
	log.Info("Number of forceBatches in the smc: ", num)

	currentBlock, err := ethClient.BlockByNumber(ctx, nil)
	if err != nil {
		log.Fatal("error getting blockByNumber. Error: ", err)
	}
	log.Debug("currentBlock.Time(): ", currentBlock.Time())

	// Get tip
	tip, err := polygonRollupManager.GetForcedBatchFee(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal("error getting tip. Error: ", err)
	}
	// Send forceBatch
	txForcedBatch, err := zkevm.ForceBatch(auth, byt, tip)
	if err != nil {
		log.Fatal("error sending forceBatch. Error: ", err)
	}

	log.Info("TxHash: ", txForcedBatch.Hash())

	time.Sleep(1 * time.Second)

	err = utils.WaitTxToBeMined(ctx, ethClient, txForcedBatch, miningTimeout*time.Second)
	if err != nil {
		log.Fatal("error: ", err)
	}

	query := ethereum.FilterQuery{
		FromBlock: currentBlock.Number(),
		Addresses: []common.Address{zkevmAddress},
	}
	logs, err := ethClient.FilterLogs(ctx, query)
	if err != nil {
		log.Fatal("error: ", err)
	}
	for _, vLog := range logs {
		fb, err := zkevm.ParseForceBatch(vLog)
		if err == nil {
			log.Debugf("log decoded: %+v", fb)
			var ger common.Hash = fb.LastGlobalExitRoot
			log.Info("GlobalExitRoot: ", ger)
			log.Info("Transactions: ", common.Bytes2Hex(fb.Transactions))
			fullBlock, err := ethClient.BlockByHash(ctx, vLog.BlockHash)
			if err != nil {
				log.Fatal("error getting hashParent. BlockNumber: %d. Error: %v", vLog.BlockNumber, err)
			}
			log.Info("MinForcedTimestamp: ", fullBlock.Time())
		}
	}
	log.Info("Success!!!!")
}

// EncodeTransactions RLP encodes the given transactions
func EncodeTransactions(txs []*types.Transaction, effectivePercentages []uint8, forkID uint64) ([]byte, error) {
	var batchL2Data []byte

	for i, tx := range txs {
		txData, err := prepareRPLTxData(tx)
		if err != nil {
			return nil, err
		}
		batchL2Data = append(batchL2Data, txData...)

		if forkID >= FORKID_DRAGONFRUIT {
			if i >= len(effectivePercentages) {
				return nil, fmt.Errorf("missing effective percentage for transaction %d", i)
			}
			effectivePercentageAsHex, err := hex.DecodeHex(fmt.Sprintf("%x", effectivePercentages[i]))
			if err != nil {
				return nil, err
			}
			batchL2Data = append(batchL2Data, effectivePercentageAsHex...)
		}
	}

	return batchL2Data, nil
}

func prepareRPLTxData(tx *types.Transaction) ([]byte, error) {
	v, r, s := tx.RawSignatureValues()
	sign := 1 - (v.Uint64() & 1)

	nonce, gasPrice, gas, to, value, data, chainID := tx.Nonce(), tx.GasPrice(), tx.Gas(), tx.To(), tx.Value(), tx.Data(), tx.ChainId()

	rlpFieldsToEncode := []interface{}{
		nonce,
		gasPrice,
		gas,
		to,
		value,
		data,
	}

	if !IsPreEIP155Tx(tx) {
		rlpFieldsToEncode = append(rlpFieldsToEncode, chainID)
		rlpFieldsToEncode = append(rlpFieldsToEncode, uint(0))
		rlpFieldsToEncode = append(rlpFieldsToEncode, uint(0))
	}

	txCodedRlp, err := rlp.EncodeToBytes(rlpFieldsToEncode)
	if err != nil {
		return nil, err
	}

	newV := new(big.Int).Add(big.NewInt(ether155V), big.NewInt(int64(sign))) // nolint:gosec
	newRPadded := fmt.Sprintf("%064s", r.Text(hex.Base))
	newSPadded := fmt.Sprintf("%064s", s.Text(hex.Base))
	newVPadded := fmt.Sprintf("%02s", newV.Text(hex.Base))
	txData, err := hex.DecodeString(hex.EncodeToString(txCodedRlp) + newRPadded + newSPadded + newVPadded)
	if err != nil {
		return nil, err
	}
	return txData, nil
}

// IsPreEIP155Tx checks if the tx is a tx that has a chainID as zero and
// V field is either 27 or 28
func IsPreEIP155Tx(tx *types.Transaction) bool {
	v, _, _ := tx.RawSignatureValues()
	return tx.ChainId().Uint64() == 0 && (v.Uint64() == 27 || v.Uint64() == 28)
}

// EncodeTransaction RLP encodes the given transaction
func EncodeTransaction(tx *types.Transaction, effectivePercentage uint8, forkID uint64) ([]byte, error) {
	return EncodeTransactions([]*types.Transaction{tx}, []uint8{effectivePercentage}, forkID)
}

// GetAuth configures and returns an auth object.
func GetAuth(privateKeyStr string, chainID uint64) (*bind.TransactOpts, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyStr, "0x"))
	if err != nil {
		return nil, err
	}

	return bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(0).SetUint64(chainID))
}
