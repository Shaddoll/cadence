// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package task

import (
	"context"
	"time"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/ndc"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/execution"
	"github.com/uber/cadence/service/history/shard"
	"github.com/uber/cadence/service/worker/archiver"
)

type (
	transferStandbyTaskExecutor struct {
		*transferTaskExecutorBase

		clusterName     string
		historyResender ndc.HistoryResender
	}
)

// NewTransferStandbyTaskExecutor creates a new task executor for standby transfer task
func NewTransferStandbyTaskExecutor(
	shard shard.Context,
	archiverClient archiver.Client,
	executionCache *execution.Cache,
	historyResender ndc.HistoryResender,
	logger log.Logger,
	clusterName string,
	config *config.Config,
) Executor {
	return &transferStandbyTaskExecutor{
		transferTaskExecutorBase: newTransferTaskExecutorBase(
			shard,
			archiverClient,
			executionCache,
			logger,
			config,
		),
		clusterName:     clusterName,
		historyResender: historyResender,
	}
}

func (t *transferStandbyTaskExecutor) Execute(
	task Task,
	shouldProcessTask bool,
) error {

	transferTask, ok := task.GetInfo().(*persistence.TransferTaskInfo)
	if !ok {
		return errUnexpectedTask
	}

	if !shouldProcessTask {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), taskDefaultTimeout)
	defer cancel()

	switch transferTask.TaskType {
	case persistence.TransferTaskTypeActivityTask:
		return t.processActivityTask(ctx, transferTask)
	case persistence.TransferTaskTypeDecisionTask:
		return t.processDecisionTask(ctx, transferTask)
	case persistence.TransferTaskTypeCloseExecution,
		persistence.TransferTaskTypeRecordWorkflowClosed:
		return t.processCloseExecution(ctx, transferTask)
	case persistence.TransferTaskTypeRecordChildExecutionCompleted,
		persistence.TransferTaskTypeApplyParentClosePolicy:
		// no action needed for standby
		// check the comment in t.processCloseExecution()
		return nil
	case persistence.TransferTaskTypeCancelExecution:
		return t.processCancelExecution(ctx, transferTask)
	case persistence.TransferTaskTypeSignalExecution:
		return t.processSignalExecution(ctx, transferTask)
	case persistence.TransferTaskTypeStartChildExecution:
		return t.processStartChildExecution(ctx, transferTask)
	case persistence.TransferTaskTypeRecordWorkflowStarted:
		return t.processRecordWorkflowStarted(ctx, transferTask)
	case persistence.TransferTaskTypeResetWorkflow:
		// no reset needed for standby
		// TODO: add error logs
		return nil
	case persistence.TransferTaskTypeUpsertWorkflowSearchAttributes:
		return t.processUpsertWorkflowSearchAttributes(ctx, transferTask)
	default:
		return errUnknownTransferTask
	}
}

func (t *transferStandbyTaskExecutor) processActivityTask(
	ctx context.Context,
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	actionFn := func(ctx context.Context, wfContext execution.Context, mutableState execution.MutableState) (interface{}, error) {

		activityInfo, ok := mutableState.GetActivityInfo(transferTask.ScheduleID)
		if !ok {
			return nil, nil
		}

		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, activityInfo.Version, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		if activityInfo.StartedID == common.EmptyEventID {
			return newPushActivityToMatchingInfo(
				activityInfo.ScheduleToStartTimeout,
			), nil
		}

		return nil, nil
	}

	return t.processTransfer(
		ctx,
		processTaskIfClosed,
		transferTask,
		actionFn,
		getStandbyPostActionFn(
			transferTask,
			t.getCurrentTime,
			t.config.StandbyTaskMissingEventsResendDelay(),
			t.config.StandbyTaskMissingEventsDiscardDelay(),
			t.pushActivity,
			t.pushActivity,
		),
	)
}

func (t *transferStandbyTaskExecutor) processDecisionTask(
	ctx context.Context,
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	actionFn := func(ctx context.Context, wfContext execution.Context, mutableState execution.MutableState) (interface{}, error) {

		decisionInfo, ok := mutableState.GetDecisionInfo(transferTask.ScheduleID)
		if !ok {
			return nil, nil
		}

		executionInfo := mutableState.GetExecutionInfo()
		workflowTimeout := executionInfo.WorkflowTimeout
		decisionTimeout := common.MinInt32(workflowTimeout, common.MaxTaskTimeout)

		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, decisionInfo.Version, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		if decisionInfo.StartedID == common.EmptyEventID {
			return newPushDecisionToMatchingInfo(
				decisionTimeout,
				types.TaskList{Name: transferTask.TaskList},
			), nil
		}

		return nil, nil
	}

	return t.processTransfer(
		ctx,
		processTaskIfClosed,
		transferTask,
		actionFn,
		getStandbyPostActionFn(
			transferTask,
			t.getCurrentTime,
			t.config.StandbyTaskMissingEventsResendDelay(),
			t.config.StandbyTaskMissingEventsDiscardDelay(),
			t.pushDecision,
			t.pushDecision,
		),
	)
}

