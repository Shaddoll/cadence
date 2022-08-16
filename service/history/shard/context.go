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

package shard

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/backoff"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/clock"
	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/engine"
	"github.com/uber/cadence/service/history/events"
	"github.com/uber/cadence/service/history/resource"
)

type (
	// Context represents a history engine shard
	Context interface {
		GetShardID() int
		GetService() resource.Resource
		GetExecutionManager() persistence.ExecutionManager
		GetHistoryManager() persistence.HistoryManager
		GetDomainCache() cache.DomainCache
		GetClusterMetadata() cluster.Metadata
		GetConfig() *config.Config
		GetEventsCache() events.Cache
		GetLogger() log.Logger
		GetThrottledLogger() log.Logger
		GetMetricsClient() metrics.Client
		GetTimeSource() clock.TimeSource
		PreviousShardOwnerWasDifferent() bool

		GetEngine() engine.Engine
		SetEngine(engine.Engine)

		GenerateTransferTaskID() (int64, error)
		GenerateTransferTaskIDs(number int) ([]int64, error)

		GetTransferMaxReadLevel() int64
		UpdateTimerMaxReadLevel(cluster string) time.Time

		SetCurrentTime(cluster string, currentTime time.Time)
		GetCurrentTime(cluster string) time.Time
		GetLastUpdatedTime() time.Time
		GetTimerMaxReadLevel(cluster string) time.Time

		GetTransferAckLevel() int64
		UpdateTransferAckLevel(ackLevel int64) error
		GetTransferClusterAckLevel(cluster string) int64
		UpdateTransferClusterAckLevel(cluster string, ackLevel int64) error
		GetTransferProcessingQueueStates(cluster string) []*types.ProcessingQueueState
		UpdateTransferProcessingQueueStates(cluster string, states []*types.ProcessingQueueState) error

		GetCrossClusterProcessingQueueStates(cluster string) []*types.ProcessingQueueState
		UpdateCrossClusterProcessingQueueStates(cluster string, states []*types.ProcessingQueueState) error

		GetClusterReplicationLevel(cluster string) int64
		UpdateClusterReplicationLevel(cluster string, lastTaskID int64) error

		GetTimerAckLevel() time.Time
		UpdateTimerAckLevel(ackLevel time.Time) error
		GetTimerClusterAckLevel(cluster string) time.Time
		UpdateTimerClusterAckLevel(cluster string, ackLevel time.Time) error
		GetTimerProcessingQueueStates(cluster string) []*types.ProcessingQueueState
		UpdateTimerProcessingQueueStates(cluster string, states []*types.ProcessingQueueState) error

		UpdateTransferFailoverLevel(failoverID string, level TransferFailoverLevel) error
		DeleteTransferFailoverLevel(failoverID string) error
		GetAllTransferFailoverLevels() map[string]TransferFailoverLevel

		UpdateTimerFailoverLevel(failoverID string, level TimerFailoverLevel) error
		DeleteTimerFailoverLevel(failoverID string) error
		GetAllTimerFailoverLevels() map[string]TimerFailoverLevel

		GetDomainNotificationVersion() int64
		UpdateDomainNotificationVersion(domainNotificationVersion int64) error

		CreateWorkflowExecution(ctx context.Context, request *persistence.CreateWorkflowExecutionRequest) (*persistence.CreateWorkflowExecutionResponse, error)
		UpdateWorkflowExecution(ctx context.Context, request *persistence.UpdateWorkflowExecutionRequest) (*persistence.UpdateWorkflowExecutionResponse, error)
		ConflictResolveWorkflowExecution(ctx context.Context, request *persistence.ConflictResolveWorkflowExecutionRequest) (*persistence.ConflictResolveWorkflowExecutionResponse, error)
		AppendHistoryV2Events(ctx context.Context, request *persistence.AppendHistoryNodesRequest, domainID string, execution types.WorkflowExecution) (int, error)

		ReplicateFailoverMarkers(ctx context.Context, markers []*persistence.FailoverMarkerTask) error
		AddingPendingFailoverMarker(*types.FailoverMarkerAttributes) error
		ValidateAndUpdateFailoverMarkers() ([]*types.FailoverMarkerAttributes, error)
	}

	// TransferFailoverLevel contains corresponding start / end level
	TransferFailoverLevel struct {
		StartTime    time.Time
		MinLevel     int64
		CurrentLevel int64
		MaxLevel     int64
		DomainIDs    map[string]struct{}
	}

	// TimerFailoverLevel contains domain IDs and corresponding start / end level
	TimerFailoverLevel struct {
		StartTime    time.Time
		MinLevel     time.Time
		CurrentLevel time.Time
		MaxLevel     time.Time
		DomainIDs    map[string]struct{}
	}

	contextImpl struct {
		resource.Resource

		shardItem        *historyShardsItem
		shardID          int
		rangeID          int64
		executionManager persistence.ExecutionManager
		eventsCache      events.Cache
		closeCallback    func(int, *historyShardsItem)
		closed           int32
		config           *config.Config
		logger           log.Logger
		throttledLogger  log.Logger
		engine           engine.Engine

		sync.RWMutex
		lastUpdated               time.Time
		shardInfo                 *persistence.ShardInfo
		transferSequenceNumber    int64
		maxTransferSequenceNumber int64
		transferMaxReadLevel      int64
		timerMaxReadLevelMap      map[string]time.Time             // cluster -> timerMaxReadLevel
		transferFailoverLevels    map[string]TransferFailoverLevel // uuid -> TransferFailoverLevel
		timerFailoverLevels       map[string]TimerFailoverLevel    // uuid -> TimerFailoverLevel

		// exist only in memory
		remoteClusterCurrentTime map[string]time.Time

		// true if previous owner was different from the acquirer's identity.
		previousShardOwnerWasDifferent bool
	}
)

var _ Context = (*contextImpl)(nil)

var (
	// ErrShardClosed is returned when shard is closed and a req cannot be processed
	ErrShardClosed = errors.New("shard closed")
)

var (
	errMaxAttemptsExceeded = errors.New("maximum attempts exceeded to update history")
)

const (
	conditionalRetryCount = 5
	// transfer/cross cluster diff/lag is in terms of taskID, which is calculated based on shard rangeID
	// on shard movement, taskID will increase by around 1 million
	logWarnTransferLevelDiff    = 3000000 // 3 million
	logWarnCrossClusterLevelLag = 3000000 // 3 million
	logWarnTimerLevelDiff       = time.Duration(30 * time.Minute)
	historySizeLogThreshold     = 10 * 1024 * 1024
	minContextTimeout           = 1 * time.Second
)

func (s *contextImpl) GetShardID() int {
	return s.shardID
}

func (s *contextImpl) GetService() resource.Resource {
	return s.Resource
}

func (s *contextImpl) GetExecutionManager() persistence.ExecutionManager {
	return s.executionManager
}

func (s *contextImpl) GetEngine() engine.Engine {
	return s.engine
}

func (s *contextImpl) SetEngine(engine engine.Engine) {
	s.engine = engine
}

func (s *contextImpl) GenerateTransferTaskID() (int64, error) {
	s.Lock()
	defer s.Unlock()

	return s.generateTransferTaskIDLocked()
}

