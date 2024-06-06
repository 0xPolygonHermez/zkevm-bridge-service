//go:build e2e_real_network
// +build e2e_real_network

package e2e

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/client"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	erc20 "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/pol"
	ops "github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type bridge2e2TestConfig struct {
	ConnectionConfig client.Config
	ChainID          int
	TestAddr         string
	TestAddrPrivate  string
}

const (
	// Tipically the time to auto-claim are 15min
	maxTimeToAutoClaim    = 30 * time.Minute
	maxTimeToClaimReady   = 30 * time.Minute
	timeBetweenCheckClaim = 60 * time.Second
	mtHeight              = 32
)

type bridge2e2TestData struct {
	// This client is used to interact with the L1 bridge contract
	L1Client        *utils.Client
	L2Client        *utils.Client
	l1BridgeService *client.RestClient
	cfg             bridge2e2TestConfig
	auth            map[operations.NetworkSID]*bind.TransactOpts
}

type ethBalances struct {
	balanceL1 *big.Int
	balanceL2 *big.Int
}

func readTestConfig() (*bridge2e2TestConfig, error) {
	configFilePath := os.Getenv("BRIDGE_TEST_CONFIG_FILE")
	if configFilePath == "" {
		log.Infof("BRIDGE_TEST_CONFIG_FILE env var not set")
		configFilePath = "../config/bridge_network_e2e/cardona.toml"
	}
	log.Infof("Reading config file from path: ", configFilePath)
	dirName, fileName := filepath.Split(configFilePath)

	fileExtension := strings.TrimPrefix(filepath.Ext(fileName), ".")
	fileNameWithoutExtension := strings.TrimSuffix(fileName, "."+fileExtension)

	viper.AddConfigPath(dirName)
	viper.SetConfigName(fileNameWithoutExtension)
	viper.SetConfigType(fileExtension)
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetEnvPrefix("BRIDGE_TEST")
	err := viper.ReadInConfig()
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		if ok {
			log.Infof("config file not found in path: ", configFilePath)
		} else {
			log.Infof("error reading config file: ", err)
			return nil, err
		}
	}
	decodeHooks := []viper.DecoderConfigOption{
		// this allows arrays to be decoded from env var separated by ",", example: MY_VAR="value1,value2,value3"
		viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(mapstructure.TextUnmarshallerHookFunc(), mapstructure.StringToSliceHookFunc(","))),
	}
	cfg := bridge2e2TestConfig{}
	err = viper.Unmarshal(&cfg, decodeHooks...)
	return &cfg, err
}

func (e *ethBalances) String() string {
	return fmt.Sprintf("Eth Balance: L1: %s, L2: %s", e.balanceL1.String(), e.balanceL2.String())
}

func NewBridge2e2TestData(ctx context.Context, cfg *bridge2e2TestConfig) (*bridge2e2TestData, error) {
	var err error
	if cfg == nil {
		cfg, err = readTestConfig()
		if err != nil {
			return nil, err
		}
	}

	l1Client, err := utils.NewClient(ctx, cfg.ConnectionConfig.L1NodeURL, cfg.ConnectionConfig.L1BridgeAddr)
	if err != nil {
		return nil, err
	}
	l2Client, err := utils.NewClient(ctx, cfg.ConnectionConfig.L2NodeURL, cfg.ConnectionConfig.L2BridgeAddr)
	if err != nil {
		return nil, err
	}
	l1BridgeService := client.NewRestClient(cfg.ConnectionConfig.BridgeURL)

	L1auth, err := l1Client.GetSigner(ctx, cfg.TestAddrPrivate)
	if err != nil {
		return nil, err
	}
	L2auth, err := l2Client.GetSigner(ctx, cfg.TestAddrPrivate)
	if err != nil {
		return nil, err
	}
	return &bridge2e2TestData{
		L1Client:        l1Client,
		L2Client:        l2Client,
		l1BridgeService: l1BridgeService,
		cfg:             *cfg,
		auth: map[operations.NetworkSID]*bind.TransactOpts{
			operations.L1: L1auth,
			operations.L2: L2auth,
		},
	}, nil
}

func getBalance(ctx context.Context, client *utils.Client, privateKey string, account *common.Address) (*big.Int, error) {
	auth, err := client.GetSigner(ctx, privateKey)
	if err != nil {
		return big.NewInt(0), nil
	}

	if account == nil {
		account = &auth.From
	}
	balance, err := client.BalanceAt(ctx, *account, nil)
	if err != nil {
		return big.NewInt(0), nil
	}
	return balance, nil
}

