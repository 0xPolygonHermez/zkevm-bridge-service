// Code generated by mockery. DO NOT EDIT.

package mock_txcompressor

import (
	context "context"

	pgx "github.com/jackc/pgx/v4"
	mock "github.com/stretchr/testify/mock"

	types "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
)

// StorageRequireInterface is an autogenerated mock type for the StorageRequireInterface type
type StorageRequireInterface struct {
	mock.Mock
}

type StorageRequireInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *StorageRequireInterface) EXPECT() *StorageRequireInterface_Expecter {
	return &StorageRequireInterface_Expecter{mock: &_m.Mock}
}

// AddMonitoredTxsGroup provides a mock function with given fields: ctx, mTxGroup, dbTx
func (_m *StorageRequireInterface) AddMonitoredTxsGroup(ctx context.Context, mTxGroup *types.MonitoredTxGroupDBEntry, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, mTxGroup, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for AddMonitoredTxsGroup")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.MonitoredTxGroupDBEntry, pgx.Tx) error); ok {
		r0 = rf(ctx, mTxGroup, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StorageRequireInterface_AddMonitoredTxsGroup_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddMonitoredTxsGroup'
type StorageRequireInterface_AddMonitoredTxsGroup_Call struct {
	*mock.Call
}

// AddMonitoredTxsGroup is a helper method to define mock.On call
//   - ctx context.Context
//   - mTxGroup *types.MonitoredTxGroupDBEntry
//   - dbTx pgx.Tx
func (_e *StorageRequireInterface_Expecter) AddMonitoredTxsGroup(ctx interface{}, mTxGroup interface{}, dbTx interface{}) *StorageRequireInterface_AddMonitoredTxsGroup_Call {
	return &StorageRequireInterface_AddMonitoredTxsGroup_Call{Call: _e.mock.On("AddMonitoredTxsGroup", ctx, mTxGroup, dbTx)}
}

func (_c *StorageRequireInterface_AddMonitoredTxsGroup_Call) Run(run func(ctx context.Context, mTxGroup *types.MonitoredTxGroupDBEntry, dbTx pgx.Tx)) *StorageRequireInterface_AddMonitoredTxsGroup_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*types.MonitoredTxGroupDBEntry), args[2].(pgx.Tx))
	})
	return _c
}

func (_c *StorageRequireInterface_AddMonitoredTxsGroup_Call) Return(_a0 error) *StorageRequireInterface_AddMonitoredTxsGroup_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StorageRequireInterface_AddMonitoredTxsGroup_Call) RunAndReturn(run func(context.Context, *types.MonitoredTxGroupDBEntry, pgx.Tx) error) *StorageRequireInterface_AddMonitoredTxsGroup_Call {
	_c.Call.Return(run)
	return _c
}

// BeginDBTransaction provides a mock function with given fields: ctx
func (_m *StorageRequireInterface) BeginDBTransaction(ctx context.Context) (pgx.Tx, error) {
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

// StorageRequireInterface_BeginDBTransaction_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'BeginDBTransaction'
type StorageRequireInterface_BeginDBTransaction_Call struct {
	*mock.Call
}

// BeginDBTransaction is a helper method to define mock.On call
//   - ctx context.Context
func (_e *StorageRequireInterface_Expecter) BeginDBTransaction(ctx interface{}) *StorageRequireInterface_BeginDBTransaction_Call {
	return &StorageRequireInterface_BeginDBTransaction_Call{Call: _e.mock.On("BeginDBTransaction", ctx)}
}

func (_c *StorageRequireInterface_BeginDBTransaction_Call) Run(run func(ctx context.Context)) *StorageRequireInterface_BeginDBTransaction_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *StorageRequireInterface_BeginDBTransaction_Call) Return(_a0 pgx.Tx, _a1 error) *StorageRequireInterface_BeginDBTransaction_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *StorageRequireInterface_BeginDBTransaction_Call) RunAndReturn(run func(context.Context) (pgx.Tx, error)) *StorageRequireInterface_BeginDBTransaction_Call {
	_c.Call.Return(run)
	return _c
}