func (s *contextImpl) GenerateTransferTaskIDs(number int) ([]int64, error) {
	s.Lock()
	defer s.Unlock()

	result := []int64{}
	for i := 0; i < number; i++ {
		id, err := s.generateTransferTaskIDLocked()
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}

func (s *contextImpl) GetTransferMaxReadLevel() int64 {
	s.RLock()
	defer s.RUnlock()
	return s.transferMaxReadLevel
}

func (s *contextImpl) GetTransferAckLevel() int64 {
	s.RLock()
	defer s.RUnlock()

	return s.shardInfo.TransferAckLevel
}

func (s *contextImpl) UpdateTransferAckLevel(ackLevel int64) error {
	s.Lock()
	defer s.Unlock()

	s.shardInfo.TransferAckLevel = ackLevel
	s.shardInfo.StolenSinceRenew = 0
	return s.updateShardInfoLocked()
}

func (s *contextImpl) GetTransferClusterAckLevel(cluster string) int64 {
	s.RLock()
	defer s.RUnlock()

	// if we can find corresponding ack level
	if ackLevel, ok := s.shardInfo.ClusterTransferAckLevel[cluster]; ok {
		return ackLevel
	}
	// otherwise, default to existing ack level, which belongs to local cluster
	// this can happen if you add more cluster
	return s.shardInfo.TransferAckLevel
}

func (s *contextImpl) UpdateTransferClusterAckLevel(cluster string, ackLevel int64) error {
	s.Lock()
	defer s.Unlock()

	s.shardInfo.ClusterTransferAckLevel[cluster] = ackLevel
	s.shardInfo.StolenSinceRenew = 0
	return s.updateShardInfoLocked()
}

func (s *contextImpl) GetTransferProcessingQueueStates(cluster string) []*types.ProcessingQueueState {
	s.RLock()
	defer s.RUnlock()

	// if we can find corresponding processing queue states
	if states, ok := s.shardInfo.TransferProcessingQueueStates.StatesByCluster[cluster]; ok {
		return states
	}

	// check if we can find corresponding ack level
	var ackLevel int64
	var ok bool
	if ackLevel, ok = s.shardInfo.ClusterTransferAckLevel[cluster]; !ok {
		// otherwise, default to existing ack level, which belongs to local cluster
		// this can happen if you add more cluster
		ackLevel = s.shardInfo.TransferAckLevel
	}

	// otherwise, create default queue state based on existing ack level,
	// which belongs to local cluster. this can happen if you add more cluster
	return []*types.ProcessingQueueState{
		{
			Level:    common.Int32Ptr(0),
			AckLevel: common.Int64Ptr(ackLevel),
			MaxLevel: common.Int64Ptr(math.MaxInt64),
			DomainFilter: &types.DomainFilter{
				ReverseMatch: true,
			},
		},
	}
}

func (s *contextImpl) UpdateTransferProcessingQueueStates(cluster string, states []*types.ProcessingQueueState) error {
	s.Lock()
	defer s.Unlock()

	if len(states) == 0 {
		return errors.New("empty transfer processing queue states")
	}

	if s.shardInfo.TransferProcessingQueueStates.StatesByCluster == nil {
		s.shardInfo.TransferProcessingQueueStates.StatesByCluster = make(map[string][]*types.ProcessingQueueState)
	}
	s.shardInfo.TransferProcessingQueueStates.StatesByCluster[cluster] = states

	// for backward compatibility
	ackLevel := states[0].GetAckLevel()
	for _, state := range states {
		ackLevel = common.MinInt64(ackLevel, state.GetAckLevel())
	}
	s.shardInfo.ClusterTransferAckLevel[cluster] = ackLevel

	s.shardInfo.StolenSinceRenew = 0
	return s.updateShardInfoLocked()
}

func (s *contextImpl) GetCrossClusterProcessingQueueStates(cluster string) []*types.ProcessingQueueState {
	s.RLock()
	defer s.RUnlock()

	// if we can find corresponding processing queue states
	if states, ok := s.shardInfo.CrossClusterProcessingQueueStates.StatesByCluster[cluster]; ok {
		return states
	}

	// otherwise, create default processing queue state
	// this can happen if you add more cluster
	return []*types.ProcessingQueueState{
		{
			Level:    common.Int32Ptr(0),
			AckLevel: common.Int64Ptr(0),
			MaxLevel: common.Int64Ptr(math.MaxInt64),
			DomainFilter: &types.DomainFilter{
				ReverseMatch: true,
			},
		},
	}
}

func (s *contextImpl) UpdateCrossClusterProcessingQueueStates(cluster string, states []*types.ProcessingQueueState) error {
	s.Lock()
	defer s.Unlock()

	if len(states) == 0 {
		return errors.New("empty cross-cluster processing queue states")
	}

	if s.shardInfo.CrossClusterProcessingQueueStates.StatesByCluster == nil {
		s.shardInfo.CrossClusterProcessingQueueStates.StatesByCluster = make(map[string][]*types.ProcessingQueueState)
	}
	s.shardInfo.CrossClusterProcessingQueueStates.StatesByCluster[cluster] = states

	s.shardInfo.StolenSinceRenew = 0
	return s.updateShardInfoLocked()
}

func (s *contextImpl) GetClusterReplicationLevel(cluster string) int64 {
	s.RLock()
	defer s.RUnlock()

	// if we can find corresponding replication level
	if replicationLevel, ok := s.shardInfo.ClusterReplicationLevel[cluster]; ok {
		return replicationLevel
	}

	// New cluster always starts from -1
	return -1
}

func (s *contextImpl) UpdateClusterReplicationLevel(cluster string, lastTaskID int64) error {
	s.Lock()
	defer s.Unlock()

	s.shardInfo.ClusterReplicationLevel[cluster] = lastTaskID
	s.shardInfo.StolenSinceRenew = 0
	return s.updateShardInfoLocked()
}

func (s *contextImpl) GetTimerAckLevel() time.Time {
	s.RLock()
	defer s.RUnlock()

	return s.shardInfo.TimerAckLevel
}

func (s *contextImpl) UpdateTimerAckLevel(ackLevel time.Time) error {
	s.Lock()
	defer s.Unlock()

	s.shardInfo.TimerAckLevel = ackLevel
	s.shardInfo.StolenSinceRenew = 0
	return s.updateShardInfoLocked()
}

func (s *contextImpl) GetTimerClusterAckLevel(cluster string) time.Time {
	s.RLock()
	defer s.RUnlock()

	// if we can find corresponding ack level
	if ackLevel, ok := s.shardInfo.ClusterTimerAckLevel[cluster]; ok {
		return ackLevel
	}
	// otherwise, default to existing ack level, which belongs to local cluster
	// this can happen if you add more cluster
	return s.shardInfo.TimerAckLevel
}

func (s *contextImpl) UpdateTimerClusterAckLevel(cluster string, ackLevel time.Time) error {
	s.Lock()
	defer s.Unlock()

	s.shardInfo.ClusterTimerAckLevel[cluster] = ackLevel
	s.shardInfo.StolenSinceRenew = 0
	return s.updateShardInfoLocked()
}

func (s *contextImpl) GetTimerProcessingQueueStates(cluster string) []*types.ProcessingQueueState {
	s.RLock()
	defer s.RUnlock()

	// if we can find corresponding processing queue states
	if states, ok := s.shardInfo.TimerProcessingQueueStates.StatesByCluster[cluster]; ok {
		return states
	}

	// check if we can find corresponding ack level
	var ackLevel time.Time
	var ok bool
	if ackLevel, ok = s.shardInfo.ClusterTimerAckLevel[cluster]; !ok {
		// otherwise, default to existing ack level, which belongs to local cluster
		// this can happen if you add more cluster
		ackLevel = s.shardInfo.TimerAckLevel
	}

	// otherwise, create default queue state based on existing ack level,
	// which belongs to local cluster. this can happen if you add more cluster
	return []*types.ProcessingQueueState{
		{
			Level:    common.Int32Ptr(0),
			AckLevel: common.Int64Ptr(ackLevel.UnixNano()),
			MaxLevel: common.Int64Ptr(math.MaxInt64),
			DomainFilter: &types.DomainFilter{
				ReverseMatch: true,
			},
		},
	}
}

func (s *contextImpl) UpdateTimerProcessingQueueStates(cluster string, states []*types.ProcessingQueueState) error {
	s.Lock()
	defer s.Unlock()

	if len(states) == 0 {
		return errors.New("empty transfer processing queue states")
	}

	if s.shardInfo.TimerProcessingQueueStates.StatesByCluster == nil {
		s.shardInfo.TimerProcessingQueueStates.StatesByCluster = make(map[string][]*types.ProcessingQueueState)
	}
	s.shardInfo.TimerProcessingQueueStates.StatesByCluster[cluster] = states

	// for backward compatibility
	ackLevel := states[0].GetAckLevel()
	for _, state := range states {
		ackLevel = common.MinInt64(ackLevel, state.GetAckLevel())
	}
	s.shardInfo.ClusterTimerAckLevel[cluster] = time.Unix(0, ackLevel)

	s.shardInfo.StolenSinceRenew = 0
	return s.updateShardInfoLocked()
}

func (s *contextImpl) UpdateTransferFailoverLevel(failoverID string, level TransferFailoverLevel) error {
	s.Lock()
	defer s.Unlock()

	s.transferFailoverLevels[failoverID] = level
	return nil
}

func (s *contextImpl) DeleteTransferFailoverLevel(failoverID string) error {
	s.Lock()
	defer s.Unlock()

	if level, ok := s.transferFailoverLevels[failoverID]; ok {
		s.GetMetricsClient().RecordTimer(metrics.ShardInfoScope, metrics.ShardInfoTransferFailoverLatencyTimer, time.Since(level.StartTime))
		delete(s.transferFailoverLevels, failoverID)
	}
	return nil
}

func (s *contextImpl) GetAllTransferFailoverLevels() map[string]TransferFailoverLevel {
	s.RLock()
	defer s.RUnlock()

	ret := map[string]TransferFailoverLevel{}
	for k, v := range s.transferFailoverLevels {
		ret[k] = v
	}
	return ret
}

func (s *contextImpl) UpdateTimerFailoverLevel(failoverID string, level TimerFailoverLevel) error {
	s.Lock()
	defer s.Unlock()

	s.timerFailoverLevels[failoverID] = level
	return nil
}

func (s *contextImpl) DeleteTimerFailoverLevel(failoverID string) error {
	s.Lock()
	defer s.Unlock()

	if level, ok := s.timerFailoverLevels[failoverID]; ok {
		s.GetMetricsClient().RecordTimer(metrics.ShardInfoScope, metrics.ShardInfoTimerFailoverLatencyTimer, time.Since(level.StartTime))
		delete(s.timerFailoverLevels, failoverID)
	}
	return nil
}

func (s *contextImpl) GetAllTimerFailoverLevels() map[string]TimerFailoverLevel {
	s.RLock()
	defer s.RUnlock()

	ret := map[string]TimerFailoverLevel{}
	for k, v := range s.timerFailoverLevels {
		ret[k] = v
	}
	return ret
}

func (s *contextImpl) GetDomainNotificationVersion() int64 {
	s.RLock()
	defer s.RUnlock()

	return s.shardInfo.DomainNotificationVersion
}

func (s *contextImpl) UpdateDomainNotificationVersion(domainNotificationVersion int64) error {
	s.Lock()
	defer s.Unlock()

	s.shardInfo.DomainNotificationVersion = domainNotificationVersion
	return s.updateShardInfoLocked()
}

func (s *contextImpl) GetTimerMaxReadLevel(cluster string) time.Time {
	s.RLock()
	defer s.RUnlock()

	return s.timerMaxReadLevelMap[cluster]
}

func (s *contextImpl) UpdateTimerMaxReadLevel(cluster string) time.Time {
	s.Lock()
	defer s.Unlock()

	currentTime := s.GetTimeSource().Now()
	if cluster != "" && cluster != s.GetClusterMetadata().GetCurrentClusterName() {
		currentTime = s.remoteClusterCurrentTime[cluster]
	}

	s.timerMaxReadLevelMap[cluster] = currentTime.Add(s.config.TimerProcessorMaxTimeShift()).Truncate(time.Millisecond)
	return s.timerMaxReadLevelMap[cluster]
}

func (s *contextImpl) CreateWorkflowExecution(
	ctx context.Context,
	request *persistence.CreateWorkflowExecutionRequest,
) (*persistence.CreateWorkflowExecutionResponse, error) {
	if s.isClosed() {
		return nil, ErrShardClosed
	}

	ctx, cancel, err := s.ensureMinContextTimeout(ctx)
	if err != nil {
		return nil, err
	}
	if cancel != nil {
		defer cancel()
	}

	domainID := request.NewWorkflowSnapshot.ExecutionInfo.DomainID
	workflowID := request.NewWorkflowSnapshot.ExecutionInfo.WorkflowID

	// do not try to get domain cache within shard lock
	domainEntry, err := s.GetDomainCache().GetDomainByID(domainID)
	if err != nil {
		return nil, err
	}

	s.Lock()
	defer s.Unlock()

	transferMaxReadLevel := int64(0)
	if err := s.allocateTaskIDsLocked(
		domainEntry,
		workflowID,
		request.NewWorkflowSnapshot.TransferTasks,
		request.NewWorkflowSnapshot.CrossClusterTasks,
		request.NewWorkflowSnapshot.ReplicationTasks,
		request.NewWorkflowSnapshot.TimerTasks,
		&transferMaxReadLevel,
	); err != nil {
		return nil, err
	}

Create_Loop:
	for attempt := 0; attempt < conditionalRetryCount && ctx.Err() == nil; attempt++ {
		if s.isClosed() {
			return nil, ErrShardClosed
		}
		currentRangeID := s.getRangeID()
		request.RangeID = currentRangeID

		response, err := s.executionManager.CreateWorkflowExecution(ctx, request)
		switch err.(type) {
		case nil:
			// Update MaxReadLevel if write to DB succeeds
			s.updateMaxReadLevelLocked(transferMaxReadLevel)
			return response, nil
		case *types.WorkflowExecutionAlreadyStartedError,
			*persistence.WorkflowExecutionAlreadyStartedError,
			*persistence.CurrentWorkflowConditionFailedError,
			*types.ServiceBusyError:
			// No special handling required for these errors
			// We know write to DB fails if these errors are returned
			return nil, err
		case *persistence.ShardOwnershipLostError:
			{
				// RangeID might have been renewed by the same host while this update was in flight
				// Retry the operation if we still have the shard ownership
				if currentRangeID != s.getRangeID() {
					continue Create_Loop
				} else {
					// Shard is stolen, trigger shutdown of history engine
					s.logger.Warn(
						"Closing shard: CreateWorkflowExecution failed due to stolen shard.",
						tag.Error(err),
					)
					s.closeShard()
					return nil, err
				}
			}
		default:
			{
				// We have no idea if the write failed or will eventually make it to
				// persistence. Increment RangeID to guarantee that subsequent reads
				// will either see that write, or know for certain that it failed.
				// This allows the callers to reliably check the outcome by performing
				// a read.
				err1 := s.renewRangeLocked(false)
				if err1 != nil {
					// At this point we have no choice but to unload the shard, so that it
					// gets a new RangeID when it's reloaded.
					s.logger.Warn(
						"Closing shard: CreateWorkflowExecution failed due to unknown error.",
						tag.Error(err),
					)
					s.closeShard()
					break Create_Loop
				}
			}
		}
		return response, err
	}

	return nil, errMaxAttemptsExceeded
}

func (s *contextImpl) getDefaultEncoding(domainName string) common.EncodingType {
	return common.EncodingType(s.config.EventEncodingType(domainName))
}

func (s *contextImpl) UpdateWorkflowExecution(
	ctx context.Context,
	request *persistence.UpdateWorkflowExecutionRequest,
) (*persistence.UpdateWorkflowExecutionResponse, error) {
	if s.isClosed() {
		return nil, ErrShardClosed
	}

	ctx, cancel, err := s.ensureMinContextTimeout(ctx)
	if err != nil {
		return nil, err
	}
	if cancel != nil {
		defer cancel()
	}

	domainID := request.UpdateWorkflowMutation.ExecutionInfo.DomainID
	workflowID := request.UpdateWorkflowMutation.ExecutionInfo.WorkflowID

	// do not try to get domain cache within shard lock
	domainEntry, err := s.GetDomainCache().GetDomainByID(domainID)
	if err != nil {
		return nil, err
	}
	request.Encoding = s.getDefaultEncoding(domainEntry.GetInfo().Name)

	s.Lock()
	defer s.Unlock()

	transferMaxReadLevel := int64(0)
	if err := s.allocateTaskIDsLocked(
		domainEntry,
		workflowID,
		request.UpdateWorkflowMutation.TransferTasks,
		request.UpdateWorkflowMutation.CrossClusterTasks,
		request.UpdateWorkflowMutation.ReplicationTasks,
		request.UpdateWorkflowMutation.TimerTasks,
		&transferMaxReadLevel,
	); err != nil {
		return nil, err
	}
	if request.NewWorkflowSnapshot != nil {
		if err := s.allocateTaskIDsLocked(
			domainEntry,
			workflowID,
			request.NewWorkflowSnapshot.TransferTasks,
			request.NewWorkflowSnapshot.CrossClusterTasks,
			request.NewWorkflowSnapshot.ReplicationTasks,
			request.NewWorkflowSnapshot.TimerTasks,
			&transferMaxReadLevel,
		); err != nil {
			return nil, err
		}
	}

Update_Loop:
	for attempt := 0; attempt < conditionalRetryCount && ctx.Err() == nil; attempt++ {
		if s.isClosed() {
			return nil, ErrShardClosed
		}
		currentRangeID := s.getRangeID()
		request.RangeID = currentRangeID

		resp, err := s.executionManager.UpdateWorkflowExecution(ctx, request)
		switch err.(type) {
		case nil:
			// Update MaxReadLevel if write to DB succeeds
			s.updateMaxReadLevelLocked(transferMaxReadLevel)
			return resp, nil
		case *persistence.ConditionFailedError,
			*types.ServiceBusyError:
			// No special handling required for these errors
			// We know write to DB fails if these errors are returned
			return nil, err
		case *persistence.ShardOwnershipLostError:
			{
				// RangeID might have been renewed by the same host while this update was in flight
				// Retry the operation if we still have the shard ownership
				if currentRangeID != s.getRangeID() {
					continue Update_Loop
				} else {
					// Shard is stolen, trigger shutdown of history engine
					s.logger.Warn(
						"Closing shard: UpdateWorkflowExecution failed due to stolen shard.",
						tag.Error(err),
					)
					s.closeShard()
					break Update_Loop
				}
			}
		default:
			{
				// We have no idea if the write failed or will eventually make it to
				// persistence. Increment RangeID to guarantee that subsequent reads
				// will either see that write, or know for certain that it failed.
				// This allows the callers to reliably check the outcome by performing
				// a read.
				err1 := s.renewRangeLocked(false)
				if err1 != nil {
					// At this point we have no choice but to unload the shard, so that it
					// gets a new RangeID when it's reloaded.
					s.logger.Warn(
						"Closing shard: UpdateWorkflowExecution failed due to unknown error.",
						tag.Error(err),
					)
					s.closeShard()
					break Update_Loop
				}
			}
		}
		return resp, err
	}

	return nil, errMaxAttemptsExceeded
}

func (s *contextImpl) ConflictResolveWorkflowExecution(
	ctx context.Context,
	request *persistence.ConflictResolveWorkflowExecutionRequest,
) (*persistence.ConflictResolveWorkflowExecutionResponse, error) {
	if s.isClosed() {
		return nil, ErrShardClosed
	}

	ctx, cancel, err := s.ensureMinContextTimeout(ctx)
	if err != nil {
		return nil, err
	}
	if cancel != nil {
		defer cancel()
	}

	domainID := request.ResetWorkflowSnapshot.ExecutionInfo.DomainID
	workflowID := request.ResetWorkflowSnapshot.ExecutionInfo.WorkflowID

	// do not try to get domain cache within shard lock
	domainEntry, err := s.GetDomainCache().GetDomainByID(domainID)
	if err != nil {
		return nil, err
	}
	request.Encoding = s.getDefaultEncoding(domainEntry.GetInfo().Name)

	s.Lock()
	defer s.Unlock()

	transferMaxReadLevel := int64(0)
	if request.CurrentWorkflowMutation != nil {
		if err := s.allocateTaskIDsLocked(
			domainEntry,
			workflowID,
			request.CurrentWorkflowMutation.TransferTasks,
			request.CurrentWorkflowMutation.CrossClusterTasks,
			request.CurrentWorkflowMutation.ReplicationTasks,
			request.CurrentWorkflowMutation.TimerTasks,
			&transferMaxReadLevel,
		); err != nil {
			return nil, err
		}
	}
	if err := s.allocateTaskIDsLocked(
		domainEntry,
		workflowID,
		request.ResetWorkflowSnapshot.TransferTasks,
		request.ResetWorkflowSnapshot.CrossClusterTasks,
		request.ResetWorkflowSnapshot.ReplicationTasks,
		request.ResetWorkflowSnapshot.TimerTasks,
		&transferMaxReadLevel,
	); err != nil {
		return nil, err
	}
	if request.NewWorkflowSnapshot != nil {
		if err := s.allocateTaskIDsLocked(
			domainEntry,
			workflowID,
			request.NewWorkflowSnapshot.TransferTasks,
			request.NewWorkflowSnapshot.CrossClusterTasks,
			request.NewWorkflowSnapshot.ReplicationTasks,
			request.NewWorkflowSnapshot.TimerTasks,
			&transferMaxReadLevel,
		); err != nil {
			return nil, err
		}
	}

Conflict_Resolve_Loop:
	for attempt := 0; attempt < conditionalRetryCount && ctx.Err() == nil; attempt++ {
		if s.isClosed() {
			return nil, ErrShardClosed
		}
		currentRangeID := s.getRangeID()
		request.RangeID = currentRangeID
		resp, err := s.executionManager.ConflictResolveWorkflowExecution(ctx, request)
		switch err.(type) {
		case nil:
			// Update MaxReadLevel if write to DB succeeds
			s.updateMaxReadLevelLocked(transferMaxReadLevel)
			return resp, nil
		case *persistence.ConditionFailedError,
			*types.ServiceBusyError:
			// No special handling required for these errors
			// We know write to DB fails if these errors are returned
			return nil, err
		case *persistence.ShardOwnershipLostError:
			{
				// RangeID might have been renewed by the same host while this update was in flight
				// Retry the operation if we still have the shard ownership
				if currentRangeID != s.getRangeID() {
					continue Conflict_Resolve_Loop
				} else {
					// Shard is stolen, trigger shutdown of history engine
					s.logger.Warn(
						"Closing shard: ConflictResolveWorkflowExecution failed due to stolen shard.",
						tag.Error(err),
					)
					s.closeShard()
					break Conflict_Resolve_Loop
				}
			}
		default:
			{
				// We have no idea if the write failed or will eventually make it to
				// persistence. Increment RangeID to guarantee that subsequent reads
				// will either see that write, or know for certain that it failed.
				// This allows the callers to reliably check the outcome by performing
				// a read.
				err1 := s.renewRangeLocked(false)
				if err1 != nil {
					// At this point we have no choice but to unload the shard, so that it
					// gets a new RangeID when it's reloaded.
					s.logger.Warn(
						"Closing shard: ConflictResolveWorkflowExecution failed due to unknown error.",
						tag.Error(err),
					)
					s.closeShard()
					break Conflict_Resolve_Loop
				}
			}
		}

		return resp, err
	}

	return nil, errMaxAttemptsExceeded
}

func (s *contextImpl) ensureMinContextTimeout(
	parent context.Context,
) (context.Context, context.CancelFunc, error) {
	if err := parent.Err(); err != nil {
		return nil, nil, err
	}

	deadline, ok := parent.Deadline()
	if !ok || deadline.Sub(s.GetTimeSource().Now()) >= minContextTimeout {
		return parent, nil, nil
	}

	childCtx, cancel := context.WithTimeout(context.Background(), minContextTimeout)
	return childCtx, cancel, nil
}

func (s *contextImpl) AppendHistoryV2Events(
	ctx context.Context,
	request *persistence.AppendHistoryNodesRequest,
	domainID string,
	execution types.WorkflowExecution,
) (int, error) {
	if s.isClosed() {
		return 0, ErrShardClosed
	}

	domainName, err := s.GetDomainCache().GetDomainName(domainID)
	if err != nil {
		return 0, err
	}

	// NOTE: do not use generateNextTransferTaskIDLocked since
	// generateNextTransferTaskIDLocked is not guarded by lock
	transactionID, err := s.GenerateTransferTaskID()
	if err != nil {
		return 0, err
	}

	request.Encoding = s.getDefaultEncoding(domainName)
	request.ShardID = common.IntPtr(s.shardID)
	request.TransactionID = transactionID

	size := 0
	defer func() {
		s.GetMetricsClient().Scope(metrics.SessionSizeStatsScope, metrics.DomainTag(domainName)).
			RecordTimer(metrics.HistorySize, time.Duration(size))
		if size >= historySizeLogThreshold {
			s.throttledLogger.Warn("history size threshold breached",
				tag.WorkflowID(execution.GetWorkflowID()),
				tag.WorkflowRunID(execution.GetRunID()),
				tag.WorkflowDomainID(domainID),
				tag.WorkflowHistorySizeBytes(size))
		}
	}()
	resp, err0 := s.GetHistoryManager().AppendHistoryNodes(ctx, request)
	if resp != nil {
		size = resp.Size
	}
	return size, err0
}

func (s *contextImpl) GetConfig() *config.Config {
	return s.config
}

func (s *contextImpl) PreviousShardOwnerWasDifferent() bool {
	return s.previousShardOwnerWasDifferent
}

func (s *contextImpl) GetEventsCache() events.Cache {
	// the shard needs to be restarted to release the shard cache once global mode is on.
	if s.config.EventsCacheGlobalEnable() {
		return s.GetEventCache()
	}
	return s.eventsCache
}

func (s *contextImpl) GetLogger() log.Logger {
	return s.logger
}

func (s *contextImpl) GetThrottledLogger() log.Logger {
	return s.throttledLogger
}

func (s *contextImpl) getRangeID() int64 {
	return s.shardInfo.RangeID
}

func (s *contextImpl) isClosed() bool {
	return atomic.LoadInt32(&s.closed) != 0
}

func (s *contextImpl) closeShard() {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return
	}

	go func() {
		s.closeCallback(s.shardID, s.shardItem)
	}()

	// fails any writes that may start after this point.
	s.shardInfo.RangeID = -1
	atomic.StoreInt64(&s.rangeID, s.shardInfo.RangeID)
}

