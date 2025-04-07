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

	"github.com/uber/cadence/common/dynamicconfig/dynamicproperties"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/ndc"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/reconciliation/invariant"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/service/history/engine"
	"github.com/uber/cadence/service/history/execution"
	"github.com/uber/cadence/service/history/queue"
	"github.com/uber/cadence/service/history/reset"
	"github.com/uber/cadence/service/history/shard"
	"github.com/uber/cadence/service/history/task"
	"github.com/uber/cadence/service/history/workflowcache"
	"github.com/uber/cadence/service/worker/archiver"
)

type (
	ProcessorFactory interface {
		NewTransferQueueProcessor(
			shard shard.Context,
			historyEngine engine.Engine,
			taskProcessor task.Processor,
			executionCache execution.Cache,
			workflowResetter reset.WorkflowResetter,
			archivalClient archiver.Client,
			executionCheck invariant.Invariant,
			wfIDCache workflowcache.WFCache,
		) queue.Processor

		NewTimerQueueProcessor(
			shard shard.Context,
			historyEngine engine.Engine,
			taskProcessor task.Processor,
			executionCache execution.Cache,
			archivalClient archiver.Client,
			executionCheck invariant.Invariant,
		) queue.Processor
	}

	factoryImpl struct{}
)

func NewProcessorFactory() ProcessorFactory {
	return &factoryImpl{}
}

func (f *factoryImpl) NewTransferQueueProcessor(
	shard shard.Context,
	historyEngine engine.Engine,
	taskProcessor task.Processor,
	executionCache execution.Cache,
	workflowResetter reset.WorkflowResetter,
	archivalClient archiver.Client,
	executionCheck invariant.Invariant,
	wfIDCache workflowcache.WFCache,
) queue.Processor {
	activeTaskExecutor := task.NewTransferActiveTaskExecutor(
		shard,
		archivalClient,
		executionCache,
		workflowResetter,
		shard.GetLogger(),
		shard.GetConfig(),
		wfIDCache,
	)

	historyResender := ndc.NewHistoryResender(
		shard.GetDomainCache(),
		shard.GetService().GetClientBean().GetRemoteAdminClient(shard.GetClusterMetadata().GetCurrentClusterName()),
		func(ctx context.Context, request *types.ReplicateEventsV2Request) error {
			return historyEngine.ReplicateEventsV2(ctx, request)
		},
		shard.GetConfig().StandbyTaskReReplicationContextTimeout,
		executionCheck,
		shard.GetLogger(),
	)
	standbyTaskExecutor := task.NewTransferStandbyTaskExecutor(
		shard,
		archivalClient,
		executionCache,
		historyResender,
		shard.GetLogger(),
		shard.GetClusterMetadata().GetCurrentClusterName(),
		shard.GetConfig(),
	)

	executorWrapper := task.NewExecutorWrapper(
		shard.GetClusterMetadata().GetCurrentClusterName(),
		shard.GetDomainCache(),
		activeTaskExecutor,
		standbyTaskExecutor,
		shard.GetLogger(),
	)
	config := shard.GetConfig()
	return NewImmediateQueue(
		shard,
		persistence.HistoryTaskCategoryTransfer,
		taskProcessor,
		executorWrapper,
		shard.GetLogger(),
		shard.GetMetricsClient(),
		shard.GetMetricsClient().Scope(metrics.TransferActiveQueueProcessorScope),
		&Options{
			PageSize:                             config.TransferTaskBatchSize,
			DeleteBatchSize:                      config.TransferTaskDeleteBatchSize,
			MaxPollRPS:                           config.TransferProcessorMaxPollRPS,
			MaxPollInterval:                      config.TransferProcessorMaxPollInterval,
			MaxPollIntervalJitterCoefficient:     config.TransferProcessorMaxPollIntervalJitterCoefficient,
			UpdateAckInterval:                    config.TransferProcessorUpdateAckInterval,
			UpdateAckIntervalJitterCoefficient:   config.TransferProcessorUpdateAckIntervalJitterCoefficient,
			MaxRedispatchQueueSize:               config.TransferProcessorMaxRedispatchQueueSize,
			SplitQueueInterval:                   config.TransferProcessorSplitQueueInterval,
			SplitQueueIntervalJitterCoefficient:  config.TransferProcessorSplitQueueIntervalJitterCoefficient,
			PollBackoffInterval:                  config.QueueProcessorPollBackoffInterval,
			PollBackoffIntervalJitterCoefficient: config.QueueProcessorPollBackoffIntervalJitterCoefficient,
			EnableValidator:                      config.TransferProcessorEnableValidator,
			ValidationInterval:                   config.TransferProcessorValidationInterval,
			EnableGracefulSyncShutdown:           config.QueueProcessorEnableGracefulSyncShutdown,
			EnableSplit:                          config.QueueProcessorEnableSplit,
			SplitMaxLevel:                        config.QueueProcessorSplitMaxLevel,
			EnableRandomSplitByDomainID:          config.QueueProcessorEnableRandomSplitByDomainID,
			RandomSplitProbability:               config.QueueProcessorRandomSplitProbability,
			EnablePendingTaskSplitByDomainID:     config.QueueProcessorEnablePendingTaskSplitByDomainID,
			PendingTaskSplitThreshold:            config.QueueProcessorPendingTaskSplitThreshold,
			EnableStuckTaskSplitByDomainID:       config.QueueProcessorEnableStuckTaskSplitByDomainID,
			StuckTaskSplitThreshold:              config.QueueProcessorStuckTaskSplitThreshold,
			SplitLookAheadDurationByDomainID:     config.QueueProcessorSplitLookAheadDurationByDomainID,
			EnablePersistQueueStates:             config.QueueProcessorEnablePersistQueueStates,
			EnableLoadQueueStates:                config.QueueProcessorEnableLoadQueueStates,
			MaxStartJitterInterval:               dynamicproperties.GetDurationPropertyFn(0),
			RedispatchInterval:                   config.ActiveTaskRedispatchInterval,
		},
	)
}