func TestCLaimAlreadyClaimedDepositL2toL1(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)
	checkBridgeServiceIsAliveAndExpectedVersion(t, testData)
	showCurrentStatus(t, ctx, testData)
	require.NoError(t, err)
	ethInitialBalances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	fmt.Println("ETH Balance ", ethInitialBalances.String())
	checkBridgeServiceIsAliveAndExpectedVersion(t, testData)
	txAssetHash := common.HexToHash("0x099252f808424cd7779f8b43038600efd47cdfdde6c077e0c1089ef742aaab36")
	deposit, err := waitDepositByTxHash(ctx, testData, txAssetHash, maxTimeToClaimReady)
	require.NoError(t, err)
	fmt.Println("Deposit: ", deposit)

	err = manualClaimDeposit(ctx, testData, deposit)
	if !isAlreadyClaimedError(err) {
		require.NoError(t, err)
	}
	ethFinalBalances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	fmt.Println("ETH Initial Balance ", ethFinalBalances.String())
	fmt.Println("ETH Final   Balance ", ethFinalBalances.String())
}

func isAlreadyClaimedError(err error) bool {
	exectionRevertedMsg := "execution reverted"
	if rpcErr, ok := err.(rpc.DataError); ok {
		if rpcErr.Error() != exectionRevertedMsg {
			return false
		}
		revertData := rpcErr.ErrorData()

		fmt.Println("Revert Data: ", revertData)
		if revertData == "0x646cf558" {
			fmt.Println("Revert Data: AlreadyClaimed()")
			return true
		}
	}
	return false
}

// ETH L1 -> L2
// Bridge Service (L1): BridgeAsset
//   - Bridge do the autoclaim ( Claim -> L2/Bridge Contract)
//   - Find the autoclaim tx
//   - done
func TestEthTransferL1toL2(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)
	checkBridgeServiceIsAliveAndExpectedVersion(t, testData)

	ethInitialBalances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	fmt.Println("ETH Balance ", ethInitialBalances.String())
	chainL2, err := testData.L2Client.ChainID(ctx)
	require.NoError(t, err)
	fmt.Println("Chain ID L2: ", chainL2.String())
	auth := testData.auth[operations.L1]

	//amount := big.NewInt(12345678)
	//amount := ethInitialBalances.balanceL1.Div(ethInitialBalances.balanceL1, big.NewInt(4))
	amount := big.NewInt(143210000000001234) // 0.14321 ETH
	txAssetHash := assetEthL1ToL2(ctx, testData, t, auth, amount)
	waitToAutoClaim(t, ctx, testData, txAssetHash, maxTimeToAutoClaim)
	ethFinalBalances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	fmt.Println("AFTER ETH Balance ", ethFinalBalances.String())
	//checkFinalBalanceL1toL2(t, ctx, testData, ethInitialBalances, amount)
}

// This case we need to do manually the claim of the asset
func TestEthTransferL2toL1(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)

	checkBridgeServiceIsAliveAndExpectedVersion(t, testData)
	showCurrentStatus(t, ctx, testData)
	ethInitialBalances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	fmt.Println("ETH Balance ", ethInitialBalances.String())
	amount := big.NewInt(12344321)
	txAssetHash := assetEthL2ToL1(ctx, testData, t, amount)
	deposit, err := waitDepositToBeReadyToClaim(ctx, testData, txAssetHash, maxTimeToClaimReady)
	require.NoError(t, err)
	err = manualClaimDeposit(ctx, testData, deposit)
	require.NoError(t, err)
}

func TestERC20TransferL1toL2(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)

	//checkBridgeServiceIsAliveAndExpectedVersion(t, testData)
	//showCurrentStatus(t, ctx, testData)
	tokenAddr, _, err := testData.L1Client.DeployERC20(ctx, "A COIN", "ACO", testData.auth[operations.L1])
	fmt.Println("Token Addr: ", tokenAddr.Hex())
	amountTokens := new(big.Int).SetUint64(1000000000000000000)
	err = testData.L1Client.ApproveERC20(ctx, tokenAddr, testData.cfg.ConnectionConfig.L1BridgeAddr, amountTokens, testData.auth[operations.L1])
	require.NoError(t, err)
	err = testData.L1Client.MintERC20(ctx, tokenAddr, amountTokens, testData.auth[operations.L1])
	require.NoError(t, err)
	erc20Balance, err := getAccountTokenBalance(ctx, testData.auth[operations.L1], testData.L1Client, tokenAddr, nil)
	require.NoError(t, err)
	fmt.Println("ERC20 Balance: ", erc20Balance.String())

}

