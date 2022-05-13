package client

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/bridgectrl/pb"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/utils"
	"google.golang.org/protobuf/encoding/protojson"
)

// NetworkSID is used to identify the network.
type NetworkSID string

const (
	l1 NetworkSID = "l1"
	l2 NetworkSID = "l2"
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
	l1Client, err := utils.NewClient(ctx, cfg.L1NodeURL)
	if err != nil {
		return nil, err
	}
	l2Client, err := utils.NewClient(ctx, cfg.L2NodeURL)
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

// SendBridge sends a bridge transaction.
func (c NodeClient) SendBridge(ctx context.Context, tokenAddr common.Address, amount *big.Int,
	destNetwork uint32, destAddr *common.Address, bridgeSCAddr common.Address,
	auth *bind.TransactOpts, network NetworkSID,
) error {
	return c.clients[network].SendBridge(ctx, tokenAddr, amount, destNetwork, destAddr, bridgeSCAddr, auth)
}

// SendClaim send a claim transaction.
func (c NodeClient) SendClaim(ctx context.Context, deposit *pb.Deposit, smtProof [][32]byte,
	globalExitRooNum *big.Int, globalExitRoot *etherman.GlobalExitRoot, bridgeSCAddr common.Address,
	auth *bind.TransactOpts, network NetworkSID,
) error {
	return c.clients[network].SendClaim(ctx, deposit, smtProof, globalExitRooNum, globalExitRoot, bridgeSCAddr, auth)
}

// GetBridges returns bridge list for the specific destination address.
func (c RestClient) GetBridges(destAddr string, offset int) ([]*pb.Deposit, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s/%s?offset=%d", c.bridgeURL, "/bridges", destAddr, offset))
	if err != nil {
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var bridgeResp pb.GetBridgesResponse
	err = protojson.Unmarshal(bodyBytes, &bridgeResp)
	if err != nil {
		return nil, err
	}
	return bridgeResp.Deposits, nil
}

// GetClaims returns claim list for the specific destination address.
func (c RestClient) GetClaims(destAddr string, offset int) ([]*pb.Claim, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s/%s?offset=%d", c.bridgeURL, "/claims", destAddr, offset))
	if err != nil {
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var claimResp pb.GetClaimsResponse
	err = protojson.Unmarshal(bodyBytes, &claimResp)
	if err != nil {
		return nil, err
	}
	return claimResp.Claims, nil
}

// GetMerkleProof returns the merkle proof for the specific bridge transaction.
func (c RestClient) GetMerkleProof(networkID uint32, depositCnt uint64) (*pb.Proof, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s?net_id=%d&deposit_cnt=%d", c.bridgeURL, "/merkle-proofs", networkID, depositCnt))
	if err != nil {
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
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

// GetClaimStatus returns the claim status whether it is able to send a claim transaction or not.
func (c RestClient) GetClaimStatus(networkID uint32, depositCnt uint64) (bool, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s?net_id=%d&deposit_cnt=%d", c.bridgeURL, "/claim-status", networkID, depositCnt))
	if err != nil {
		return false, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	var claimStatusResp pb.GetClaimStatusResponse
	err = protojson.Unmarshal(bodyBytes, &claimStatusResp)
	if err != nil {
		return false, err
	}
	return claimStatusResp.Ready, nil
}

// GetWrappedToken returns the wrapped token address.
func (c RestClient) GetWrappedToken(origNet uint32, origTokenAddr string) (*pb.TokenWrapped, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s?orig_net=%d&orig_token_addr=%s", c.bridgeURL, "/tokenwrapped", origNet, origTokenAddr))
	if err != nil {
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
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
	bodyBytes, err := ioutil.ReadAll(resp.Body)
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