func (t *transferStandbyTaskExecutor) processCloseExecution(
	ctx context.Context,
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := true
	actionFn := func(ctx context.Context, wfContext execution.Context, mutableState execution.MutableState) (interface{}, error) {

		if mutableState.IsWorkflowExecutionRunning() {
			// this can happen if workflow is reset.
			return nil, nil
		}

		completionEvent, err := mutableState.GetCompletionEvent(ctx)
		if err != nil {
			return nil, err
		}
		wfCloseTime := completionEvent.GetTimestamp()

		executionInfo := mutableState.GetExecutionInfo()
		workflowTypeName := executionInfo.WorkflowTypeName
		workflowCloseTimestamp := wfCloseTime
		workflowCloseStatus := persistence.ToInternalWorkflowExecutionCloseStatus(executionInfo.CloseStatus)
		workflowHistoryLength := mutableState.GetNextEventID() - 1
		startEvent, err := mutableState.GetStartEvent(ctx)
		if err != nil {
			return nil, err
		}
		workflowStartTimestamp := startEvent.GetTimestamp()
		workflowExecutionTimestamp := getWorkflowExecutionTimestamp(mutableState, startEvent)
		visibilityMemo := getWorkflowMemo(executionInfo.Memo)
		searchAttr := executionInfo.SearchAttributes
		isCron := len(executionInfo.CronSchedule) > 0

		lastWriteVersion, err := mutableState.GetLastWriteVersion()
		if err != nil {
			return nil, err
		}
		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, lastWriteVersion, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		domainEntry, err := t.shard.GetDomainCache().GetDomainByID(transferTask.DomainID)
		if err != nil {
			return nil, err
		}
		numClusters := (int16)(len(domainEntry.GetReplicationConfig().Clusters))

		// DO NOT REPLY TO PARENT
		// since event replication should be done by active cluster
		return nil, t.recordWorkflowClosed(
			ctx,
			transferTask.DomainID,
			transferTask.WorkflowID,
			transferTask.RunID,
			workflowTypeName,
			workflowStartTimestamp,
			workflowExecutionTimestamp.UnixNano(),
			workflowCloseTimestamp,
			*workflowCloseStatus,
			workflowHistoryLength,
			transferTask.GetTaskID(),
			visibilityMemo,
			executionInfo.TaskList,
			isCron,
			numClusters,
			searchAttr,
		)
	}

	return t.processTransfer(
		ctx,
		processTaskIfClosed,
		transferTask,
		actionFn,
		standbyTaskPostActionNoOp,
	) // no op post action, since the entire workflow is finished
}

func (t *transferStandbyTaskExecutor) processCancelExecution(
	ctx context.Context,
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	actionFn := func(ctx context.Context, wfContext execution.Context, mutableState execution.MutableState) (interface{}, error) {

		requestCancelInfo, ok := mutableState.GetRequestCancelInfo(transferTask.ScheduleID)
		if !ok {
			return nil, nil
		}

		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, requestCancelInfo.Version, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		return getHistoryResendInfo(mutableState)
	}

	return t.processTransfer(
		ctx,
		processTaskIfClosed,
		transferTask,
		actionFn,
		getStandbyPostActionFn(
			transferTask,
			t.getCurrentTime,
			t.config.StandbyTaskMissingEventsResendDelay(),
			t.config.StandbyTaskMissingEventsDiscardDelay(),
			t.fetchHistoryFromRemote,
			standbyTransferTaskPostActionTaskDiscarded,
		),
	)
}

func (t *transferStandbyTaskExecutor) processSignalExecution(
	ctx context.Context,
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	actionFn := func(ctx context.Context, wfContext execution.Context, mutableState execution.MutableState) (interface{}, error) {

		signalInfo, ok := mutableState.GetSignalInfo(transferTask.ScheduleID)
		if !ok {
			return nil, nil
		}

		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, signalInfo.Version, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		return getHistoryResendInfo(mutableState)
	}

	return t.processTransfer(
		ctx,
		processTaskIfClosed,
		transferTask,
		actionFn,
		getStandbyPostActionFn(
			transferTask,
			t.getCurrentTime,
			t.config.StandbyTaskMissingEventsResendDelay(),
			t.config.StandbyTaskMissingEventsDiscardDelay(),
			t.fetchHistoryFromRemote,
			standbyTransferTaskPostActionTaskDiscarded,
		),
	)
}

