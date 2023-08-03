package main

import (
	"context"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	clientUtils "github.com/0xPolygonHermez/zkevm-bridge-service/test/client"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/hex"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/state"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	l2BridgeAddr = "0xff0EE8ea08cEf5cb4322777F5CC3E8A584B8A4A0"
	zkevmAddr    = "0x610178dA211FEF7D417bC0e6FeD39F05609AD788"

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

	var smt [mtHeight][32]byte
	for i := 0; i < len(proof.MerkleProof); i++ {
		log.Debug("smt: ", proof.MerkleProof[i])
		smt[i] = common.HexToHash(proof.MerkleProof[i])
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
		LeafType:           uint8(bridgeData.LeafType),
		OriginalNetwork:    uint(bridgeData.OrigNet),
		OriginalAddress:    common.HexToAddress(bridgeData.OrigAddr),
		Amount:             a,
		DestinationNetwork: uint(bridgeData.DestNet),
		DestinationAddress: common.HexToAddress(bridgeData.DestAddr),
		DepositCount:       uint(bridgeData.DepositCnt),
		BlockNumber:        bridgeData.BlockNum,
		NetworkID:          uint(bridgeData.NetworkId),
		TxHash:             common.HexToHash(bridgeData.TxHash),
		Metadata:           metadata,
		ReadyForClaim:      bridgeData.ReadyForClaim,
	}
	tx, err := c.BuildSendClaim(ctx, &e, smt, globalExitRoot, 0, 0, l2GasLimit, auth)
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
	byt, err := state.EncodeTransaction(*tx, state.MaxEffectivePercentage, forkID)
	if err != nil {
		log.Fatal("error: ", err)
	}
	log.Info("forcedBatch content: ", hex.EncodeToHex(byt))

	log.Info("Using address: ", auth.From)

	// Connect to ethereum node
	ethClient, err := ethclient.Dial(l1NetworkURL)
	if err != nil {
		log.Fatalf("error connecting to %s: %+v", l1NetworkURL, err)
	}
	chainID, err := ethClient.ChainID(ctx)
	if err != nil {
		log.Fatal("error getting l1 chainID: ", err)
	}
	auth, err = operations.GetAuth(accHexPrivateKey, chainID.Uint64())
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
	tip, err := zkevm.GetForcedBatchFee(&bind.CallOpts{Pending: false})
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

	err = operations.WaitTxToBeMined(ctx, ethClient, txForcedBatch, miningTimeout*time.Second)
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
