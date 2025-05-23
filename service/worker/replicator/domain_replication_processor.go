// The MIT License (MIT)
//
// Copyright (c) 2017-2020 Uber Technologies Inc.
//
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

package replicator

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uber/cadence/client/admin"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/backoff"
	"github.com/uber/cadence/common/clock"
	"github.com/uber/cadence/common/domain"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/membership"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/service"
	"github.com/uber/cadence/common/types"
)

const (
	fetchTaskRequestTimeout                   = 10 * time.Second
	pollTimerJitterCoefficient                = 0.2
	pollIntervalSecs                          = 1
	taskProcessorErrorRetryWait               = time.Second
	taskProcessorErrorRetryBackoffCoefficient = 1
)

type (
	domainReplicationProcessor struct {
		hostInfo               membership.HostInfo
		membershipResolver     membership.Resolver
		status                 int32
		sourceCluster          string
		currentCluster         string
		logger                 log.Logger
		remotePeer             admin.Client
		taskExecutor           domain.ReplicationTaskExecutor
		metricsClient          metrics.Client
		throttleRetry          *backoff.ThrottleRetry
		lastProcessedMessageID int64
		lastRetrievedMessageID int64
		ctx                    context.Context
		cancelFn               context.CancelFunc
		wg                     sync.WaitGroup
		timeSource             clock.TimeSource
		domainReplicationQueue domain.ReplicationQueue
	}
)

func newDomainReplicationProcessor(
	sourceCluster string,
	currentCluster string,
	logger log.Logger,
	remotePeer admin.Client,
	metricsClient metrics.Client,
	taskExecutor domain.ReplicationTaskExecutor,
	hostInfo membership.HostInfo,
	resolver membership.Resolver,
	domainReplicationQueue domain.ReplicationQueue,
	replicationMaxRetry time.Duration,
	timeSource clock.TimeSource,
) *domainReplicationProcessor {
	retryPolicy := backoff.NewExponentialRetryPolicy(taskProcessorErrorRetryWait)
	retryPolicy.SetBackoffCoefficient(taskProcessorErrorRetryBackoffCoefficient)
	retryPolicy.SetExpirationInterval(replicationMaxRetry)
	throttleRetry := backoff.NewThrottleRetry(
		backoff.WithRetryPolicy(retryPolicy),
		backoff.WithRetryableError(isTransientRetryableError),
	)

	ctx, cancel := context.WithCancel(context.Background())
	return &domainReplicationProcessor{
		hostInfo:               hostInfo,
		membershipResolver:     resolver,
		status:                 common.DaemonStatusInitialized,
		sourceCluster:          sourceCluster,
		currentCluster:         currentCluster,
		logger:                 logger,
		remotePeer:             remotePeer,
		taskExecutor:           taskExecutor,
		metricsClient:          metricsClient,
		throttleRetry:          throttleRetry,
		lastProcessedMessageID: -1,
		lastRetrievedMessageID: -1,
		ctx:                    ctx,
		cancelFn:               cancel,
		timeSource:             timeSource,
		domainReplicationQueue: domainReplicationQueue,
	}
}

func (p *domainReplicationProcessor) Start() {
	if !atomic.CompareAndSwapInt32(&p.status, common.DaemonStatusInitialized, common.DaemonStatusStarted) {
		return
	}

	p.wg.Add(1)
	go p.processorLoop()
	p.logger.Info("Domain replication processor started.")
}