func (t *transferStandbyTaskExecutor) processStartChildExecution(
	ctx context.Context,
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	actionFn := func(ctx context.Context, wfContext execution.Context, mutableState execution.MutableState) (interface{}, error) {

		childWorkflowInfo, ok := mutableState.GetChildExecutionInfo(transferTask.ScheduleID)
		if !ok {
			return nil, nil
		}

		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, childWorkflowInfo.Version, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		if childWorkflowInfo.StartedID != common.EmptyEventID {
			return nil, nil
		}

		return getHistoryResendInfo(mutableState)
	}

	return t.processTransfer(
		ctx,
		processTaskIfClosed,
		transferTask,
		actionFn,
		getStandbyPostActionFn(
			transferTask,
			t.getCurrentTime,
			t.config.StandbyTaskMissingEventsResendDelay(),
			t.config.StandbyTaskMissingEventsDiscardDelay(),
			t.fetchHistoryFromRemote,
			standbyTransferTaskPostActionTaskDiscarded,
		),
	)
}

func (t *transferStandbyTaskExecutor) processRecordWorkflowStarted(
	ctx context.Context,
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	return t.processTransfer(
		ctx,
		processTaskIfClosed,
		transferTask,
		func(ctx context.Context, wfContext execution.Context, mutableState execution.MutableState) (interface{}, error) {
			return nil, t.processRecordWorkflowStartedOrUpsertHelper(ctx, transferTask, mutableState, true)
		},
		standbyTaskPostActionNoOp,
	)
}

func (t *transferStandbyTaskExecutor) processUpsertWorkflowSearchAttributes(
	ctx context.Context,
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	return t.processTransfer(
		ctx,
		processTaskIfClosed,
		transferTask,
		func(ctx context.Context, wfContext execution.Context, mutableState execution.MutableState) (interface{}, error) {
			return nil, t.processRecordWorkflowStartedOrUpsertHelper(ctx, transferTask, mutableState, false)
		},
		standbyTaskPostActionNoOp,
	)
}

func (t *transferStandbyTaskExecutor) processRecordWorkflowStartedOrUpsertHelper(
	ctx context.Context,
	transferTask *persistence.TransferTaskInfo,
	mutableState execution.MutableState,
	isRecordStart bool,
) error {

	// verify task version for RecordWorkflowStarted.
	// upsert doesn't require verifyTask, because it is just a sync of mutableState.
	if isRecordStart {
		startVersion, err := mutableState.GetStartVersion()
		if err != nil {
			return err
		}
		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, startVersion, transferTask.Version, transferTask)
		if err != nil || !ok {
			return err
		}
	}

	executionInfo := mutableState.GetExecutionInfo()
	workflowTimeout := executionInfo.WorkflowTimeout
	wfTypeName := executionInfo.WorkflowTypeName
	startEvent, err := mutableState.GetStartEvent(ctx)
	if err != nil {
		return err
	}
	startTimestamp := startEvent.GetTimestamp()
	executionTimestamp := getWorkflowExecutionTimestamp(mutableState, startEvent)
	visibilityMemo := getWorkflowMemo(executionInfo.Memo)
	searchAttr := copySearchAttributes(executionInfo.SearchAttributes)
	isCron := len(executionInfo.CronSchedule) > 0

	domainEntry, err := t.shard.GetDomainCache().GetDomainByID(transferTask.DomainID)
	if err != nil {
		return err
	}
	numClusters := (int16)(len(domainEntry.GetReplicationConfig().Clusters))

	if isRecordStart {
		return t.recordWorkflowStarted(
			ctx,
			transferTask.DomainID,
			transferTask.WorkflowID,
			transferTask.RunID,
			wfTypeName,
			startTimestamp,
			executionTimestamp.UnixNano(),
			workflowTimeout,
			transferTask.GetTaskID(),
			executionInfo.TaskList,
			isCron,
			numClusters,
			visibilityMemo,
			searchAttr,
		)
	}
	return t.upsertWorkflowExecution(
		ctx,
		transferTask.DomainID,
		transferTask.WorkflowID,
		transferTask.RunID,
		wfTypeName,
		startTimestamp,
		executionTimestamp.UnixNano(),
		workflowTimeout,
		transferTask.GetTaskID(),
		executionInfo.TaskList,
		visibilityMemo,
		isCron,
		numClusters,
		searchAttr,
	)

}

