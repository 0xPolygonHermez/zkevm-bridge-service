package operations

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/encoding"
	erc20 "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/pol"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmglobalexitroot"
	"github.com/0xPolygonHermez/zkevm-node/test/contracts/bin/ERC20"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// NetworkSID is used to identify the network.
type NetworkSID string

// NetworkSID constants
const (
	L1  NetworkSID = "l1"
	L2  NetworkSID = "l2"
	L22 NetworkSID = "l22"

	waitRootSyncDeadline = 120 * time.Second
)

const (
	// PolTokenAddress token address
	PolTokenAddress = "0x5FbDB2315678afecb367f032d93F642f64180aa3" //nolint:gosec
	l1BridgeAddr    = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
	l2BridgeAddr    = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"

	l1AccHexAddress = "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"

	sequencerAddress = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

	makeCmd = "make"
	cmdDir  = "../.."

	mtHeight = 32
	rollupID = 1
)

var accHexPrivateKeys = map[NetworkSID]string{
	L1:  "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a", //0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC
	L2:  "0xdfd01798f92667dbf91df722434e8fbe96af0211d4d1b82bbbbc8f1def7a814f", //0xc949254d682d8c9ad5682521675b8f43b102aec4
	L22: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", //0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
}

// Config is the main Manager configuration.
type Config struct {
	L1NetworkURL string
	L2NetworkURL string
	L2NetworkID  uint
	Storage      db.Config
	BT           bridgectrl.Config
	BS           server.Config
}

// Manager controls operations and has knowledge about how to set up and tear
// down a functional environment.
type Manager struct {
	cfg *Config
	ctx context.Context

	storage       StorageInterface
	bridgetree    *bridgectrl.BridgeController
	bridgeService BridgeServiceInterface

	clients map[NetworkSID]*utils.Client
}

// NewManager returns a manager ready to be used and a potential error caused
// during its creation (which can come from the setup of the db connection).
func NewManager(ctx context.Context, cfg *Config) (*Manager, error) {
	opsman := &Manager{
		cfg: cfg,
		ctx: ctx,
	}

	pgst, err := pgstorage.NewPostgresStorage(pgstorage.Config{
		Name:     cfg.Storage.Name,
		User:     cfg.Storage.User,
		Password: cfg.Storage.Password,
		Host:     cfg.Storage.Host,
		Port:     cfg.Storage.Port,
		MaxConns: cfg.Storage.MaxConns,
	})
	if err != nil {
		return nil, err
	}
	st, err := db.NewStorage(cfg.Storage)
	if err != nil {
		return nil, err
	}
	bt, err := bridgectrl.NewBridgeController(ctx, cfg.BT, []uint{0, cfg.L2NetworkID}, pgst)
	if err != nil {
		return nil, err
	}
	l1Client, err := utils.NewClient(ctx, cfg.L1NetworkURL, common.HexToAddress(l1BridgeAddr))
	if err != nil {
		return nil, err
	}
	l2Client, err := utils.NewClient(ctx, cfg.L2NetworkURL, common.HexToAddress(l2BridgeAddr))
	if err != nil {
		return nil, err
	}
	bService := server.NewBridgeService(cfg.BS, cfg.BT.Height, []uint{0, cfg.L2NetworkID}, pgst)
	opsman.storage = st.(StorageInterface)
	opsman.bridgetree = bt
	opsman.bridgeService = bService
	opsman.clients = make(map[NetworkSID]*utils.Client)
	opsman.clients[L1] = l1Client
	opsman.clients[L2] = l2Client
	return opsman, err
}

// CheckClaim checks if the claim is already in the network
func (m *Manager) CheckClaim(ctx context.Context, deposit *pb.Deposit) error {
	return operations.Poll(defaultInterval, defaultDeadline, func() (bool, error) {
		return m.claimChecker(ctx, deposit)
	})
}

