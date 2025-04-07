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
	"context"
	"sync"
	"time"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/backoff"
	"github.com/uber/cadence/common/clock"
	"github.com/uber/cadence/common/dynamicconfig/dynamicproperties"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/service/history/queue"
	"github.com/uber/cadence/service/history/shard"
	"github.com/uber/cadence/service/history/task"
)

type (
	QueueState struct {
		VirtualQueueStates map[int][]VirtualSliceState
		AckLevelTaskKey    persistence.HistoryTaskKey
	}

	Options struct {
		PageSize                             dynamicproperties.IntPropertyFn
		DeleteBatchSize                      dynamicproperties.IntPropertyFn
		MaxPollRPS                           dynamicproperties.IntPropertyFn
		MaxPollInterval                      dynamicproperties.DurationPropertyFn
		MaxPollIntervalJitterCoefficient     dynamicproperties.FloatPropertyFn
		UpdateAckInterval                    dynamicproperties.DurationPropertyFn
		UpdateAckIntervalJitterCoefficient   dynamicproperties.FloatPropertyFn
		RedispatchInterval                   dynamicproperties.DurationPropertyFn
		MaxRedispatchQueueSize               dynamicproperties.IntPropertyFn
		MaxStartJitterInterval               dynamicproperties.DurationPropertyFn
		SplitQueueInterval                   dynamicproperties.DurationPropertyFn
		SplitQueueIntervalJitterCoefficient  dynamicproperties.FloatPropertyFn
		EnableSplit                          dynamicproperties.BoolPropertyFn
		SplitMaxLevel                        dynamicproperties.IntPropertyFn
		EnableRandomSplitByDomainID          dynamicproperties.BoolPropertyFnWithDomainIDFilter
		RandomSplitProbability               dynamicproperties.FloatPropertyFn
		EnablePendingTaskSplitByDomainID     dynamicproperties.BoolPropertyFnWithDomainIDFilter
		PendingTaskSplitThreshold            dynamicproperties.MapPropertyFn
		EnableStuckTaskSplitByDomainID       dynamicproperties.BoolPropertyFnWithDomainIDFilter
		StuckTaskSplitThreshold              dynamicproperties.MapPropertyFn
		SplitLookAheadDurationByDomainID     dynamicproperties.DurationPropertyFnWithDomainIDFilter
		PollBackoffInterval                  dynamicproperties.DurationPropertyFn
		PollBackoffIntervalJitterCoefficient dynamicproperties.FloatPropertyFn
		EnablePersistQueueStates             dynamicproperties.BoolPropertyFn
		EnableLoadQueueStates                dynamicproperties.BoolPropertyFn
		EnableGracefulSyncShutdown           dynamicproperties.BoolPropertyFn
		EnableValidator                      dynamicproperties.BoolPropertyFn
		ValidationInterval                   dynamicproperties.DurationPropertyFn
		// MaxPendingTaskSize is used in cross cluster queue to limit the pending task count
		MaxPendingTaskSize dynamicproperties.IntPropertyFn
	}

	queueBase struct {
		shard           shard.Context
		taskProcessor   task.Processor
		logger          log.Logger
		metricsClient   metrics.Client
		metricsScope    metrics.Scope
		category        persistence.HistoryTaskCategory
		options         *Options
		timeSource      clock.TimeSource
		taskInitializer task.Initializer

		status                int32
		shutdownWG            sync.WaitGroup
		ctx                   context.Context
		cancel                func()
		redispatcher          task.Redispatcher
		queueReader           QueueReader
		pollTimer             clock.Timer
		updateQueueStateTimer clock.Timer
		lastPollTime          time.Time
		virtualQueueManager   VirtualQueueManager
		ackLevelTaskKey       persistence.HistoryTaskKey

		newVirtualSliceState VirtualSliceState
	}
)

