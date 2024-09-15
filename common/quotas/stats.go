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

package quotas

import (
	"sync"
	"time"

	"go.uber.org/atomic"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/clock"
)

type (
	emaFixedWindowQPSReporter struct {
		timeSource            clock.TimeSource
		exp                   float64
		bucketInterval        time.Duration
		bucketIntervalSeconds float64
		wg                    sync.WaitGroup
		done                  chan struct{}
		status                *atomic.Int32

		firstBucket bool
		qps         *atomic.Float64
		counter     *atomic.Int64
	}
)

func NewEmaFixedWindowQPSReporter(timeSource clock.TimeSource, exp float64, bucketInterval time.Duration) StatsReporter {
	return &emaFixedWindowQPSReporter{
		timeSource:            timeSource,
		exp:                   exp,
		bucketInterval:        bucketInterval,
		bucketIntervalSeconds: float64(bucketInterval) / float64(time.Second),
		done:                  make(chan struct{}),
		status:                atomic.NewInt32(common.DaemonStatusInitialized),
		firstBucket:           true,
		counter:               atomic.NewInt64(0),
		qps:                   atomic.NewFloat64(0),
	}
}

func (r *emaFixedWindowQPSReporter) Start() {
	if !r.status.CompareAndSwap(common.DaemonStatusInitialized, common.DaemonStatusStarted) {
		return
	}
	r.wg.Add(1)
	go r.reportLoop()
}

func (r *emaFixedWindowQPSReporter) reportLoop() {
	defer r.wg.Done()
	ticker := r.timeSource.NewTicker(r.bucketInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.Chan():
			r.report()
		case <-r.done:
			return
		}
	}
}

func (r *emaFixedWindowQPSReporter) report() {
	if r.firstBucket {
		counter := r.counter.Swap(0)
		r.qps.Store(float64(counter) / r.bucketIntervalSeconds)
		r.firstBucket = false
		return
	}
	counter := r.counter.Swap(0)
	qps := r.qps.Load()
	r.qps.Store(qps*(1-r.exp) + float64(counter)*r.exp/r.bucketIntervalSeconds)
}

func (r *emaFixedWindowQPSReporter) Stop() {
	if !r.status.CompareAndSwap(common.DaemonStatusStarted, common.DaemonStatusStopped) {
		return
	}
	close(r.done)
	r.wg.Wait()
}

func (r *emaFixedWindowQPSReporter) ReportCounter(delta int64) {
	r.counter.Add(delta)
}

func (r *emaFixedWindowQPSReporter) QPS() float64 {
	return r.qps.Load()
}

type (
	rollingWindowQPSReporter struct {
		timeSource     clock.TimeSource
		bucketInterval time.Duration
		buckets        []int64
		wg             sync.WaitGroup
		done           chan struct{}
		status         *atomic.Int32

		counter      *atomic.Int64
		bucketIndex  int
		totalCounter *atomic.Int64
	}
)

func NewRollingWindowQPSReporter(timeSource clock.TimeSource, bucketInterval time.Duration, numBuckets int) StatsReporter {
	return &rollingWindowQPSReporter{
		timeSource:     timeSource,
		bucketInterval: bucketInterval,
		buckets:        make([]int64, numBuckets),
		done:           make(chan struct{}),
		status:         atomic.NewInt32(common.DaemonStatusInitialized),
		counter:        atomic.NewInt64(0),
		totalCounter:   atomic.NewInt64(0),
	}
}

func (r *rollingWindowQPSReporter) Start() {
	if !r.status.CompareAndSwap(common.DaemonStatusInitialized, common.DaemonStatusStarted) {
		return
	}
	r.bucketIndex = r.getCurrentBucketIndex()
	r.wg.Add(1)
	go r.reportLoop()
}

func (r *rollingWindowQPSReporter) Stop() {
	if !r.status.CompareAndSwap(common.DaemonStatusStarted, common.DaemonStatusStopped) {
		return
	}
	close(r.done)
	r.wg.Wait()
}

func (r *rollingWindowQPSReporter) reportLoop() {
	defer r.wg.Done()
	ticker := r.timeSource.NewTicker(r.bucketInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.Chan():
			r.report()
		case <-r.done:
			return
		}
	}
}

func (r *rollingWindowQPSReporter) report() {
	old := r.buckets[r.bucketIndex]
	new := r.counter.Swap(0)
	r.buckets[r.bucketIndex] = new
	r.totalCounter.Add(new - old)
	r.bucketIndex++
	if r.bucketIndex == len(r.buckets) {
		r.bucketIndex = 0
	}
}

func (r *rollingWindowQPSReporter) getCurrentBucketIndex() int {
	now := r.timeSource.Now()
	return int(now.UnixNano()/int64(r.bucketInterval)) % len(r.buckets)
}

func (r *rollingWindowQPSReporter) ReportCounter(delta int64) {
	r.counter.Add(delta)
}

func (r *rollingWindowQPSReporter) QPS() float64 {
	return float64(r.totalCounter.Load()) / float64(r.bucketInterval) * float64(time.Second)
}