func (m *Manager) claimChecker(ctx context.Context, deposit *pb.Deposit) (bool, error) {
	// Check that claim exist on GetClaims endpoint
	req := pb.GetClaimsRequest{
		DestAddr: deposit.DestAddr,
	}
	claims, err := m.bridgeService.GetClaims(ctx, &req)
	if err != nil {
		return false, err
	}
	idx, succ := big.NewInt(0).SetString(deposit.GlobalIndex, 10) //nolint:gomnd
	if !succ {
		return false, errors.New("error setting big int")
	}
	mainnetFlag, rollupIndex, _, err := etherman.DecodeGlobalIndex(idx)
	if err != nil {
		return false, err
	}
	claimFound := false
	var claimTxHash string
	for _, c := range claims.Claims {
		if c.Index == deposit.DepositCnt && c.MainnetFlag == mainnetFlag && c.RollupIndex == rollupIndex {
			log.Debugf("deposit claimed with hash: %s", c.TxHash)
			claimFound = true
			claimTxHash = c.TxHash
			break
		}
	}
	if !claimFound {
		return false, nil
	}

	// Check that claim tx has been added on GetBridges response
	reqB := &pb.GetBridgesRequest{
		DestAddr: deposit.DestAddr,
	}
	bridges, err := m.bridgeService.GetBridges(ctx, reqB)
	if err != nil {
		return false, err
	}
	claimFound = false
	for _, d := range bridges.Deposits {
		dIdx, succ := big.NewInt(0).SetString(deposit.GlobalIndex, 10) //nolint:gomnd
		if !succ {
			return false, errors.New("error setting big int")
		}
		dMainnetFlag, dRollupIndex, _, err := etherman.DecodeGlobalIndex(dIdx)
		if err != nil {
			return false, err
		}
		if d.DepositCnt == deposit.DepositCnt && dMainnetFlag == mainnetFlag && dRollupIndex == rollupIndex {
			if d.ClaimTxHash == claimTxHash {
				claimFound = true
				break
			} else {
				return false, errors.New("claim tx not linked to the deposit")
			}
		}
	}
	return claimFound, nil
}

// CustomCheckClaim checks if the claim is already in the L2 network.
func (m *Manager) CustomCheckClaim(ctx context.Context, deposit *pb.Deposit, interval, deadline time.Duration) error {
	return operations.Poll(interval, deadline, func() (bool, error) {
		return m.claimChecker(ctx, deposit)
	})
}

// GetNumberClaims get the number of claim events synced
func (m *Manager) GetNumberClaims(ctx context.Context, destAddr string) (int, error) {
	const limit = 100
	claims, err := m.storage.GetClaims(ctx, destAddr, limit, 0, nil)
	if err != nil {
		return 0, err
	}
	return len(claims), nil
}

// SendL1Deposit sends a deposit from l1 to l2.
func (m *Manager) SendL1Deposit(ctx context.Context, tokenAddr common.Address, amount *big.Int, destNetwork uint32, destAddr *common.Address) error {
	client := m.clients[L1]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[L1])
	if err != nil {
		return err
	}

	orgExitRoot, err := m.storage.GetLatestExitRoot(ctx, 0, nil)
	if err != nil && err != gerror.ErrStorageNotFound {
		return err
	}

	err = client.SendBridgeAsset(ctx, tokenAddr, amount, destNetwork, destAddr, []byte{}, auth)
	if err != nil {
		return err
	}

	// sync for new exit root
	return m.WaitExitRootToBeSynced(ctx, orgExitRoot, uint(destNetwork))
}

// SendMultipleL1Deposit sends a deposit from l1 to l2.
func (m *Manager) SendMultipleL1Deposit(ctx context.Context, tokenAddr common.Address, amount *big.Int,
	destNetwork uint32, destAddr *common.Address, numberDeposits int,
) error {
	if numberDeposits == 0 {
		return fmt.Errorf("error: numberDeposits is 0")
	}
	client := m.clients[L1]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[L1])
	if err != nil {
		log.Error("error getting signer: ", err)
		return err
	}

	for i := 0; i < numberDeposits; i++ {
		err = client.SendBridgeAsset(ctx, tokenAddr, big.NewInt(0).Add(amount, big.NewInt(int64(i))), destNetwork, destAddr, []byte{}, auth)
		if err != nil {
			log.Error("error sending bridge asset: ", err)
			return err
		}
	}
	return nil
}

