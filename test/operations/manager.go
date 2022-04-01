package operations

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"runtime"
	"path"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hermeznetwork/hermez-bridge/bridgectrl"
	"github.com/hermeznetwork/hermez-bridge/bridgectrl/pb"
	"github.com/hermeznetwork/hermez-bridge/db"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-core/encoding"
	"github.com/hermeznetwork/hermez-core/log"
	"github.com/hermeznetwork/hermez-core/test/operations"
	"github.com/hermeznetwork/hermez-core/etherman/smartcontracts/bridge"
	"github.com/hermeznetwork/hermez-core/etherman/smartcontracts/globalexitrootmanager"
	erc20 "github.com/hermeznetwork/hermez-core/etherman/smartcontracts/matic"
	"github.com/hermeznetwork/hermez-bridge/etherman"
)

const (
	l1NetworkURL = "http://localhost:8545"
	l2NetworkURL = "http://localhost:8123"

	poeAddress        = "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
	MaticTokenAddress = "0x5FbDB2315678afecb367f032d93F642f64180aa3" //nolint:gosec
	l1BridgeAddr      = "0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9"
	l2BridgeAddr      = "0x5FbDB2315678afecb367f032d93F642f64180aa3"

	l1AccHexAddress    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	l1AccHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	sequencerAddress    = "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"

	makeCmd = "make"
	cmdDir  = "../.."
)

var (
	dbConfig = pgstorage.NewConfigFromEnv()
	networks = []uint{0,2}
)


// Config is the main Manager configuration.
type Config struct {
	// Arity   uint8
	Storage db.Config
	BT      bridgectrl.Config
}

// Manager controls operations and has knowledge about how to set up and tear
// down a functional environment.
type Manager struct {
	cfg *Config
	ctx context.Context

	storage       db.Storage
	bridgetree    *bridgectrl.BridgeController
	bridgeService pb.BridgeServiceServer
	wait          *Wait
}

// NewManager returns a manager ready to be used and a potential error caused
// during its creation (which can come from the setup of the db connection).
func NewManager(ctx context.Context, cfg *Config) (*Manager, error) {
	// Init database instance
	err := pgstorage.InitOrReset(dbConfig)
	if err != nil {
		return nil, err
	}

	opsman := &Manager{
		cfg:  cfg,
		ctx:  ctx,
		wait: NewWait(),
	}
	//Init storage and mt
	pgst, err := pgstorage.NewPostgresStorage(dbConfig)
	if err != nil {
		return nil, err
	}
	st, err := db.NewStorage(cfg.Storage)
	if err != nil {
		return nil, err
	}
	bt, err := bridgectrl.NewBridgeController(cfg.BT, networks, pgst, pgst)
	if err != nil {
		return nil, err
	}
	bService := bridgectrl.NewBridgeService(pgst, bt)
	opsman.storage = st
	opsman.bridgetree = bt
	opsman.bridgeService = bService

	return opsman, nil
}

// SendL1Deposit sends a deposit from l1 to l2
func (m *Manager) SendL1Deposit(ctx context.Context, tokenAddr common.Address, amount *big.Int,
	destNetwork uint32, destAddr *common.Address,
	) error {
	client, auth, err := initClientConnection(ctx, "l1")
	if err != nil {
		log.Error(1)
		return err
	}
	emptyAddr := common.Address{}
	if tokenAddr == emptyAddr {
		auth.Value = amount
	}
	if destAddr == nil {
		destAddr = &auth.From
	}
	br, err := bridge.NewBridge(common.HexToAddress(l1BridgeAddr), client)
	if err != nil {
		log.Error(2)
		return nil
	}
	tx, err := br.Bridge(auth, tokenAddr, amount, destNetwork, *destAddr)
	if err != nil {
		log.Error(3, err)
		return err
	}

	// wait matic transfer to be mined
	log.Infof("Waiting L1Deposit to be mined")
	const txTimeout = 15 * time.Second
	_, err = m.WaitTxToBeMined(ctx, client, tx.Hash(), txTimeout)
	if err != nil {
		return err
	}
	//Wait to process tx and sync it
	time.Sleep(15 * time.Second)
	return nil
}

