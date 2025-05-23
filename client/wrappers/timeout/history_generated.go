package timeout

// Code generated by gowrap. DO NOT EDIT.
// template: ../../templates/timeout.tmpl
// gowrap: http://github.com/hexdigest/gowrap

import (
	"context"
	"time"

	"go.uber.org/yarpc"

	"github.com/uber/cadence/client/history"
	"github.com/uber/cadence/common/types"
)

var _ history.Client = (*historyClient)(nil)

// historyClient implements the history.Client interface instrumented with timeouts
type historyClient struct {
	client  history.Client
	timeout time.Duration
}

// NewHistoryClient creates a new historyClient instance
func NewHistoryClient(
	client history.Client,
	timeout time.Duration,
) history.Client {
	return &historyClient{
		client:  client,
		timeout: timeout,
	}
}

func (c *historyClient) CloseShard(ctx context.Context, cp1 *types.CloseShardRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.CloseShard(ctx, cp1, p1...)
}

func (c *historyClient) CountDLQMessages(ctx context.Context, cp1 *types.CountDLQMessagesRequest, p1 ...yarpc.CallOption) (hp1 *types.HistoryCountDLQMessagesResponse, err error) {
	return c.client.CountDLQMessages(ctx, cp1, p1...)
}

func (c *historyClient) DescribeHistoryHost(ctx context.Context, dp1 *types.DescribeHistoryHostRequest, p1 ...yarpc.CallOption) (dp2 *types.DescribeHistoryHostResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.DescribeHistoryHost(ctx, dp1, p1...)
}

func (c *historyClient) DescribeMutableState(ctx context.Context, dp1 *types.DescribeMutableStateRequest, p1 ...yarpc.CallOption) (dp2 *types.DescribeMutableStateResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.DescribeMutableState(ctx, dp1, p1...)
}

func (c *historyClient) DescribeQueue(ctx context.Context, dp1 *types.DescribeQueueRequest, p1 ...yarpc.CallOption) (dp2 *types.DescribeQueueResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.DescribeQueue(ctx, dp1, p1...)
}

func (c *historyClient) DescribeWorkflowExecution(ctx context.Context, hp1 *types.HistoryDescribeWorkflowExecutionRequest, p1 ...yarpc.CallOption) (dp1 *types.DescribeWorkflowExecutionResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.DescribeWorkflowExecution(ctx, hp1, p1...)
}

func (c *historyClient) GetCrossClusterTasks(ctx context.Context, gp1 *types.GetCrossClusterTasksRequest, p1 ...yarpc.CallOption) (gp2 *types.GetCrossClusterTasksResponse, err error) {
	return c.client.GetCrossClusterTasks(ctx, gp1, p1...)
}

func (c *historyClient) GetDLQReplicationMessages(ctx context.Context, gp1 *types.GetDLQReplicationMessagesRequest, p1 ...yarpc.CallOption) (gp2 *types.GetDLQReplicationMessagesResponse, err error) {
	return c.client.GetDLQReplicationMessages(ctx, gp1, p1...)
}

func (c *historyClient) GetFailoverInfo(ctx context.Context, gp1 *types.GetFailoverInfoRequest, p1 ...yarpc.CallOption) (gp2 *types.GetFailoverInfoResponse, err error) {
	return c.client.GetFailoverInfo(ctx, gp1, p1...)
}

func (c *historyClient) GetMutableState(ctx context.Context, gp1 *types.GetMutableStateRequest, p1 ...yarpc.CallOption) (gp2 *types.GetMutableStateResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.GetMutableState(ctx, gp1, p1...)
}

func (c *historyClient) GetReplicationMessages(ctx context.Context, gp1 *types.GetReplicationMessagesRequest, p1 ...yarpc.CallOption) (gp2 *types.GetReplicationMessagesResponse, err error) {
	return c.client.GetReplicationMessages(ctx, gp1, p1...)
}