// SendL2Deposit sends a deposit from l2 to l1.
func (m *Manager) SendL2Deposit(ctx context.Context, tokenAddr common.Address, amount *big.Int, destNetwork uint32, destAddr *common.Address, l2 NetworkSID) error {
	client := m.clients[L2]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[l2])
	if err != nil {
		return err
	}
	networkID, err := client.Bridge.NetworkID(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Error("error getting networkID: ", networkID)
		return err
	}
	orgExitRoot, err := m.storage.GetLatestExitRoot(ctx, uint(networkID), nil)
	if err != nil && err != gerror.ErrStorageNotFound {
		return err
	}

	err = client.SendBridgeAsset(ctx, tokenAddr, amount, destNetwork, destAddr, []byte{}, auth)
	if err != nil {
		return err
	}

	// sync for new exit root
	return m.WaitExitRootToBeSynced(ctx, orgExitRoot, uint(destNetwork))
}

// SendL1BridgeMessage bridges a message from l1 to l2.
func (m *Manager) SendL1BridgeMessage(ctx context.Context, destAddr common.Address, destNetwork uint32, amount *big.Int, metadata []byte, privKey *string) error {
	client := m.clients[L1]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[L1])
	if err != nil {
		return err
	}
	if privKey != nil {
		auth, err = client.GetSigner(ctx, *privKey)
		if err != nil {
			return err
		}
	}

	orgExitRoot, err := m.storage.GetLatestExitRoot(ctx, 0, nil)
	if err != nil && err != gerror.ErrStorageNotFound {
		return err
	}

	auth.Value = amount
	err = client.SendBridgeMessage(ctx, destNetwork, destAddr, metadata, auth)
	if err != nil {
		return err
	}

	// sync for new exit root
	return m.WaitExitRootToBeSynced(ctx, orgExitRoot, uint(destNetwork))
}

// SendL2BridgeMessage bridges a message from l2 to l1.
func (m *Manager) SendL2BridgeMessage(ctx context.Context, destAddr common.Address, destNetwork uint32, amount *big.Int, metadata []byte) error {
	client := m.clients[L2]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[L2])
	if err != nil {
		return err
	}

	networkID, err := client.Bridge.NetworkID(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Error("error getting networkID: ", networkID)
		return err
	}

	orgExitRoot, err := m.storage.GetLatestExitRoot(ctx, uint(networkID), nil)
	if err != nil && err != gerror.ErrStorageNotFound {
		return err
	}

	auth.Value = amount
	err = client.SendBridgeMessage(ctx, destNetwork, destAddr, metadata, auth)
	if err != nil {
		return err
	}

	// sync for new exit root
	return m.WaitExitRootToBeSynced(ctx, orgExitRoot, uint(destNetwork))
}

// Setup creates all the required components and initializes them according to
// the manager config.
func (m *Manager) Setup() error {
	// Run network container
	err := m.startNetwork()
	if err != nil {
		log.Error("network start failed")
		return err
	}
	const t time.Duration = 3
	time.Sleep(t * time.Second)

	// Start prover container
	err = m.startProver()
	if err != nil {
		log.Error("prover start failed")
		return err
	}

	//Send funds to zkevm node
	err = m.AddFunds(m.ctx)
	if err != nil {
		log.Error("addfunds failed")
		return err
	}

	// Run zkevm node container
	err = m.startZKEVMNode()
	if err != nil {
		log.Error("zkevm node start failed")
		return err
	}
	//Wait for set the genesis and sync
	time.Sleep(t * time.Second)

	// Run bridge container
	err = StartBridge()
	if err != nil {
		log.Error("bridge start failed")
		// return err
	}

	//Wait for sync
	const t2 time.Duration = 5
	time.Sleep(t2 * time.Second)

	return nil
}

// AddFunds adds pol and eth to the zkevm node wallet.
func (m *Manager) AddFunds(ctx context.Context) error {
	// Eth client
	log.Infof("Connecting to l1")
	client := m.clients[L1]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[L1])
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
	err = WaitTxToBeMined(ctx, client.Client, signedTx, txETHTransferTimeout)
	if err != nil {
		return err
	}

	// Create pol polTokenSC sc instance
	log.Infof("Loading pol token SC instance")
	polAddr := common.HexToAddress(PolTokenAddress)
	polTokenSC, err := operations.NewToken(polAddr, client)
	if err != nil {
		return err
	}

	// Send pol to sequencer
	log.Infof("Transferring pol tokens to sequencer")
	polAmount, _ := big.NewInt(0).SetString("200000000000000000000000", encoding.Base10)
	tx, err = polTokenSC.Transfer(auth, toAddress, polAmount)
	if err != nil {
		return err
	}

	// wait pol transfer to be mined
	log.Infof("Waiting tx to be mined")
	const txPolTransferTimeout = 5 * time.Second
	return WaitTxToBeMined(ctx, client.Client, tx, txPolTransferTimeout)
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
	return poll(defaultInterval, defaultDeadline, m.networkUpCondition)
}

