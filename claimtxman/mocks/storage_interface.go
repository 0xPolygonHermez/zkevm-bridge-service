// Code generated by mockery. DO NOT EDIT.

package mock_txcompressor

import (
	context "context"

	etherman "github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	mock "github.com/stretchr/testify/mock"

	pgx "github.com/jackc/pgx/v4"

	types "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
)

// StorageInterface is an autogenerated mock type for the StorageInterface type
type StorageInterface struct {
	mock.Mock
}

type StorageInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *StorageInterface) EXPECT() *StorageInterface_Expecter {
	return &StorageInterface_Expecter{mock: &_m.Mock}
}

// AddBlock provides a mock function with given fields: ctx, block, dbTx
func (_m *StorageInterface) AddBlock(ctx context.Context, block *etherman.Block, dbTx pgx.Tx) (uint64, error) {
	ret := _m.Called(ctx, block, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for AddBlock")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *etherman.Block, pgx.Tx) (uint64, error)); ok {
		return rf(ctx, block, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *etherman.Block, pgx.Tx) uint64); ok {
		r0 = rf(ctx, block, dbTx)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *etherman.Block, pgx.Tx) error); ok {
		r1 = rf(ctx, block, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StorageInterface_AddBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddBlock'
type StorageInterface_AddBlock_Call struct {
	*mock.Call
}

// AddBlock is a helper method to define mock.On call
//   - ctx context.Context
//   - block *etherman.Block
//   - dbTx pgx.Tx
func (_e *StorageInterface_Expecter) AddBlock(ctx interface{}, block interface{}, dbTx interface{}) *StorageInterface_AddBlock_Call {
	return &StorageInterface_AddBlock_Call{Call: _e.mock.On("AddBlock", ctx, block, dbTx)}
}

func (_c *StorageInterface_AddBlock_Call) Run(run func(ctx context.Context, block *etherman.Block, dbTx pgx.Tx)) *StorageInterface_AddBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*etherman.Block), args[2].(pgx.Tx))
	})
	return _c
}

func (_c *StorageInterface_AddBlock_Call) Return(_a0 uint64, _a1 error) *StorageInterface_AddBlock_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *StorageInterface_AddBlock_Call) RunAndReturn(run func(context.Context, *etherman.Block, pgx.Tx) (uint64, error)) *StorageInterface_AddBlock_Call {
	_c.Call.Return(run)
	return _c
}

// AddClaimTx provides a mock function with given fields: ctx, mTx, dbTx
func (_m *StorageInterface) AddClaimTx(ctx context.Context, mTx types.MonitoredTx, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, mTx, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for AddClaimTx")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.MonitoredTx, pgx.Tx) error); ok {
		r0 = rf(ctx, mTx, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StorageInterface_AddClaimTx_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddClaimTx'
type StorageInterface_AddClaimTx_Call struct {
	*mock.Call
}

// AddClaimTx is a helper method to define mock.On call
//   - ctx context.Context
//   - mTx types.MonitoredTx
//   - dbTx pgx.Tx
func (_e *StorageInterface_Expecter) AddClaimTx(ctx interface{}, mTx interface{}, dbTx interface{}) *StorageInterface_AddClaimTx_Call {
	return &StorageInterface_AddClaimTx_Call{Call: _e.mock.On("AddClaimTx", ctx, mTx, dbTx)}
}

func (_c *StorageInterface_AddClaimTx_Call) Run(run func(ctx context.Context, mTx types.MonitoredTx, dbTx pgx.Tx)) *StorageInterface_AddClaimTx_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(types.MonitoredTx), args[2].(pgx.Tx))
	})
	return _c
}

func (_c *StorageInterface_AddClaimTx_Call) Return(_a0 error) *StorageInterface_AddClaimTx_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StorageInterface_AddClaimTx_Call) RunAndReturn(run func(context.Context, types.MonitoredTx, pgx.Tx) error) *StorageInterface_AddClaimTx_Call {
	_c.Call.Return(run)
	return _c
}

