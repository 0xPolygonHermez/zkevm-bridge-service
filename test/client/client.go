package client

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"net/http"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/encoding/protojson"
)

// NetworkSID is used to identify the network.
type NetworkSID string

const (
	l1 NetworkSID = "l1"
	l2 NetworkSID = "l2"

	mtHeight = 32
)

// RestClient is a client for the rest api.
type RestClient struct {
	bridgeURL string
}

// NewRestClient creates new rest api client.
func NewRestClient(url string) *RestClient {
	return &RestClient{
		bridgeURL: url,
	}
}

// NodeClient is for the ethclient.
type NodeClient struct {
	clients map[NetworkSID]*utils.Client
}

// Client is for the outside test.
type Client struct {
	NodeClient
	RestClient
}

// NewClient returns a new client.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	l1Client, err := utils.NewClient(ctx, cfg.L1NodeURL, cfg.L1BridgeAddr)
	if err != nil {
		return nil, err
	}
	l2Client, err := utils.NewClient(ctx, cfg.L2NodeURL, cfg.L2BridgeAddr)
	if err != nil {
		return nil, err
	}

	clients := make(map[NetworkSID]*utils.Client)
	clients[l1] = l1Client
	clients[l2] = l2Client

	return &Client{
		NodeClient{
			clients: clients,
		},
		RestClient{
			bridgeURL: cfg.BridgeURL,
		},
	}, nil
}

// GetChainID returns a chain ID of the dedicated node.
func (c NodeClient) GetChainID(ctx context.Context, network NetworkSID) (*big.Int, error) {
	return c.clients[network].ChainID(ctx)
}

// DeployERC20 deploys erc20 smc.
func (c NodeClient) DeployERC20(ctx context.Context, name, symbol string, auth *bind.TransactOpts, network NetworkSID) (common.Address, error) {
	addr, _, err := c.clients[network].DeployERC20(ctx, name, symbol, auth)
	return addr, err
}

// ApproveERC20 approves erc20 tokens.
func (c NodeClient) ApproveERC20(ctx context.Context, erc20Addr, spender common.Address, amount *big.Int, auth *bind.TransactOpts, network NetworkSID) error {
	return c.clients[network].ApproveERC20(ctx, erc20Addr, spender, amount, auth)
}

// MintERC20 mint erc20 tokens.
func (c NodeClient) MintERC20(ctx context.Context, erc20Addr common.Address, amount *big.Int, auth *bind.TransactOpts, network NetworkSID) error {
	return c.clients[network].MintERC20(ctx, erc20Addr, amount, auth)
}

// SendBridgeAsset sends a bridge asset transaction.
func (c NodeClient) SendBridgeAsset(ctx context.Context, tokenAddr common.Address, amount *big.Int,
	destNetwork uint32, destAddr *common.Address, metadata []byte, auth *bind.TransactOpts, network NetworkSID,
) error {
	return c.clients[network].SendBridgeAsset(ctx, tokenAddr, amount, destNetwork, destAddr, metadata, auth)
}

// SendBridgeMessage sends a bridge message transaction.
func (c NodeClient) SendBridgeMessage(ctx context.Context, destNetwork uint32, destAddr common.Address, metadata []byte,
	auth *bind.TransactOpts, network NetworkSID,
) error {
	return c.clients[network].SendBridgeMessage(ctx, destNetwork, destAddr, metadata, auth)
}

// SendClaim send a claim transaction.
func (c NodeClient) SendClaim(ctx context.Context, deposit *pb.Deposit, smtProof [mtHeight][32]byte, globalExitRoot *etherman.GlobalExitRoot,
	auth *bind.TransactOpts, network NetworkSID,
) error {
	return c.clients[network].SendClaim(ctx, deposit, smtProof, globalExitRoot, auth)
}

// GetBridges returns bridge list for the specific destination address.
func (c RestClient) GetBridges(destAddr string, offset, limit uint) ([]*pb.Deposit, uint64, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s/%s?offset=%d&limit=%d", c.bridgeURL, "/bridges", destAddr, offset, limit))
	if err != nil {
		return nil, 0, err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	var bridgeResp pb.GetBridgesResponse
	err = protojson.Unmarshal(bodyBytes, &bridgeResp)
	if err != nil {
		return nil, 0, err
	}
	return bridgeResp.Deposits, bridgeResp.TotalCnt, nil
}

// GetClaims returns claim list for the specific destination address.
func (c RestClient) GetClaims(destAddr string, offset, limit uint) ([]*pb.Claim, uint64, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s/%s?offset=%d&limit=%d", c.bridgeURL, "/claims", destAddr, offset, limit))
	if err != nil {
		return nil, 0, err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	var claimResp pb.GetClaimsResponse
	err = protojson.Unmarshal(bodyBytes, &claimResp)
	if err != nil {
		return nil, 0, err
	}
	return claimResp.Claims, claimResp.TotalCnt, nil
}

// GetMerkleProof returns the merkle proof for the specific bridge transaction.
func (c RestClient) GetMerkleProof(networkID uint32, depositCnt uint64) (*pb.Proof, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s?net_id=%d&deposit_cnt=%d", c.bridgeURL, "/merkle-proof", networkID, depositCnt))
	if err != nil {
		return nil, err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var proofResp pb.GetProofResponse
	err = protojson.Unmarshal(bodyBytes, &proofResp)
	if err != nil {
		return nil, err
	}
	return proofResp.Proof, nil
}

// GetBridge returns the specific bridge info.
func (c RestClient) GetBridge(networkID uint32, depositCnt uint64) (*pb.Deposit, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s?net_id=%d&deposit_cnt=%d", c.bridgeURL, "/bridge", networkID, depositCnt))
	if err != nil {
		return nil, err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var bridgeResp pb.GetBridgeResponse
	err = protojson.Unmarshal(bodyBytes, &bridgeResp)
	if err != nil {
		return nil, err
	}
	return bridgeResp.Deposit, nil
}

// GetWrappedToken returns the wrapped token address.
func (c RestClient) GetWrappedToken(origNet uint32, origTokenAddr string) (*pb.TokenWrapped, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s?orig_net=%d&orig_token_addr=%s", c.bridgeURL, "/tokenwrapped", origNet, origTokenAddr))
	if err != nil {
		return nil, err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tokenWrappedResp pb.GetTokenWrappedResponse
	err = protojson.Unmarshal(bodyBytes, &tokenWrappedResp)
	if err != nil {
		return nil, err
	}
	return tokenWrappedResp.Tokenwrapped, nil
}

// GetVersion returns the api version.
func (c RestClient) GetVersion() (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s", c.bridgeURL, "/api"))
	if err != nil {
		return "", err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var apiResp pb.CheckAPIResponse
	err = protojson.Unmarshal(bodyBytes, &apiResp)
	if err != nil {
		return "", err
	}
	return apiResp.Api, nil
}