func getAccountTokenBalance(ctx context.Context, auth *bind.TransactOpts, client *utils.Client, tokenAddr common.Address, account *common.Address) (*big.Int, error) {

	if account == nil {
		account = &auth.From
	}
	erc20Token, err := erc20.NewPol(tokenAddr, client)
	if err != nil {
		return big.NewInt(0), nil
	}
	balance, err := erc20Token.BalanceOf(&bind.CallOpts{Pending: false}, *account)
	if err != nil {
		return big.NewInt(0), nil
	}
	return balance, nil
}

func checkBridgeServiceIsAliveAndExpectedVersion(t *testing.T, testData *bridge2e2TestData) {
	version, err := testData.l1BridgeService.GetVersion()
	require.NoError(t, err)
	require.Equal(t, "v1", version, "bridge service version is not as expected, so I can't execute test")
}

func showCurrentStatus(t *testing.T, ctx context.Context, testData *bridge2e2TestData) {
	chainL2, err := testData.L2Client.ChainID(ctx)
	require.NoError(t, err)
	fmt.Println("Chain ID L2: ", chainL2.String())

	deposits, _, err := testData.l1BridgeService.GetBridges(testData.cfg.TestAddr, 0, 10)
	require.NoError(t, err)
	for _, deposit := range deposits {
		fmt.Println("Deposit: ", deposit)
	}
}

func manualClaimDeposit(ctx context.Context, testData *bridge2e2TestData, deposit *pb.Deposit) error {
	proof, err := testData.l1BridgeService.GetMerkleProof(deposit.NetworkId, deposit.DepositCnt)
	if err != nil {
		log.Fatal("error: ", err)
	}
	log.Debug("deposit: ", deposit)
	log.Debug("mainnetExitRoot: ", proof.MainExitRoot)
	log.Debug("rollupExitRoot: ", proof.RollupExitRoot)

	smtProof := convertMerkleProof(proof.MerkleProof)
	printMkerkleProof(smtProof, "smtProof: ")
	smtRollupProof := convertMerkleProof(proof.RollupMerkleProof)
	printMkerkleProof(smtRollupProof, "smtRollupProof: ")

	ger := &etherman.GlobalExitRoot{
		ExitRoots: []common.Hash{common.HexToHash(proof.MainExitRoot), common.HexToHash(proof.RollupExitRoot)},
	}
	err = testData.L1Client.SendClaim(ctx, deposit, smtProof, smtRollupProof, ger, testData.auth[operations.L1])
	// rollupID := uint(1)
	// globalIndex := generateGlobalIndex(deposit, rollupID)
	// emptyAddr := common.Address{}
	// zeroAmount := big.NewInt(0)
	//_, err = testData.L1Client.Bridge.ClaimAsset(testData.auth[operations.L1], smtProof, smtRollupProof,
	//	globalIndex, common.HexToHash(proof.MainExitRoot), common.HexToHash(proof.RollupExitRoot), deposit.NetworkId, emptyAddr, deposit.DestNet, emptyAddr, zeroAmount, []byte{})
	return err
}

func generateGlobalIndex(deposit *pb.Deposit, rollupID uint) *big.Int {
	mainnetFlag := deposit.NetworkId == 0
	rollupIndex := rollupID - 1
	localExitRootIndex := uint(deposit.DepositCnt)
	globalIndex := etherman.GenerateGlobalIndex(mainnetFlag, rollupIndex, localExitRootIndex)
	return globalIndex
}

func convertMerkleProof(mkProof []string) [mtHeight][32]byte {
	var smtProof [mtHeight][32]byte
	for i := 0; i < len(mkProof); i++ {
		smtProof[i] = common.HexToHash(mkProof[i])
	}
	return smtProof
}

func printMkerkleProof(mkProof [mtHeight][32]byte, title string) {
	for i := 0; i < len(mkProof); i++ {
		fmt.Println(title, "[", i, "]", mkProof[i])
	}
}