func newQueueBase(
	shard shard.Context,
	taskProcessor task.Processor,
	logger log.Logger,
	metricsClient metrics.Client,
	metricsScope metrics.Scope,
	category persistence.HistoryTaskCategory,
	taskExecutor task.Executor,
	options *Options,
) *queueBase {
	ctx, cancel := context.WithCancel(context.Background())
	redispatcher := task.NewRedispatcher(
		taskProcessor,
		shard.GetTimeSource(),
		&task.RedispatcherOptions{
			TaskRedispatchInterval: options.RedispatchInterval,
		},
		logger,
		metricsScope,
	)
	var queueType task.QueueType
	if category == persistence.HistoryTaskCategoryTransfer {
		queueType = task.QueueTypeActiveTransfer
	} else if category == persistence.HistoryTaskCategoryTimer {
		queueType = task.QueueTypeActiveTimer
	}
	taskInitializer := func(t persistence.Task) task.Task {
		return task.NewTimerTask(
			shard,
			t,
			queueType,
			task.InitializeLoggerForTask(shard.GetShardID(), t, logger),
			func(task persistence.Task) (bool, error) { return true, nil },
			taskExecutor,
			taskProcessor,
			redispatcher.AddTask,
			shard.GetConfig().TaskCriticalRetryCount,
		)
	}
	queueReader := NewQueueReader(shard, category)

	virtualQueueOptions := &VirtualQueueOptions{
		PageSize: options.PageSize,
	}

	virtualQueueStates := make(map[int][]VirtualSliceState)
	var inclusiveNextTaskKey persistence.HistoryTaskKey
	if category == persistence.HistoryTaskCategoryTransfer {
		inclusiveNextTaskKey.TaskID = shard.GetTransferAckLevel() + 1
		virtualQueueStates[rootQueueID] = []VirtualSliceState{
			{
				Range: Range{
					InclusiveMinTaskKey: persistence.HistoryTaskKey{
						TaskID: shard.GetTransferAckLevel() + 1,
					},
					ExclusiveMaxTaskKey: persistence.HistoryTaskKey{
						TaskID: shard.GetTransferMaxReadLevel() + 1,
					},
				},
				Predicate: NewUniversalPredicate(),
			},
		}
	} else if category == persistence.HistoryTaskCategoryTimer {
		inclusiveNextTaskKey.ScheduledTime = shard.UpdateTimerMaxReadLevel(shard.GetClusterMetadata().GetCurrentClusterName())
		virtualQueueStates[rootQueueID] = []VirtualSliceState{
			{
				Range: Range{
					InclusiveMinTaskKey: persistence.HistoryTaskKey{
						ScheduledTime: shard.GetTimerAckLevel(),
					},
					ExclusiveMaxTaskKey: persistence.HistoryTaskKey{
						ScheduledTime: inclusiveNextTaskKey.ScheduledTime,
					},
				},
				Predicate: NewUniversalPredicate(),
			},
		}
	}
	virtualQueueManager := NewVirtualQueueManager(taskProcessor, redispatcher, taskInitializer, queueReader, logger, metricsScope, virtualQueueOptions, virtualQueueStates)
	return &queueBase{
		shard:               shard,
		taskProcessor:       taskProcessor,
		logger:              logger,
		metricsClient:       metricsClient,
		metricsScope:        metricsScope,
		category:            category,
		options:             options,
		timeSource:          shard.GetTimeSource(),
		taskInitializer:     taskInitializer,
		redispatcher:        redispatcher,
		ctx:                 ctx,
		cancel:              cancel,
		status:              common.DaemonStatusInitialized,
		shutdownWG:          sync.WaitGroup{},
		queueReader:         queueReader,
		virtualQueueManager: virtualQueueManager,
		newVirtualSliceState: VirtualSliceState{
			Range: Range{
				InclusiveMinTaskKey: inclusiveNextTaskKey,
				ExclusiveMaxTaskKey: persistence.MaxHistoryTaskKey,
			},
			Predicate: NewUniversalPredicate(),
		},
	}
}

func (q *queueBase) Start() {
	q.redispatcher.Start()
	q.virtualQueueManager.Start()

	q.pollTimer = q.timeSource.NewTimer(backoff.JitDuration(
		q.options.MaxPollInterval(),
		q.options.MaxPollIntervalJitterCoefficient(),
	))
	q.updateQueueStateTimer = q.timeSource.NewTimer(backoff.JitDuration(
		q.options.UpdateAckInterval(),
		q.options.UpdateAckIntervalJitterCoefficient(),
	))
}

func (q *queueBase) Stop() {
	q.updateQueueStateTimer.Stop()
	q.pollTimer.Stop()
	q.virtualQueueManager.Stop()
	q.redispatcher.Stop()
}

func (q *queueBase) Category() persistence.HistoryTaskCategory {
	return q.category
}

func (q *queueBase) FailoverDomain(domainIDs map[string]struct{}) {}

func (q *queueBase) HandleAction(ctx context.Context, clusterName string, action *queue.Action) (*queue.ActionResult, error) {
	return nil, nil
}

func (q *queueBase) LockTaskProcessing() {}

func (q *queueBase) UnlockTaskProcessing() {}

func (q *queueBase) processNewTasks() {
	var newExclusiveMaxTaskKey persistence.HistoryTaskKey
	if q.category == persistence.HistoryTaskCategoryTransfer {
		newExclusiveMaxTaskKey.TaskID = q.shard.GetTransferMaxReadLevel() + 1
	} else if q.category == persistence.HistoryTaskCategoryTimer {
		newExclusiveMaxTaskKey.ScheduledTime = q.shard.UpdateTimerMaxReadLevel(q.shard.GetClusterMetadata().GetCurrentClusterName())
	}

	var newVirtualSliceState VirtualSliceState
	var ok bool
	newVirtualSliceState, q.newVirtualSliceState, ok = q.newVirtualSliceState.TrySplitByTaskKey(newExclusiveMaxTaskKey)
	if !ok {
		q.logger.Warn("Failed to split new virtual slice", tag.Value(newExclusiveMaxTaskKey))
		return
	}
	q.lastPollTime = q.timeSource.Now()

	newVirtualSlice := NewVirtualSlice(newVirtualSliceState, q.taskInitializer, q.queueReader, NewPendingTaskTracker())

	q.virtualQueueManager.AddNewVirtualSlice(newVirtualSlice)
}