// SendL2Deposit sends a deposit from l2 to l1
func (m *Manager) SendL2Deposit(ctx context.Context, tokenAddr common.Address, amount *big.Int,
	destNetwork uint32, destAddr *common.Address,
	) error {
	client, auth, err := initClientConnection(ctx, "l2")
	if err != nil {
		return err
	}
	emptyAddr := common.Address{}
	if tokenAddr == emptyAddr {
		auth.Value = amount
	}
	if destAddr == nil {
		destAddr = &auth.From
	}
	br, err := bridge.NewBridge(common.HexToAddress(l2BridgeAddr), client)
	if err != nil {
		return nil
	}
	tx, err := br.Bridge(auth, tokenAddr, amount, destNetwork, *destAddr)
	if err != nil {
		return err
	}

	// wait matic transfer to be mined
	log.Infof("Waiting tx to be mined")
	const txTimeout = 15 * time.Second
	_, err = m.WaitTxToBeMined(ctx, client, tx.Hash(), txTimeout)
	return err
}

// GetAuth configures and returns an auth object.
func GetAuth(privateKeyStr string, chainID *big.Int) (*bind.TransactOpts, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyStr, "0x"))
	if err != nil {
		return nil, err
	}

	return bind.NewKeyedTransactorWithChainID(privateKey, chainID)
}

// Setup creates all the required components and initializes them according to
// the manager config.
func (m *Manager) Setup() error {
	// Run network container
	err := m.startNetwork()
	if err != nil {
		return err
	}

	// Start prover container
	err = m.startProver()
	if err != nil {
		return err
	}

	//Send funds to hermezCore
	err = m.AddFunds(m.ctx)
	if err != nil {
		return err
	}

	// Run core container
	err = m.startCore()
	if err != nil {
		return err
	}
	time.Sleep(15 * time.Second)

	//Deploy Bridge and GlobalExitRoot Smc
	bridgeAddr, globlaExitRootL2Addr, err := m.DeploySmc(m.ctx)
	if err != nil {
		log.Error("Error deploying", err)
		panic(err)
		return err
	}
	log.Warn(bridgeAddr, globlaExitRootL2Addr)

	time.Sleep(15 * time.Second)

	// Run bridge container
	err = m.startBridge()
	if err != nil {
		return err
	}
	log.Warn("Bridge started without any error")
	time.Sleep(30 * time.Second)
	return nil
}

// AddFunds adds matic and eth to the hermez core wallet.
func (m *Manager) AddFunds(ctx context.Context) error {
	// Eth client
	log.Infof("Connecting to l1")
	client, auth, err := initClientConnection(ctx, "l1")
	if err != nil {
		return err
	}

	// Getting l1 info
	log.Infof("Getting L1 info")
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return err
	}

	// Send some Ether from l1Acc to sequencer acc
	log.Infof("Transferring ETH to the sequencer")
	fromAddress := common.HexToAddress(l1AccHexAddress)
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return err
	}
	const gasLimit = 21000
	toAddress := common.HexToAddress(sequencerAddress)
	ethAmount, _ := big.NewInt(0).SetString("200000000000000000000", encoding.Base10)
	tx := types.NewTransaction(nonce, toAddress, ethAmount, gasLimit, gasPrice, nil)
	signedTx, err := auth.Signer(auth.From, tx)
	if err != nil {
		return err
	}
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return err
	}

	// Wait eth transfer to be mined
	log.Infof("Waiting tx to be mined")
	const txETHTransferTimeout = 5 * time.Second
	_, err = m.WaitTxToBeMined(ctx, client, signedTx.Hash(), txETHTransferTimeout)
	if err != nil {
		return err
	}

	// Create matic maticTokenSC sc instance
	log.Infof("Loading Matic token SC instance")
	maticAddr := common.HexToAddress(MaticTokenAddress)
	maticTokenSC, err := operations.NewToken(maticAddr, client)
	if err != nil {
		return err
	}

	// Send matic to sequencer
	log.Infof("Transferring MATIC tokens to sequencer")
	maticAmount, _ := big.NewInt(0).SetString("200000000000000000000000", encoding.Base10)
	tx, err = maticTokenSC.Transfer(auth, toAddress, maticAmount)
	if err != nil {
		return err
	}

	// wait matic transfer to be mined
	log.Infof("Waiting tx to be mined")
	const txMaticTransferTimeout = 5 * time.Second
	_, err = m.WaitTxToBeMined(ctx, client, tx.Hash(), txMaticTransferTimeout)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) DeploySmc(ctx context.Context) (common.Address, common.Address, error) {
	bridgeByteCode, err := readBytecode("bridge.bin")
	if err != nil {
		log.Error(1, err)
		return common.Address{}, common.Address{}, err
	}
	globalExitRootL2ByteCode, err := readBytecode("globalexitroot.bin")
	if err != nil {
		log.Error(2)
		return common.Address{}, common.Address{}, err
	}

	// Eth client
	log.Info("Connecting to l2")
	client, auth, err := initClientConnection(ctx, "l2")
	if err != nil {
		return common.Address{}, common.Address{}, err
	}

	// Send some Ether from l1Acc to sequencer acc
	log.Infof("Transferring ETH to the sequencer")
	fromAddress := common.HexToAddress(l1AccHexAddress)
	nonce, err := client.NonceAt(ctx, fromAddress, nil)
	// nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		log.Error(7)
		return common.Address{}, common.Address{}, err
	}
	// Simulate future address
	calculatedglobalExitRootL2Addr := crypto.CreateAddress(auth.From, nonce+1)
	// calculatedBridgeAddr := crypto.CreateAddress2()

	//Build deployment bytecode bridgeL2
	globalExitRootPaddedAddr := common.LeftPadBytes(calculatedglobalExitRootL2Addr.Bytes(), 32)
	networkID := "0000000000000000000000000000000000000000000000000000000000000002"
	fullByteCode := bridgeByteCode+networkID+hex.EncodeToString(globalExitRootPaddedAddr)

	bridgeL2Addr, err := m.DeploySC(ctx, client, common.Hex2Bytes(fullByteCode), 80000000, nonce, auth)
	if err != nil {
		log.Error(9, err)
		return common.Address{}, common.Address{}, err
	}

	BridgePaddedAddr := common.LeftPadBytes(bridgeL2Addr.Bytes(), 32)
	fullByteCode = globalExitRootL2ByteCode+hex.EncodeToString(BridgePaddedAddr)
	globalExitRootL2Addr, err := m.DeploySC(ctx, client, common.Hex2Bytes(fullByteCode), 80000000, nonce+1, auth)
	if err != nil {
		log.Error(10, err)
		return common.Address{}, common.Address{}, err
	}
	return bridgeL2Addr, globalExitRootL2Addr, nil
}

