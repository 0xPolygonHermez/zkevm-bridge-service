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

type Client struct {
	clients   map[string]*utils.Client
	bridgeURL string
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	l1Client, err := utils.NewClient(ctx, cfg.L1NodeURL)
	if err != nil {
		return nil, err
	}
	l2Client, err := utils.NewClient(ctx, cfg.L2NodeURL)

	clients := make(map[string]*utils.Client)
	clients["l1"] = l1Client
	clients["l2"] = l2Client

	return &Client{
		clients:   clients,
		bridgeURL: cfg.BridgeURL,
	}, err
}

// SendBridge sends a bridge transaction.
func (c Client) SendBridge(ctx context.Context, tokenAddr common.Address, amount *big.Int,
	destNetwork uint32, destAddr *common.Address, bridgeSCAddr common.Address,
	auth *bind.TransactOpts, network string,
) error {
	return c.clients[network].SendBridge(ctx, tokenAddr, amount, destNetwork, destAddr, bridgeSCAddr, auth)
}

// SendClaim send a claim transaction.
func (c Client) SendClaim(ctx context.Context, deposit *pb.Deposit, smtProof [][32]byte,
	globalExitRooNum *big.Int, globalExitRoot *etherman.GlobalExitRoot, bridgeSCAddr common.Address,
	auth *bind.TransactOpts, network string,
) error {
	return c.clients[network].SendClaim(ctx, deposit, smtProof, globalExitRooNum, globalExitRoot, bridgeSCAddr, auth)
}

// GetBridges returns bridge list for the specific destination address
func (c Client) GetBridges(destAddr string, offset int) ([]*pb.Deposit, error) {
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

// GetClaims returns claim list for the specific destination address
func (c Client) GetClaims(destAddr string, offset int) ([]*pb.Claim, error) {
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
