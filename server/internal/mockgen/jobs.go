// Code generated by MockGen. DO NOT EDIT.
// Source: jobs.go

// Package mockgen is a generated GoMock package.
package mockgen

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockJobController is a mock of JobController interface.
type MockJobController struct {
	ctrl     *gomock.Controller
	recorder *MockJobControllerMockRecorder
}

// MockJobControllerMockRecorder is the mock recorder for MockJobController.
type MockJobControllerMockRecorder struct {
	mock *MockJobController
}

// NewMockJobController creates a new mock instance.
func NewMockJobController(ctrl *gomock.Controller) *MockJobController {
	mock := &MockJobController{ctrl: ctrl}
	mock.recorder = &MockJobControllerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockJobController) EXPECT() *MockJobControllerMockRecorder {
	return m.recorder
}

// EnqueueJob mocks base method.
func (m *MockJobController) EnqueueJob(ctx context.Context, queue string, data interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnqueueJob", ctx, queue, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// EnqueueJob indicates an expected call of EnqueueJob.
func (mr *MockJobControllerMockRecorder) EnqueueJob(ctx, queue, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnqueueJob", reflect.TypeOf((*MockJobController)(nil).EnqueueJob), ctx, queue, data)
}