// BeginDBTransaction provides a mock function with given fields: ctx
func (_m *StorageInterface) BeginDBTransaction(ctx context.Context) (pgx.Tx, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for BeginDBTransaction")
	}

	var r0 pgx.Tx
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (pgx.Tx, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) pgx.Tx); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(pgx.Tx)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StorageInterface_BeginDBTransaction_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'BeginDBTransaction'
type StorageInterface_BeginDBTransaction_Call struct {
	*mock.Call
}

// BeginDBTransaction is a helper method to define mock.On call
//   - ctx context.Context
func (_e *StorageInterface_Expecter) BeginDBTransaction(ctx interface{}) *StorageInterface_BeginDBTransaction_Call {
	return &StorageInterface_BeginDBTransaction_Call{Call: _e.mock.On("BeginDBTransaction", ctx)}
}

func (_c *StorageInterface_BeginDBTransaction_Call) Run(run func(ctx context.Context)) *StorageInterface_BeginDBTransaction_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *StorageInterface_BeginDBTransaction_Call) Return(_a0 pgx.Tx, _a1 error) *StorageInterface_BeginDBTransaction_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *StorageInterface_BeginDBTransaction_Call) RunAndReturn(run func(context.Context) (pgx.Tx, error)) *StorageInterface_BeginDBTransaction_Call {
	_c.Call.Return(run)
	return _c
}

// Commit provides a mock function with given fields: ctx, dbTx
func (_m *StorageInterface) Commit(ctx context.Context, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for Commit")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx) error); ok {
		r0 = rf(ctx, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StorageInterface_Commit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Commit'
type StorageInterface_Commit_Call struct {
	*mock.Call
}

// Commit is a helper method to define mock.On call
//   - ctx context.Context
//   - dbTx pgx.Tx
func (_e *StorageInterface_Expecter) Commit(ctx interface{}, dbTx interface{}) *StorageInterface_Commit_Call {
	return &StorageInterface_Commit_Call{Call: _e.mock.On("Commit", ctx, dbTx)}
}

func (_c *StorageInterface_Commit_Call) Run(run func(ctx context.Context, dbTx pgx.Tx)) *StorageInterface_Commit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(pgx.Tx))
	})
	return _c
}

func (_c *StorageInterface_Commit_Call) Return(_a0 error) *StorageInterface_Commit_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StorageInterface_Commit_Call) RunAndReturn(run func(context.Context, pgx.Tx) error) *StorageInterface_Commit_Call {
	_c.Call.Return(run)
	return _c
}

// GetClaimTxsByStatus provides a mock function with given fields: ctx, statuses, rollupID, dbTx
func (_m *StorageInterface) GetClaimTxsByStatus(ctx context.Context, statuses []types.MonitoredTxStatus, rollupID uint, dbTx pgx.Tx) ([]types.MonitoredTx, error) {
	ret := _m.Called(ctx, statuses, rollupID, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for GetClaimTxsByStatus")
	}

	var r0 []types.MonitoredTx
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []types.MonitoredTxStatus, uint, pgx.Tx) ([]types.MonitoredTx, error)); ok {
		return rf(ctx, statuses, rollupID, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []types.MonitoredTxStatus, uint, pgx.Tx) []types.MonitoredTx); ok {
		r0 = rf(ctx, statuses, rollupID, dbTx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.MonitoredTx)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []types.MonitoredTxStatus, uint, pgx.Tx) error); ok {
		r1 = rf(ctx, statuses, rollupID, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StorageInterface_GetClaimTxsByStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetClaimTxsByStatus'
type StorageInterface_GetClaimTxsByStatus_Call struct {
	*mock.Call
}

// GetClaimTxsByStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - statuses []types.MonitoredTxStatus
//   - rollupID uint
//   - dbTx pgx.Tx
func (_e *StorageInterface_Expecter) GetClaimTxsByStatus(ctx interface{}, statuses interface{}, rollupID interface{}, dbTx interface{}) *StorageInterface_GetClaimTxsByStatus_Call {
	return &StorageInterface_GetClaimTxsByStatus_Call{Call: _e.mock.On("GetClaimTxsByStatus", ctx, statuses, rollupID, dbTx)}
}

func (_c *StorageInterface_GetClaimTxsByStatus_Call) Run(run func(ctx context.Context, statuses []types.MonitoredTxStatus, rollupID uint, dbTx pgx.Tx)) *StorageInterface_GetClaimTxsByStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]types.MonitoredTxStatus), args[2].(uint), args[3].(pgx.Tx))
	})
	return _c
}