func (p *domainReplicationProcessor) processorLoop() {
	defer p.wg.Done()
	dur := getWaitDuration()
	timer := p.timeSource.NewTimer(dur)
	defer timer.Stop()

	for {
		select {
		case <-timer.Chan():
			p.fetchDomainReplicationTasks()
			timer.Reset(getWaitDuration())
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *domainReplicationProcessor) fetchDomainReplicationTasks() {
	// The following is a best effort to make sure only one worker is processing tasks for a
	// particular source cluster. When the ring is under reconfiguration, it is possible that
	// for a small period of time two or more workers think they are the owner and try to execute
	// the processing logic. This will not result in correctness issue as domain replication task
	// processing will be protected by version check.
	info, err := p.membershipResolver.Lookup(service.Worker, p.sourceCluster)
	if err != nil {
		p.logger.Info("Failed to lookup host info. Skip current run.")
		return
	}

	if info.Identity() != p.hostInfo.Identity() {
		p.logger.Debug(fmt.Sprintf("Worker not responsible for source cluster %v.", p.sourceCluster))
		return
	}

	ctx, cancel := context.WithTimeout(p.ctx, fetchTaskRequestTimeout)
	request := &types.GetDomainReplicationMessagesRequest{
		LastRetrievedMessageID: common.Int64Ptr(p.lastRetrievedMessageID),
		LastProcessedMessageID: common.Int64Ptr(p.lastProcessedMessageID),
		ClusterName:            p.currentCluster,
	}
	response, err := p.remotePeer.GetDomainReplicationMessages(ctx, request)
	defer cancel()

	if err != nil {
		p.logger.Error("Failed to get replication tasks", tag.Error(err))
		return
	}

	p.logger.Debug("Successfully fetched domain replication tasks.", tag.Counter(len(response.Messages.ReplicationTasks)))

	for taskIndex := range response.Messages.ReplicationTasks {
		task := response.Messages.ReplicationTasks[taskIndex]
		err := p.throttleRetry.Do(p.ctx, func(ctx context.Context) error {
			return p.handleDomainReplicationTask(task)
		})

		if err != nil {
			p.logger.Error("Failed to apply domain replication tasks", tag.Error(err))
			dlqErr := p.throttleRetry.Do(context.Background(), func(ctx context.Context) error {
				return p.putDomainReplicationTaskToDLQ(task)
			})
			if dlqErr != nil {
				p.logger.Error("Failed to put replication tasks to DLQ", tag.Error(dlqErr))
				p.metricsClient.IncCounter(metrics.DomainReplicationTaskScope, metrics.ReplicatorDLQFailures)
				return
			}
		}
	}

	p.lastProcessedMessageID = response.Messages.GetLastRetrievedMessageID()
	p.lastRetrievedMessageID = response.Messages.GetLastRetrievedMessageID()
}

func (p *domainReplicationProcessor) putDomainReplicationTaskToDLQ(task *types.ReplicationTask) error {
	domainAttribute := task.GetDomainTaskAttributes()
	if domainAttribute == nil {
		return &types.InternalServiceError{
			Message: "Domain replication task does not set domain task attribute",
		}
	}
	p.metricsClient.Scope(
		metrics.DomainReplicationTaskScope,
		metrics.DomainTag(domainAttribute.GetInfo().GetName()),
	).IncCounter(metrics.DomainReplicationEnqueueDLQCount)
	return p.domainReplicationQueue.PublishToDLQ(context.Background(), task)
}

func (p *domainReplicationProcessor) handleDomainReplicationTask(
	task *types.ReplicationTask,
) error {
	p.metricsClient.IncCounter(metrics.DomainReplicationTaskScope, metrics.ReplicatorMessages)
	sw := p.metricsClient.StartTimer(metrics.DomainReplicationTaskScope, metrics.ReplicatorLatency)
	defer sw.Stop()

	err := p.taskExecutor.Execute(task.DomainTaskAttributes)
	if err != nil {
		p.metricsClient.IncCounter(metrics.DomainReplicationTaskScope, metrics.ReplicatorFailures)
	}
	return err
}

func (p *domainReplicationProcessor) Stop() {
	p.logger.Info("Domain replication processor stopping.")
	p.cancelFn()
	p.wg.Wait()
	p.logger.Info("Domain replication processor stopped.")
}

func getWaitDuration() time.Duration {
	return backoff.JitDuration(time.Duration(pollIntervalSecs)*time.Second, pollTimerJitterCoefficient)
}

func isTransientRetryableError(err error) bool {
	switch err.(type) {
	case *types.BadRequestError:
		return false
	default:
		return true
	}
}