// Commit provides a mock function with given fields: ctx, dbTx
func (_m *StorageRequireInterface) Commit(ctx context.Context, dbTx pgx.Tx) error {
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

// StorageRequireInterface_Commit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Commit'
type StorageRequireInterface_Commit_Call struct {
	*mock.Call
}

// Commit is a helper method to define mock.On call
//   - ctx context.Context
//   - dbTx pgx.Tx
func (_e *StorageRequireInterface_Expecter) Commit(ctx interface{}, dbTx interface{}) *StorageRequireInterface_Commit_Call {
	return &StorageRequireInterface_Commit_Call{Call: _e.mock.On("Commit", ctx, dbTx)}
}

func (_c *StorageRequireInterface_Commit_Call) Run(run func(ctx context.Context, dbTx pgx.Tx)) *StorageRequireInterface_Commit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(pgx.Tx))
	})
	return _c
}

func (_c *StorageRequireInterface_Commit_Call) Return(_a0 error) *StorageRequireInterface_Commit_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StorageRequireInterface_Commit_Call) RunAndReturn(run func(context.Context, pgx.Tx) error) *StorageRequireInterface_Commit_Call {
	_c.Call.Return(run)
	return _c
}

// GetClaimTxsByStatus provides a mock function with given fields: ctx, statuses, dbTx
func (_m *StorageRequireInterface) GetClaimTxsByStatus(ctx context.Context, statuses []types.MonitoredTxStatus, dbTx pgx.Tx) ([]types.MonitoredTx, error) {
	ret := _m.Called(ctx, statuses, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for GetClaimTxsByStatus")
	}

	var r0 []types.MonitoredTx
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []types.MonitoredTxStatus, pgx.Tx) ([]types.MonitoredTx, error)); ok {
		return rf(ctx, statuses, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []types.MonitoredTxStatus, pgx.Tx) []types.MonitoredTx); ok {
		r0 = rf(ctx, statuses, dbTx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.MonitoredTx)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []types.MonitoredTxStatus, pgx.Tx) error); ok {
		r1 = rf(ctx, statuses, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StorageRequireInterface_GetClaimTxsByStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetClaimTxsByStatus'
type StorageRequireInterface_GetClaimTxsByStatus_Call struct {
	*mock.Call
}

// GetClaimTxsByStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - statuses []types.MonitoredTxStatus
//   - dbTx pgx.Tx
func (_e *StorageRequireInterface_Expecter) GetClaimTxsByStatus(ctx interface{}, statuses interface{}, dbTx interface{}) *StorageRequireInterface_GetClaimTxsByStatus_Call {
	return &StorageRequireInterface_GetClaimTxsByStatus_Call{Call: _e.mock.On("GetClaimTxsByStatus", ctx, statuses, dbTx)}
}

func (_c *StorageRequireInterface_GetClaimTxsByStatus_Call) Run(run func(ctx context.Context, statuses []types.MonitoredTxStatus, dbTx pgx.Tx)) *StorageRequireInterface_GetClaimTxsByStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]types.MonitoredTxStatus), args[2].(pgx.Tx))
	})
	return _c
}

func (_c *StorageRequireInterface_GetClaimTxsByStatus_Call) Return(_a0 []types.MonitoredTx, _a1 error) *StorageRequireInterface_GetClaimTxsByStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *StorageRequireInterface_GetClaimTxsByStatus_Call) RunAndReturn(run func(context.Context, []types.MonitoredTxStatus, pgx.Tx) ([]types.MonitoredTx, error)) *StorageRequireInterface_GetClaimTxsByStatus_Call {
	_c.Call.Return(run)
	return _c
}

// GetLatestMonitoredTxGroupID provides a mock function with given fields: ctx, dbTx
func (_m *StorageRequireInterface) GetLatestMonitoredTxGroupID(ctx context.Context, dbTx pgx.Tx) (uint64, error) {
	ret := _m.Called(ctx, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for GetLatestMonitoredTxGroupID")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx) (uint64, error)); ok {
		return rf(ctx, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, pgx.Tx) uint64); ok {
		r0 = rf(ctx, dbTx)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, pgx.Tx) error); ok {
		r1 = rf(ctx, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StorageRequireInterface_GetLatestMonitoredTxGroupID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetLatestMonitoredTxGroupID'
type StorageRequireInterface_GetLatestMonitoredTxGroupID_Call struct {
	*mock.Call
}

// GetLatestMonitoredTxGroupID is a helper method to define mock.On call
//   - ctx context.Context
//   - dbTx pgx.Tx
func (_e *StorageRequireInterface_Expecter) GetLatestMonitoredTxGroupID(ctx interface{}, dbTx interface{}) *StorageRequireInterface_GetLatestMonitoredTxGroupID_Call {
	return &StorageRequireInterface_GetLatestMonitoredTxGroupID_Call{Call: _e.mock.On("GetLatestMonitoredTxGroupID", ctx, dbTx)}
}

func (_c *StorageRequireInterface_GetLatestMonitoredTxGroupID_Call) Run(run func(ctx context.Context, dbTx pgx.Tx)) *StorageRequireInterface_GetLatestMonitoredTxGroupID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(pgx.Tx))
	})
	return _c
}

