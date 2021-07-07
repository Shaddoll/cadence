// The MIT License (MIT)

// Copyright (c) 2017-2020 Uber Technologies Inc.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Code generated by MockGen. DO NOT EDIT.
// Source: adminHandler.go

// Package frontend is a generated GoMock package.
package frontend

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	types "github.com/uber/cadence/common/types"
)

// MockAdminHandler is a mock of AdminHandler interface
type MockAdminHandler struct {
	ctrl     *gomock.Controller
	recorder *MockAdminHandlerMockRecorder
}

// MockAdminHandlerMockRecorder is the mock recorder for MockAdminHandler
type MockAdminHandlerMockRecorder struct {
	mock *MockAdminHandler
}

// NewMockAdminHandler creates a new mock instance
func NewMockAdminHandler(ctrl *gomock.Controller) *MockAdminHandler {
	mock := &MockAdminHandler{ctrl: ctrl}
	mock.recorder = &MockAdminHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAdminHandler) EXPECT() *MockAdminHandlerMockRecorder {
	return m.recorder
}

// Start mocks base method
func (m *MockAdminHandler) Start() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start")
}

// Start indicates an expected call of Start
func (mr *MockAdminHandlerMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockAdminHandler)(nil).Start))
}

// Stop mocks base method
func (m *MockAdminHandler) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop
func (mr *MockAdminHandlerMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockAdminHandler)(nil).Stop))
}

