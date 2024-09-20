package blockchainmanager

import(
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/common"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman/smartcontracts/claimcompressor"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	// LeafTypeAsset represents a bridge asset
	LeafTypeAsset uint32 = 0
	// LeafTypeMessage represents a bridge message
	LeafTypeMessage uint32 = 1

	mtHeight = 32
	keyLen   = 32
)

// Client is a simple implementation of EtherMan.
type Client struct {
	EtherClient                *ethclient.Client
	PolygonBridge              *polygonzkevmbridge.Polygonzkevmbridge
	ClaimCompressor            *claimcompressor.Claimcompressor
	NetworkID                  uint32
	polygonBridgeAddr          common.Address
	logger                     *log.Logger
}

// NewClient creates a new etherman for L2.
func NewClient(url string, polygonBridgeAddr, claimCompressorAddress common.Address) (*Client, error) {
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(url)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", url, err)
		return nil, err
	}
	// Create smc clients
	bridge, err := polygonzkevmbridge.NewPolygonzkevmbridge(polygonBridgeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	var claimCompressor *claimcompressor.Claimcompressor
	if claimCompressorAddress == (common.Address{}) {
		log.Warn("Claim compressor Address not configured")
	} else {
		log.Infof("Grouping claims allowed, claimCompressor=%s", claimCompressorAddress.String())
		claimCompressor, err = claimcompressor.NewClaimcompressor(claimCompressorAddress, ethClient)
		if err != nil {
			log.Errorf("error creating claimCompressor: %+v", err)
			return nil, err
		}
	}
	networkID, err := bridge.NetworkID(&bind.CallOpts{Pending: false})
	if err != nil {
		return nil, err
	}
	logger := log.WithFields("networkID", networkID)

	return &Client{
		logger:            logger,
		EtherClient:       ethClient,
		PolygonBridge:     bridge,
		ClaimCompressor:   claimCompressor,
		polygonBridgeAddr: polygonBridgeAddr,
		NetworkID:         networkID,
	}, nil
}

func (bm *Client) SendCompressedClaims(auth *bind.TransactOpts, compressedTxData []byte) (*types.Transaction, error) {
	claimTx, err := bm.ClaimCompressor.SendCompressedClaims(auth, compressedTxData)
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
func (bm *Client) SendClaim(ctx context.Context,
	leafType, origNet uint32,
    origAddr      common.Address,
    amount        *big.Int,
    destNet       uint32,
    destAddr      common.Address,
    networkId     uint32,
    metadata      []byte,
    globalIndex   *big.Int,
	smtProof [mtHeight][keyLen]byte,
	smtRollupProof [mtHeight][keyLen]byte,
	mainnetExitRoot, rollupExitRoot common.Hash,
	auth *bind.TransactOpts,
	) error {
	var (
		tx  *types.Transaction
		err error
	)
	if leafType == LeafTypeAsset {
		tx, err = bm.PolygonBridge.ClaimAsset(auth, smtProof, smtRollupProof, globalIndex, mainnetExitRoot, rollupExitRoot, origNet, origAddr, destNet, destAddr, amount, metadata)
		if err != nil {
			a, _ := polygonzkevmbridge.PolygonzkevmbridgeMetaData.GetAbi()
			input, err3 := a.Pack("claimAsset", smtProof, smtRollupProof, globalIndex, mainnetExitRoot, rollupExitRoot, origNet, origAddr, destNet, destAddr, amount, metadata)
			if err3 != nil {
				log.Error("error packing call. Error: ", err3)
			}
			log.Warnf(`Use the next command to debug it manually.
			curl --location --request POST 'http://localhost:8123' \
			--header 'Content-Type: application/json' \
			--data-raw '{
				"jsonrpc": "2.0",
				"method": "eth_call",
				"params": [{"from": "%s","to":"%s","data":"0x%s"},"latest"],
				"id": 1
			}'`, auth.From, bm.polygonBridgeAddr.String(), common.Bytes2Hex(input))
		}
	} else if leafType == LeafTypeMessage {
		tx, err = bm.PolygonBridge.ClaimMessage(auth, smtProof, smtRollupProof, globalIndex, mainnetExitRoot, rollupExitRoot, origNet, origAddr, destNet, destAddr, amount, metadata)
	}
	if err != nil {
		txHash := ""
		if tx != nil {
			txHash = tx.Hash().String()
		}
		log.Error("Error: ", err, ". Tx Hash: ", txHash)
		return err
	}
	return nil
}