func (_c *StorageRequireInterface_GetLatestMonitoredTxGroupID_Call) Return(_a0 uint64, _a1 error) *StorageRequireInterface_GetLatestMonitoredTxGroupID_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *StorageRequireInterface_GetLatestMonitoredTxGroupID_Call) RunAndReturn(run func(context.Context, pgx.Tx) (uint64, error)) *StorageRequireInterface_GetLatestMonitoredTxGroupID_Call {
	_c.Call.Return(run)
	return _c
}

// GetMonitoredTxsGroups provides a mock function with given fields: ctx, groupIds, dbTx
func (_m *StorageRequireInterface) GetMonitoredTxsGroups(ctx context.Context, groupIds []uint64, dbTx pgx.Tx) (map[uint64]types.MonitoredTxGroupDBEntry, error) {
	ret := _m.Called(ctx, groupIds, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for GetMonitoredTxsGroups")
	}

	var r0 map[uint64]types.MonitoredTxGroupDBEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []uint64, pgx.Tx) (map[uint64]types.MonitoredTxGroupDBEntry, error)); ok {
		return rf(ctx, groupIds, dbTx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []uint64, pgx.Tx) map[uint64]types.MonitoredTxGroupDBEntry); ok {
		r0 = rf(ctx, groupIds, dbTx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[uint64]types.MonitoredTxGroupDBEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []uint64, pgx.Tx) error); ok {
		r1 = rf(ctx, groupIds, dbTx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StorageRequireInterface_GetMonitoredTxsGroups_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetMonitoredTxsGroups'
type StorageRequireInterface_GetMonitoredTxsGroups_Call struct {
	*mock.Call
}

// GetMonitoredTxsGroups is a helper method to define mock.On call
//   - ctx context.Context
//   - groupIds []uint64
//   - dbTx pgx.Tx
func (_e *StorageRequireInterface_Expecter) GetMonitoredTxsGroups(ctx interface{}, groupIds interface{}, dbTx interface{}) *StorageRequireInterface_GetMonitoredTxsGroups_Call {
	return &StorageRequireInterface_GetMonitoredTxsGroups_Call{Call: _e.mock.On("GetMonitoredTxsGroups", ctx, groupIds, dbTx)}
}

func (_c *StorageRequireInterface_GetMonitoredTxsGroups_Call) Run(run func(ctx context.Context, groupIds []uint64, dbTx pgx.Tx)) *StorageRequireInterface_GetMonitoredTxsGroups_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]uint64), args[2].(pgx.Tx))
	})
	return _c
}

func (_c *StorageRequireInterface_GetMonitoredTxsGroups_Call) Return(_a0 map[uint64]types.MonitoredTxGroupDBEntry, _a1 error) *StorageRequireInterface_GetMonitoredTxsGroups_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *StorageRequireInterface_GetMonitoredTxsGroups_Call) RunAndReturn(run func(context.Context, []uint64, pgx.Tx) (map[uint64]types.MonitoredTxGroupDBEntry, error)) *StorageRequireInterface_GetMonitoredTxsGroups_Call {
	_c.Call.Return(run)
	return _c
}

// Rollback provides a mock function with given fields: ctx, dbTx
func (_m *StorageRequireInterface) Rollback(ctx context.Context, dbTx pgx.Tx) error {
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

// StorageRequireInterface_Rollback_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Rollback'
type StorageRequireInterface_Rollback_Call struct {
	*mock.Call
}

// Rollback is a helper method to define mock.On call
//   - ctx context.Context
//   - dbTx pgx.Tx
func (_e *StorageRequireInterface_Expecter) Rollback(ctx interface{}, dbTx interface{}) *StorageRequireInterface_Rollback_Call {
	return &StorageRequireInterface_Rollback_Call{Call: _e.mock.On("Rollback", ctx, dbTx)}
}

func (_c *StorageRequireInterface_Rollback_Call) Run(run func(ctx context.Context, dbTx pgx.Tx)) *StorageRequireInterface_Rollback_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(pgx.Tx))
	})
	return _c
}