func (s *contextImpl) generateTransferTaskIDLocked() (int64, error) {
	if err := s.updateRangeIfNeededLocked(); err != nil {
		return -1, err
	}

	taskID := s.transferSequenceNumber
	s.transferSequenceNumber++

	return taskID, nil
}

func (s *contextImpl) updateRangeIfNeededLocked() error {
	if s.transferSequenceNumber < s.maxTransferSequenceNumber {
		return nil
	}

	return s.renewRangeLocked(false)
}

func (s *contextImpl) renewRangeLocked(isStealing bool) error {
	updatedShardInfo := s.shardInfo.Copy()
	updatedShardInfo.RangeID++
	if isStealing {
		updatedShardInfo.StolenSinceRenew++
	}

	var err error
	var attempt int32
Retry_Loop:
	for attempt = 0; attempt < conditionalRetryCount; attempt++ {
		if s.isClosed() {
			err = ErrShardClosed
			break Retry_Loop
		}
		err = s.GetShardManager().UpdateShard(context.Background(), &persistence.UpdateShardRequest{
			ShardInfo:       updatedShardInfo,
			PreviousRangeID: s.shardInfo.RangeID})
		switch err.(type) {
		case nil:
			break Retry_Loop
		case *persistence.ShardOwnershipLostError:
			// Shard is stolen, trigger history engine shutdown
			s.logger.Warn(
				"Closing shard: renewRangeLocked failed due to stolen shard.",
				tag.Error(err),
				tag.Attempt(attempt),
			)
			s.closeShard()
			break Retry_Loop
		default:
			s.logger.Warn("UpdateShard failed with an unknown error.",
				tag.Error(err),
				tag.ShardRangeID(updatedShardInfo.RangeID),
				tag.PreviousShardRangeID(s.shardInfo.RangeID),
				tag.Attempt(attempt))
		}
	}
	if err != nil {
		// Failure in updating shard to grab new RangeID
		s.logger.Error("renewRangeLocked failed.",
			tag.StoreOperationUpdateShard,
			tag.Error(err),
			tag.ShardRangeID(updatedShardInfo.RangeID),
			tag.PreviousShardRangeID(s.shardInfo.RangeID),
			tag.Attempt(attempt))
		return err
	}

	// Range is successfully updated in cassandra now update shard context to reflect new range
	s.transferSequenceNumber = updatedShardInfo.RangeID << s.config.RangeSizeBits
	s.maxTransferSequenceNumber = (updatedShardInfo.RangeID + 1) << s.config.RangeSizeBits
	s.transferMaxReadLevel = s.transferSequenceNumber - 1
	atomic.StoreInt64(&s.rangeID, updatedShardInfo.RangeID)
	s.shardInfo = updatedShardInfo

	s.logger.Info("Range updated for shardID",
		tag.ShardRangeID(s.shardInfo.RangeID),
		tag.Number(s.transferSequenceNumber),
		tag.NextNumber(s.maxTransferSequenceNumber))
	return nil
}

