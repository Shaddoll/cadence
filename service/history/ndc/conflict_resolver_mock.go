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
// Source: conflict_resolver.go

// Package ndc is a generated GoMock package.
package ndc

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	execution "github.com/uber/cadence/service/history/execution"
)

// MockconflictResolver is a mock of conflictResolver interface
type MockconflictResolver struct {
	ctrl     *gomock.Controller
	recorder *MockconflictResolverMockRecorder
}

// MockconflictResolverMockRecorder is the mock recorder for MockconflictResolver
type MockconflictResolverMockRecorder struct {
	mock *MockconflictResolver
}

// NewMockconflictResolver creates a new mock instance
func NewMockconflictResolver(ctrl *gomock.Controller) *MockconflictResolver {
	mock := &MockconflictResolver{ctrl: ctrl}
	mock.recorder = &MockconflictResolverMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockconflictResolver) EXPECT() *MockconflictResolverMockRecorder {
	return m.recorder
}

// prepareMutableState mocks base method
func (m *MockconflictResolver) prepareMutableState(ctx context.Context, branchIndex int, incomingVersion int64) (execution.MutableState, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "prepareMutableState", ctx, branchIndex, incomingVersion)
	ret0, _ := ret[0].(execution.MutableState)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// prepareMutableState indicates an expected call of prepareMutableState
func (mr *MockconflictResolverMockRecorder) prepareMutableState(ctx, branchIndex, incomingVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "prepareMutableState", reflect.TypeOf((*MockconflictResolver)(nil).prepareMutableState), ctx, branchIndex, incomingVersion)
}