func stopNetwork() error {
	cmd := exec.Command(makeCmd, "stop-network")
	return runCmd(cmd)
}

func (m *Manager) startZKEVMNode() error {
	if err := stopZKEVMNode(); err != nil {
		return err
	}
	cmd := exec.Command(makeCmd, "run-node")
	err := runCmd(cmd)
	if err != nil {
		return err
	}
	// Wait zkevm node to be ready
	return poll(defaultInterval, defaultDeadline, m.zkevmNodeUpCondition)
}

func stopZKEVMNode() error {
	cmd := exec.Command(makeCmd, "stop-node")
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
	return poll(defaultInterval, defaultDeadline, proverUpCondition)
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

// StartBridge3 restarts the bridge service.
func StartBridge3() error {
	if err := StopBridge3(); err != nil {
		return err
	}
	cmd := exec.Command(makeCmd, "run-bridge-3")
	err := runCmd(cmd)
	if err != nil {
		return err
	}
	// Wait bridge to be ready
	return poll(defaultInterval, defaultDeadline, bridgeUpCondition)
}

// StopBridge3 stops the bridge service.
func StopBridge3() error {
	cmd := exec.Command(makeCmd, "stop-bridge-3")
	return runCmd(cmd)
}

// StartBridge restarts the bridge service.
func StartBridge() error {
	if err := StopBridge(); err != nil {
		return err
	}
	cmd := exec.Command(makeCmd, "run-bridge")
	err := runCmd(cmd)
	if err != nil {
		return err
	}
	// Wait bridge to be ready
	return poll(defaultInterval, defaultDeadline, bridgeUpCondition)
}

// StopBridge stops the bridge service.
func StopBridge() error {
	cmd := exec.Command(makeCmd, "stop-bridge")
	return runCmd(cmd)
}

// CheckAccountBalance checks the balance by address
func (m *Manager) CheckAccountBalance(ctx context.Context, network NetworkSID, account *common.Address) (*big.Int, error) {
	client := m.clients[network]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[network])
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
func (m *Manager) CheckAccountTokenBalance(ctx context.Context, network NetworkSID, tokenAddr common.Address, account *common.Address) (*big.Int, error) {
	client := m.clients[network]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[network])
	if err != nil {
		return big.NewInt(0), nil
	}

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

// GetClaimData gets the claim data
func (m *Manager) GetClaimData(ctx context.Context, networkID, depositCount uint) ([mtHeight][bridgectrl.KeyLen]byte, [mtHeight][bridgectrl.KeyLen]byte, *etherman.GlobalExitRoot, error) {
	res, err := m.bridgeService.GetProof(context.Background(), &pb.GetProofRequest{
		NetId:      uint32(networkID),
		DepositCnt: uint64(depositCount),
	})
	if err != nil {
		return [mtHeight][32]byte{}, [mtHeight][32]byte{}, nil, err
	}
	merkleproof := [mtHeight][bridgectrl.KeyLen]byte{}
	rollupMerkleProof := [mtHeight][bridgectrl.KeyLen]byte{}
	for i, p := range res.Proof.MerkleProof {
		var proof [bridgectrl.KeyLen]byte
		copy(proof[:], common.FromHex(p))
		merkleproof[i] = proof
		var rollupProof [bridgectrl.KeyLen]byte
		copy(rollupProof[:], common.FromHex(res.Proof.RollupMerkleProof[i]))
		rollupMerkleProof[i] = rollupProof
	}
	return merkleproof, rollupMerkleProof, &etherman.GlobalExitRoot{
		ExitRoots: []common.Hash{
			common.HexToHash(res.Proof.MainExitRoot),
			common.HexToHash(res.Proof.RollupExitRoot),
		},
	}, nil
}

// GetBridgeInfoByDestAddr gets the bridge info
func (m *Manager) GetBridgeInfoByDestAddr(ctx context.Context, addr *common.Address) ([]*pb.Deposit, error) {
	if addr == nil {
		auth, err := m.clients[L2].GetSigner(ctx, accHexPrivateKeys[L2])
		if err != nil {
			return []*pb.Deposit{}, err
		}
		addr = &auth.From
	}
	req := pb.GetBridgesRequest{
		DestAddr: addr.String(),
	}
	res, err := m.bridgeService.GetBridges(ctx, &req)
	if err != nil {
		return []*pb.Deposit{}, err
	}
	return res.Deposits, nil
}

// SendL1Claim send an L1 claim
func (m *Manager) SendL1Claim(ctx context.Context, deposit *pb.Deposit, smtProof, smtRollupProof [mtHeight][32]byte, globalExitRoot *etherman.GlobalExitRoot) error {
	client := m.clients[L1]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[L1])
	if err != nil {
		return err
	}

	return client.SendClaim(ctx, deposit, smtProof, smtRollupProof, globalExitRoot, auth)
}

