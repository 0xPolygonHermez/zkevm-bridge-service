package operations

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// SetL2TokensAllowed set l2 tokens allowed set
func (m *Manager) SetL2TokensAllowed(ctx context.Context, allowed bool) error {
	client := m.clients[L2]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[L1])
	if err != nil {
		return err
	}

	err = client.SetL2TokensAllowed(ctx, allowed, auth)
	if err != nil {
		return errors.Wrap(err, "SetL2TokensAllowed")
	}

	return nil
}

// ApproveERC20OKB approves erc20 tokens
func (m *Manager) ApproveERC20OKB(ctx context.Context, okbAddress common.Address, amount *big.Int) error {
	client := m.clients[L1]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[L1])
	if err != nil {
		return err
	}
	return client.ApproveERC20(ctx, okbAddress, common.HexToAddress(l1BridgeAddr), amount, auth)
}

// ApproveERC20X1 approves erc20 tokens
func (m *Manager) ApproveERC20X1(ctx context.Context, erc20Addr common.Address, amount *big.Int, network NetworkSID) error {
	client := m.clients[network]
	auth, err := client.GetSigner(ctx, accHexPrivateKeys[network])
	if err != nil {
		return err
	}

	bridgeAddr := common.HexToAddress(l1BridgeAddr)
	if network == L2 {
		bridgeAddr = common.HexToAddress(l2BridgeAddr)
	}

	return client.ApproveERC20(ctx, erc20Addr, bridgeAddr, amount, auth)
}