func (q *queueBase) processPollTimer() {
	if q.lastPollTime.Add(q.options.PollBackoffInterval()).Before(q.timeSource.Now()) {
		q.processNewTasks()
	}

	q.pollTimer.Reset(backoff.JitDuration(
		q.options.MaxPollInterval(),
		q.options.MaxPollIntervalJitterCoefficient(),
	))
}

func (q *queueBase) updateQueueState() {

	q.virtualQueueManager.UpdateState()
	queueState := q.virtualQueueManager.GetState()

	if queueState.AckLevelTaskKey != persistence.MaxHistoryTaskKey && queueState.AckLevelTaskKey.Compare(q.ackLevelTaskKey) > 0 {
		for {
			pageSize := q.options.DeleteBatchSize()
			resp, err := q.shard.GetExecutionManager().RangeCompleteHistoryTask(q.ctx, &persistence.RangeCompleteHistoryTaskRequest{
				TaskCategory:        q.category,
				InclusiveMinTaskKey: q.ackLevelTaskKey,
				ExclusiveMaxTaskKey: queueState.AckLevelTaskKey,
				PageSize:            pageSize,
			})
			if err != nil {
				return
			}
			if !persistence.HasMoreRowsToDelete(resp.TasksCompleted, pageSize) {
				break
			}
		}
		q.ackLevelTaskKey = queueState.AckLevelTaskKey
	}

	// persist queue state
	if q.category == persistence.HistoryTaskCategoryTransfer {
		persistenceQueueState := toPersistenceTransferQueueState(queueState)
		clusterInfo := q.shard.GetClusterMetadata().GetAllClusterInfo()
		for clusterName := range clusterInfo {
			if err := q.shard.UpdateTransferProcessingQueueStates(clusterName, persistenceQueueState); err != nil {
				q.logger.Error("Failed to update transfer processing queue states", tag.Error(err))
			}
		}
		if err := q.shard.UpdateTransferAckLevel(q.ackLevelTaskKey.TaskID); err != nil {
			q.logger.Error("Failed to update transfer ack level", tag.Error(err))
		}
	} else if q.category == persistence.HistoryTaskCategoryTimer {
		persistenceQueueState := toPersistenceTimerQueueState(queueState)
		clusterInfo := q.shard.GetClusterMetadata().GetAllClusterInfo()
		for clusterName := range clusterInfo {
			if err := q.shard.UpdateTimerProcessingQueueStates(clusterName, persistenceQueueState); err != nil {
				q.logger.Error("Failed to update timer processing queue states", tag.Error(err))
			}
		}
		if err := q.shard.UpdateTimerAckLevel(q.ackLevelTaskKey.ScheduledTime); err != nil {
			q.logger.Error("Failed to update timer ack level", tag.Error(err))
		}
	}
}

func toPersistenceTimerQueueState(state QueueState) []*types.ProcessingQueueState {
	persistenceQueueState := make([]*types.ProcessingQueueState, 0, len(state.VirtualQueueStates))
	for level, virtualQueueState := range state.VirtualQueueStates {
		for _, virtualSliceState := range virtualQueueState {
			persistenceQueueState = append(persistenceQueueState, &types.ProcessingQueueState{
				Level:    common.Ptr(int32(level)),
				AckLevel: common.Ptr(virtualSliceState.Range.InclusiveMinTaskKey.ScheduledTime.UnixNano()),
				MaxLevel: common.Ptr(virtualSliceState.Range.ExclusiveMaxTaskKey.ScheduledTime.UnixNano()),
			})
		}
	}
	return persistenceQueueState
}

func toPersistenceTransferQueueState(state QueueState) []*types.ProcessingQueueState {
	persistenceQueueState := make([]*types.ProcessingQueueState, 0, len(state.VirtualQueueStates))
	for level, virtualQueueState := range state.VirtualQueueStates {
		for _, virtualSliceState := range virtualQueueState {
			persistenceQueueState = append(persistenceQueueState, &types.ProcessingQueueState{
				Level:    common.Ptr(int32(level)),
				AckLevel: common.Ptr(virtualSliceState.Range.InclusiveMinTaskKey.TaskID),
				MaxLevel: common.Ptr(virtualSliceState.Range.ExclusiveMaxTaskKey.TaskID),
			})
		}
	}
	return persistenceQueueState
}
