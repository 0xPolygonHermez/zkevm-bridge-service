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
	"github.com/hermeznetwork/hermez-bridge/db"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-core/encoding"
	"github.com/hermeznetwork/hermez-core/log"
	"github.com/hermeznetwork/hermez-core/test/operations"
)

const (
	l1NetworkURL = "http://localhost:8545"
	l2NetworkURL = "http://localhost:8123"

	poeAddress        = "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
	maticTokenAddress = "0x5FbDB2315678afecb367f032d93F642f64180aa3" //nolint:gosec
	bridgeAddress     = "0x0000000000000000000000000000000000000000"

	l1AccHexAddress    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	l1AccHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	sequencerAddress    = "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"

	makeCmd = "make"
	cmdDir  = "../.."
)

var (
	dbConfig = pgstorage.NewConfigFromEnv()
	networks = []uint{1,2}
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

	storage    *db.Storage
	bridgetree *bridgectrl.BridgeController
	wait       *Wait
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
	opsman.storage = &st
	opsman.bridgetree = bt

	return opsman, nil
}

// Storage is a getter for the storage field.
func (m *Manager) Storage() *db.Storage {
	return m.storage
}

// CheckVirtualRoot verifies if the given root is the current root of the
// merkletree for virtual state.
func (m *Manager) CheckGlobalExitRoot(expectedRoot string) error {
	// root, err := m.st.GetStateRoot(m.ctx, true)
	// if err != nil {
	// 	return err
	// }
	// return m.checkRoot(root, expectedRoot)
	return nil
}

// SendL2Txs sends the given L2 txs and waits for them to be consolidated
func (m *Manager) SendL2Txs() error {
	//TODO Send L2

	var currentBatchNumber uint64
	// Wait for sequencer to select txs from pool and propose a new batch
	// Wait for the synchronizer to update state
	err := m.wait.Poll(defaultInterval, defaultDeadline, func() (bool, error) {
		// TODO call hermez core rpc to check current batch
		var latestBatchNumber uint64
		done := latestBatchNumber > currentBatchNumber
		return done, nil
	})
	return err
}

// SendL1Txs sends the given L1 txs and waits for them to be consolidated
func (m *Manager) SendL1Txs() error {
	//TODO Send L2

	var currentBatchNumber uint64
	// Wait for sequencer to select txs from pool and propose a new batch
	// Wait for the synchronizer to update state
	err := m.wait.Poll(defaultInterval, defaultDeadline, func() (bool, error) {
		// TODO call hermez core rpc to check current batch
		var latestBatchNumber uint64
		done := latestBatchNumber > currentBatchNumber
		return done, nil
	})
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
	err = AddFunds(m.ctx)
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
	bridgeAddr, globlaExitRootL2Addr, err := DeploySmc(m.ctx)
	log.Warn(bridgeAddr, globlaExitRootL2Addr)

	time.Sleep(30 * time.Second)

	// Run bridge container
	err = m.startBridge()
	if err != nil {
		return err
	}

	return nil
}

// AddFunds adds matic and eth to the hermez core wallet.
func AddFunds(ctx context.Context) error {
	// Eth client
	log.Infof("Connecting to l1")
	client, err := ethclient.Dial(l1NetworkURL)
	if err != nil {
		return err
	}

	// Get network chain id
	log.Infof("Getting chainID")
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return err
	}

	// Preparing l1 acc info
	log.Infof("Preparing authorization")
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(l1AccHexPrivateKey, "0x"))
	if err != nil {
		return err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
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
	_, err = waitTxToBeMined(ctx, client, signedTx.Hash(), txETHTransferTimeout)
	if err != nil {
		return err
	}

	// Create matic maticTokenSC sc instance
	log.Infof("Loading Matic token SC instance")
	maticAddr := common.HexToAddress(maticTokenAddress)
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
	_, err = waitTxToBeMined(ctx, client, tx.Hash(), txMaticTransferTimeout)
	if err != nil {
		return err
	}
	return nil
}

func DeploySmc(ctx context.Context) (common.Address, common.Address, error) {
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
	client, err := ethclient.Dial(l2NetworkURL)
	if err != nil {
		log.Error(3)
		return common.Address{}, common.Address{}, err
	}
	// Get network chain id
	log.Infof("Getting chainID")
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		log.Error(4)
		return common.Address{}, common.Address{}, err
	}

	// Preparing l1 acc info
	log.Infof("Preparing authorization")
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(l1AccHexPrivateKey, "0x"))
	if err != nil {
		log.Error(5)
		return common.Address{}, common.Address{}, err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Error(6)
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

	bridgeL2Addr, err := deploySC(ctx, client, common.Hex2Bytes(fullByteCode), 80000000, nonce, auth)
	if err != nil {
		log.Error(9, err)
		return common.Address{}, common.Address{}, err
	}

	BridgePaddedAddr := common.LeftPadBytes(bridgeL2Addr.Bytes(), 32)
	fullByteCode = globalExitRootL2ByteCode+hex.EncodeToString(BridgePaddedAddr)
	globalExitRootL2Addr, err := deploySC(ctx, client, common.Hex2Bytes(fullByteCode), 80000000, nonce+1, auth)
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

func deploySC(ctx context.Context, client *ethclient.Client, scByteCode []byte, gasLimit, nonce uint64, auth *bind.TransactOpts) (common.Address, error) {
	

	// log.Infof("reading nonce for account: %v", auth.From.Hex())
	// nonce, err := client.NonceAt(ctx, auth.From, nil)
	// log.Infof("nonce: %v", nonce)
	// chkErr(err)

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
	r, err := waitTxToBeMined(ctx, client, signedTx.Hash(), txMinedTimeoutLimit)
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

func waitTxToBeMined(ctx context.Context, client *ethclient.Client, hash common.Hash, timeout time.Duration) (*types.Receipt, error) {
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