// SendL2Claim send an L2 claim
func (m *Manager) SendL2Claim(ctx context.Context, deposit *pb.Deposit, smtProof, smtRollupProof [mtHeight][32]byte, globalExitRoot *etherman.GlobalExitRoot, l2 NetworkSID) error {
	client := m.clients[L2]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[l2])
	if err != nil {
		return err
	}

	err = client.SendClaim(ctx, deposit, smtProof, smtRollupProof, globalExitRoot, auth)
	return err
}

// GetTrustedGlobalExitRootSynced reads the latest globalexitroot of a batch proposal from db
func (m *Manager) GetTrustedGlobalExitRootSynced(ctx context.Context, networkID uint) (*etherman.GlobalExitRoot, error) {
	return m.storage.GetLatestTrustedExitRoot(ctx, networkID, nil)
}

// GetLatestGlobalExitRootFromL1 reads the latest synced globalexitroot in l1 from db
func (m *Manager) GetLatestGlobalExitRootFromL1(ctx context.Context) (*etherman.GlobalExitRoot, error) {
	return m.storage.GetLatestL1SyncedExitRoot(ctx, nil)
}

// GetCurrentGlobalExitRootFromSmc reads the globalexitroot from the smc
func (m *Manager) GetCurrentGlobalExitRootFromSmc(ctx context.Context) (*etherman.GlobalExitRoot, error) {
	client := m.clients[L1]
	br, err := polygonzkevmbridge.NewPolygonzkevmbridge(common.HexToAddress(l1BridgeAddr), client)
	if err != nil {
		return nil, err
	}
	GlobalExitRootManAddr, err := br.GlobalExitRootManager(&bind.CallOpts{Pending: false})
	if err != nil {
		return nil, err
	}
	globalManager, err := polygonzkevmglobalexitroot.NewPolygonzkevmglobalexitroot(GlobalExitRootManAddr, client)
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
	result := etherman.GlobalExitRoot{
		ExitRoots: []common.Hash{gMainnet, gRollup},
	}
	return &result, nil
}

// DeployERC20 deploys erc20 smc
func (m *Manager) DeployERC20(ctx context.Context, name, symbol string, network NetworkSID) (common.Address, *ERC20.ERC20, error) {
	client := m.clients[network]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[network])
	if err != nil {
		return common.Address{}, nil, err
	}

	return client.DeployERC20(ctx, name, symbol, auth)
}

// DeployBridgeMessageReceiver deploys the brdige message receiver smc.
func (m *Manager) DeployBridgeMessageReceiver(ctx context.Context, network NetworkSID) (common.Address, error) {
	client := m.clients[network]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[network])
	if err != nil {
		return common.Address{}, err
	}

	return client.DeployBridgeMessageReceiver(ctx, auth)
}