func waitDepositToBeReadyToClaim(ctx context.Context, testData *bridge2e2TestData, asssetTxHash common.Hash, timeout time.Duration) (*pb.Deposit, error) {
	startTime := time.Now()
	for true {
		fmt.Println("Waiting to deposit fo assetTx: ", asssetTxHash.Hex(), "...")
		deposits, _, err := testData.l1BridgeService.GetBridges(testData.cfg.TestAddr, 0, 10)
		if err != nil {
			return nil, err
		}

		for _, deposit := range deposits {
			depositHash := common.HexToHash(deposit.TxHash)
			if depositHash == asssetTxHash {
				fmt.Println("Deposit: ", deposit)

				if deposit.ReadyForClaim {
					fmt.Println("Found claim! Claim Is ready  Elapsed time: ", time.Since(startTime))
					return deposit, nil
				}
			}
		}
		if time.Since(startTime) > timeout {
			return nil, fmt.Errorf("Timeout waiting for deposit to be ready to be claimed")
		}
		fmt.Println("Sleeping ", timeBetweenCheckClaim.String(), "Elapsed time: ", time.Since(startTime))
		time.Sleep(timeBetweenCheckClaim)
	}
	return nil, nil
}

func waitDepositByTxHash(ctx context.Context, testData *bridge2e2TestData, asssetTxHash common.Hash, timeout time.Duration) (*pb.Deposit, error) {
	startTime := time.Now()
	for true {
		fmt.Println("Waiting to deposit fo assetTx: ", asssetTxHash.Hex(), "...")
		deposits, _, err := testData.l1BridgeService.GetBridges(testData.cfg.TestAddr, 0, 10)
		if err != nil {
			return nil, err
		}

		for _, deposit := range deposits {
			depositHash := common.HexToHash(deposit.TxHash)
			if depositHash == asssetTxHash {
				fmt.Println("Found Deposit: ", deposit)
				return deposit, nil
			}
		}
		if time.Since(startTime) > timeout {
			return nil, fmt.Errorf("Timeout waiting for deposit  for txHash: %s", asssetTxHash.Hex())
		}
		fmt.Println("Sleeping ", timeBetweenCheckClaim.String(), "Elapsed time: ", time.Since(startTime))
		time.Sleep(timeBetweenCheckClaim)
	}
	return nil, nil
}
func getcurrentBalance(ctx context.Context, testData *bridge2e2TestData) (*ethBalances, error) {
	balanceL1, err := getBalance(ctx, testData.L1Client, testData.cfg.TestAddrPrivate, nil)
	if err != nil {
		return nil, err
	}
	balanceL2, err := getBalance(ctx, testData.L2Client, testData.cfg.TestAddrPrivate, nil)
	if err != nil {
		return nil, err
	}
	result := &ethBalances{
		balanceL1: balanceL1,
		balanceL2: balanceL2,
	}
	return result, nil
}

func checkFinalBalanceL1toL2(t *testing.T, ctx context.Context, testData *bridge2e2TestData, initialBalance *ethBalances, amount *big.Int) {
	ethFinalBalances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)

	finalBalanceL1 := ethFinalBalances.balanceL1
	finalBalanceL2 := ethFinalBalances.balanceL2

	fmt.Println("Final Balance L1: ", finalBalanceL1.String(), " L2:", finalBalanceL2.String())
	require.Equal(t, initialBalance.balanceL1.Sub(initialBalance.balanceL1, amount).String(), finalBalanceL1.String())
	require.Equal(t, initialBalance.balanceL2.Add(initialBalance.balanceL2, amount).String(), finalBalanceL2.String())
}

func assetEthOld(ctx context.Context, testData *bridge2e2TestData, t *testing.T, auth *bind.TransactOpts, amount *big.Int) common.Hash {
	l2NetworkId, err := testData.L2Client.Bridge.NetworkID(nil)
	require.NoError(t, err)
	fmt.Println("L2 Network ID: ", l2NetworkId)

	destAddr := auth.From
	auth.Value = amount
	tx, err := testData.L1Client.Bridge.BridgeAsset(auth, l2NetworkId, destAddr, amount, common.Address{}, true, []byte{})
	require.NoError(t, err)
	fmt.Println("Tx: ", tx.Hash().Hex())
	ops.WaitTxToBeMined(ctx, testData.L1Client.Client, tx, 60*time.Second)
	return tx.Hash()
}