func (_c *StorageInterface_GetClaimTxsByStatus_Call) Return(_a0 []types.MonitoredTx, _a1 error) *StorageInterface_GetClaimTxsByStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *StorageInterface_GetClaimTxsByStatus_Call) RunAndReturn(run func(context.Context, []types.MonitoredTxStatus, uint, pgx.Tx) ([]types.MonitoredTx, error)) *StorageInterface_GetClaimTxsByStatus_Call {
	_c.Call.Return(run)
	return _c
}

// Rollback provides a mock function with given fields: ctx, dbTx
func (_m *StorageInterface) Rollback(ctx context.Context, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for Rollback")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx) error); ok {
		r0 = rf(ctx, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StorageInterface_Rollback_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Rollback'
type StorageInterface_Rollback_Call struct {
	*mock.Call
}

// Rollback is a helper method to define mock.On call
//   - ctx context.Context
//   - dbTx pgx.Tx
func (_e *StorageInterface_Expecter) Rollback(ctx interface{}, dbTx interface{}) *StorageInterface_Rollback_Call {
	return &StorageInterface_Rollback_Call{Call: _e.mock.On("Rollback", ctx, dbTx)}
}

func (_c *StorageInterface_Rollback_Call) Run(run func(ctx context.Context, dbTx pgx.Tx)) *StorageInterface_Rollback_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(pgx.Tx))
	})
	return _c
}

func (_c *StorageInterface_Rollback_Call) Return(_a0 error) *StorageInterface_Rollback_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StorageInterface_Rollback_Call) RunAndReturn(run func(context.Context, pgx.Tx) error) *StorageInterface_Rollback_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateClaimTx provides a mock function with given fields: ctx, mTx, dbTx
func (_m *StorageInterface) UpdateClaimTx(ctx context.Context, mTx types.MonitoredTx, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, mTx, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateClaimTx")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.MonitoredTx, pgx.Tx) error); ok {
		r0 = rf(ctx, mTx, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StorageInterface_UpdateClaimTx_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateClaimTx'
type StorageInterface_UpdateClaimTx_Call struct {
	*mock.Call
}

// UpdateClaimTx is a helper method to define mock.On call
//   - ctx context.Context
//   - mTx types.MonitoredTx
//   - dbTx pgx.Tx
func (_e *StorageInterface_Expecter) UpdateClaimTx(ctx interface{}, mTx interface{}, dbTx interface{}) *StorageInterface_UpdateClaimTx_Call {
	return &StorageInterface_UpdateClaimTx_Call{Call: _e.mock.On("UpdateClaimTx", ctx, mTx, dbTx)}
}

func (_c *StorageInterface_UpdateClaimTx_Call) Run(run func(ctx context.Context, mTx types.MonitoredTx, dbTx pgx.Tx)) *StorageInterface_UpdateClaimTx_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(types.MonitoredTx), args[2].(pgx.Tx))
	})
	return _c
}