func (s *contextImpl) updateMaxReadLevelLocked(rl int64) {
	if rl > s.transferMaxReadLevel {
		s.logger.Debug(fmt.Sprintf("Updating MaxReadLevel: %v", rl))
		s.transferMaxReadLevel = rl
	}
}

func (s *contextImpl) updateShardInfoLocked() error {
	return s.persistShardInfoLocked(false)
}

func (s *contextImpl) forceUpdateShardInfoLocked() error {
	return s.persistShardInfoLocked(true)
}

func (s *contextImpl) persistShardInfoLocked(
	isForced bool,
) error {

	if s.isClosed() {
		return ErrShardClosed
	}

	var err error
	now := clock.NewRealTimeSource().Now()
	if !isForced && s.lastUpdated.Add(s.config.ShardUpdateMinInterval()).After(now) {
		return nil
	}
	updatedShardInfo := s.shardInfo.Copy()
	s.emitShardInfoMetricsLogsLocked()

	err = s.GetShardManager().UpdateShard(context.Background(), &persistence.UpdateShardRequest{
		ShardInfo:       updatedShardInfo,
		PreviousRangeID: s.shardInfo.RangeID,
	})

	if err != nil {
		// Shard is stolen, trigger history engine shutdown
		if _, ok := err.(*persistence.ShardOwnershipLostError); ok {
			s.logger.Warn(
				"Closing shard: updateShardInfoLocked failed due to stolen shard.",
				tag.Error(err),
			)
			s.closeShard()
		}
	} else {
		s.lastUpdated = now
	}

	return err
}