// AddSearchAttribute mocks base method
func (m *MockAdminHandler) AddSearchAttribute(arg0 context.Context, arg1 *types.AddSearchAttributeRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddSearchAttribute", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddSearchAttribute indicates an expected call of AddSearchAttribute
func (mr *MockAdminHandlerMockRecorder) AddSearchAttribute(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddSearchAttribute", reflect.TypeOf((*MockAdminHandler)(nil).AddSearchAttribute), arg0, arg1)
}

// CloseShard mocks base method
func (m *MockAdminHandler) CloseShard(arg0 context.Context, arg1 *types.CloseShardRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseShard", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseShard indicates an expected call of CloseShard
func (mr *MockAdminHandlerMockRecorder) CloseShard(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseShard", reflect.TypeOf((*MockAdminHandler)(nil).CloseShard), arg0, arg1)
}

// DescribeCluster mocks base method
func (m *MockAdminHandler) DescribeCluster(arg0 context.Context) (*types.DescribeClusterResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DescribeCluster", arg0)
	ret0, _ := ret[0].(*types.DescribeClusterResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DescribeCluster indicates an expected call of DescribeCluster
func (mr *MockAdminHandlerMockRecorder) DescribeCluster(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DescribeCluster", reflect.TypeOf((*MockAdminHandler)(nil).DescribeCluster), arg0)
}

// DescribeShardDistribution mocks base method
func (m *MockAdminHandler) DescribeShardDistribution(arg0 context.Context, arg1 *types.DescribeShardDistributionRequest) (*types.DescribeShardDistributionResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DescribeShardDistribution", arg0, arg1)
	ret0, _ := ret[0].(*types.DescribeShardDistributionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DescribeShardDistribution indicates an expected call of DescribeShardDistribution
func (mr *MockAdminHandlerMockRecorder) DescribeShardDistribution(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DescribeShardDistribution", reflect.TypeOf((*MockAdminHandler)(nil).DescribeShardDistribution), arg0, arg1)
}

// DescribeHistoryHost mocks base method
func (m *MockAdminHandler) DescribeHistoryHost(arg0 context.Context, arg1 *types.DescribeHistoryHostRequest) (*types.DescribeHistoryHostResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DescribeHistoryHost", arg0, arg1)
	ret0, _ := ret[0].(*types.DescribeHistoryHostResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DescribeHistoryHost indicates an expected call of DescribeHistoryHost
func (mr *MockAdminHandlerMockRecorder) DescribeHistoryHost(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DescribeHistoryHost", reflect.TypeOf((*MockAdminHandler)(nil).DescribeHistoryHost), arg0, arg1)
}

// DescribeQueue mocks base method
func (m *MockAdminHandler) DescribeQueue(arg0 context.Context, arg1 *types.DescribeQueueRequest) (*types.DescribeQueueResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DescribeQueue", arg0, arg1)
	ret0, _ := ret[0].(*types.DescribeQueueResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DescribeQueue indicates an expected call of DescribeQueue
func (mr *MockAdminHandlerMockRecorder) DescribeQueue(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DescribeQueue", reflect.TypeOf((*MockAdminHandler)(nil).DescribeQueue), arg0, arg1)
}

// DescribeWorkflowExecution mocks base method
func (m *MockAdminHandler) DescribeWorkflowExecution(arg0 context.Context, arg1 *types.AdminDescribeWorkflowExecutionRequest) (*types.AdminDescribeWorkflowExecutionResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DescribeWorkflowExecution", arg0, arg1)
	ret0, _ := ret[0].(*types.AdminDescribeWorkflowExecutionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DescribeWorkflowExecution indicates an expected call of DescribeWorkflowExecution
func (mr *MockAdminHandlerMockRecorder) DescribeWorkflowExecution(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DescribeWorkflowExecution", reflect.TypeOf((*MockAdminHandler)(nil).DescribeWorkflowExecution), arg0, arg1)
}

// GetDLQReplicationMessages mocks base method
func (m *MockAdminHandler) GetDLQReplicationMessages(arg0 context.Context, arg1 *types.GetDLQReplicationMessagesRequest) (*types.GetDLQReplicationMessagesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDLQReplicationMessages", arg0, arg1)
	ret0, _ := ret[0].(*types.GetDLQReplicationMessagesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDLQReplicationMessages indicates an expected call of GetDLQReplicationMessages
func (mr *MockAdminHandlerMockRecorder) GetDLQReplicationMessages(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDLQReplicationMessages", reflect.TypeOf((*MockAdminHandler)(nil).GetDLQReplicationMessages), arg0, arg1)
}

// GetDomainReplicationMessages mocks base method
func (m *MockAdminHandler) GetDomainReplicationMessages(arg0 context.Context, arg1 *types.GetDomainReplicationMessagesRequest) (*types.GetDomainReplicationMessagesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDomainReplicationMessages", arg0, arg1)
	ret0, _ := ret[0].(*types.GetDomainReplicationMessagesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDomainReplicationMessages indicates an expected call of GetDomainReplicationMessages
func (mr *MockAdminHandlerMockRecorder) GetDomainReplicationMessages(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDomainReplicationMessages", reflect.TypeOf((*MockAdminHandler)(nil).GetDomainReplicationMessages), arg0, arg1)
}

// GetReplicationMessages mocks base method
func (m *MockAdminHandler) GetReplicationMessages(arg0 context.Context, arg1 *types.GetReplicationMessagesRequest) (*types.GetReplicationMessagesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetReplicationMessages", arg0, arg1)
	ret0, _ := ret[0].(*types.GetReplicationMessagesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetReplicationMessages indicates an expected call of GetReplicationMessages
func (mr *MockAdminHandlerMockRecorder) GetReplicationMessages(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetReplicationMessages", reflect.TypeOf((*MockAdminHandler)(nil).GetReplicationMessages), arg0, arg1)
}

// GetWorkflowExecutionRawHistoryV2 mocks base method
func (m *MockAdminHandler) GetWorkflowExecutionRawHistoryV2(arg0 context.Context, arg1 *types.GetWorkflowExecutionRawHistoryV2Request) (*types.GetWorkflowExecutionRawHistoryV2Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorkflowExecutionRawHistoryV2", arg0, arg1)
	ret0, _ := ret[0].(*types.GetWorkflowExecutionRawHistoryV2Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWorkflowExecutionRawHistoryV2 indicates an expected call of GetWorkflowExecutionRawHistoryV2
func (mr *MockAdminHandlerMockRecorder) GetWorkflowExecutionRawHistoryV2(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorkflowExecutionRawHistoryV2", reflect.TypeOf((*MockAdminHandler)(nil).GetWorkflowExecutionRawHistoryV2), arg0, arg1)
}

// MergeDLQMessages mocks base method
func (m *MockAdminHandler) MergeDLQMessages(arg0 context.Context, arg1 *types.MergeDLQMessagesRequest) (*types.MergeDLQMessagesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MergeDLQMessages", arg0, arg1)
	ret0, _ := ret[0].(*types.MergeDLQMessagesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MergeDLQMessages indicates an expected call of MergeDLQMessages
func (mr *MockAdminHandlerMockRecorder) MergeDLQMessages(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MergeDLQMessages", reflect.TypeOf((*MockAdminHandler)(nil).MergeDLQMessages), arg0, arg1)
}

// PurgeDLQMessages mocks base method
func (m *MockAdminHandler) PurgeDLQMessages(arg0 context.Context, arg1 *types.PurgeDLQMessagesRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PurgeDLQMessages", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PurgeDLQMessages indicates an expected call of PurgeDLQMessages
func (mr *MockAdminHandlerMockRecorder) PurgeDLQMessages(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PurgeDLQMessages", reflect.TypeOf((*MockAdminHandler)(nil).PurgeDLQMessages), arg0, arg1)
}

// ReadDLQMessages mocks base method
func (m *MockAdminHandler) ReadDLQMessages(arg0 context.Context, arg1 *types.ReadDLQMessagesRequest) (*types.ReadDLQMessagesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadDLQMessages", arg0, arg1)
	ret0, _ := ret[0].(*types.ReadDLQMessagesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadDLQMessages indicates an expected call of ReadDLQMessages
func (mr *MockAdminHandlerMockRecorder) ReadDLQMessages(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadDLQMessages", reflect.TypeOf((*MockAdminHandler)(nil).ReadDLQMessages), arg0, arg1)
}

// ReapplyEvents mocks base method
func (m *MockAdminHandler) ReapplyEvents(arg0 context.Context, arg1 *types.ReapplyEventsRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReapplyEvents", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReapplyEvents indicates an expected call of ReapplyEvents
func (mr *MockAdminHandlerMockRecorder) ReapplyEvents(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReapplyEvents", reflect.TypeOf((*MockAdminHandler)(nil).ReapplyEvents), arg0, arg1)
}

// RefreshWorkflowTasks mocks base method
func (m *MockAdminHandler) RefreshWorkflowTasks(arg0 context.Context, arg1 *types.RefreshWorkflowTasksRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RefreshWorkflowTasks", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// RefreshWorkflowTasks indicates an expected call of RefreshWorkflowTasks
func (mr *MockAdminHandlerMockRecorder) RefreshWorkflowTasks(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RefreshWorkflowTasks", reflect.TypeOf((*MockAdminHandler)(nil).RefreshWorkflowTasks), arg0, arg1)
}

// RemoveTask mocks base method
func (m *MockAdminHandler) RemoveTask(arg0 context.Context, arg1 *types.RemoveTaskRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveTask", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveTask indicates an expected call of RemoveTask
func (mr *MockAdminHandlerMockRecorder) RemoveTask(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveTask", reflect.TypeOf((*MockAdminHandler)(nil).RemoveTask), arg0, arg1)
}

// ResendReplicationTasks mocks base method
func (m *MockAdminHandler) ResendReplicationTasks(arg0 context.Context, arg1 *types.ResendReplicationTasksRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResendReplicationTasks", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// ResendReplicationTasks indicates an expected call of ResendReplicationTasks
func (mr *MockAdminHandlerMockRecorder) ResendReplicationTasks(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResendReplicationTasks", reflect.TypeOf((*MockAdminHandler)(nil).ResendReplicationTasks), arg0, arg1)
}

// ResetQueue mocks base method
func (m *MockAdminHandler) ResetQueue(arg0 context.Context, arg1 *types.ResetQueueRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResetQueue", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// ResetQueue indicates an expected call of ResetQueue
func (mr *MockAdminHandlerMockRecorder) ResetQueue(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetQueue", reflect.TypeOf((*MockAdminHandler)(nil).ResetQueue), arg0, arg1)
}

// GetCrossClusterTasks mocks base method
func (m *MockAdminHandler) GetCrossClusterTasks(arg0 context.Context, arg1 *types.GetCrossClusterTasksRequest) (*types.GetCrossClusterTasksResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCrossClusterTasks", arg0, arg1)
	ret0, _ := ret[0].(*types.GetCrossClusterTasksResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCrossClusterTasks indicates an expected call of GetCrossClusterTasks
func (mr *MockAdminHandlerMockRecorder) GetCrossClusterTasks(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCrossClusterTasks", reflect.TypeOf((*MockAdminHandler)(nil).GetCrossClusterTasks), arg0, arg1)
}