func (c *historyClient) MergeDLQMessages(ctx context.Context, mp1 *types.MergeDLQMessagesRequest, p1 ...yarpc.CallOption) (mp2 *types.MergeDLQMessagesResponse, err error) {
	return c.client.MergeDLQMessages(ctx, mp1, p1...)
}

func (c *historyClient) NotifyFailoverMarkers(ctx context.Context, np1 *types.NotifyFailoverMarkersRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.NotifyFailoverMarkers(ctx, np1, p1...)
}

func (c *historyClient) PollMutableState(ctx context.Context, pp1 *types.PollMutableStateRequest, p1 ...yarpc.CallOption) (pp2 *types.PollMutableStateResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.PollMutableState(ctx, pp1, p1...)
}

func (c *historyClient) PurgeDLQMessages(ctx context.Context, pp1 *types.PurgeDLQMessagesRequest, p1 ...yarpc.CallOption) (err error) {
	return c.client.PurgeDLQMessages(ctx, pp1, p1...)
}

func (c *historyClient) QueryWorkflow(ctx context.Context, hp1 *types.HistoryQueryWorkflowRequest, p1 ...yarpc.CallOption) (hp2 *types.HistoryQueryWorkflowResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.QueryWorkflow(ctx, hp1, p1...)
}

func (c *historyClient) RatelimitUpdate(ctx context.Context, request *types.RatelimitUpdateRequest, opts ...yarpc.CallOption) (rp1 *types.RatelimitUpdateResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RatelimitUpdate(ctx, request, opts...)
}

func (c *historyClient) ReadDLQMessages(ctx context.Context, rp1 *types.ReadDLQMessagesRequest, p1 ...yarpc.CallOption) (rp2 *types.ReadDLQMessagesResponse, err error) {
	return c.client.ReadDLQMessages(ctx, rp1, p1...)
}

func (c *historyClient) ReapplyEvents(ctx context.Context, hp1 *types.HistoryReapplyEventsRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.ReapplyEvents(ctx, hp1, p1...)
}

func (c *historyClient) RecordActivityTaskHeartbeat(ctx context.Context, hp1 *types.HistoryRecordActivityTaskHeartbeatRequest, p1 ...yarpc.CallOption) (rp1 *types.RecordActivityTaskHeartbeatResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RecordActivityTaskHeartbeat(ctx, hp1, p1...)
}

func (c *historyClient) RecordActivityTaskStarted(ctx context.Context, rp1 *types.RecordActivityTaskStartedRequest, p1 ...yarpc.CallOption) (rp2 *types.RecordActivityTaskStartedResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RecordActivityTaskStarted(ctx, rp1, p1...)
}

func (c *historyClient) RecordChildExecutionCompleted(ctx context.Context, rp1 *types.RecordChildExecutionCompletedRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RecordChildExecutionCompleted(ctx, rp1, p1...)
}

func (c *historyClient) RecordDecisionTaskStarted(ctx context.Context, rp1 *types.RecordDecisionTaskStartedRequest, p1 ...yarpc.CallOption) (rp2 *types.RecordDecisionTaskStartedResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RecordDecisionTaskStarted(ctx, rp1, p1...)
}

func (c *historyClient) RefreshWorkflowTasks(ctx context.Context, hp1 *types.HistoryRefreshWorkflowTasksRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RefreshWorkflowTasks(ctx, hp1, p1...)
}

func (c *historyClient) RemoveSignalMutableState(ctx context.Context, rp1 *types.RemoveSignalMutableStateRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RemoveSignalMutableState(ctx, rp1, p1...)
}

func (c *historyClient) RemoveTask(ctx context.Context, rp1 *types.RemoveTaskRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RemoveTask(ctx, rp1, p1...)
}

func (c *historyClient) ReplicateEventsV2(ctx context.Context, rp1 *types.ReplicateEventsV2Request, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.ReplicateEventsV2(ctx, rp1, p1...)
}

func (c *historyClient) RequestCancelWorkflowExecution(ctx context.Context, hp1 *types.HistoryRequestCancelWorkflowExecutionRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RequestCancelWorkflowExecution(ctx, hp1, p1...)
}