// MintERC20 mint erc20 tokens
func (m *Manager) MintERC20(ctx context.Context, erc20Addr common.Address, amount *big.Int, network NetworkSID) error {
	client := m.clients[network]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[network])
	if err != nil {
		return err
	}

	var bridgeAddress = l1BridgeAddr
	if network == L2 {
		bridgeAddress = l2BridgeAddr
	}

	err = client.ApproveERC20(ctx, erc20Addr, common.HexToAddress(bridgeAddress), amount, auth)
	if err != nil {
		return err
	}

	return client.MintERC20(ctx, erc20Addr, amount, auth)
}

// ApproveERC20 approves erc20 tokens
func (m *Manager) ApproveERC20(ctx context.Context, erc20Addr, bridgeAddr common.Address, amount *big.Int, network NetworkSID) error {
	client := m.clients[network]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[network])
	if err != nil {
		return err
	}
	return client.ApproveERC20(ctx, erc20Addr, bridgeAddr, amount, auth)
}

// GetTokenWrapped get token wrapped info
func (m *Manager) GetTokenWrapped(ctx context.Context, originNetwork uint, originalTokenAddr common.Address, isCreated bool) (*etherman.TokenWrapped, error) {
	if isCreated {
		if err := operations.Poll(defaultInterval, defaultDeadline, func() (bool, error) {
			wrappedToken, err := m.storage.GetTokenWrapped(ctx, originNetwork, originalTokenAddr, nil)
			if err != nil {
				return false, err
			}
			return wrappedToken != nil, nil
		}); err != nil {
			return nil, err
		}
	}
	return m.storage.GetTokenWrapped(ctx, originNetwork, originalTokenAddr, nil)
}

// UpdateBlocksForTesting updates the hash of blocks.
func (m *Manager) UpdateBlocksForTesting(ctx context.Context, networkID uint, blockNum uint64) error {
	return m.storage.UpdateBlocksForTesting(ctx, networkID, blockNum, nil)
}

// WaitExitRootToBeSynced waits until new exit root is synced.
func (m *Manager) WaitExitRootToBeSynced(ctx context.Context, orgExitRoot *etherman.GlobalExitRoot, networkID uint) error {
	log.Debugf("WaitExitRootToBeSynced: %+v", orgExitRoot)
	if orgExitRoot == nil {
		orgExitRoot = &etherman.GlobalExitRoot{
			ExitRoots: []common.Hash{{}, {}},
		}
	}
	return operations.Poll(defaultInterval, waitRootSyncDeadline, func() (bool, error) {
		exitRoot, err := m.storage.GetLatestExitRoot(ctx, networkID, nil)
		if err != nil {
			if err == gerror.ErrStorageNotFound {
				return false, nil
			}
			return false, err
		}
		tID := 0
		if networkID == 0 {
			tID = 1
		}
		return exitRoot.ExitRoots[tID] != orgExitRoot.ExitRoots[tID], nil
	})
}

func (m *Manager) GetLatestMonitoredTxGroupID(ctx context.Context) (uint64, error) {
	return m.storage.GetLatestMonitoredTxGroupID(ctx, nil)
}

// MintPOL mint POL tokens
func (m *Manager) MintPOL(ctx context.Context, erc20Addr common.Address, amount *big.Int, network NetworkSID) error {
	client := m.clients[network]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[network])
	if err != nil {
		return err
	}

	var bridgeAddress = l1BridgeAddr
	if network == L2 {
		bridgeAddress = l2BridgeAddr
	}

	err = client.ApproveERC20(ctx, erc20Addr, common.HexToAddress(bridgeAddress), amount, auth)
	if err != nil {
		return err
	}

	return client.MintPOL(ctx, erc20Addr, amount, auth)
}

// ERC20Transfer send tokens
func (m *Manager) ERC20Transfer(ctx context.Context, erc20Addr, to common.Address, amount *big.Int, network NetworkSID) error {
	client := m.clients[network]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[network])
	if err != nil {
		return err
	}

	return client.ERC20Transfer(ctx, erc20Addr, to, amount, auth)
}

