// Code generated by MockGen. DO NOT EDIT.
// Source: client.go
//
// Generated by this command:
//
//	mockgen -source=client.go -package=mockgen -destination=../internal/mockgen/platypus_client.go Client
//

// Package mockgen is a generated GoMock package.
package mockgen

import (
	context "context"
	reflect "reflect"
	time "time"

	platypus "github.com/monetr/monetr/server/platypus"
	gomock "go.uber.org/mock/gomock"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// GetAccounts mocks base method.
func (m *MockClient) GetAccounts(ctx context.Context, accountIds ...string) ([]platypus.BankAccount, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx}
	for _, a := range accountIds {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetAccounts", varargs...)
	ret0, _ := ret[0].([]platypus.BankAccount)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccounts indicates an expected call of GetAccounts.
func (mr *MockClientMockRecorder) GetAccounts(ctx any, accountIds ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx}, accountIds...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccounts", reflect.TypeOf((*MockClient)(nil).GetAccounts), varargs...)
}

// GetAllTransactions mocks base method.
func (m *MockClient) GetAllTransactions(ctx context.Context, start, end time.Time, accountIds []string) ([]platypus.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllTransactions", ctx, start, end, accountIds)
	ret0, _ := ret[0].([]platypus.Transaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllTransactions indicates an expected call of GetAllTransactions.
func (mr *MockClientMockRecorder) GetAllTransactions(ctx, start, end, accountIds any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllTransactions", reflect.TypeOf((*MockClient)(nil).GetAllTransactions), ctx, start, end, accountIds)
}

// RefeshTransactions mocks base method.
func (m *MockClient) RefeshTransactions(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RefeshTransactions", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// RefeshTransactions indicates an expected call of RefeshTransactions.
func (mr *MockClientMockRecorder) RefeshTransactions(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RefeshTransactions", reflect.TypeOf((*MockClient)(nil).RefeshTransactions), ctx)
}

// RemoveItem mocks base method.
func (m *MockClient) RemoveItem(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveItem", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveItem indicates an expected call of RemoveItem.
func (mr *MockClientMockRecorder) RemoveItem(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveItem", reflect.TypeOf((*MockClient)(nil).RemoveItem), ctx)
}

// Sync mocks base method.
func (m *MockClient) Sync(ctx context.Context, cursor *string) (*platypus.SyncResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Sync", ctx, cursor)
	ret0, _ := ret[0].(*platypus.SyncResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Sync indicates an expected call of Sync.
func (mr *MockClientMockRecorder) Sync(ctx, cursor any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Sync", reflect.TypeOf((*MockClient)(nil).Sync), ctx, cursor)
}

// UpdateItem mocks base method.
func (m *MockClient) UpdateItem(ctx context.Context, updateAccountSelection bool) (platypus.LinkToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateItem", ctx, updateAccountSelection)
	ret0, _ := ret[0].(platypus.LinkToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateItem indicates an expected call of UpdateItem.
func (mr *MockClientMockRecorder) UpdateItem(ctx, updateAccountSelection any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateItem", reflect.TypeOf((*MockClient)(nil).UpdateItem), ctx, updateAccountSelection)
}