func assetEthL1ToL2(ctx context.Context, testData *bridge2e2TestData, t *testing.T, auth *bind.TransactOpts, amount *big.Int) common.Hash {
	l2NetworkId, err := testData.L2Client.Bridge.NetworkID(nil)
	require.NoError(t, err)

	fmt.Printf("L2 Network ID: %d. Moving %+v from L1 -> L2 (addr=%s)\n", l2NetworkId, amount, auth.From.String())
	txHash, err := assetEthGeneric(ctx, testData.L1Client, l2NetworkId, auth, amount)
	require.NoError(t, err)
	return txHash
}

func assetEthL2ToL1(ctx context.Context, testData *bridge2e2TestData, t *testing.T, amount *big.Int) common.Hash {
	destNetworkId, err := testData.L1Client.Bridge.NetworkID(nil)
	require.NoError(t, err)
	fmt.Printf("L1 Network ID: %d. Moving %+v from L2- > L1 (addr=%s)\n", destNetworkId, amount, testData.auth[operations.L2].From.String())
	txHash, err := assetEthGeneric(ctx, testData.L2Client, destNetworkId, testData.auth[operations.L2], amount)
	require.NoError(t, err)
	return txHash
}

func assetEthGeneric(ctx context.Context, client *utils.Client, destNetwork uint32, auth *bind.TransactOpts, amount *big.Int) (common.Hash, error) {
	destAddr := auth.From
	auth.Value = amount
	tx, err := client.Bridge.BridgeAsset(auth, destNetwork, destAddr, amount, common.Address{}, true, []byte{})
	if err != nil {
		return common.Hash{}, err
	}
	fmt.Println("Tx: ", tx.Hash().Hex())
	err = ops.WaitTxToBeMined(ctx, client.Client, tx, 60*time.Second)
	return tx.Hash(), err
}

func waitToAutoClaim(t *testing.T, ctx context.Context, testData *bridge2e2TestData, asssetTxHash common.Hash, timeout time.Duration) {
	startTime := time.Now()
	for true {
		fmt.Println("Waiting to deposit fo assetTx: ", asssetTxHash.Hex(), "...")
		deposits, _, err := testData.l1BridgeService.GetBridges(testData.cfg.TestAddr, 0, 10)
		require.NoError(t, err)

		for _, deposit := range deposits {
			depositHash := common.HexToHash(deposit.TxHash)
			if depositHash == asssetTxHash {
				fmt.Println("Deposit: ", deposit, " Elapsed time: ", time.Since(startTime))
				claimTxHash := common.HexToHash(deposit.ClaimTxHash)
				emptyHash := common.Hash{}
				if claimTxHash != emptyHash {
					fmt.Println("Found claim! Claim Tx Hash: ", claimTxHash.Hex())
					// The claim from L1 -> L2 is done by the bridge service to L2
					receipt, err := waitTxToBeMinedByTxHash(ctx, testData.L2Client, claimTxHash, 60*time.Second)
					require.NoError(t, err)
					fmt.Println("Receipt: ", receipt, " Elapsed time: ", time.Since(startTime))
					return
				}
			}
		}
		if time.Since(startTime) > timeout {
			require.Fail(t, "Timeout waiting for deposit to be automatically claimed by Bridge Service")
		}
		time.Sleep(5 * time.Second)
	}
}

// WaitMined waits for tx to be mined on the blockchain.
// It stops waiting when the context is canceled.
func waitMinedByTxHash(ctx context.Context, client *utils.Client, txHash common.Hash) (*types.Receipt, error) {
	queryTicker := time.NewTicker(time.Second)
	defer queryTicker.Stop()

	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}

		if errors.Is(err, ethereum.NotFound) {
			log.Debug("Transaction not yet mined")
		} else {
			log.Debug("Receipt retrieval failed", "err", err)
		}

		// Wait for the next round.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}

// WaitTxToBeMined waits until a tx has been mined or the given timeout expires.
func waitTxToBeMinedByTxHash(parentCtx context.Context, client *utils.Client, txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()
	receipt, err := waitMinedByTxHash(ctx, client, txHash)
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	} else if err != nil {
		log.Errorf("error waiting tx %s to be mined: %w", txHash, err)
		return nil, err
	}
	if receipt.Status == types.ReceiptStatusFailed {
		reason := " reverted "
		return nil, fmt.Errorf("transaction has failed, reason: %s, receipt: %+v. txHash:%s", reason, receipt, txHash.Hex())
	}
	log.Debug("Transaction successfully mined: ", txHash)
	return receipt, nil
}
