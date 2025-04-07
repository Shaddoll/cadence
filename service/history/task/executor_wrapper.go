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

package task

import (
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
)

type (
	executorWrapper struct {
		currentClusterName string
		registry           cache.DomainCache
		activeExecutor     Executor
		standbyExecutor    Executor
		logger             log.Logger
	}
)

func NewExecutorWrapper(
	currentClusterName string,
	registry cache.DomainCache,
	activeExecutor Executor,
	standbyExecutor Executor,
	logger log.Logger,
) Executor {
	return &executorWrapper{
		currentClusterName: currentClusterName,
		registry:           registry,
		activeExecutor:     activeExecutor,
		standbyExecutor:    standbyExecutor,
		logger:             logger,
	}
}

func (e *executorWrapper) Stop() {
	e.activeExecutor.Stop()
	e.standbyExecutor.Stop()
}

func (e *executorWrapper) Execute(task Task) (metrics.Scope, error) {
	if e.isActiveTask(task) {
		return e.activeExecutor.Execute(task)
	}

	return e.standbyExecutor.Execute(task)
}

func (e *executorWrapper) isActiveTask(
	task Task,
) bool {
	domainID := task.GetDomainID()
	entry, err := e.registry.GetDomainByID(domainID)
	if err != nil {
		e.logger.Warn("Unable to find namespace, process task as active.", tag.WorkflowDomainID(domainID), tag.Value(task.GetInfo()), tag.Error(err))
		return true
	}

	if isActive, err := entry.IsActiveIn(e.currentClusterName); err != nil || !isActive {
		e.logger.Debug("Process task as standby.", tag.WorkflowDomainID(domainID), tag.Value(task.GetInfo()))
		return false
	}

	e.logger.Debug("Process task as active.", tag.WorkflowDomainID(domainID), tag.Value(task.GetInfo()))
	return true
}
