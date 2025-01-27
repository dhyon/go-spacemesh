// Code generated by MockGen. DO NOT EDIT.
// Source: ./interface.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	types "github.com/spacemeshos/go-spacemesh/common/types"
)

// MockproposalOracle is a mock of proposalOracle interface.
type MockproposalOracle struct {
	ctrl     *gomock.Controller
	recorder *MockproposalOracleMockRecorder
}

// MockproposalOracleMockRecorder is the mock recorder for MockproposalOracle.
type MockproposalOracleMockRecorder struct {
	mock *MockproposalOracle
}

// NewMockproposalOracle creates a new mock instance.
func NewMockproposalOracle(ctrl *gomock.Controller) *MockproposalOracle {
	mock := &MockproposalOracle{ctrl: ctrl}
	mock.recorder = &MockproposalOracleMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockproposalOracle) EXPECT() *MockproposalOracleMockRecorder {
	return m.recorder
}

// GetProposalEligibility mocks base method.
func (m *MockproposalOracle) GetProposalEligibility(arg0 types.LayerID, arg1 types.Beacon) (types.ATXID, []types.ATXID, []types.VotingEligibilityProof, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProposalEligibility", arg0, arg1)
	ret0, _ := ret[0].(types.ATXID)
	ret1, _ := ret[1].([]types.ATXID)
	ret2, _ := ret[2].([]types.VotingEligibilityProof)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// GetProposalEligibility indicates an expected call of GetProposalEligibility.
func (mr *MockproposalOracleMockRecorder) GetProposalEligibility(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProposalEligibility", reflect.TypeOf((*MockproposalOracle)(nil).GetProposalEligibility), arg0, arg1)
}

// MockproposalDB is a mock of proposalDB interface.
type MockproposalDB struct {
	ctrl     *gomock.Controller
	recorder *MockproposalDBMockRecorder
}

// MockproposalDBMockRecorder is the mock recorder for MockproposalDB.
type MockproposalDBMockRecorder struct {
	mock *MockproposalDB
}

// NewMockproposalDB creates a new mock instance.
func NewMockproposalDB(ctrl *gomock.Controller) *MockproposalDB {
	mock := &MockproposalDB{ctrl: ctrl}
	mock.recorder = &MockproposalDBMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockproposalDB) EXPECT() *MockproposalDBMockRecorder {
	return m.recorder
}

// AddProposal mocks base method.
func (m *MockproposalDB) AddProposal(arg0 context.Context, arg1 *types.Proposal) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddProposal", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddProposal indicates an expected call of AddProposal.
func (mr *MockproposalDBMockRecorder) AddProposal(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddProposal", reflect.TypeOf((*MockproposalDB)(nil).AddProposal), arg0, arg1)
}

// MocktxPool is a mock of txPool interface.
type MocktxPool struct {
	ctrl     *gomock.Controller
	recorder *MocktxPoolMockRecorder
}

// MocktxPoolMockRecorder is the mock recorder for MocktxPool.
type MocktxPoolMockRecorder struct {
	mock *MocktxPool
}

// NewMocktxPool creates a new mock instance.
func NewMocktxPool(ctrl *gomock.Controller) *MocktxPool {
	mock := &MocktxPool{ctrl: ctrl}
	mock.recorder = &MocktxPoolMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MocktxPool) EXPECT() *MocktxPoolMockRecorder {
	return m.recorder
}

// SelectTopNTransactions mocks base method.
func (m *MocktxPool) SelectTopNTransactions(numOfTxs int, getState func(types.Address) (uint64, uint64, error)) ([]types.TransactionID, []*types.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SelectTopNTransactions", numOfTxs, getState)
	ret0, _ := ret[0].([]types.TransactionID)
	ret1, _ := ret[1].([]*types.Transaction)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// SelectTopNTransactions indicates an expected call of SelectTopNTransactions.
func (mr *MocktxPoolMockRecorder) SelectTopNTransactions(numOfTxs, getState interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelectTopNTransactions", reflect.TypeOf((*MocktxPool)(nil).SelectTopNTransactions), numOfTxs, getState)
}

// MockbaseBallotProvider is a mock of baseBallotProvider interface.
type MockbaseBallotProvider struct {
	ctrl     *gomock.Controller
	recorder *MockbaseBallotProviderMockRecorder
}

// MockbaseBallotProviderMockRecorder is the mock recorder for MockbaseBallotProvider.
type MockbaseBallotProviderMockRecorder struct {
	mock *MockbaseBallotProvider
}

// NewMockbaseBallotProvider creates a new mock instance.
func NewMockbaseBallotProvider(ctrl *gomock.Controller) *MockbaseBallotProvider {
	mock := &MockbaseBallotProvider{ctrl: ctrl}
	mock.recorder = &MockbaseBallotProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockbaseBallotProvider) EXPECT() *MockbaseBallotProviderMockRecorder {
	return m.recorder
}

// BaseBallot mocks base method.
func (m *MockbaseBallotProvider) BaseBallot(arg0 context.Context) (*types.Votes, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BaseBallot", arg0)
	ret0, _ := ret[0].(*types.Votes)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BaseBallot indicates an expected call of BaseBallot.
func (mr *MockbaseBallotProviderMockRecorder) BaseBallot(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BaseBallot", reflect.TypeOf((*MockbaseBallotProvider)(nil).BaseBallot), arg0)
}

// MockactivationDB is a mock of activationDB interface.
type MockactivationDB struct {
	ctrl     *gomock.Controller
	recorder *MockactivationDBMockRecorder
}

// MockactivationDBMockRecorder is the mock recorder for MockactivationDB.
type MockactivationDBMockRecorder struct {
	mock *MockactivationDB
}

// NewMockactivationDB creates a new mock instance.
func NewMockactivationDB(ctrl *gomock.Controller) *MockactivationDB {
	mock := &MockactivationDB{ctrl: ctrl}
	mock.recorder = &MockactivationDBMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockactivationDB) EXPECT() *MockactivationDBMockRecorder {
	return m.recorder
}

// GetAtxHeader mocks base method.
func (m *MockactivationDB) GetAtxHeader(arg0 types.ATXID) (*types.ActivationTxHeader, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAtxHeader", arg0)
	ret0, _ := ret[0].(*types.ActivationTxHeader)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAtxHeader indicates an expected call of GetAtxHeader.
func (mr *MockactivationDBMockRecorder) GetAtxHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAtxHeader", reflect.TypeOf((*MockactivationDB)(nil).GetAtxHeader), arg0)
}

// GetEpochWeight mocks base method.
func (m *MockactivationDB) GetEpochWeight(arg0 types.EpochID) (uint64, []types.ATXID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEpochWeight", arg0)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].([]types.ATXID)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetEpochWeight indicates an expected call of GetEpochWeight.
func (mr *MockactivationDBMockRecorder) GetEpochWeight(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEpochWeight", reflect.TypeOf((*MockactivationDB)(nil).GetEpochWeight), arg0)
}

// GetNodeAtxIDForEpoch mocks base method.
func (m *MockactivationDB) GetNodeAtxIDForEpoch(nodeID types.NodeID, targetEpoch types.EpochID) (types.ATXID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodeAtxIDForEpoch", nodeID, targetEpoch)
	ret0, _ := ret[0].(types.ATXID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNodeAtxIDForEpoch indicates an expected call of GetNodeAtxIDForEpoch.
func (mr *MockactivationDBMockRecorder) GetNodeAtxIDForEpoch(nodeID, targetEpoch interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodeAtxIDForEpoch", reflect.TypeOf((*MockactivationDB)(nil).GetNodeAtxIDForEpoch), nodeID, targetEpoch)
}

// Mockprojector is a mock of projector interface.
type Mockprojector struct {
	ctrl     *gomock.Controller
	recorder *MockprojectorMockRecorder
}

// MockprojectorMockRecorder is the mock recorder for Mockprojector.
type MockprojectorMockRecorder struct {
	mock *Mockprojector
}

// NewMockprojector creates a new mock instance.
func NewMockprojector(ctrl *gomock.Controller) *Mockprojector {
	mock := &Mockprojector{ctrl: ctrl}
	mock.recorder = &MockprojectorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockprojector) EXPECT() *MockprojectorMockRecorder {
	return m.recorder
}

// GetProjection mocks base method.
func (m *Mockprojector) GetProjection(arg0 types.Address) (uint64, uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProjection", arg0)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(uint64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetProjection indicates an expected call of GetProjection.
func (mr *MockprojectorMockRecorder) GetProjection(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProjection", reflect.TypeOf((*Mockprojector)(nil).GetProjection), arg0)
}