// ReadBytecode reads the bytecode of the given contract.
func readBytecode(contractPath string) (string, error) {
	const basePath = "../l2contracts"

	_, currentFilename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("Could not get name of current file")
	}
	fullBasePath := path.Join(path.Dir(currentFilename), basePath)

	content, err := os.ReadFile(path.Join(fullBasePath, contractPath))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (m *Manager) DeploySC(ctx context.Context, client *ethclient.Client, scByteCode []byte, gasLimit, nonce uint64, auth *bind.TransactOpts) (common.Address, error) {
	// we need to use this method to send `TO` field as `NULL`,
	// so the explorer can detect this is a smart contract creation

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       nil,
		Value:    new(big.Int),
		Gas:      gasLimit,
		GasPrice: new(big.Int).SetUint64(1),
		Data:     scByteCode,
	})

	signedTx, err := auth.Signer(auth.From, tx)
	if err != nil {
		log.Error(9.1, err)
		return common.Address{}, err
	}
	log.Infof("sending tx to deploy sc")

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Error(9.2, err)
		return common.Address{}, err
	}
	log.Infof("tx sent: %v", signedTx.Hash().Hex())
	txMinedTimeoutLimit := 60 * time.Second
	r, err := m.WaitTxToBeMined(ctx, client, signedTx.Hash(), txMinedTimeoutLimit)
	if err != nil {
		log.Error(9.3, err)
		return common.Address{}, err
	}
	log.Infof("SC Deployed to address: %v", r.ContractAddress.Hex())

	return r.ContractAddress, nil
}

// Teardown stops all the components.
func Teardown() error {
	err := stopBridge()
	if err != nil {
		return err  
	}

	err = stopCore()
	if err != nil {
		return err
	}

	err = stopProver()
	if err != nil {
		return err
	}

	err = stopNetwork()
	if err != nil {
		return err
	}


	return nil
}

func (m *Manager) startNetwork() error {
	if err := stopNetwork(); err != nil {
		return err
	}
	cmd := exec.Command(makeCmd, "run-network")
	err := runCmd(cmd)
	if err != nil {
		return err
	}
	// Wait network to be ready
	return m.wait.Poll(defaultInterval, defaultDeadline, networkUpCondition)
}

func stopNetwork() error {
	cmd := exec.Command(makeCmd, "stop-network")
	return runCmd(cmd)
}

