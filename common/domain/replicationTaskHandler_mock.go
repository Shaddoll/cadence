// Code generated by MockGen. DO NOT EDIT.
// Source: replicationTaskExecutor.go
//
// Generated by this command:
//
//	mockgen -package domain -source replicationTaskExecutor.go -destination replicationTaskHandler_mock.go
//

// Package domain is a generated GoMock package.
package domain

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"

	types "github.com/uber/cadence/common/types"
)

// MockReplicationTaskExecutor is a mock of ReplicationTaskExecutor interface.
type MockReplicationTaskExecutor struct {
	ctrl     *gomock.Controller
	recorder *MockReplicationTaskExecutorMockRecorder
	isgomock struct{}
}

// MockReplicationTaskExecutorMockRecorder is the mock recorder for MockReplicationTaskExecutor.
type MockReplicationTaskExecutorMockRecorder struct {
	mock *MockReplicationTaskExecutor
}

// NewMockReplicationTaskExecutor creates a new mock instance.
func NewMockReplicationTaskExecutor(ctrl *gomock.Controller) *MockReplicationTaskExecutor {
	mock := &MockReplicationTaskExecutor{ctrl: ctrl}
	mock.recorder = &MockReplicationTaskExecutorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockReplicationTaskExecutor) EXPECT() *MockReplicationTaskExecutorMockRecorder {
	return m.recorder
}

// Execute mocks base method.
func (m *MockReplicationTaskExecutor) Execute(task *types.DomainTaskAttributes) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Execute", task)
	ret0, _ := ret[0].(error)
	return ret0
}

// Execute indicates an expected call of Execute.
func (mr *MockReplicationTaskExecutorMockRecorder) Execute(task any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Execute", reflect.TypeOf((*MockReplicationTaskExecutor)(nil).Execute), task)
}
