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

package matching

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/dynamicconfig"
	"github.com/uber/cadence/common/log/testlogger"
	"github.com/uber/cadence/common/types"
)

func TestPollerWeight(t *testing.T) {
	n := 4
	pw := newWeightSelector(n)
	// uninitialized weights should return -1
	assert.Equal(t, -1, pw.pick())
	// all 0 weights should return -1
	for i := 0; i < n; i++ {
		pw.update(n, i, 0)
		assert.Equal(t, -1, pw.pick())
	}
	// if only one item has non-zero weight, always pick that item
	pw.update(n, 3, 4)
	for i := 0; i < 100; i++ {
		assert.Equal(t, 3, pw.pick())
	}
	pw.update(n, 2, 3)
	pw.update(n, 1, 2)
	pw.update(n, 0, 1)
	// test pick probabilities
	testPickProbHelper(t, pw, time.Now().UnixNano())

	// shrink size and test pick probabilities
	pw.update(n-1, 2, 2)
	testPickProbHelper(t, pw, time.Now().UnixNano())

	// expand size and test pick probabilities
	pw.update(n, 3, 3)
	pw.update(n+1, 4, 4)
	testPickProbHelper(t, pw, time.Now().UnixNano())
}

func testPickProbHelper(t *testing.T, pw *weightSelector, seed int64) {
	t.Helper()
	rand.Seed(seed)
	// Collect pick results
	results := make(map[int]int)
	numPicks := 1000000
	for i := 0; i < numPicks; i++ {
		index := pw.pick()
		results[index]++
	}
	// Calculate expected probabilities
	totalWeight := int64(0)
	for _, w := range pw.weights {
		totalWeight += w
	}
	expectedProbs := make([]float64, len(pw.weights))
	for i, w := range pw.weights {
		expectedProbs[i] = float64(w) / float64(totalWeight)
	}
	// Check that pick results are approximately proportional to weights
	for i := 0; i < len(pw.weights); i++ {
		expectedCount := expectedProbs[i] * float64(numPicks)
		actualCount := float64(results[i])
		delta := expectedCount * 0.02 // Allow 2% error margin
		if actualCount < expectedCount-delta || actualCount > expectedCount+delta {
			t.Errorf("Index %d: expected count approximately %.0f, got %d", i, expectedCount, results[i])
		}
	}
}

func TestNewWeightedLoadBalancer(t *testing.T) {
	ctrl := gomock.NewController(t)
	roundRobinMock := NewMockLoadBalancer(ctrl)
	domainIDToName := func(domainID string) (string, error) {
		return "testDomainName", nil
	}
	dc := dynamicconfig.NewCollection(dynamicconfig.NewNopClient(), testlogger.New(t))
	logger := testlogger.New(t)
	lb := NewWeightedLoadBalancer(roundRobinMock, domainIDToName, dc, logger)
	assert.NotNil(t, lb)
	weightedLB, ok := lb.(*weightedLoadBalancer)
	assert.NotNil(t, weightedLB)
	assert.True(t, ok)
	assert.NotNil(t, weightedLB.fallbackLoadBalancer)
	assert.NotNil(t, weightedLB.domainIDToName)
	assert.NotNil(t, weightedLB.nReadPartitions)
	assert.NotNil(t, weightedLB.weightCache)
	assert.NotNil(t, weightedLB.logger)
}