func (_c *StorageRequireInterface_Rollback_Call) Return(_a0 error) *StorageRequireInterface_Rollback_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StorageRequireInterface_Rollback_Call) RunAndReturn(run func(context.Context, pgx.Tx) error) *StorageRequireInterface_Rollback_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateClaimTx provides a mock function with given fields: ctx, mTx, dbTx
func (_m *StorageRequireInterface) UpdateClaimTx(ctx context.Context, mTx types.MonitoredTx, dbTx pgx.Tx) error {
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

// StorageRequireInterface_UpdateClaimTx_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateClaimTx'
type StorageRequireInterface_UpdateClaimTx_Call struct {
	*mock.Call
}

// UpdateClaimTx is a helper method to define mock.On call
//   - ctx context.Context
//   - mTx types.MonitoredTx
//   - dbTx pgx.Tx
func (_e *StorageRequireInterface_Expecter) UpdateClaimTx(ctx interface{}, mTx interface{}, dbTx interface{}) *StorageRequireInterface_UpdateClaimTx_Call {
	return &StorageRequireInterface_UpdateClaimTx_Call{Call: _e.mock.On("UpdateClaimTx", ctx, mTx, dbTx)}
}

func (_c *StorageRequireInterface_UpdateClaimTx_Call) Run(run func(ctx context.Context, mTx types.MonitoredTx, dbTx pgx.Tx)) *StorageRequireInterface_UpdateClaimTx_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(types.MonitoredTx), args[2].(pgx.Tx))
	})
	return _c
}

func (_c *StorageRequireInterface_UpdateClaimTx_Call) Return(_a0 error) *StorageRequireInterface_UpdateClaimTx_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StorageRequireInterface_UpdateClaimTx_Call) RunAndReturn(run func(context.Context, types.MonitoredTx, pgx.Tx) error) *StorageRequireInterface_UpdateClaimTx_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateMonitoredTxsGroup provides a mock function with given fields: ctx, mTxGroup, dbTx
func (_m *StorageRequireInterface) UpdateMonitoredTxsGroup(ctx context.Context, mTxGroup *types.MonitoredTxGroupDBEntry, dbTx pgx.Tx) error {
	ret := _m.Called(ctx, mTxGroup, dbTx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateMonitoredTxsGroup")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.MonitoredTxGroupDBEntry, pgx.Tx) error); ok {
		r0 = rf(ctx, mTxGroup, dbTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StorageRequireInterface_UpdateMonitoredTxsGroup_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateMonitoredTxsGroup'
type StorageRequireInterface_UpdateMonitoredTxsGroup_Call struct {
	*mock.Call
}

// UpdateMonitoredTxsGroup is a helper method to define mock.On call
//   - ctx context.Context
//   - mTxGroup *types.MonitoredTxGroupDBEntry
//   - dbTx pgx.Tx
func (_e *StorageRequireInterface_Expecter) UpdateMonitoredTxsGroup(ctx interface{}, mTxGroup interface{}, dbTx interface{}) *StorageRequireInterface_UpdateMonitoredTxsGroup_Call {
	return &StorageRequireInterface_UpdateMonitoredTxsGroup_Call{Call: _e.mock.On("UpdateMonitoredTxsGroup", ctx, mTxGroup, dbTx)}
}

func (_c *StorageRequireInterface_UpdateMonitoredTxsGroup_Call) Run(run func(ctx context.Context, mTxGroup *types.MonitoredTxGroupDBEntry, dbTx pgx.Tx)) *StorageRequireInterface_UpdateMonitoredTxsGroup_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*types.MonitoredTxGroupDBEntry), args[2].(pgx.Tx))
	})
	return _c
}

func (_c *StorageRequireInterface_UpdateMonitoredTxsGroup_Call) Return(_a0 error) *StorageRequireInterface_UpdateMonitoredTxsGroup_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StorageRequireInterface_UpdateMonitoredTxsGroup_Call) RunAndReturn(run func(context.Context, *types.MonitoredTxGroupDBEntry, pgx.Tx) error) *StorageRequireInterface_UpdateMonitoredTxsGroup_Call {
	_c.Call.Return(run)
	return _c
}

// NewStorageRequireInterface creates a new instance of StorageRequireInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewStorageRequireInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *StorageRequireInterface {
	mock := &StorageRequireInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