func (s *contextImpl) emitShardInfoMetricsLogsLocked() {
	currentCluster := s.GetClusterMetadata().GetCurrentClusterName()
	clusterInfo := s.GetClusterMetadata().GetAllClusterInfo()

	minTransferLevel := s.shardInfo.ClusterTransferAckLevel[currentCluster]
	maxTransferLevel := s.shardInfo.ClusterTransferAckLevel[currentCluster]
	for clusterName, v := range s.shardInfo.ClusterTransferAckLevel {
		if !clusterInfo[clusterName].Enabled {
			continue
		}

		if v < minTransferLevel {
			minTransferLevel = v
		}
		if v > maxTransferLevel {
			maxTransferLevel = v
		}
	}
	diffTransferLevel := maxTransferLevel - minTransferLevel

	minTimerLevel := s.shardInfo.ClusterTimerAckLevel[currentCluster]
	maxTimerLevel := s.shardInfo.ClusterTimerAckLevel[currentCluster]
	for clusterName, v := range s.shardInfo.ClusterTimerAckLevel {
		if !clusterInfo[clusterName].Enabled {
			continue
		}

		if v.Before(minTimerLevel) {
			minTimerLevel = v
		}
		if v.After(maxTimerLevel) {
			maxTimerLevel = v
		}
	}
	diffTimerLevel := maxTimerLevel.Sub(minTimerLevel)

	replicationLag := s.transferMaxReadLevel - s.shardInfo.ReplicationAckLevel
	transferLag := s.transferMaxReadLevel - s.shardInfo.TransferAckLevel
	timerLag := time.Since(s.shardInfo.TimerAckLevel)

	minCrossClusterLevel := int64(math.MaxInt64)
	crossClusterLevelByCluster := make(map[string]int64)
	for cluster, pqs := range s.shardInfo.CrossClusterProcessingQueueStates.StatesByCluster {
		crossClusterLevel := int64(math.MaxInt64)
		for _, queueState := range pqs {
			crossClusterLevel = common.MinInt64(crossClusterLevel, queueState.GetAckLevel())
		}
		crossClusterLevelByCluster[cluster] = crossClusterLevel
		minCrossClusterLevel = common.MinInt64(minCrossClusterLevel, crossClusterLevel)
	}
	crossClusterLag := s.transferMaxReadLevel - minCrossClusterLevel
	if minCrossClusterLevel == 0 {
		crossClusterLag = 0
	}

	transferFailoverInProgress := len(s.transferFailoverLevels)
	timerFailoverInProgress := len(s.timerFailoverLevels)

	if s.config.EmitShardDiffLog() &&
		(logWarnTransferLevelDiff < diffTransferLevel ||
			logWarnTimerLevelDiff < diffTimerLevel ||
			logWarnTransferLevelDiff < transferLag ||
			logWarnTimerLevelDiff < timerLag ||
			logWarnCrossClusterLevelLag < crossClusterLag) {

		logger := s.logger.WithTags(
			tag.ShardTime(s.remoteClusterCurrentTime),
			tag.ShardReplicationAck(s.shardInfo.ReplicationAckLevel),
			tag.ShardTimerAcks(s.shardInfo.ClusterTimerAckLevel),
			tag.ShardTransferAcks(s.shardInfo.ClusterTransferAckLevel),
			tag.ShardCrossClusterAcks(crossClusterLevelByCluster),
		)

		logger.Warn("Shard ack levels diff exceeds warn threshold.")
	}

	metricsScope := s.GetMetricsClient().Scope(metrics.ShardInfoScope)
	metricsScope.RecordTimer(metrics.ShardInfoTransferDiffTimer, time.Duration(diffTransferLevel))
	metricsScope.RecordTimer(metrics.ShardInfoTimerDiffTimer, diffTimerLevel)

	metricsScope.RecordTimer(metrics.ShardInfoReplicationLagTimer, time.Duration(replicationLag))
	metricsScope.RecordTimer(metrics.ShardInfoTransferLagTimer, time.Duration(transferLag))
	metricsScope.RecordTimer(metrics.ShardInfoTimerLagTimer, timerLag)
	for cluster, crossClusterLag := range crossClusterLevelByCluster {
		metricsScope.Tagged(metrics.TargetClusterTag(cluster)).RecordTimer(metrics.ShardInfoCrossClusterLagTimer, time.Duration(crossClusterLag))
	}

	metricsScope.RecordTimer(metrics.ShardInfoTransferFailoverInProgressTimer, time.Duration(transferFailoverInProgress))
	metricsScope.RecordTimer(metrics.ShardInfoTimerFailoverInProgressTimer, time.Duration(timerFailoverInProgress))
}