func (m *Manager) startCore() error {
	if err := stopCore(); err != nil {
		return err
	}
	cmd := exec.Command(makeCmd, "run-core")
	err := runCmd(cmd)
	if err != nil {
		return err
	}
	// Wait core to be ready
	return m.wait.Poll(defaultInterval, defaultDeadline, coreUpCondition)
}

func stopCore() error {
	cmd := exec.Command(makeCmd, "stop-core")
	return runCmd(cmd)
}

func (m *Manager) startProver() error {
	if err := stopProver(); err != nil {
		return err
	}
	cmd := exec.Command(makeCmd, "run-prover")
	err := runCmd(cmd)
	if err != nil {
		return err
	}
	// Wait prover to be ready
	return m.wait.Poll(defaultInterval, defaultDeadline, proverUpCondition)
}

func stopProver() error {
	cmd := exec.Command(makeCmd, "stop-prover")
	return runCmd(cmd)
}

func runCmd(c *exec.Cmd) error {
	c.Dir = cmdDir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func (m *Manager) startBridge() error {
	if err := stopBridge(); err != nil {
		return err
	}
	cmd := exec.Command(makeCmd, "run-bridge")
	err := runCmd(cmd)
	if err != nil {
		return err
	}
	// Wait bridge to be ready
	return m.wait.Poll(defaultInterval, defaultDeadline, bridgeUpCondition)
}

func stopBridge() error {
	cmd := exec.Command(makeCmd, "stop-bridge")
	return runCmd(cmd)
}

//WaitTxToBeMined waits until a tx is mined or forged
func (m *Manager) WaitTxToBeMined(ctx context.Context, client *ethclient.Client, hash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return nil, errors.New("timeout exceed")
		}

		time.Sleep(1 * time.Second)

		_, isPending, err := client.TransactionByHash(ctx, hash)
		if err == ethereum.NotFound {
			continue
		}

		if err != nil {
			return nil, err
		}

		if !isPending {
			r, err := client.TransactionReceipt(ctx, hash)
			if err != nil {
				return nil, err
			}

			if r.Status == types.ReceiptStatusFailed {
				return nil, fmt.Errorf("transaction has failed: %s", string(r.PostState))
			}

			return r, nil
		}
	}
}

// CheckAccountBalance checks the balance by address
func (m *Manager) CheckAccountBalance(ctx context.Context, network string, account *common.Address) (*big.Int, error) {
	client, auth, err := initClientConnection(ctx, network)
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

// CheckAccountTokenBalance checks the balance by address
func (m *Manager) CheckAccountTokenBalance(ctx context.Context, network string, tokenAddr common.Address, account *common.Address) (*big.Int, error) {
	client, auth, err := initClientConnection(ctx, network)
	if err != nil {
		return big.NewInt(0), nil
	}
	if account == nil {
		account = &auth.From
	}
	erc20Token, err := erc20.NewMatic(tokenAddr, client)
	if err != nil {
		return big.NewInt(0), nil
	}
	balance, err := erc20Token.BalanceOf(&bind.CallOpts{Pending: false}, *account)
	if err != nil {
		return big.NewInt(0), nil
	}
	return balance, nil
}

var (
	l2Client *ethclient.Client
	l1Client *ethclient.Client
)

func initClientConnection(ctx context.Context, network string) (*ethclient.Client, *bind.TransactOpts, error) {
	var (
		client *ethclient.Client
		err error
	)
	if network == "l2" || network == "L2" {
		if l2Client != nil {
			client = l2Client
		} else {
			// Eth client
			log.Info("Connecting...")
			client, err = ethclient.Dial(l2NetworkURL)
			if err != nil {
				return nil, nil, err
			}
			l2Client = client
		}
	} else {
		if l1Client != nil {
			client = l1Client
		} else {
			// Eth client
			log.Info("Connecting...")
			client, err = ethclient.Dial(l1NetworkURL)
			if err != nil {
				return nil, nil, err
			}
			l1Client = client
		}
	}

	// Get network chain id
	log.Infof("Getting chainID")
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return nil, nil, err

	}

	// Preparing l1 acc info
	log.Infof("Preparing authorization")
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(l1AccHexPrivateKey, "0x"))
	if err != nil {
		return nil, nil, err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, nil, err
	}
	return client, auth, nil
}

