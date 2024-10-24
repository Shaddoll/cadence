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

//go:generate mockgen -package $GOPACKAGE -source $GOFILE -destination partition_config_provider_mock.go -package matching github.com/uber/cadence/client/matching PartitionConfigProvider

package matching

import (
	"sync"

	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/dynamicconfig"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/types"
)

type (
	// PartitionConfigProvider is the interface for implementers of
	// component that provides partition configuration for task list
	// partitions
	PartitionConfigProvider interface {
		// GetNumberOfReadPartitions returns the number of read partitions
		GetNumberOfReadPartitions(domainID string, taskList types.TaskList, taskListType int) int
		// GetNumberOfWritePartitions returns the number of write partitions
		GetNumberOfWritePartitions(domainID string, taskList types.TaskList, taskListType int) int
		// UpdatePartitionConfig updates the partition configuration for a task list
		UpdatePartitionConfig(domainID string, taskList types.TaskList, taskListType int, config *types.TaskListPartitionConfig)
	}

	syncedTaskListPartitionConfig struct {
		sync.RWMutex
		types.TaskListPartitionConfig
	}

	partitionConfigProviderImpl struct {
		configCache         cache.Cache
		logger              log.Logger
		domainIDToName      func(string) (string, error)
		enableReadFromCache dynamicconfig.BoolPropertyFnWithTaskListInfoFilters
		nReadPartitions     dynamicconfig.IntPropertyFnWithTaskListInfoFilters
		nWritePartitions    dynamicconfig.IntPropertyFnWithTaskListInfoFilters
	}
)

func (c *syncedTaskListPartitionConfig) updateConfig(newConfig types.TaskListPartitionConfig) {
	c.Lock()
	defer c.Unlock()
	if c.Version < newConfig.Version {
		c.TaskListPartitionConfig = newConfig
	}
}

func NewPartitionConfigProvider(
	logger log.Logger,
	domainIDToName func(string) (string, error),
	dc *dynamicconfig.Collection,
) PartitionConfigProvider {
	return &partitionConfigProviderImpl{
		logger:              logger,
		domainIDToName:      domainIDToName,
		enableReadFromCache: dc.GetBoolPropertyFilteredByTaskListInfo(dynamicconfig.MatchingEnableGetNumberOfPartitionsFromCache),
		nReadPartitions:     dc.GetIntPropertyFilteredByTaskListInfo(dynamicconfig.MatchingNumTasklistReadPartitions),
		nWritePartitions:    dc.GetIntPropertyFilteredByTaskListInfo(dynamicconfig.MatchingNumTasklistWritePartitions),
		configCache: cache.New(&cache.Options{
			TTL:             0,
			InitialCapacity: 100,
			Pin:             false,
			MaxCount:        3000,
			ActivelyEvict:   false,
		}),
	}
}

func (p *partitionConfigProviderImpl) GetNumberOfReadPartitions(domainID string, taskList types.TaskList, taskListType int) int {
	domainName, err := p.domainIDToName(domainID)
	if err != nil {
		return 1
	}
	if !p.enableReadFromCache(domainName, taskList.GetName(), taskListType) {
		return p.nReadPartitions(domainName, taskList.GetName(), taskListType)
	}
	c := p.getPartitionConfig(domainID, taskList, taskListType)
	if c == nil {
		return 1
	}
	c.RLock()
	defer c.RUnlock()
	return int(c.NumReadPartitions)
}

func (p *partitionConfigProviderImpl) GetNumberOfWritePartitions(domainID string, taskList types.TaskList, taskListType int) int {
	domainName, err := p.domainIDToName(domainID)
	if err != nil {
		return 1
	}
	if !p.enableReadFromCache(domainName, taskList.GetName(), taskListType) {
		nPartitions := p.nWritePartitions(domainName, taskList.GetName(), taskListType)

		// checks to make sure number of writes never exceeds number of reads
		if nRead := p.nReadPartitions(domainName, taskList.GetName(), taskListType); nPartitions > nRead {
			nPartitions = nRead
		}
		return nPartitions
	}
	c := p.getPartitionConfig(domainID, taskList, taskListType)
	if c == nil {
		return 1
	}
	c.RLock()
	defer c.RUnlock()
	result := c.NumWritePartitions
	if result > c.NumReadPartitions {
		result = c.NumReadPartitions
	}
	return int(result)
}

func (p *partitionConfigProviderImpl) UpdatePartitionConfig(domainID string, taskList types.TaskList, taskListType int, config *types.TaskListPartitionConfig) {
	if config == nil || taskList.GetKind() != types.TaskListKindNormal {
		return
	}
	taskListKey := key{
		domainID:     domainID,
		taskListName: taskList.Name,
		taskListType: taskListType,
	}
	var err error
	cI := p.configCache.Get(taskListKey)
	if cI == nil {
		cI, err = p.configCache.PutIfNotExist(taskListKey, &syncedTaskListPartitionConfig{TaskListPartitionConfig: *config})
		if err != nil {
			return
		}
	}
	c, ok := cI.(*syncedTaskListPartitionConfig)
	if !ok {
		return
	}
	c.updateConfig(*config)
}

func (p *partitionConfigProviderImpl) getPartitionConfig(domainID string, taskList types.TaskList, taskListType int) *syncedTaskListPartitionConfig {
	if taskList.GetKind() != types.TaskListKindNormal {
		return nil
	}
	taskListKey := key{
		domainID:     domainID,
		taskListName: taskList.Name,
		taskListType: taskListType,
	}
	cI := p.configCache.Get(taskListKey)
	if cI == nil {
		return nil
	}
	c, ok := cI.(*syncedTaskListPartitionConfig)
	if !ok {
		return nil
	}
	return c
}