func (t *transferStandbyTaskExecutor) processTransfer(
	ctx context.Context,
	processTaskIfClosed bool,
	taskInfo Info,
	actionFn standbyActionFn,
	postActionFn standbyPostActionFn,
) (retError error) {

	transferTask := taskInfo.(*persistence.TransferTaskInfo)
	wfContext, release, err := t.executionCache.GetOrCreateWorkflowExecutionWithTimeout(
		transferTask.DomainID,
		getWorkflowExecution(transferTask),
		taskGetExecutionContextTimeout,
	)
	if err != nil {
		if err == context.DeadlineExceeded {
			return errWorkflowBusy
		}
		return err
	}
	defer func() {
		if retError == ErrTaskRedispatch {
			release(nil)
		} else {
			release(retError)
		}
	}()

	mutableState, err := loadMutableStateForTransferTask(ctx, wfContext, transferTask, t.metricsClient, t.logger)
	if err != nil || mutableState == nil {
		return err
	}

	if !mutableState.IsWorkflowExecutionRunning() && !processTaskIfClosed {
		// workflow already finished, no need to process the timer
		return nil
	}

	historyResendInfo, err := actionFn(ctx, wfContext, mutableState)
	if err != nil {
		return err
	}

	release(nil)
	return postActionFn(ctx, taskInfo, historyResendInfo, t.logger)
}

func (t *transferStandbyTaskExecutor) pushActivity(
	ctx context.Context,
	task Info,
	postActionInfo interface{},
	logger log.Logger,
) error {

	if postActionInfo == nil {
		return nil
	}

	pushActivityInfo := postActionInfo.(*pushActivityToMatchingInfo)
	timeout := common.MinInt32(pushActivityInfo.activityScheduleToStartTimeout, common.MaxTaskTimeout)
	return t.transferTaskExecutorBase.pushActivity(
		ctx,
		task.(*persistence.TransferTaskInfo),
		timeout,
	)
}

func (t *transferStandbyTaskExecutor) pushDecision(
	ctx context.Context,
	task Info,
	postActionInfo interface{},
	logger log.Logger,
) error {

	if postActionInfo == nil {
		return nil
	}

	pushDecisionInfo := postActionInfo.(*pushDecisionToMatchingInfo)
	timeout := common.MinInt32(pushDecisionInfo.decisionScheduleToStartTimeout, common.MaxTaskTimeout)
	return t.transferTaskExecutorBase.pushDecision(
		ctx,
		task.(*persistence.TransferTaskInfo),
		&pushDecisionInfo.tasklist,
		timeout,
	)
}

func (t *transferStandbyTaskExecutor) fetchHistoryFromRemote(
	_ context.Context,
	taskInfo Info,
	postActionInfo interface{},
	log log.Logger,
) error {

	if postActionInfo == nil {
		return nil
	}

	transferTask := taskInfo.(*persistence.TransferTaskInfo)
	resendInfo := postActionInfo.(*historyResendInfo)

	t.metricsClient.IncCounter(metrics.HistoryRereplicationByTransferTaskScope, metrics.CadenceClientRequests)
	stopwatch := t.metricsClient.StartTimer(metrics.HistoryRereplicationByTransferTaskScope, metrics.CadenceClientLatency)
	defer stopwatch.Stop()

	var err error
	if resendInfo.lastEventID != nil && resendInfo.lastEventVersion != nil {
		// note history resender doesn't take in a context parameter, there's a separate dynamicconfig for
		// controlling the timeout for resending history.
		err = t.historyResender.SendSingleWorkflowHistory(
			transferTask.DomainID,
			transferTask.WorkflowID,
			transferTask.RunID,
			resendInfo.lastEventID,
			resendInfo.lastEventVersion,
			nil,
			nil,
		)
	} else {
		err = &types.InternalServiceError{
			Message: "transferQueueStandbyProcessor encounter empty historyResendInfo",
		}
	}

	if err != nil {
		t.logger.Error("Error re-replicating history from remote.",
			tag.ShardID(t.shard.GetShardID()),
			tag.WorkflowDomainID(transferTask.DomainID),
			tag.WorkflowID(transferTask.WorkflowID),
			tag.WorkflowRunID(transferTask.RunID),
			tag.SourceCluster(t.clusterName))
	}

	// return error so task processing logic will retry
	return ErrTaskRedispatch
}

func (t *transferStandbyTaskExecutor) getCurrentTime() time.Time {
	return t.shard.GetCurrentTime(t.clusterName)
}
