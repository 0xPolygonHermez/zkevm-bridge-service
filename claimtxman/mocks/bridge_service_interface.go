// Code generated by mockery. DO NOT EDIT.

package mock_txcompressor

import (
	context "context"

	common "github.com/ethereum/go-ethereum/common"

	etherman "github.com/0xPolygonHermez/zkevm-bridge-service/etherman"

	mock "github.com/stretchr/testify/mock"

	pgx "github.com/jackc/pgx/v4"
)

// bridgeServiceInterface is an autogenerated mock type for the bridgeServiceInterface type
type bridgeServiceInterface struct {
	mock.Mock
}

type bridgeServiceInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *bridgeServiceInterface) EXPECT() *bridgeServiceInterface_Expecter {
	return &bridgeServiceInterface_Expecter{mock: &_m.Mock}
}

// GetClaimProofForCompressed provides a mock function with given fields: ger, depositCnt, networkID, dbTx
func (_m *bridgeServiceInterface) GetClaimProofForCompressed(ger common.Hash, depositCnt uint, networkID uint, dbTx pgx.Tx) (*etherman.GlobalExitRoot, [][32]byte, [][32]byte, error) {
	ret := _m.Called(ger, depositCnt, networkID, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for GetClaimProofForCompressed")
	}

	var r0 *etherman.GlobalExitRoot
	var r1 [][32]byte
	var r2 [][32]byte
	var r3 error
	if rf, ok := ret.Get(0).(func(common.Hash, uint, uint, pgx.Tx) (*etherman.GlobalExitRoot, [][32]byte, [][32]byte, error)); ok {
		return rf(ger, depositCnt, networkID, dbTx)
	}
	if rf, ok := ret.Get(0).(func(common.Hash, uint, uint, pgx.Tx) *etherman.GlobalExitRoot); ok {
		r0 = rf(ger, depositCnt, networkID, dbTx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*etherman.GlobalExitRoot)
		}
	}

	if rf, ok := ret.Get(1).(func(common.Hash, uint, uint, pgx.Tx) [][32]byte); ok {
		r1 = rf(ger, depositCnt, networkID, dbTx)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([][32]byte)
		}
	}

	if rf, ok := ret.Get(2).(func(common.Hash, uint, uint, pgx.Tx) [][32]byte); ok {
		r2 = rf(ger, depositCnt, networkID, dbTx)
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Get(2).([][32]byte)
		}
	}

	if rf, ok := ret.Get(3).(func(common.Hash, uint, uint, pgx.Tx) error); ok {
		r3 = rf(ger, depositCnt, networkID, dbTx)
	} else {
		r3 = ret.Error(3)
	}

	return r0, r1, r2, r3
}

// bridgeServiceInterface_GetClaimProofForCompressed_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetClaimProofForCompressed'
type bridgeServiceInterface_GetClaimProofForCompressed_Call struct {
	*mock.Call
}

// GetClaimProofForCompressed is a helper method to define mock.On call
//   - ger common.Hash
//   - depositCnt uint
//   - networkID uint
//   - dbTx pgx.Tx
func (_e *bridgeServiceInterface_Expecter) GetClaimProofForCompressed(ger interface{}, depositCnt interface{}, networkID interface{}, dbTx interface{}) *bridgeServiceInterface_GetClaimProofForCompressed_Call {
	return &bridgeServiceInterface_GetClaimProofForCompressed_Call{Call: _e.mock.On("GetClaimProofForCompressed", ger, depositCnt, networkID, dbTx)}
}

func (_c *bridgeServiceInterface_GetClaimProofForCompressed_Call) Run(run func(ger common.Hash, depositCnt uint, networkID uint, dbTx pgx.Tx)) *bridgeServiceInterface_GetClaimProofForCompressed_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(common.Hash), args[1].(uint), args[2].(uint), args[3].(pgx.Tx))
	})
	return _c
}

func (_c *bridgeServiceInterface_GetClaimProofForCompressed_Call) Return(_a0 *etherman.GlobalExitRoot, _a1 [][32]byte, _a2 [][32]byte, _a3 error) *bridgeServiceInterface_GetClaimProofForCompressed_Call {
	_c.Call.Return(_a0, _a1, _a2, _a3)
	return _c
}

func (_c *bridgeServiceInterface_GetClaimProofForCompressed_Call) RunAndReturn(run func(common.Hash, uint, uint, pgx.Tx) (*etherman.GlobalExitRoot, [][32]byte, [][32]byte, error)) *bridgeServiceInterface_GetClaimProofForCompressed_Call {
	_c.Call.Return(run)
	return _c
}

// GetDepositStatus provides a mock function with given fields: ctx, depositCount, networkID, destNetworkID
func (_m *bridgeServiceInterface) GetDepositStatus(ctx context.Context, depositCount uint, networkID uint, destNetworkID uint) (string, error) {
	ret := _m.Called(ctx, depositCount, networkID, destNetworkID)

	if len(ret) == 0 {
		panic("no return value specified for GetDepositStatus")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint, uint) (string, error)); ok {
		return rf(ctx, depositCount, networkID, destNetworkID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint, uint, uint) string); ok {
		r0 = rf(ctx, depositCount, networkID, destNetworkID)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint, uint, uint) error); ok {
		r1 = rf(ctx, depositCount, networkID, destNetworkID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// bridgeServiceInterface_GetDepositStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetDepositStatus'
type bridgeServiceInterface_GetDepositStatus_Call struct {
	*mock.Call
}

// GetDepositStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - depositCount uint
//   - networkID uint
//   - destNetworkID uint
func (_e *bridgeServiceInterface_Expecter) GetDepositStatus(ctx interface{}, depositCount interface{}, networkID interface{}, destNetworkID interface{}) *bridgeServiceInterface_GetDepositStatus_Call {
	return &bridgeServiceInterface_GetDepositStatus_Call{Call: _e.mock.On("GetDepositStatus", ctx, depositCount, networkID, destNetworkID)}
}

func (_c *bridgeServiceInterface_GetDepositStatus_Call) Run(run func(ctx context.Context, depositCount uint, networkID uint, destNetworkID uint)) *bridgeServiceInterface_GetDepositStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(uint), args[2].(uint), args[3].(uint))
	})
	return _c
}

func (_c *bridgeServiceInterface_GetDepositStatus_Call) Return(_a0 string, _a1 error) *bridgeServiceInterface_GetDepositStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *bridgeServiceInterface_GetDepositStatus_Call) RunAndReturn(run func(context.Context, uint, uint, uint) (string, error)) *bridgeServiceInterface_GetDepositStatus_Call {
	_c.Call.Return(run)
	return _c
}

// newBridgeServiceInterface creates a new instance of bridgeServiceInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newBridgeServiceInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *bridgeServiceInterface {
	mock := &bridgeServiceInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
