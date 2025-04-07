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

package queuev2

import (
	"sync"
	"sync/atomic"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/service/history/task"
)

const (
	rootQueueID = 0
)

type (
	VirtualQueueManager interface {
		common.Daemon
		GetState() QueueState
		UpdateState()
		AddNewVirtualSlice(VirtualSlice)
	}

	virtualQueueManagerImpl struct {
		processor       task.Processor
		taskInitializer task.Initializer
		redispatcher    task.Redispatcher
		queueReader     QueueReader
		logger          log.Logger
		metricsScope    metrics.Scope
		options         *VirtualQueueOptions

		sync.RWMutex
		status        int32
		virtualQueues map[int]VirtualQueue
	}
)

func NewVirtualQueueManager(
	processor task.Processor,
	redispatcher task.Redispatcher,
	taskInitializer task.Initializer,
	queueReader QueueReader,
	logger log.Logger,
	metricsScope metrics.Scope,
	options *VirtualQueueOptions,
	virtualQueueStates map[int][]VirtualSliceState,
) VirtualQueueManager {
	virtualQueues := make(map[int]VirtualQueue)
	for queueID, states := range virtualQueueStates {
		virtualSlices := make([]VirtualSlice, len(states))
		for i, state := range states {
			virtualSlices[i] = NewVirtualSlice(state, taskInitializer, queueReader, NewPendingTaskTracker())
		}
		virtualQueues[queueID] = NewVirtualQueue(processor, redispatcher, logger, metricsScope, virtualSlices, options)
	}
	return &virtualQueueManagerImpl{
		processor:       processor,
		taskInitializer: taskInitializer,
		queueReader:     queueReader,
		redispatcher:    redispatcher,
		logger:          logger,
		metricsScope:    metricsScope,
		options:         options,
		status:          common.DaemonStatusInitialized,
		virtualQueues:   virtualQueues,
	}
}

func (m *virtualQueueManagerImpl) Start() {
	if !atomic.CompareAndSwapInt32(&m.status, common.DaemonStatusInitialized, common.DaemonStatusStarted) {
		return
	}

	m.RLock()
	defer m.RUnlock()

	for _, vq := range m.virtualQueues {
		vq.Start()
	}
}

func (m *virtualQueueManagerImpl) Stop() {
	if !atomic.CompareAndSwapInt32(&m.status, common.DaemonStatusStarted, common.DaemonStatusStopped) {
		return
	}

	m.RLock()
	defer m.RUnlock()

	for _, vq := range m.virtualQueues {
		vq.Stop()
	}
}

func (m *virtualQueueManagerImpl) GetState() QueueState {
	m.RLock()
	defer m.RUnlock()

	virtualQueueStates := make(map[int][]VirtualSliceState)
	ackLevelTaskKey := persistence.MaxHistoryTaskKey
	for key, vq := range m.virtualQueues {
		state := vq.GetState()
		if len(state) > 0 {
			virtualQueueStates[key] = state
			if state[0].Range.InclusiveMinTaskKey.Compare(ackLevelTaskKey) < 0 {
				ackLevelTaskKey = state[0].Range.InclusiveMinTaskKey
			}
		}
	}

	return QueueState{
		VirtualQueueStates: virtualQueueStates,
		AckLevelTaskKey:    ackLevelTaskKey,
	}
}

func (m *virtualQueueManagerImpl) UpdateState() {
	m.RLock()
	defer m.RUnlock()

	for _, vq := range m.virtualQueues {
		vq.UpdateState()
	}
}

func (m *virtualQueueManagerImpl) AddNewVirtualSlice(s VirtualSlice) {
	m.RLock()
	if vq, ok := m.virtualQueues[rootQueueID]; ok {
		m.RUnlock()
		vq.MergeSlices(s)
		return
	}
	m.RUnlock()

	m.Lock()
	defer m.Unlock()
	if vq, ok := m.virtualQueues[rootQueueID]; ok {
		vq.MergeSlices(s)
		return
	}

	m.virtualQueues[rootQueueID] = NewVirtualQueue(m.processor, m.redispatcher, m.logger, m.metricsScope, []VirtualSlice{s}, m.options)
	m.virtualQueues[rootQueueID].Start()
}