func (_c *StorageInterface_UpdateClaimTx_Call) Return(_a0 error) *StorageInterface_UpdateClaimTx_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StorageInterface_UpdateClaimTx_Call) RunAndReturn(run func(context.Context, types.MonitoredTx, pgx.Tx) error) *StorageInterface_UpdateClaimTx_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateL1DepositsStatus provides a mock function with given fields: ctx, exitRoot, destinationNetwork, dbTx
func (_m *StorageInterface) UpdateL1DepositsStatus(ctx context.Context, exitRoot []byte, destinationNetwork uint, dbTx pgx.Tx) ([]*etherman.Deposit, error) {
	ret := _m.Called(ctx, exitRoot, destinationNetwork, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateL1DepositsStatus")
	}

	var r0 []*etherman.Deposit
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, uint, pgx.Tx) ([]*etherman.Deposit, error)); ok {
		return rf(ctx, exitRoot, destinationNetwork, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte, uint, pgx.Tx) []*etherman.Deposit); ok {
		r0 = rf(ctx, exitRoot, destinationNetwork, dbTx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*etherman.Deposit)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte, uint, pgx.Tx) error); ok {
		r1 = rf(ctx, exitRoot, destinationNetwork, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StorageInterface_UpdateL1DepositsStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateL1DepositsStatus'
type StorageInterface_UpdateL1DepositsStatus_Call struct {
	*mock.Call
}

// UpdateL1DepositsStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - exitRoot []byte
//   - destinationNetwork uint
//   - dbTx pgx.Tx
func (_e *StorageInterface_Expecter) UpdateL1DepositsStatus(ctx interface{}, exitRoot interface{}, destinationNetwork interface{}, dbTx interface{}) *StorageInterface_UpdateL1DepositsStatus_Call {
	return &StorageInterface_UpdateL1DepositsStatus_Call{Call: _e.mock.On("UpdateL1DepositsStatus", ctx, exitRoot, destinationNetwork, dbTx)}
}

func (_c *StorageInterface_UpdateL1DepositsStatus_Call) Run(run func(ctx context.Context, exitRoot []byte, destinationNetwork uint, dbTx pgx.Tx)) *StorageInterface_UpdateL1DepositsStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]byte), args[2].(uint), args[3].(pgx.Tx))
	})
	return _c
}

func (_c *StorageInterface_UpdateL1DepositsStatus_Call) Return(_a0 []*etherman.Deposit, _a1 error) *StorageInterface_UpdateL1DepositsStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *StorageInterface_UpdateL1DepositsStatus_Call) RunAndReturn(run func(context.Context, []byte, uint, pgx.Tx) ([]*etherman.Deposit, error)) *StorageInterface_UpdateL1DepositsStatus_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateL2DepositsStatus provides a mock function with given fields: ctx, exitRoot, rollupID, networkID, dbTx
func (_m *StorageInterface) UpdateL2DepositsStatus(ctx context.Context, exitRoot []byte, rollupID uint, networkID uint, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, exitRoot, rollupID, networkID, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateL2DepositsStatus")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, uint, uint, pgx.Tx) error); ok {
		r0 = rf(ctx, exitRoot, rollupID, networkID, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StorageInterface_UpdateL2DepositsStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateL2DepositsStatus'
type StorageInterface_UpdateL2DepositsStatus_Call struct {
	*mock.Call
}

// UpdateL2DepositsStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - exitRoot []byte
//   - rollupID uint
//   - networkID uint
//   - dbTx pgx.Tx
func (_e *StorageInterface_Expecter) UpdateL2DepositsStatus(ctx interface{}, exitRoot interface{}, rollupID interface{}, networkID interface{}, dbTx interface{}) *StorageInterface_UpdateL2DepositsStatus_Call {
	return &StorageInterface_UpdateL2DepositsStatus_Call{Call: _e.mock.On("UpdateL2DepositsStatus", ctx, exitRoot, rollupID, networkID, dbTx)}
}

func (_c *StorageInterface_UpdateL2DepositsStatus_Call) Run(run func(ctx context.Context, exitRoot []byte, rollupID uint, networkID uint, dbTx pgx.Tx)) *StorageInterface_UpdateL2DepositsStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]byte), args[2].(uint), args[3].(uint), args[4].(pgx.Tx))
	})
	return _c
}

func (_c *StorageInterface_UpdateL2DepositsStatus_Call) Return(_a0 error) *StorageInterface_UpdateL2DepositsStatus_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StorageInterface_UpdateL2DepositsStatus_Call) RunAndReturn(run func(context.Context, []byte, uint, uint, pgx.Tx) error) *StorageInterface_UpdateL2DepositsStatus_Call {
	_c.Call.Return(run)
	return _c
}

// NewStorageInterface creates a new instance of StorageInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewStorageInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *StorageInterface {
	mock := &StorageInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
