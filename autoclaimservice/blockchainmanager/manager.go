package blockchainmanager

import(
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/common"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman/smartcontracts/claimcompressor"
	zkevmtypes "github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	// LeafTypeAsset represents a bridge asset
	LeafTypeAsset uint32 = 0
	// LeafTypeMessage represents a bridge message
	LeafTypeMessage uint32 = 1

	MtHeight = 32
	KeyLen   = 32
)

// Client is a simple implementation of EtherMan.
type Client struct {
	EtherClient                *ethclient.Client
	PolygonBridge              *polygonzkevmbridge.Polygonzkevmbridge
	ClaimCompressor            *claimcompressor.Claimcompressor
	NetworkID                  uint32
	cfg                        *Config
	logger                     *log.Logger
	auth 					   *bind.TransactOpts
	ctx                        context.Context
}

// NewClient creates a new etherman for L2.
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(cfg.L2RPC)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", cfg.L2RPC, err)
		return nil, err
	}
	// Create smc clients
	bridge, err := polygonzkevmbridge.NewPolygonzkevmbridge(cfg.PolygonBridgeAddress, ethClient)
	if err != nil {
		return nil, err
	}
	var claimCompressor *claimcompressor.Claimcompressor
	if cfg.ClaimCompressorAddress == (common.Address{}) {
		log.Info("Claim compressor Address not configured")
	} else {
		log.Infof("Grouping claims configured, claimCompressor=%s", cfg.ClaimCompressorAddress.String())
		claimCompressor, err = claimcompressor.NewClaimcompressor(cfg.ClaimCompressorAddress, ethClient)
		if err != nil {
			log.Errorf("error creating claimCompressor: %+v", err)
			return nil, err
		}
	}
	networkID, err := bridge.NetworkID(&bind.CallOpts{Pending: false})
	if err != nil {
		return nil, err
	}
	auth, err := GetSignerFromKeystore(ctx, ethClient, cfg.PrivateKey)
	if err != nil {
		log.Errorf("error creating signer. URL: %s. Error: %v", cfg.L2RPC, err)
		return nil, err
	}
	logger := log.WithFields("networkID", networkID)

	return &Client{
		ctx:               ctx,
		logger:            logger,
		EtherClient:       ethClient,
		PolygonBridge:     bridge,
		ClaimCompressor:   claimCompressor,
		cfg:               cfg,
		NetworkID:         networkID,
		auth:              auth,
	}, nil
}

// GetSignerFromKeystore returns a transaction signer from the keystore file.
func GetSignerFromKeystore(ctx context.Context, ethClient *ethclient.Client, ks zkevmtypes.KeystoreFileConfig) (*bind.TransactOpts, error) {
	keystoreEncrypted, err := os.ReadFile(filepath.Clean(ks.Path))
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(keystoreEncrypted, ks.Password)
	if err != nil {
		return nil, err
	}
	chainID, err := ethClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	return bind.NewKeyedTransactorWithChainID(key.PrivateKey, chainID)
}

func (bm *Client) SendCompressedClaims(compressedTxData []byte) (*types.Transaction, error) {
	claimTx, err := bm.ClaimCompressor.SendCompressedClaims(bm.auth, compressedTxData)
	if err != nil {
		bm.logger.Error("failed to call SMC SendCompressedClaims: %v", err)
		return nil, err
	}
	return claimTx, err
}

func (bm *Client) CompressClaimCall(mainnetExitRoot, rollupExitRoot common.Hash, claimData []claimcompressor.ClaimCompressorCompressClaimCallData) ([]byte, error) {
	compressedData, err := bm.ClaimCompressor.CompressClaimCall(&bind.CallOpts{Pending: false}, mainnetExitRoot, rollupExitRoot, claimData)
	if err != nil {
		bm.logger.Errorf("fails call to claimCompressorSMC. Error: %v", err)
		return []byte{}, nil
	}
	return compressedData, nil
}

// SendClaim sends a claim transaction.
func (bm *Client) SendClaim(leafType, origNet uint32,
    origAddr      common.Address,
    amount        *big.Int,
    destNet       uint32,
    destAddr      common.Address,
    networkId     uint32,
    metadata      []byte,
    globalIndex   *big.Int,
	smtProof [MtHeight][KeyLen]byte,
	smtRollupProof [MtHeight][KeyLen]byte,
	mainnetExitRoot, rollupExitRoot common.Hash,
	) (*types.Transaction, error) {
	var (
		tx  *types.Transaction
		err error
	)
	if leafType == LeafTypeAsset {
		tx, err = bm.PolygonBridge.ClaimAsset(bm.auth, smtProof, smtRollupProof, globalIndex, mainnetExitRoot, rollupExitRoot, origNet, origAddr, destNet, destAddr, amount, metadata)
		if err != nil {
			a, _ := polygonzkevmbridge.PolygonzkevmbridgeMetaData.GetAbi()
			input, err3 := a.Pack("claimAsset", smtProof, smtRollupProof, globalIndex, mainnetExitRoot, rollupExitRoot, origNet, origAddr, destNet, destAddr, amount, metadata)
			if err3 != nil {
				bm.logger.Error("error packing call. Error: ", err3)
			}
			bm.logger.Warnf(`Use the next command to debug it manually.
			curl --location --request POST 'http://localhost:8123' \
			--header 'Content-Type: application/json' \
			--data-raw '{
				"jsonrpc": "2.0",
				"method": "eth_call",
				"params": [{"from": "%s","to":"%s","data":"0x%s"},"latest"],
				"id": 1
			}'`, bm.auth.From, bm.cfg.PolygonBridgeAddress.String(), common.Bytes2Hex(input))
		}
	} else if leafType == LeafTypeMessage {
		tx, err = bm.PolygonBridge.ClaimMessage(bm.auth, smtProof, smtRollupProof, globalIndex, mainnetExitRoot, rollupExitRoot, origNet, origAddr, destNet, destAddr, amount, metadata)
	}
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (bm *Client) EstimateGasClaim(metadata []byte, amount *big.Int, originalAddress, destinationAddress common.Address, originalNetwork, destinationNetwork uint32, mainnetExitRoot, rollupExitRoot common.Hash, leafType uint32, globalIndex *big.Int, smtProof [MtHeight][KeyLen]byte, smtRollupProof [MtHeight][KeyLen]byte) (*types.Transaction, error) {
	opts := *bm.auth
	opts.NoSend = true
	var (
		tx  *types.Transaction
		err error
	)
	if leafType == LeafTypeAsset {
		tx, err = bm.PolygonBridge.ClaimAsset(&opts, smtProof, smtRollupProof,
			globalIndex, mainnetExitRoot, rollupExitRoot, originalNetwork, originalAddress, destinationNetwork, destinationAddress, amount, metadata)
	} else if leafType == LeafTypeMessage {
		tx, err = bm.PolygonBridge.ClaimMessage(&opts, smtProof, smtRollupProof, globalIndex, mainnetExitRoot, rollupExitRoot, originalNetwork, originalAddress, destinationNetwork, destinationAddress, amount, metadata)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas tx, err: %v", err)
	}
	return tx, nil
}
