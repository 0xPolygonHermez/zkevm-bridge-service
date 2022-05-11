package client

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/utils"
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

func (c Client) SendBridge(ctx context.Context, tokenAddr common.Address, amount *big.Int,
	destNetwork uint32, destAddr *common.Address, bridgeSCAddr common.Address,
	auth *bind.TransactOpts, network string,
) error {
	return c.clients[network].SendBridge(ctx, tokenAddr, amount, destNetwork, destAddr, bridgeSCAddr, auth)
}