func (s *contextImpl) allocateTaskIDsLocked(
	domainEntry *cache.DomainCacheEntry,
	workflowID string,
	transferTasks []persistence.Task,
	crossClusterTasks []persistence.Task,
	replicationTasks []persistence.Task,
	timerTasks []persistence.Task,
	transferMaxReadLevel *int64,
) error {

	if err := s.allocateTransferIDsLocked(
		transferTasks,
		transferMaxReadLevel,
	); err != nil {
		return err
	}
	if err := s.allocateTransferIDsLocked(
		crossClusterTasks,
		transferMaxReadLevel,
	); err != nil {
		return err
	}
	if err := s.allocateTransferIDsLocked(
		replicationTasks,
		transferMaxReadLevel,
	); err != nil {
		return err
	}
	return s.allocateTimerIDsLocked(
		domainEntry,
		workflowID,
		timerTasks,
	)
}

func (s *contextImpl) allocateTransferIDsLocked(
	tasks []persistence.Task,
	transferMaxReadLevel *int64,
) error {
	now := s.GetTimeSource().Now()

	for _, task := range tasks {
		id, err := s.generateTransferTaskIDLocked()
		if err != nil {
			return err
		}
		s.logger.Debug(fmt.Sprintf("Assigning task ID: %v", id))
		task.SetTaskID(id)
		// only set task visibility timestamp if it's not set
		if task.GetVisibilityTimestamp().IsZero() {
			task.SetVisibilityTimestamp(now)
		}
		*transferMaxReadLevel = id
	}
	return nil
}