func (m *Manager) GetTokenAddress(ctx context.Context, network NetworkSID, originalNetwork uint, originalTokenAddr common.Address) (common.Address, error) {
	zeroAddr := common.Address{}
	if network == L1 {
		if originalNetwork == 0 {
			return originalTokenAddr, nil
		}
		token, err := m.storage.GetTokenWrapped(ctx, uint(originalNetwork), originalTokenAddr, nil)
		if err != nil {
			return common.Address{}, err
		}
		return token.WrappedTokenAddress, nil
	} else if network == L2 {
		if originalNetwork == 0 && originalTokenAddr == zeroAddr {
			return zeroAddr, nil
		}
		networkID, err := m.clients[network].GetNetworkID()
		if err != nil {
			return common.Address{}, err
		}
		if originalNetwork == uint(networkID) {
			return originalTokenAddr, nil
		}
		token, err := m.storage.GetTokenWrapped(ctx, uint(originalNetwork), originalTokenAddr, nil)
		if err != nil {
			return common.Address{}, err
		}
		return token.WrappedTokenAddress, nil
	} else {
		return common.Address{}, errors.New("unexpected network")
	}
}

func (m *Manager) GetBalances(ctx context.Context, originalNetwork uint32, originalTokenAddr, l1Holder, l2Holder common.Address) (l1Balance, l2Balance *big.Int, err error) {
	l1Balance, err = m.GetL1Balance(ctx, originalNetwork, originalTokenAddr, l1Holder)
	if err != nil {
		return
	}
	l2Balance, err = m.GetL2Balance(ctx, originalNetwork, originalTokenAddr, l2Holder)
	return
}

func (m *Manager) GetL1Balance(ctx context.Context, originalNetwork uint32, originalTokenAddr, holder common.Address) (*big.Int, error) {
	zeroAddrr := common.Address{}
	if originalNetwork == 0 {
		if originalTokenAddr == zeroAddrr {
			return m.CheckAccountBalance(ctx, L1, &holder)
		} else {
			return m.CheckAccountTokenBalance(ctx, L1, originalTokenAddr, &holder)
		}
	} else {
		token, err := m.storage.GetTokenWrapped(ctx, uint(originalNetwork), originalTokenAddr, nil)
		if err == gerror.ErrStorageNotFound {
			return big.NewInt(0), nil
		} else if err != nil {
			return nil, err
		}
		return m.CheckAccountTokenBalance(ctx, L1, token.WrappedTokenAddress, &holder)
	}
}

func (m *Manager) GetL2Balance(ctx context.Context, originalNetwork uint32, originalTokenAddr, holder common.Address) (*big.Int, error) {
	zeroAddr := common.Address{}
	if originalNetwork == 0 && originalTokenAddr == zeroAddr {
		return m.CheckAccountBalance(ctx, L2, &holder)
	}
	var rollupAddr common.Address
	networkID, err := m.clients[L2].GetNetworkID()
	if err != nil {
		return nil, err
	}
	if originalNetwork == networkID {
		rollupAddr = originalTokenAddr
	} else {
		// If the token is not created on L1 or in this rollup, it's needed to calculate
		// the addr of the token on the rollup
		token, err := m.storage.GetTokenWrapped(ctx, uint(originalNetwork), originalTokenAddr, nil)
		if err == gerror.ErrStorageNotFound {
			return big.NewInt(0), nil
		} else if err != nil {
			return nil, err
		}
		rollupAddr = token.WrappedTokenAddress
	}
	return m.CheckAccountTokenBalance(ctx, L2, rollupAddr, &holder)
}

func GetOpsman(ctx context.Context, l2NetworkURL, dbName, bridgeServiceHTTPPort, bridgeServiceGRPCPort, port string, networkID uint) (*Manager, error) {
	//nolint:gomnd
	opsCfg := &Config{
		L1NetworkURL: "http://localhost:8545",
		L2NetworkURL: l2NetworkURL,
		L2NetworkID:  networkID,
		Storage: db.Config{
			Database: "postgres",
			Name:     dbName,
			User:     "test_user",
			Password: "test_password",
			Host:     "localhost",
			Port:     port,
			MaxConns: 10,
		},
		BT: bridgectrl.Config{
			Store:  "postgres",
			Height: uint8(32),
		},
		BS: server.Config{
			GRPCPort:         bridgeServiceGRPCPort,
			HTTPPort:         bridgeServiceHTTPPort,
			CacheSize:        100000,
			DefaultPageLimit: 25,
			MaxPageLimit:     100,
			BridgeVersion:    "v1",
		},
	}
	return NewManager(ctx, opsCfg)
}
