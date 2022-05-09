package client

import (
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	l1Client  *ethclient.Client
	l2Client  *ethclient.Client
	bridgeURL string
}