// NOTE: allocateTimerIDsLocked should always been called after assigning taskID for transferTasks when assigning taskID together,
// because Cadence Indexer assume timer taskID of deleteWorkflowExecution is larger than transfer taskID of closeWorkflowExecution
// for a given workflow.
func (s *contextImpl) allocateTimerIDsLocked(
	domainEntry *cache.DomainCacheEntry,
	workflowID string,
	timerTasks []persistence.Task,
) error {

	// assign IDs for the timer tasks. They need to be assigned under shard lock.
	currentCluster := s.GetClusterMetadata().GetCurrentClusterName()
	for _, task := range timerTasks {
		ts := task.GetVisibilityTimestamp()
		if task.GetVersion() != common.EmptyVersion {
			// cannot use version to determine the corresponding cluster for timer task
			// this is because during failover, timer task should be created as active
			// or otherwise, failover + active processing logic may not pick up the task.
			currentCluster = domainEntry.GetReplicationConfig().ActiveClusterName
		}
		readCursorTS := s.timerMaxReadLevelMap[currentCluster]
		if ts.Before(readCursorTS) {
			// This can happen if shard move and new host have a time SKU, or there is db write delay.
			// We generate a new timer ID using timerMaxReadLevel.
			s.logger.Warn("New timer generated is less than read level",
				tag.WorkflowDomainID(domainEntry.GetInfo().ID),
				tag.WorkflowID(workflowID),
				tag.Timestamp(ts),
				tag.CursorTimestamp(readCursorTS),
				tag.ValueShardAllocateTimerBeforeRead)
			task.SetVisibilityTimestamp(s.timerMaxReadLevelMap[currentCluster].Add(time.Millisecond))
		}

		seqNum, err := s.generateTransferTaskIDLocked()
		if err != nil {
			return err
		}
		task.SetTaskID(seqNum)
		visibilityTs := task.GetVisibilityTimestamp()
		s.logger.Debug(fmt.Sprintf("Assigning new timer (timestamp: %v, seq: %v)) ackLeveL: %v",
			visibilityTs, task.GetTaskID(), s.shardInfo.TimerAckLevel))
	}
	return nil
}

func (s *contextImpl) SetCurrentTime(cluster string, currentTime time.Time) {
	s.Lock()
	defer s.Unlock()
	if cluster != s.GetClusterMetadata().GetCurrentClusterName() {
		prevTime := s.remoteClusterCurrentTime[cluster]
		if prevTime.Before(currentTime) {
			s.remoteClusterCurrentTime[cluster] = currentTime
		}
	} else {
		panic("Cannot set current time for current cluster")
	}
}

func (s *contextImpl) GetCurrentTime(cluster string) time.Time {
	s.RLock()
	defer s.RUnlock()
	if cluster != s.GetClusterMetadata().GetCurrentClusterName() {
		return s.remoteClusterCurrentTime[cluster]
	}
	return s.GetTimeSource().Now()
}

func (s *contextImpl) GetLastUpdatedTime() time.Time {
	s.RLock()
	defer s.RUnlock()
	return s.lastUpdated
}

func (s *contextImpl) ReplicateFailoverMarkers(
	ctx context.Context,
	markers []*persistence.FailoverMarkerTask,
) error {
	if s.isClosed() {
		return ErrShardClosed
	}

	tasks := make([]persistence.Task, 0, len(markers))
	for _, marker := range markers {
		tasks = append(tasks, marker)
	}

	s.Lock()
	defer s.Unlock()

	transferMaxReadLevel := int64(0)
	if err := s.allocateTransferIDsLocked(
		tasks,
		&transferMaxReadLevel,
	); err != nil {
		return err
	}

	var err error
Retry_Loop:
	for attempt := int32(0); attempt < conditionalRetryCount && ctx.Err() == nil; attempt++ {
		if s.isClosed() {
			return ErrShardClosed
		}
		err = s.executionManager.CreateFailoverMarkerTasks(
			ctx,
			&persistence.CreateFailoverMarkersRequest{
				RangeID: s.getRangeID(),
				Markers: markers,
			},
		)
		switch err.(type) {
		case nil:
			// Update MaxReadLevel if write to DB succeeds
			s.updateMaxReadLevelLocked(transferMaxReadLevel)
			break Retry_Loop
		case *persistence.ShardOwnershipLostError:
			// do not retry on ShardOwnershipLostError
			s.logger.Warn(
				"Closing shard: ReplicateFailoverMarkers failed due to stolen shard.",
				tag.Error(err),
			)
			s.closeShard()
			break Retry_Loop
		default:
			s.logger.Error(
				"Failed to insert the failover marker into replication queue.",
				tag.Error(err),
				tag.Attempt(attempt),
			)
		}
	}
	return err
}

func (s *contextImpl) AddingPendingFailoverMarker(
	marker *types.FailoverMarkerAttributes,
) error {

	domainEntry, err := s.GetDomainCache().GetDomainByID(marker.GetDomainID())
	if err != nil {
		return err
	}
	// domain is active, the marker is expired
	isActive, _ := domainEntry.IsActiveIn(s.GetClusterMetadata().GetCurrentClusterName())
	if isActive || domainEntry.GetFailoverVersion() > marker.GetFailoverVersion() {
		s.logger.Info("Skipped out-of-date failover marker", tag.WorkflowDomainName(domainEntry.GetInfo().Name))
		return nil
	}

	s.Lock()
	defer s.Unlock()

	s.shardInfo.PendingFailoverMarkers = append(s.shardInfo.PendingFailoverMarkers, marker)
	return s.forceUpdateShardInfoLocked()
}

