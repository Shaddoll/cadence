// Copyright (c) 2017 Uber Technologies, Inc.
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

package dynamodb

import (
	"testing"

	"github.com/uber/cadence/common/config"
	"github.com/uber/cadence/common/persistence/nosql/nosqlplugin/dynamodb"
)

// This is to make sure adding new noop method when adding new nosql interfaces
// Remove it when any other tests are implemented.
func TestDynamoDBNoopStruct(t *testing.T) {
	_, _ = dynamodb.NewDynamoDB(config.NoSQL{}, nil)
}

func TestDynamoDBHistoryPersistence(t *testing.T) {
	// s := new(persistencetests.HistoryV2PersistenceSuite)
	// s.TestBase = public.NewTestBaseWithDynamoDB(&persistencetests.TestBaseOptions{})
	// s.TestBase.Setup()
	// suite.Run(t, s)
}

func TestDynamoDBMatchingPersistence(t *testing.T) {
	// s := new(persistencetests.MatchingPersistenceSuite)
	// s.TestBase = public.NewTestBaseWithDynamoDB(&persistencetests.TestBaseOptions{})
	// s.TestBase.Setup()
	// suite.Run(t, s)
}

func TestDynamoDBDomainPersistence(t *testing.T) {
	// s := new(persistencetests.MetadataPersistenceSuiteV2)
	// s.TestBase = public.NewTestBaseWithDynamoDB(&persistencetests.TestBaseOptions{})
	// s.TestBase.Setup()
	// suite.Run(t, s)
}

func TestDynamoDBQueuePersistence(t *testing.T) {
	// s := new(persistencetests.QueuePersistenceSuite)
	// s.TestBase = public.NewTestBaseWithDynamoDB(&persistencetests.TestBaseOptions{})
	// s.TestBase.Setup()
	// suite.Run(t, s)
}