func (c *historyClient) ResetQueue(ctx context.Context, rp1 *types.ResetQueueRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.ResetQueue(ctx, rp1, p1...)
}

func (c *historyClient) ResetStickyTaskList(ctx context.Context, hp1 *types.HistoryResetStickyTaskListRequest, p1 ...yarpc.CallOption) (hp2 *types.HistoryResetStickyTaskListResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.ResetStickyTaskList(ctx, hp1, p1...)
}

func (c *historyClient) ResetWorkflowExecution(ctx context.Context, hp1 *types.HistoryResetWorkflowExecutionRequest, p1 ...yarpc.CallOption) (rp1 *types.ResetWorkflowExecutionResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.ResetWorkflowExecution(ctx, hp1, p1...)
}

func (c *historyClient) RespondActivityTaskCanceled(ctx context.Context, hp1 *types.HistoryRespondActivityTaskCanceledRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RespondActivityTaskCanceled(ctx, hp1, p1...)
}

func (c *historyClient) RespondActivityTaskCompleted(ctx context.Context, hp1 *types.HistoryRespondActivityTaskCompletedRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RespondActivityTaskCompleted(ctx, hp1, p1...)
}

func (c *historyClient) RespondActivityTaskFailed(ctx context.Context, hp1 *types.HistoryRespondActivityTaskFailedRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RespondActivityTaskFailed(ctx, hp1, p1...)
}

func (c *historyClient) RespondCrossClusterTasksCompleted(ctx context.Context, rp1 *types.RespondCrossClusterTasksCompletedRequest, p1 ...yarpc.CallOption) (rp2 *types.RespondCrossClusterTasksCompletedResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RespondCrossClusterTasksCompleted(ctx, rp1, p1...)
}

func (c *historyClient) RespondDecisionTaskCompleted(ctx context.Context, hp1 *types.HistoryRespondDecisionTaskCompletedRequest, p1 ...yarpc.CallOption) (hp2 *types.HistoryRespondDecisionTaskCompletedResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RespondDecisionTaskCompleted(ctx, hp1, p1...)
}

func (c *historyClient) RespondDecisionTaskFailed(ctx context.Context, hp1 *types.HistoryRespondDecisionTaskFailedRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.RespondDecisionTaskFailed(ctx, hp1, p1...)
}

func (c *historyClient) ScheduleDecisionTask(ctx context.Context, sp1 *types.ScheduleDecisionTaskRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.ScheduleDecisionTask(ctx, sp1, p1...)
}

func (c *historyClient) SignalWithStartWorkflowExecution(ctx context.Context, hp1 *types.HistorySignalWithStartWorkflowExecutionRequest, p1 ...yarpc.CallOption) (sp1 *types.StartWorkflowExecutionResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.SignalWithStartWorkflowExecution(ctx, hp1, p1...)
}

func (c *historyClient) SignalWorkflowExecution(ctx context.Context, hp1 *types.HistorySignalWorkflowExecutionRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.SignalWorkflowExecution(ctx, hp1, p1...)
}

func (c *historyClient) StartWorkflowExecution(ctx context.Context, hp1 *types.HistoryStartWorkflowExecutionRequest, p1 ...yarpc.CallOption) (sp1 *types.StartWorkflowExecutionResponse, err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.StartWorkflowExecution(ctx, hp1, p1...)
}

func (c *historyClient) SyncActivity(ctx context.Context, sp1 *types.SyncActivityRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.SyncActivity(ctx, sp1, p1...)
}

func (c *historyClient) SyncShardStatus(ctx context.Context, sp1 *types.SyncShardStatusRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.SyncShardStatus(ctx, sp1, p1...)
}

func (c *historyClient) TerminateWorkflowExecution(ctx context.Context, hp1 *types.HistoryTerminateWorkflowExecutionRequest, p1 ...yarpc.CallOption) (err error) {
	ctx, cancel := createContext(ctx, c.timeout)
	defer cancel()
	return c.client.TerminateWorkflowExecution(ctx, hp1, p1...)
}