func (s *contextImpl) ValidateAndUpdateFailoverMarkers() ([]*types.FailoverMarkerAttributes, error) {

	completedFailoverMarkers := make(map[*types.FailoverMarkerAttributes]struct{})
	s.RLock()
	for _, marker := range s.shardInfo.PendingFailoverMarkers {
		domainEntry, err := s.GetDomainCache().GetDomainByID(marker.GetDomainID())
		if err != nil {
			s.RUnlock()
			return nil, err
		}
		isActive, _ := domainEntry.IsActiveIn(s.GetClusterMetadata().GetCurrentClusterName())
		if isActive || domainEntry.GetFailoverVersion() > marker.GetFailoverVersion() {
			completedFailoverMarkers[marker] = struct{}{}
		}
	}

	if len(completedFailoverMarkers) == 0 {
		s.RUnlock()
		return s.shardInfo.PendingFailoverMarkers, nil
	}
	s.RUnlock()

	// clean up all pending failover tasks
	s.Lock()
	defer s.Unlock()

	for idx, marker := range s.shardInfo.PendingFailoverMarkers {
		if _, ok := completedFailoverMarkers[marker]; ok {
			s.shardInfo.PendingFailoverMarkers[idx] = s.shardInfo.PendingFailoverMarkers[len(s.shardInfo.PendingFailoverMarkers)-1]
			s.shardInfo.PendingFailoverMarkers[len(s.shardInfo.PendingFailoverMarkers)-1] = nil
			s.shardInfo.PendingFailoverMarkers = s.shardInfo.PendingFailoverMarkers[:len(s.shardInfo.PendingFailoverMarkers)-1]
		}
	}
	if err := s.updateShardInfoLocked(); err != nil {
		return nil, err
	}

	return s.shardInfo.PendingFailoverMarkers, nil
}

func acquireShard(
	shardItem *historyShardsItem,
	closeCallback func(int, *historyShardsItem),
) (Context, error) {

	var shardInfo *persistence.ShardInfo

	retryPolicy := backoff.NewExponentialRetryPolicy(50 * time.Millisecond)
	retryPolicy.SetMaximumInterval(time.Second)
	retryPolicy.SetExpirationInterval(5 * time.Second)

	retryPredicate := func(err error) bool {
		if persistence.IsTransientError(err) {
			return true
		}
		_, ok := err.(*persistence.ShardAlreadyExistError)
		return ok
	}

	getShard := func() error {
		resp, err := shardItem.GetShardManager().GetShard(context.Background(), &persistence.GetShardRequest{
			ShardID: shardItem.shardID,
		})
		if err == nil {
			shardInfo = resp.ShardInfo
			return nil
		}
		if _, ok := err.(*types.EntityNotExistsError); !ok {
			return err
		}

		// EntityNotExistsError error
		shardInfo = &persistence.ShardInfo{
			ShardID:          shardItem.shardID,
			RangeID:          0,
			TransferAckLevel: 0,
		}
		return shardItem.GetShardManager().CreateShard(context.Background(), &persistence.CreateShardRequest{ShardInfo: shardInfo})
	}

	throttleRetry := backoff.NewThrottleRetry(
		backoff.WithRetryPolicy(retryPolicy),
		backoff.WithRetryableError(retryPredicate),
	)
	err := throttleRetry.Do(context.Background(), getShard)
	if err != nil {
		shardItem.logger.Error("Fail to acquire shard.", tag.Error(err))
		return nil, err
	}

	updatedShardInfo := shardInfo.Copy()
	ownershipChanged := shardInfo.Owner != shardItem.GetHostInfo().Identity()
	updatedShardInfo.Owner = shardItem.GetHostInfo().Identity()

	// initialize the cluster current time to be the same as ack level
	remoteClusterCurrentTime := make(map[string]time.Time)
	timerMaxReadLevelMap := make(map[string]time.Time)
	for clusterName, info := range shardItem.GetClusterMetadata().GetAllClusterInfo() {
		if !info.Enabled {
			continue
		}

		if clusterName != shardItem.GetClusterMetadata().GetCurrentClusterName() {
			if currentTime, ok := shardInfo.ClusterTimerAckLevel[clusterName]; ok {
				remoteClusterCurrentTime[clusterName] = currentTime
				timerMaxReadLevelMap[clusterName] = currentTime
			} else {
				remoteClusterCurrentTime[clusterName] = shardInfo.TimerAckLevel
				timerMaxReadLevelMap[clusterName] = shardInfo.TimerAckLevel
			}
		} else { // active cluster
			timerMaxReadLevelMap[clusterName] = shardInfo.TimerAckLevel
		}

		timerMaxReadLevelMap[clusterName] = timerMaxReadLevelMap[clusterName].Truncate(time.Millisecond)
	}

	if updatedShardInfo.TransferProcessingQueueStates == nil {
		updatedShardInfo.TransferProcessingQueueStates = &types.ProcessingQueueStates{
			StatesByCluster: make(map[string][]*types.ProcessingQueueState),
		}
	}
	if updatedShardInfo.CrossClusterProcessingQueueStates == nil {
		updatedShardInfo.CrossClusterProcessingQueueStates = &types.ProcessingQueueStates{
			StatesByCluster: make(map[string][]*types.ProcessingQueueState),
		}
	}
	if updatedShardInfo.TimerProcessingQueueStates == nil {
		updatedShardInfo.TimerProcessingQueueStates = &types.ProcessingQueueStates{
			StatesByCluster: make(map[string][]*types.ProcessingQueueState),
		}
	}

	executionMgr, err := shardItem.GetExecutionManager(shardItem.shardID)
	if err != nil {
		return nil, err
	}

	context := &contextImpl{
		Resource:                       shardItem.Resource,
		shardItem:                      shardItem,
		shardID:                        shardItem.shardID,
		executionManager:               executionMgr,
		shardInfo:                      updatedShardInfo,
		closeCallback:                  closeCallback,
		config:                         shardItem.config,
		remoteClusterCurrentTime:       remoteClusterCurrentTime,
		timerMaxReadLevelMap:           timerMaxReadLevelMap, // use ack to init read level
		transferFailoverLevels:         make(map[string]TransferFailoverLevel),
		timerFailoverLevels:            make(map[string]TimerFailoverLevel),
		logger:                         shardItem.logger,
		throttledLogger:                shardItem.throttledLogger,
		previousShardOwnerWasDifferent: ownershipChanged,
	}

	// TODO remove once migrated to global event cache
	context.eventsCache = events.NewCache(
		context.shardID,
		context.Resource.GetHistoryManager(),
		context.config,
		context.logger,
		context.Resource.GetMetricsClient(),
	)

	context.logger.Debug(fmt.Sprintf("Global event cache mode: %v", context.config.EventsCacheGlobalEnable()))

	err1 := context.renewRangeLocked(true)
	if err1 != nil {
		return nil, err1
	}

	return context, nil
}