func (f *factoryImpl) NewTimerQueueProcessor(
	shard shard.Context,
	historyEngine engine.Engine,
	taskProcessor task.Processor,
	executionCache execution.Cache,
	archivalClient archiver.Client,
	executionCheck invariant.Invariant,
) queue.Processor {
	activeTaskExecutor := task.NewTimerActiveTaskExecutor(
		shard,
		archivalClient,
		executionCache,
		shard.GetLogger(),
		shard.GetMetricsClient(),
		shard.GetConfig(),
	)
	historyResender := ndc.NewHistoryResender(
		shard.GetDomainCache(),
		shard.GetService().GetClientBean().GetRemoteAdminClient(shard.GetClusterMetadata().GetCurrentClusterName()),
		func(ctx context.Context, request *types.ReplicateEventsV2Request) error {
			return historyEngine.ReplicateEventsV2(ctx, request)
		},
		shard.GetConfig().StandbyTaskReReplicationContextTimeout,
		executionCheck,
		shard.GetLogger(),
	)
	standbyTaskExecutor := task.NewTimerStandbyTaskExecutor(
		shard,
		archivalClient,
		executionCache,
		historyResender,
		shard.GetLogger(),
		shard.GetMetricsClient(),
		shard.GetClusterMetadata().GetCurrentClusterName(),
		shard.GetConfig(),
	)
	executorWrapper := task.NewExecutorWrapper(
		shard.GetClusterMetadata().GetCurrentClusterName(),
		shard.GetDomainCache(),
		activeTaskExecutor,
		standbyTaskExecutor,
		shard.GetLogger(),
	)
	config := shard.GetConfig()
	return NewScheduledQueue(
		shard,
		persistence.HistoryTaskCategoryTimer,
		taskProcessor,
		executorWrapper,
		shard.GetLogger(),
		shard.GetMetricsClient(),
		shard.GetMetricsClient().Scope(metrics.TimerActiveQueueProcessorScope),
		&Options{
			PageSize:                             config.TimerTaskBatchSize,
			DeleteBatchSize:                      config.TimerTaskDeleteBatchSize,
			MaxPollRPS:                           config.TimerProcessorMaxPollRPS,
			MaxPollInterval:                      config.TimerProcessorMaxPollInterval,
			MaxPollIntervalJitterCoefficient:     config.TimerProcessorMaxPollIntervalJitterCoefficient,
			UpdateAckInterval:                    config.TimerProcessorUpdateAckInterval,
			UpdateAckIntervalJitterCoefficient:   config.TimerProcessorUpdateAckIntervalJitterCoefficient,
			MaxRedispatchQueueSize:               config.TimerProcessorMaxRedispatchQueueSize,
			SplitQueueInterval:                   config.TimerProcessorSplitQueueInterval,
			SplitQueueIntervalJitterCoefficient:  config.TimerProcessorSplitQueueIntervalJitterCoefficient,
			PollBackoffInterval:                  config.QueueProcessorPollBackoffInterval,
			PollBackoffIntervalJitterCoefficient: config.QueueProcessorPollBackoffIntervalJitterCoefficient,
			EnableGracefulSyncShutdown:           config.QueueProcessorEnableGracefulSyncShutdown,
			EnableSplit:                          config.QueueProcessorEnableSplit,
			SplitMaxLevel:                        config.QueueProcessorSplitMaxLevel,
			EnableRandomSplitByDomainID:          config.QueueProcessorEnableRandomSplitByDomainID,
			RandomSplitProbability:               config.QueueProcessorRandomSplitProbability,
			EnablePendingTaskSplitByDomainID:     config.QueueProcessorEnablePendingTaskSplitByDomainID,
			PendingTaskSplitThreshold:            config.QueueProcessorPendingTaskSplitThreshold,
			EnableStuckTaskSplitByDomainID:       config.QueueProcessorEnableStuckTaskSplitByDomainID,
			StuckTaskSplitThreshold:              config.QueueProcessorStuckTaskSplitThreshold,
			SplitLookAheadDurationByDomainID:     config.QueueProcessorSplitLookAheadDurationByDomainID,
			EnablePersistQueueStates:             config.QueueProcessorEnablePersistQueueStates,
			EnableLoadQueueStates:                config.QueueProcessorEnableLoadQueueStates,
			MaxStartJitterInterval:               dynamicproperties.GetDurationPropertyFn(0),
			RedispatchInterval:                   config.ActiveTaskRedispatchInterval,
		},
	)
}