func TestWeightedLoadBalancer_PickWritePartition(t *testing.T) {
	testCases := []struct {
		name           string
		domainID       string
		taskList       types.TaskList
		taskListType   int
		forwardedFrom  string
		expectedResult string
		setupMock      func(m *MockLoadBalancer)
	}{
		{
			name:     "Basic case",
			domainID: "domainA",
			taskList: types.TaskList{Name: "taskListA"},
			setupMock: func(m *MockLoadBalancer) {
				m.EXPECT().
					PickWritePartition("domainA", types.TaskList{Name: "taskListA"}, 0, "").
					Return("partitionA")
			},
			expectedResult: "partitionA",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockFallbackLB := NewMockLoadBalancer(ctrl)
			if tc.setupMock != nil {
				tc.setupMock(mockFallbackLB)
			}

			lb := &weightedLoadBalancer{
				fallbackLoadBalancer: mockFallbackLB,
			}

			result := lb.PickWritePartition(tc.domainID, tc.taskList, tc.taskListType, tc.forwardedFrom)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestWeightedLoadBalancer_PickReadPartition(t *testing.T) {
	testCases := []struct {
		name               string
		domainID           string
		taskList           types.TaskList
		taskListType       int
		forwardedFrom      string
		weightCacheReturn  interface{}
		fallbackReturn     string
		expectedResult     string
		expectFallbackCall bool
	}{
		{
			name:               "WeightCache returns nil",
			domainID:           "domainA",
			taskList:           types.TaskList{Name: "taskListA"},
			weightCacheReturn:  nil,
			fallbackReturn:     "fallbackPartition",
			expectedResult:     "fallbackPartition",
			expectFallbackCall: true,
		},
		{
			name:               "WeightCache returns invalid type",
			domainID:           "domainB",
			taskList:           types.TaskList{Name: "taskListB"},
			weightCacheReturn:  "invalidType",
			fallbackReturn:     "fallbackPartition",
			expectedResult:     "fallbackPartition",
			expectFallbackCall: true,
		},
		{
			name:               "WeightSelector pick returns negative",
			domainID:           "domainC",
			taskList:           types.TaskList{Name: "taskListC"},
			weightCacheReturn:  newWeightSelector(2),
			fallbackReturn:     "fallbackPartition",
			expectedResult:     "fallbackPartition",
			expectFallbackCall: true,
		},
		{
			name:     "WeightSelector pick returns non-negative",
			domainID: "domainD",
			taskList: types.TaskList{Name: "taskListD"},
			weightCacheReturn: func() *weightSelector {
				pw := newWeightSelector(2)
				pw.update(2, 0, 0)
				pw.update(2, 1, 1)
				return pw
			}(),
			expectedResult:     getPartitionTaskListName("taskListD", 1),
			expectFallbackCall: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			// Create mocks.
			mockWeightCache := cache.NewMockCache(ctrl)
			mockFallbackLoadBalancer := NewMockLoadBalancer(ctrl)

			// Set up the mocks.
			taskListKey := key{
				domainID:     tc.domainID,
				taskListName: tc.taskList.GetName(),
				taskListType: tc.taskListType,
			}
			mockWeightCache.EXPECT().
				Get(taskListKey).
				Return(tc.weightCacheReturn)

			if tc.expectFallbackCall {
				mockFallbackLoadBalancer.EXPECT().
					PickReadPartition(tc.domainID, tc.taskList, tc.taskListType, tc.forwardedFrom).
					Return(tc.fallbackReturn)
			}

			logger := testlogger.New(t)

			// Create the weightedLoadBalancer instance.
			lb := &weightedLoadBalancer{
				weightCache:          mockWeightCache,
				fallbackLoadBalancer: mockFallbackLoadBalancer,
				logger:               logger,
			}

			// Call the method under test.
			result := lb.PickReadPartition(tc.domainID, tc.taskList, tc.taskListType, tc.forwardedFrom)

			// Assert the result.
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestWeightedLoadBalancer_UpdateWeight(t *testing.T) {
	testCases := []struct {
		name         string
		domainID     string
		taskList     types.TaskList
		taskListType int
		forwardedFrom string
		partition    string
		weight       int64
		nPartitions  int
		setupCache   func(mockCache *cache.MockCache)
	}{
		{
			name:     "Sticky task list",
			domainID: "domainA",
			taskList: types.TaskList{Name: "a", Kind: types.TaskListKindSticky.Ptr()},
		},
		{
			name:     "forwarded request",
			domainID: "domainA",
			taskList: types.TaskList{Name: "a"},
			forwardedFrom: "tasklist",
		},
		{
			name:     "partitioned task list",
			domainID: "domainA",
			taskList: types.TaskList{Name: "/__cadence_sys/aaa/1"},
		},
		{
			name:     "domain Name lookup error",
			domainID: "invalid-domainID",
			taskList: types.TaskList{Name: "a"},
		},
		{
			name:        "1 partition",
			domainID:    "domainA",
			taskList:    types.TaskList{Name: "a"},
			partition:   "a",
			nPartitions: 1,
			setupCache: func(mockCache *cache.MockCache) {
				mockCache.EXPECT().Delete(key{
					domainID:     "domainA",
					taskListName: "a",
					taskListType: 0,
				})
			},
		},
		{
			name:        "partition 0",
			domainID:    "domainA",
			taskList:    types.TaskList{Name: "a"},
			partition:   "a",
			weight:      1,
			nPartitions: 2,
			setupCache: func(mockCache *cache.MockCache) {
				mockCache.EXPECT().Get(key{
					domainID:     "domainA",
					taskListName: "a",
					taskListType: 0,
				}).Return(nil)
				mockCache.EXPECT().PutIfNotExist(key{
					domainID:     "domainA",
					taskListName: "a",
					taskListType: 0,
				}, newWeightSelector(2)).Return(newWeightSelector(2), nil)
			},
		},
		{
			name:        "partition 1",
			domainID:    "domainA",
			taskList:    types.TaskList{Name: "a"},
			partition:   "/__cadence_sys/a/1",
			weight:      1,
			nPartitions: 2,
			setupCache: func(mockCache *cache.MockCache) {
				mockCache.EXPECT().Get(key{
					domainID:     "domainA",
					taskListName: "a",
					taskListType: 0,
				}).Return(newWeightSelector(2))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockWeightCache := cache.NewMockCache(ctrl)
			lb := &weightedLoadBalancer{
				domainIDToName: func(domainID string) (string, error) {
					if domainID == "invalid-domainID" {
						return "", errors.New("invalid domainID")
					}
					return "testDomainName", nil
				},
				nReadPartitions: func(domainName, taskListName string, taskListType int) int {
					return tc.nPartitions
				},
				weightCache: mockWeightCache,
				logger:      testlogger.New(t),
			}
			if tc.setupCache != nil {
				tc.setupCache(mockWeightCache)
			}

			lb.UpdateWeight(tc.domainID, tc.taskList, tc.taskListType, tc.forwardedFrom, tc.partition, tc.weight)
		})
	}
}
