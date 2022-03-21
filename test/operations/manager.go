package operations

import (
	"context"
	"math/big"
	"os"
	"os/exec"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/bridgetree"
	"github.com/hermeznetwork/hermez-bridge/db"
)

const (
	l1NetworkURL = "http://localhost:8545"
	l2NetworkURL = "http://localhost:8123"

	poeAddress        = "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
	maticTokenAddress = "0x5FbDB2315678afecb367f032d93F642f64180aa3" //nolint:gosec
	bridgeAddress     = "0x0000000000000000000000000000000000000000"

	l1AccHexAddress    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	l1AccHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	makeCmd = "make"
	cmdDir  = "../.."
)

var (
	dbConfig = pgstorage.NewConfigFromEnv()
	networks = []uint64{1,2}
)


// Config is the main Manager configuration.
type Config struct {
	// Arity   uint8
	Storage db.Config
	BT      bridgetree.Config
}

// Manager controls operations and has knowledge about how to set up and tear
// down a functional environment.
type Manager struct {
	cfg *Config
	ctx context.Context

	storage    *db.Storage
	bridgetree *bridgetree.BridgeTree
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
	bt, err := bridgetree.NewBridgeTree(cfg.BT, networks, pgst, pgst)
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

	// Run core container
	err = m.startCore()
	if err != nil {
		return err
	}

	// Run bridge container
	err = m.startBridge()
	if err != nil {
		return err
	}

	return nil
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