func (m *Manager) GetClaimData(networkID, depositCount uint) ([][bridgectrl.KeyLen]byte, *etherman.GlobalExitRoot, error) {
	return m.bridgetree.GetClaim(networkID, depositCount)
}

func (m *Manager) GetBridgeInfoByDestAddr(ctx context.Context, addr *common.Address) ([]*pb.Deposit, error) {
	_, auth, err := initClientConnection(ctx, "l2")
	if err != nil {
		return []*pb.Deposit{}, err
	}
	if addr == nil {
		addr = &auth.From
	}
	req := pb.GetBridgesRequest{
		EtherAddr: addr.String(),
	}
	res, err := m.bridgeService.GetBridges(ctx, &req)
	if err != nil {
		return []*pb.Deposit{}, err
	}
	return res.Deposits, nil
}

func (m *Manager) SendL1Claim(ctx context.Context, deposit *pb.Deposit, smtProof [][32]byte, globalExitRoot *etherman.GlobalExitRoot) error {
	client, auth, err := initClientConnection(ctx, "l1")
	if err != nil {
		return err
	}
	br, err := bridge.NewBridge(common.HexToAddress(l1BridgeAddr), client)
	if err != nil {
		return err
	}
	amount, _ := new(big.Int).SetString(deposit.Amount, encoding.Base10)
	tx, err := br.Claim(auth, common.HexToAddress(deposit.TokenAddr), amount, deposit.OrigNet, deposit.DestNet,
	common.HexToAddress(deposit.DestAddr), smtProof, uint32(deposit.DepositCnt), globalExitRoot.GlobalExitRootNum,
	globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1])
	if err != nil {
		return err
	}

	// wait matic transfer to be mined
	log.Infof("Waiting tx to be mined")
	const txTimeout = 15 * time.Second
	_, err = m.WaitTxToBeMined(ctx, client, tx.Hash(), txTimeout)
	return err

}

func (m *Manager) SendL2Claim(ctx context.Context, deposit *pb.Deposit, smtProof [][32]byte, globalExitRoot *etherman.GlobalExitRoot) error {
	client, auth, err := initClientConnection(ctx, "l2")
	if err != nil {
		return err
	}
	br, err := bridge.NewBridge(common.HexToAddress(l2BridgeAddr), client)
	if err != nil {
		return err
	}
	amount, _ := new(big.Int).SetString(deposit.Amount, encoding.Base10)
	tx, err := br.Claim(auth, common.HexToAddress(deposit.TokenAddr), amount, deposit.OrigNet, deposit.DestNet,
	common.HexToAddress(deposit.DestAddr), smtProof, uint32(deposit.DepositCnt), globalExitRoot.GlobalExitRootNum,
	globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1])
	if err != nil {
		return err
	}

	// wait matic transfer to be mined
	log.Infof("Waiting tx to be mined")
	const txTimeout = 15 * time.Second
	_, err = m.WaitTxToBeMined(ctx, client, tx.Hash(), txTimeout)
	return err

}

func (m *Manager) GetCurrentGlobalExitRootSynced(ctx context.Context) (*etherman.GlobalExitRoot, error) {
	return m.storage.GetLatestExitRoot(ctx)
}

func (m *Manager) GetCurrentGlobalExitRootFromSmc(ctx context.Context) (*etherman.GlobalExitRoot, error) {
	client, _, err := initClientConnection(ctx, "l1")
	if err != nil {
		return nil, err
	}
	br, err := bridge.NewBridge(common.HexToAddress(l1BridgeAddr), client)
	if err != nil {
		return nil, err
	}
	GlobalExitRootManAddr, err := br.GlobalExitRootManager(&bind.CallOpts{Pending: false})
	if err != nil {
		return nil, err
	}
	globalManager, err := globalexitrootmanager.NewGlobalexitrootmanager(GlobalExitRootManAddr, client)
	if err != nil {
		return nil, err
	}
	gNum, err := globalManager.LastGlobalExitRootNum(&bind.CallOpts{Pending: false})
	if err != nil {
		return nil, err
	}
	gMainnet, err := globalManager.LastMainnetExitRoot(&bind.CallOpts{Pending: false})
	if err != nil {
		return nil, err
	}
	gRollup, err := globalManager.LastRollupExitRoot(&bind.CallOpts{Pending: false})
	if err != nil {
		return nil, err
	}
	result := etherman.GlobalExitRoot {
		GlobalExitRootNum: gNum,
		ExitRoots: []common.Hash{gMainnet, gRollup},
	}
	return &result, nil
}