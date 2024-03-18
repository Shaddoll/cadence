// Copyright (c) 2018 Uber Technologies, Inc.
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

package sql

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/log/testlogger"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/persistence/serialization"
	"github.com/uber/cadence/common/persistence/sql/sqlplugin"
	"github.com/uber/cadence/common/types"
)

func TestDeleteCurrentWorkflowExecution(t *testing.T) {
	shardID := int64(100)
	testCases := []struct {
		name      string
		req       *persistence.DeleteCurrentWorkflowExecutionRequest
		mockSetup func(*sqlplugin.MockDB)
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.DeleteCurrentWorkflowExecutionRequest{
				DomainID:   "abdcea69-61d5-44c3-9d55-afe23505a542",
				WorkflowID: "aaaa",
				RunID:      "fd65967f-777d-45de-8dee-be49dfda6716",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().DeleteFromCurrentExecutions(gomock.Any(), &sqlplugin.CurrentExecutionsFilter{
					ShardID:    shardID,
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "aaaa",
					RunID:      serialization.MustParseUUID("fd65967f-777d-45de-8dee-be49dfda6716"),
				}).Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.DeleteCurrentWorkflowExecutionRequest{
				DomainID:   "abdcea69-61d5-44c3-9d55-afe23505a542",
				WorkflowID: "aaaa",
				RunID:      "fd65967f-777d-45de-8dee-be49dfda6716",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().DeleteFromCurrentExecutions(gomock.Any(), &sqlplugin.CurrentExecutionsFilter{
					ShardID:    shardID,
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "aaaa",
					RunID:      serialization.MustParseUUID("fd65967f-777d-45de-8dee-be49dfda6716"),
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			err = store.DeleteCurrentWorkflowExecution(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
			}
		})
	}
}

func TestGetCurrentExecution(t *testing.T) {
	shardID := int64(100)
	testCases := []struct {
		name      string
		req       *persistence.GetCurrentExecutionRequest
		mockSetup func(*sqlplugin.MockDB)
		want      *persistence.GetCurrentExecutionResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.GetCurrentExecutionRequest{
				DomainID:   "abdcea69-61d5-44c3-9d55-afe23505a542",
				WorkflowID: "aaaa",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().SelectFromCurrentExecutions(gomock.Any(), &sqlplugin.CurrentExecutionsFilter{
					ShardID:    shardID,
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "aaaa",
				}).Return(&sqlplugin.CurrentExecutionsRow{
					ShardID:          shardID,
					DomainID:         serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID:       "aaaa",
					RunID:            serialization.MustParseUUID("fd65967f-777d-45de-8dee-be49dfda6716"),
					CreateRequestID:  "create",
					State:            2,
					CloseStatus:      3,
					LastWriteVersion: 9,
				}, nil)
			},
			want: &persistence.GetCurrentExecutionResponse{
				StartRequestID:   "create",
				RunID:            "fd65967f-777d-45de-8dee-be49dfda6716",
				State:            2,
				CloseStatus:      3,
				LastWriteVersion: 9,
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.GetCurrentExecutionRequest{
				DomainID:   "abdcea69-61d5-44c3-9d55-afe23505a542",
				WorkflowID: "aaaa",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().SelectFromCurrentExecutions(gomock.Any(), &sqlplugin.CurrentExecutionsFilter{
					ShardID:    shardID,
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "aaaa",
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			got, err := store.GetCurrentExecution(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestGetTransferTasks(t *testing.T) {
	shardID := 1
	testCases := []struct {
		name      string
		req       *persistence.GetTransferTasksRequest
		mockSetup func(*sqlplugin.MockDB, *serialization.MockParser)
		want      *persistence.GetTransferTasksResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.GetTransferTasksRequest{
				ReadLevel:     1,
				MaxReadLevel:  99,
				BatchSize:     1,
				NextPageToken: serializePageToken(11),
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockDB.EXPECT().SelectFromTransferTasks(gomock.Any(), &sqlplugin.TransferTasksFilter{
					ShardID:   shardID,
					MinTaskID: 11,
					MaxTaskID: 99,
					PageSize:  1,
				}).Return([]sqlplugin.TransferTasksRow{
					{
						ShardID:      shardID,
						TaskID:       12,
						Data:         []byte(`transfer`),
						DataEncoding: "transfer",
					},
				}, nil)
				mockParser.EXPECT().TransferTaskInfoFromBlob([]byte(`transfer`), "transfer").Return(&serialization.TransferTaskInfo{
					DomainID:                serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID:              "test",
					RunID:                   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a54a"),
					VisibilityTimestamp:     time.Unix(1, 1),
					TargetDomainID:          serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a542"),
					TargetDomainIDs:         []serialization.UUID{serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a54a")},
					TargetWorkflowID:        "xyz",
					TargetRunID:             serialization.MustParseUUID("dbdcea69-61d5-44c3-9d55-afe23505a54a"),
					TargetChildWorkflowOnly: true,
					TaskList:                "tl",
					TaskType:                1,
					ScheduleID:              19,
					Version:                 202,
				}, nil)
			},
			want: &persistence.GetTransferTasksResponse{
				Tasks: []*persistence.TransferTaskInfo{
					{
						TaskID:                  12,
						DomainID:                "abdcea69-61d5-44c3-9d55-afe23505a542",
						WorkflowID:              "test",
						RunID:                   "abdcea69-61d5-44c3-9d55-afe23505a54a",
						VisibilityTimestamp:     time.Unix(1, 1),
						TargetDomainID:          "bbdcea69-61d5-44c3-9d55-afe23505a542",
						TargetWorkflowID:        "xyz",
						TargetDomainIDs:         map[string]struct{}{"bbdcea69-61d5-44c3-9d55-afe23505a54a": struct{}{}},
						TargetRunID:             "dbdcea69-61d5-44c3-9d55-afe23505a54a",
						TargetChildWorkflowOnly: true,
						TaskList:                "tl",
						TaskType:                1,
						ScheduleID:              19,
						Version:                 202,
					},
				},
				NextPageToken: serializePageToken(12),
			},
			wantErr: false,
		},
		{
			name: "Error case - failed to get from database",
			req: &persistence.GetTransferTasksRequest{
				ReadLevel:     1,
				MaxReadLevel:  99,
				BatchSize:     1,
				NextPageToken: serializePageToken(11),
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				err := errors.New("some error")
				mockDB.EXPECT().SelectFromTransferTasks(gomock.Any(), &sqlplugin.TransferTasksFilter{
					ShardID:   shardID,
					MinTaskID: 11,
					MaxTaskID: 99,
					PageSize:  1,
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
		{
			name: "Error case - failed to decode data",
			req: &persistence.GetTransferTasksRequest{
				ReadLevel:     1,
				MaxReadLevel:  99,
				BatchSize:     1,
				NextPageToken: serializePageToken(11),
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockDB.EXPECT().SelectFromTransferTasks(gomock.Any(), &sqlplugin.TransferTasksFilter{
					ShardID:   shardID,
					MinTaskID: 11,
					MaxTaskID: 99,
					PageSize:  1,
				}).Return([]sqlplugin.TransferTasksRow{
					{
						ShardID:      shardID,
						TaskID:       12,
						Data:         []byte(`transfer`),
						DataEncoding: "transfer",
					},
				}, nil)
				mockParser.EXPECT().TransferTaskInfoFromBlob([]byte(`transfer`), "transfer").Return(nil, errors.New("some error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			mockParser := serialization.NewMockParser(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), mockParser, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB, mockParser)

			got, err := store.GetTransferTasks(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestCompleteTransferTask(t *testing.T) {
	shardID := 100
	testCases := []struct {
		name      string
		req       *persistence.CompleteTransferTaskRequest
		mockSetup func(*sqlplugin.MockDB)
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.CompleteTransferTaskRequest{
				TaskID: 123,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().DeleteFromTransferTasks(gomock.Any(), &sqlplugin.TransferTasksFilter{
					ShardID: shardID,
					TaskID:  123,
				}).Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.CompleteTransferTaskRequest{
				TaskID: 123,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().DeleteFromTransferTasks(gomock.Any(), &sqlplugin.TransferTasksFilter{
					ShardID: shardID,
					TaskID:  123,
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			err = store.CompleteTransferTask(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
			}
		})
	}
}

func TestRangeCompleteTransferTask(t *testing.T) {
	shardID := 100
	testCases := []struct {
		name      string
		req       *persistence.RangeCompleteTransferTaskRequest
		mockSetup func(*sqlplugin.MockDB)
		want      *persistence.RangeCompleteTransferTaskResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.RangeCompleteTransferTaskRequest{
				ExclusiveBeginTaskID: 123,
				InclusiveEndTaskID:   345,
				PageSize:             1000,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().RangeDeleteFromTransferTasks(gomock.Any(), &sqlplugin.TransferTasksFilter{
					ShardID:   shardID,
					MinTaskID: 123,
					MaxTaskID: 345,
					PageSize:  1000,
				}).Return(&sqlResult{rowsAffected: 1000}, nil)
			},
			want: &persistence.RangeCompleteTransferTaskResponse{
				TasksCompleted: 1000,
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.RangeCompleteTransferTaskRequest{
				ExclusiveBeginTaskID: 123,
				InclusiveEndTaskID:   345,
				PageSize:             1000,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().RangeDeleteFromTransferTasks(gomock.Any(), &sqlplugin.TransferTasksFilter{
					ShardID:   shardID,
					MinTaskID: 123,
					MaxTaskID: 345,
					PageSize:  1000,
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			got, err := store.RangeCompleteTransferTask(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestGetCrossClusterTasks(t *testing.T) {
	shardID := 1
	testCases := []struct {
		name      string
		req       *persistence.GetCrossClusterTasksRequest
		mockSetup func(*sqlplugin.MockDB, *serialization.MockParser)
		want      *persistence.GetCrossClusterTasksResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.GetCrossClusterTasksRequest{
				NextPageToken: serializePageToken(100),
				TargetCluster: "target",
				MaxReadLevel:  199,
				BatchSize:     10,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockDB.EXPECT().SelectFromCrossClusterTasks(gomock.Any(), &sqlplugin.CrossClusterTasksFilter{
					TargetCluster: "target",
					ShardID:       shardID,
					MinTaskID:     100,
					MaxTaskID:     199,
					PageSize:      10,
				}).Return([]sqlplugin.CrossClusterTasksRow{
					{
						TargetCluster: "target",
						ShardID:       shardID,
						TaskID:        101,
						Data:          []byte(`cross`),
						DataEncoding:  "cross",
					},
				}, nil)
				mockParser.EXPECT().CrossClusterTaskInfoFromBlob([]byte(`cross`), "cross").Return(&serialization.CrossClusterTaskInfo{
					DomainID:                serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID:              "test",
					RunID:                   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a54a"),
					VisibilityTimestamp:     time.Unix(1, 1),
					TargetDomainID:          serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a542"),
					TargetDomainIDs:         []serialization.UUID{serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a54a")},
					TargetWorkflowID:        "xyz",
					TargetRunID:             serialization.MustParseUUID("dbdcea69-61d5-44c3-9d55-afe23505a54a"),
					TargetChildWorkflowOnly: true,
					TaskList:                "tl",
					TaskType:                1,
					ScheduleID:              19,
					Version:                 202,
				}, nil)
			},
			want: &persistence.GetCrossClusterTasksResponse{
				Tasks: []*persistence.CrossClusterTaskInfo{
					{
						TaskID:                  101,
						DomainID:                "abdcea69-61d5-44c3-9d55-afe23505a542",
						WorkflowID:              "test",
						RunID:                   "abdcea69-61d5-44c3-9d55-afe23505a54a",
						VisibilityTimestamp:     time.Unix(1, 1),
						TargetDomainID:          "bbdcea69-61d5-44c3-9d55-afe23505a542",
						TargetWorkflowID:        "xyz",
						TargetDomainIDs:         map[string]struct{}{"bbdcea69-61d5-44c3-9d55-afe23505a54a": struct{}{}},
						TargetRunID:             "dbdcea69-61d5-44c3-9d55-afe23505a54a",
						TargetChildWorkflowOnly: true,
						TaskList:                "tl",
						TaskType:                1,
						ScheduleID:              19,
						Version:                 202,
					},
				},
				NextPageToken: serializePageToken(101),
			},
		},
		{
			name: "Error case - failed to get from database",
			req: &persistence.GetCrossClusterTasksRequest{
				ReadLevel:     1,
				MaxReadLevel:  99,
				BatchSize:     1,
				NextPageToken: serializePageToken(11),
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				err := errors.New("some error")
				mockDB.EXPECT().SelectFromCrossClusterTasks(gomock.Any(), gomock.Any()).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
		{
			name: "Error case - failed to decode data",
			req: &persistence.GetCrossClusterTasksRequest{
				ReadLevel:     1,
				MaxReadLevel:  99,
				BatchSize:     1,
				NextPageToken: serializePageToken(11),
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockDB.EXPECT().SelectFromCrossClusterTasks(gomock.Any(), gomock.Any()).Return([]sqlplugin.CrossClusterTasksRow{
					{
						ShardID:      shardID,
						TaskID:       12,
						Data:         []byte(`cross`),
						DataEncoding: "cross",
					},
				}, nil)
				mockParser.EXPECT().CrossClusterTaskInfoFromBlob([]byte(`cross`), "cross").Return(nil, errors.New("some error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			mockParser := serialization.NewMockParser(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), mockParser, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB, mockParser)

			got, err := store.GetCrossClusterTasks(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestCompleteCrossClusterTask(t *testing.T) {
	shardID := 100
	testCases := []struct {
		name      string
		req       *persistence.CompleteCrossClusterTaskRequest
		mockSetup func(*sqlplugin.MockDB)
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.CompleteCrossClusterTaskRequest{
				TaskID:        123,
				TargetCluster: "test",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().DeleteFromCrossClusterTasks(gomock.Any(), &sqlplugin.CrossClusterTasksFilter{
					TargetCluster: "test",
					ShardID:       shardID,
					TaskID:        123,
				}).Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.CompleteCrossClusterTaskRequest{
				TaskID:        123,
				TargetCluster: "test",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().DeleteFromCrossClusterTasks(gomock.Any(), &sqlplugin.CrossClusterTasksFilter{
					TargetCluster: "test",
					ShardID:       shardID,
					TaskID:        123,
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			err = store.CompleteCrossClusterTask(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
			}
		})
	}
}

func TestRangeCompleteCrossClusterTask(t *testing.T) {
	shardID := 100
	testCases := []struct {
		name      string
		req       *persistence.RangeCompleteCrossClusterTaskRequest
		mockSetup func(*sqlplugin.MockDB)
		want      *persistence.RangeCompleteCrossClusterTaskResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.RangeCompleteCrossClusterTaskRequest{
				ExclusiveBeginTaskID: 123,
				InclusiveEndTaskID:   345,
				TargetCluster:        "test",
				PageSize:             10,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().RangeDeleteFromCrossClusterTasks(gomock.Any(), &sqlplugin.CrossClusterTasksFilter{
					TargetCluster: "test",
					ShardID:       shardID,
					MinTaskID:     123,
					MaxTaskID:     345,
					PageSize:      10,
				}).Return(&sqlResult{rowsAffected: 10}, nil)
			},
			want: &persistence.RangeCompleteCrossClusterTaskResponse{
				TasksCompleted: 10,
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.RangeCompleteCrossClusterTaskRequest{
				ExclusiveBeginTaskID: 123,
				InclusiveEndTaskID:   345,
				TargetCluster:        "test",
				PageSize:             10,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().RangeDeleteFromCrossClusterTasks(gomock.Any(), &sqlplugin.CrossClusterTasksFilter{
					TargetCluster: "test",
					ShardID:       shardID,
					MinTaskID:     123,
					MaxTaskID:     345,
					PageSize:      10,
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			got, err := store.RangeCompleteCrossClusterTask(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestGetReplicationTasks(t *testing.T) {
	shardID := 0
	testCases := []struct {
		name      string
		req       *persistence.GetReplicationTasksRequest
		mockSetup func(*sqlplugin.MockDB, *serialization.MockParser)
		want      *persistence.InternalGetReplicationTasksResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.GetReplicationTasksRequest{
				NextPageToken: serializePageToken(100),
				MaxReadLevel:  199,
				BatchSize:     1000,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockDB.EXPECT().SelectFromReplicationTasks(gomock.Any(), &sqlplugin.ReplicationTasksFilter{
					ShardID:   shardID,
					MinTaskID: 100,
					MaxTaskID: 1100,
					PageSize:  1000,
				}).Return([]sqlplugin.ReplicationTasksRow{
					{
						ShardID:      shardID,
						TaskID:       101,
						Data:         []byte(`replication`),
						DataEncoding: "replication",
					},
				}, nil)
				mockParser.EXPECT().ReplicationTaskInfoFromBlob([]byte(`replication`), "replication").Return(&serialization.ReplicationTaskInfo{
					DomainID:          serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID:        "test",
					RunID:             serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a54a"),
					TaskType:          1,
					FirstEventID:      10,
					NextEventID:       101,
					Version:           202,
					ScheduledID:       19,
					BranchToken:       []byte(`bt`),
					NewRunBranchToken: []byte(`nbt`),
					CreationTimestamp: time.Unix(1, 1),
				}, nil)
			},
			want: &persistence.InternalGetReplicationTasksResponse{
				Tasks: []*persistence.InternalReplicationTaskInfo{
					{
						TaskID:            101,
						DomainID:          "abdcea69-61d5-44c3-9d55-afe23505a542",
						WorkflowID:        "test",
						RunID:             "abdcea69-61d5-44c3-9d55-afe23505a54a",
						TaskType:          1,
						FirstEventID:      10,
						NextEventID:       101,
						Version:           202,
						ScheduledID:       19,
						BranchToken:       []byte(`bt`),
						NewRunBranchToken: []byte(`nbt`),
						CreationTime:      time.Unix(1, 1),
					},
				},
				NextPageToken: serializePageToken(101),
			},
			wantErr: false,
		},
		{
			name: "Error case - failed to load from database",
			req: &persistence.GetReplicationTasksRequest{
				NextPageToken: serializePageToken(100),
				MaxReadLevel:  199,
				BatchSize:     1000,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				err := errors.New("some error")
				mockDB.EXPECT().SelectFromReplicationTasks(gomock.Any(), &sqlplugin.ReplicationTasksFilter{
					ShardID:   shardID,
					MinTaskID: 100,
					MaxTaskID: 1100,
					PageSize:  1000,
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
		{
			name: "Error case - failed to decode data",
			req: &persistence.GetReplicationTasksRequest{
				NextPageToken: serializePageToken(100),
				MaxReadLevel:  199,
				BatchSize:     1000,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockDB.EXPECT().SelectFromReplicationTasks(gomock.Any(), &sqlplugin.ReplicationTasksFilter{
					ShardID:   shardID,
					MinTaskID: 100,
					MaxTaskID: 1100,
					PageSize:  1000,
				}).Return([]sqlplugin.ReplicationTasksRow{
					{
						ShardID:      shardID,
						TaskID:       101,
						Data:         []byte(`replication`),
						DataEncoding: "replication",
					},
				}, nil)
				mockParser.EXPECT().ReplicationTaskInfoFromBlob([]byte(`replication`), "replication").Return(nil, errors.New("some error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			mockParser := serialization.NewMockParser(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), mockParser, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB, mockParser)

			got, err := store.GetReplicationTasks(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestCompleteReplicationTask(t *testing.T) {
	shardID := 100
	testCases := []struct {
		name      string
		req       *persistence.CompleteReplicationTaskRequest
		mockSetup func(*sqlplugin.MockDB)
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.CompleteReplicationTaskRequest{
				TaskID: 123,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().DeleteFromReplicationTasks(gomock.Any(), &sqlplugin.ReplicationTasksFilter{
					ShardID: shardID,
					TaskID:  123,
				}).Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.CompleteReplicationTaskRequest{
				TaskID: 123,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().DeleteFromReplicationTasks(gomock.Any(), &sqlplugin.ReplicationTasksFilter{
					ShardID: shardID,
					TaskID:  123,
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			err = store.CompleteReplicationTask(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
			}
		})
	}
}

func TestRangeCompleteReplicationTask(t *testing.T) {
	shardID := 100
	testCases := []struct {
		name      string
		req       *persistence.RangeCompleteReplicationTaskRequest
		mockSetup func(*sqlplugin.MockDB)
		want      *persistence.RangeCompleteReplicationTaskResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.RangeCompleteReplicationTaskRequest{
				InclusiveEndTaskID: 345,
				PageSize:           10,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().RangeDeleteFromReplicationTasks(gomock.Any(), &sqlplugin.ReplicationTasksFilter{
					ShardID:            shardID,
					InclusiveEndTaskID: 345,
					PageSize:           10,
				}).Return(&sqlResult{rowsAffected: 10}, nil)
			},
			want: &persistence.RangeCompleteReplicationTaskResponse{
				TasksCompleted: 10,
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.RangeCompleteReplicationTaskRequest{
				InclusiveEndTaskID: 345,
				PageSize:           10,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().RangeDeleteFromReplicationTasks(gomock.Any(), &sqlplugin.ReplicationTasksFilter{
					ShardID:            shardID,
					InclusiveEndTaskID: 345,
					PageSize:           10,
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			got, err := store.RangeCompleteReplicationTask(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestGetReplicationTasksFromDLQ(t *testing.T) {
	shardID := 0
	testCases := []struct {
		name      string
		req       *persistence.GetReplicationTasksFromDLQRequest
		mockSetup func(*sqlplugin.MockDB, *serialization.MockParser)
		want      *persistence.InternalGetReplicationTasksFromDLQResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.GetReplicationTasksFromDLQRequest{
				SourceClusterName: "source",
				GetReplicationTasksRequest: persistence.GetReplicationTasksRequest{
					NextPageToken: serializePageToken(100),
					MaxReadLevel:  199,
					BatchSize:     1000,
				},
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockDB.EXPECT().SelectFromReplicationTasksDLQ(gomock.Any(), &sqlplugin.ReplicationTasksDLQFilter{
					ReplicationTasksFilter: sqlplugin.ReplicationTasksFilter{
						ShardID:   shardID,
						MinTaskID: 100,
						MaxTaskID: 1100,
						PageSize:  1000,
					},
					SourceClusterName: "source",
				}).Return([]sqlplugin.ReplicationTasksRow{
					{
						ShardID:      shardID,
						TaskID:       101,
						Data:         []byte(`replication`),
						DataEncoding: "replication",
					},
				}, nil)
				mockParser.EXPECT().ReplicationTaskInfoFromBlob([]byte(`replication`), "replication").Return(&serialization.ReplicationTaskInfo{
					DomainID:          serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID:        "test",
					RunID:             serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a54a"),
					TaskType:          1,
					FirstEventID:      10,
					NextEventID:       101,
					Version:           202,
					ScheduledID:       19,
					BranchToken:       []byte(`bt`),
					NewRunBranchToken: []byte(`nbt`),
					CreationTimestamp: time.Unix(1, 1),
				}, nil)
			},
			want: &persistence.InternalGetReplicationTasksFromDLQResponse{
				Tasks: []*persistence.InternalReplicationTaskInfo{
					{
						TaskID:            101,
						DomainID:          "abdcea69-61d5-44c3-9d55-afe23505a542",
						WorkflowID:        "test",
						RunID:             "abdcea69-61d5-44c3-9d55-afe23505a54a",
						TaskType:          1,
						FirstEventID:      10,
						NextEventID:       101,
						Version:           202,
						ScheduledID:       19,
						BranchToken:       []byte(`bt`),
						NewRunBranchToken: []byte(`nbt`),
						CreationTime:      time.Unix(1, 1),
					},
				},
				NextPageToken: serializePageToken(101),
			},
			wantErr: false,
		},
		{
			name: "Error case - failed to load from database",
			req: &persistence.GetReplicationTasksFromDLQRequest{
				SourceClusterName: "source",
				GetReplicationTasksRequest: persistence.GetReplicationTasksRequest{
					NextPageToken: serializePageToken(100),
					MaxReadLevel:  199,
					BatchSize:     1000,
				},
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				err := errors.New("some error")
				mockDB.EXPECT().SelectFromReplicationTasksDLQ(gomock.Any(), &sqlplugin.ReplicationTasksDLQFilter{
					ReplicationTasksFilter: sqlplugin.ReplicationTasksFilter{
						ShardID:   shardID,
						MinTaskID: 100,
						MaxTaskID: 1100,
						PageSize:  1000,
					},
					SourceClusterName: "source",
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
		{
			name: "Error case - failed to decode data",
			req: &persistence.GetReplicationTasksFromDLQRequest{
				SourceClusterName: "source",
				GetReplicationTasksRequest: persistence.GetReplicationTasksRequest{
					NextPageToken: serializePageToken(100),
					MaxReadLevel:  199,
					BatchSize:     1000,
				},
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockDB.EXPECT().SelectFromReplicationTasksDLQ(gomock.Any(), &sqlplugin.ReplicationTasksDLQFilter{
					ReplicationTasksFilter: sqlplugin.ReplicationTasksFilter{
						ShardID:   shardID,
						MinTaskID: 100,
						MaxTaskID: 1100,
						PageSize:  1000,
					},
					SourceClusterName: "source",
				}).Return([]sqlplugin.ReplicationTasksRow{
					{
						ShardID:      shardID,
						TaskID:       101,
						Data:         []byte(`replication`),
						DataEncoding: "replication",
					},
				}, nil)
				mockParser.EXPECT().ReplicationTaskInfoFromBlob([]byte(`replication`), "replication").Return(nil, errors.New("some error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			mockParser := serialization.NewMockParser(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), mockParser, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB, mockParser)

			got, err := store.GetReplicationTasksFromDLQ(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestGetReplicationDLQSize(t *testing.T) {
	shardID := 9
	testCases := []struct {
		name      string
		req       *persistence.GetReplicationDLQSizeRequest
		mockSetup func(*sqlplugin.MockDB)
		want      *persistence.GetReplicationDLQSizeResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.GetReplicationDLQSizeRequest{
				SourceClusterName: "source",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().SelectFromReplicationDLQ(gomock.Any(), &sqlplugin.ReplicationTaskDLQFilter{
					SourceClusterName: "source",
					ShardID:           shardID,
				}).Return(int64(1), nil)
			},
			want: &persistence.GetReplicationDLQSizeResponse{
				Size: 1,
			},
			wantErr: false,
		},
		{
			name: "Success case - no row",
			req: &persistence.GetReplicationDLQSizeRequest{
				SourceClusterName: "source",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().SelectFromReplicationDLQ(gomock.Any(), &sqlplugin.ReplicationTaskDLQFilter{
					SourceClusterName: "source",
					ShardID:           shardID,
				}).Return(int64(0), sql.ErrNoRows)
			},
			want: &persistence.GetReplicationDLQSizeResponse{
				Size: 0,
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.GetReplicationDLQSizeRequest{
				SourceClusterName: "source",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().SelectFromReplicationDLQ(gomock.Any(), &sqlplugin.ReplicationTaskDLQFilter{
					SourceClusterName: "source",
					ShardID:           shardID,
				}).Return(int64(0), err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			got, err := store.GetReplicationDLQSize(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestDeleteReplicationTaskFromDLQ(t *testing.T) {
	shardID := 100
	testCases := []struct {
		name      string
		req       *persistence.DeleteReplicationTaskFromDLQRequest
		mockSetup func(*sqlplugin.MockDB)
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.DeleteReplicationTaskFromDLQRequest{
				TaskID:            123,
				SourceClusterName: "source",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().DeleteMessageFromReplicationTasksDLQ(gomock.Any(), &sqlplugin.ReplicationTasksDLQFilter{
					ReplicationTasksFilter: sqlplugin.ReplicationTasksFilter{
						ShardID: shardID,
						TaskID:  123,
					},
					SourceClusterName: "source",
				}).Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.DeleteReplicationTaskFromDLQRequest{
				TaskID:            123,
				SourceClusterName: "source",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().DeleteMessageFromReplicationTasksDLQ(gomock.Any(), &sqlplugin.ReplicationTasksDLQFilter{
					ReplicationTasksFilter: sqlplugin.ReplicationTasksFilter{
						ShardID: shardID,
						TaskID:  123,
					},
					SourceClusterName: "source",
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			err = store.DeleteReplicationTaskFromDLQ(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
			}
		})
	}
}

func TestRangeDeleteReplicationTaskFromDLQ(t *testing.T) {
	shardID := 100
	testCases := []struct {
		name      string
		req       *persistence.RangeDeleteReplicationTaskFromDLQRequest
		mockSetup func(*sqlplugin.MockDB)
		want      *persistence.RangeDeleteReplicationTaskFromDLQResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.RangeDeleteReplicationTaskFromDLQRequest{
				ExclusiveBeginTaskID: 123,
				InclusiveEndTaskID:   345,
				PageSize:             10,
				SourceClusterName:    "source",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().RangeDeleteMessageFromReplicationTasksDLQ(gomock.Any(), &sqlplugin.ReplicationTasksDLQFilter{
					ReplicationTasksFilter: sqlplugin.ReplicationTasksFilter{
						ShardID:            shardID,
						TaskID:             123,
						InclusiveEndTaskID: 345,
						PageSize:           10,
					},
					SourceClusterName: "source",
				}).Return(&sqlResult{rowsAffected: 10}, nil)
			},
			want: &persistence.RangeDeleteReplicationTaskFromDLQResponse{
				TasksCompleted: 10,
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.RangeDeleteReplicationTaskFromDLQRequest{
				ExclusiveBeginTaskID: 123,
				InclusiveEndTaskID:   345,
				PageSize:             10,
				SourceClusterName:    "source",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().RangeDeleteMessageFromReplicationTasksDLQ(gomock.Any(), &sqlplugin.ReplicationTasksDLQFilter{
					ReplicationTasksFilter: sqlplugin.ReplicationTasksFilter{
						ShardID:            shardID,
						TaskID:             123,
						PageSize:           10,
						InclusiveEndTaskID: 345,
					},
					SourceClusterName: "source",
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			got, err := store.RangeDeleteReplicationTaskFromDLQ(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestGetTimerIndexTasks(t *testing.T) {
	shardID := 1
	testCases := []struct {
		name      string
		req       *persistence.GetTimerIndexTasksRequest
		mockSetup func(*sqlplugin.MockDB, *serialization.MockParser)
		want      *persistence.GetTimerIndexTasksResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.GetTimerIndexTasksRequest{
				NextPageToken: func() []byte {
					ti := &timerTaskPageToken{TaskID: 101, Timestamp: time.Unix(1000, 1).UTC()}
					token, err := ti.serialize()
					require.NoError(t, err, "failed to serialize timer task page token")
					return token
				}(),
				MaxTimestamp: time.Unix(2000, 0).UTC(),
				BatchSize:    1,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockDB.EXPECT().SelectFromTimerTasks(gomock.Any(), &sqlplugin.TimerTasksFilter{
					ShardID:                shardID,
					MinVisibilityTimestamp: time.Unix(1000, 1).UTC(),
					TaskID:                 101,
					MaxVisibilityTimestamp: time.Unix(2000, 0).UTC(),
					PageSize:               2,
				}).Return([]sqlplugin.TimerTasksRow{
					{
						ShardID:             shardID,
						VisibilityTimestamp: time.Unix(1001, 0).UTC(),
						TaskID:              11,
						Data:                []byte(`timer`),
						DataEncoding:        "timer",
					},
					{
						ShardID:             shardID,
						VisibilityTimestamp: time.Unix(1003, 0).UTC(),
						TaskID:              22,
						Data:                []byte(`timer`),
						DataEncoding:        "timer",
					},
				}, nil)
				mockParser.EXPECT().TimerTaskInfoFromBlob([]byte(`timer`), "timer").Return(&serialization.TimerTaskInfo{
					DomainID:        serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID:      "test",
					RunID:           serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a54a"),
					TaskType:        1,
					Version:         202,
					EventID:         10,
					ScheduleAttempt: 9,
					TimeoutType:     common.Int16Ptr(12),
				}, nil).Times(2)
			},
			want: &persistence.GetTimerIndexTasksResponse{
				Timers: []*persistence.TimerTaskInfo{
					{
						VisibilityTimestamp: time.Unix(1001, 0).UTC(),
						TaskID:              11,
						DomainID:            "abdcea69-61d5-44c3-9d55-afe23505a542",
						WorkflowID:          "test",
						RunID:               "abdcea69-61d5-44c3-9d55-afe23505a54a",
						TaskType:            1,
						Version:             202,
						EventID:             10,
						ScheduleAttempt:     9,
						TimeoutType:         12,
					},
				},
				NextPageToken: func() []byte {
					ti := &timerTaskPageToken{TaskID: 22, Timestamp: time.Unix(1003, 0).UTC()}
					token, err := ti.serialize()
					require.NoError(t, err, "failed to serialize timer page token")
					return token
				}(),
			},
			wantErr: false,
		},
		{
			name: "Error case - failed to load from database",
			req: &persistence.GetTimerIndexTasksRequest{
				NextPageToken: func() []byte {
					ti := &timerTaskPageToken{TaskID: 101, Timestamp: time.Unix(1000, 1).UTC()}
					token, err := ti.serialize()
					require.NoError(t, err, "failed to serialize timer task page token")
					return token
				}(),
				MaxTimestamp: time.Unix(2000, 0).UTC(),
				BatchSize:    10,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				err := errors.New("some error")
				mockDB.EXPECT().SelectFromTimerTasks(gomock.Any(), &sqlplugin.TimerTasksFilter{
					ShardID:                shardID,
					MinVisibilityTimestamp: time.Unix(1000, 1).UTC(),
					TaskID:                 101,
					MaxVisibilityTimestamp: time.Unix(2000, 0).UTC(),
					PageSize:               11,
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
		{
			name: "Error case - failed to decode data",
			req: &persistence.GetTimerIndexTasksRequest{
				NextPageToken: func() []byte {
					ti := &timerTaskPageToken{TaskID: 101, Timestamp: time.Unix(1000, 1).UTC()}
					token, err := ti.serialize()
					require.NoError(t, err, "failed to serialize timer task page token")
					return token
				}(),
				MaxTimestamp: time.Unix(2000, 0).UTC(),
				BatchSize:    10,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockDB.EXPECT().SelectFromTimerTasks(gomock.Any(), &sqlplugin.TimerTasksFilter{
					ShardID:                shardID,
					MinVisibilityTimestamp: time.Unix(1000, 1).UTC(),
					TaskID:                 101,
					MaxVisibilityTimestamp: time.Unix(2000, 0).UTC(),
					PageSize:               11,
				}).Return([]sqlplugin.TimerTasksRow{
					{
						ShardID:             shardID,
						VisibilityTimestamp: time.Unix(1001, 0).UTC(),
						TaskID:              11,
						Data:                []byte(`timer`),
						DataEncoding:        "timer",
					},
				}, nil)
				mockParser.EXPECT().TimerTaskInfoFromBlob([]byte(`timer`), "timer").Return(nil, errors.New("some error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			mockParser := serialization.NewMockParser(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), mockParser, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB, mockParser)

			got, err := store.GetTimerIndexTasks(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestCompleteTimerTask(t *testing.T) {
	shardID := 100
	testCases := []struct {
		name      string
		req       *persistence.CompleteTimerTaskRequest
		mockSetup func(*sqlplugin.MockDB)
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.CompleteTimerTaskRequest{
				TaskID:              123,
				VisibilityTimestamp: time.Unix(9, 1),
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().DeleteFromTimerTasks(gomock.Any(), &sqlplugin.TimerTasksFilter{
					ShardID:             shardID,
					TaskID:              123,
					VisibilityTimestamp: time.Unix(9, 1),
				}).Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.CompleteTimerTaskRequest{
				TaskID:              123,
				VisibilityTimestamp: time.Unix(9, 1),
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().DeleteFromTimerTasks(gomock.Any(), &sqlplugin.TimerTasksFilter{
					ShardID:             shardID,
					TaskID:              123,
					VisibilityTimestamp: time.Unix(9, 1),
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			err = store.CompleteTimerTask(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
			}
		})
	}
}

func TestRangeCompleteTimerTask(t *testing.T) {
	shardID := 100
	testCases := []struct {
		name      string
		req       *persistence.RangeCompleteTimerTaskRequest
		mockSetup func(*sqlplugin.MockDB)
		want      *persistence.RangeCompleteTimerTaskResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.RangeCompleteTimerTaskRequest{
				InclusiveBeginTimestamp: time.Unix(9, 1),
				ExclusiveEndTimestamp:   time.Unix(99, 1),
				PageSize:                12,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				mockDB.EXPECT().RangeDeleteFromTimerTasks(gomock.Any(), &sqlplugin.TimerTasksFilter{
					ShardID:                shardID,
					MinVisibilityTimestamp: time.Unix(9, 1),
					MaxVisibilityTimestamp: time.Unix(99, 1),
					PageSize:               12,
				}).Return(&sqlResult{rowsAffected: 11}, nil)
			},
			want: &persistence.RangeCompleteTimerTaskResponse{
				TasksCompleted: 11,
			},
			wantErr: false,
		},
		{
			name: "Error case",
			req: &persistence.RangeCompleteTimerTaskRequest{
				InclusiveBeginTimestamp: time.Unix(9, 1),
				ExclusiveEndTimestamp:   time.Unix(99, 1),
				PageSize:                12,
			},
			mockSetup: func(mockDB *sqlplugin.MockDB) {
				err := errors.New("some error")
				mockDB.EXPECT().RangeDeleteFromTimerTasks(gomock.Any(), &sqlplugin.TimerTasksFilter{
					ShardID:                shardID,
					MinVisibilityTimestamp: time.Unix(9, 1),
					MaxVisibilityTimestamp: time.Unix(99, 1),
					PageSize:               12,
				}).Return(nil, err)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB)

			got, err := store.RangeCompleteTimerTask(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestPutReplicationTaskToDLQ(t *testing.T) {
	shardID := 100
	testCases := []struct {
		name      string
		req       *persistence.InternalPutReplicationTaskToDLQRequest
		mockSetup func(*sqlplugin.MockDB, *serialization.MockParser)
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.InternalPutReplicationTaskToDLQRequest{
				SourceClusterName: "source",
				TaskInfo: &persistence.InternalReplicationTaskInfo{
					DomainID:          "abdcea69-61d5-44c3-9d55-afe23505a542",
					WorkflowID:        "test",
					RunID:             "abdcea69-61d5-44c3-9d55-afe23505a54a",
					TaskType:          1,
					TaskID:            101,
					Version:           202,
					FirstEventID:      10,
					NextEventID:       101,
					ScheduledID:       19,
					BranchToken:       []byte(`bt`),
					NewRunBranchToken: []byte(`nbt`),
					CreationTime:      time.Unix(1, 1),
				},
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockParser.EXPECT().ReplicationTaskInfoToBlob(&serialization.ReplicationTaskInfo{
					DomainID:                serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID:              "test",
					RunID:                   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a54a"),
					TaskType:                1,
					Version:                 202,
					FirstEventID:            10,
					NextEventID:             101,
					ScheduledID:             19,
					EventStoreVersion:       persistence.EventStoreVersion,
					NewRunEventStoreVersion: persistence.EventStoreVersion,
					BranchToken:             []byte(`bt`),
					NewRunBranchToken:       []byte(`nbt`),
					CreationTimestamp:       time.Unix(1, 1),
				}).Return(persistence.DataBlob{Data: []byte(`replication`), Encoding: "replication"}, nil)
				mockDB.EXPECT().InsertIntoReplicationTasksDLQ(gomock.Any(), &sqlplugin.ReplicationTaskDLQRow{
					SourceClusterName: "source",
					ShardID:           shardID,
					TaskID:            101,
					Data:              []byte(`replication`),
					DataEncoding:      "replication",
				}).Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name: "Error case - failed to encode data",
			req: &persistence.InternalPutReplicationTaskToDLQRequest{
				SourceClusterName: "source",
				TaskInfo: &persistence.InternalReplicationTaskInfo{
					DomainID:          "abdcea69-61d5-44c3-9d55-afe23505a542",
					WorkflowID:        "test",
					RunID:             "abdcea69-61d5-44c3-9d55-afe23505a54a",
					TaskType:          1,
					TaskID:            101,
					Version:           202,
					FirstEventID:      10,
					NextEventID:       101,
					ScheduledID:       19,
					BranchToken:       []byte(`bt`),
					NewRunBranchToken: []byte(`nbt`),
					CreationTime:      time.Unix(1, 1),
				},
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockParser.EXPECT().ReplicationTaskInfoToBlob(gomock.Any()).Return(persistence.DataBlob{}, errors.New("some error"))
			},
			wantErr: true,
		},
		{
			name: "Error case - failed to insert into database",
			req: &persistence.InternalPutReplicationTaskToDLQRequest{
				SourceClusterName: "source",
				TaskInfo: &persistence.InternalReplicationTaskInfo{
					DomainID:          "abdcea69-61d5-44c3-9d55-afe23505a542",
					WorkflowID:        "test",
					RunID:             "abdcea69-61d5-44c3-9d55-afe23505a54a",
					TaskType:          1,
					TaskID:            101,
					Version:           202,
					FirstEventID:      10,
					NextEventID:       101,
					ScheduledID:       19,
					BranchToken:       []byte(`bt`),
					NewRunBranchToken: []byte(`nbt`),
					CreationTime:      time.Unix(1, 1),
				},
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockParser *serialization.MockParser) {
				mockParser.EXPECT().ReplicationTaskInfoToBlob(gomock.Any()).Return(persistence.DataBlob{Data: []byte(`replication`), Encoding: "replication"}, nil)
				err := errors.New("some error")
				mockDB.EXPECT().InsertIntoReplicationTasksDLQ(gomock.Any(), gomock.Any()).Return(nil, err)
				mockDB.EXPECT().IsDupEntryError(err).Return(false)
				mockDB.EXPECT().IsNotFoundError(err).Return(true)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			mockParser := serialization.NewMockParser(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), mockParser, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB, mockParser)

			err = store.PutReplicationTaskToDLQ(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
			}
		})
	}
}

func TestDeleteWorkflowExecution(t *testing.T) {
	shardID := int64(100)
	testCases := []struct {
		name      string
		req       *persistence.DeleteWorkflowExecutionRequest
		mockSetup func(*sqlplugin.MockDB, *sqlplugin.MockTx)
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.DeleteWorkflowExecutionRequest{
				DomainID:   "abdcea69-61d5-44c3-9d55-afe23505a542",
				WorkflowID: "wid",
				RunID:      "bbdcea69-61d5-44c3-9d55-afe23505a542",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockTx *sqlplugin.MockTx) {
				mockDB.EXPECT().GetTotalNumDBShards().Return(1)
				mockDB.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil)
				mockTx.EXPECT().DeleteFromExecutions(gomock.Any(), &sqlplugin.ExecutionsFilter{
					ShardID:    int(shardID),
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "wid",
					RunID:      serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a542"),
				}).Return(nil, nil)
				mockTx.EXPECT().DeleteFromActivityInfoMaps(gomock.Any(), &sqlplugin.ActivityInfoMapsFilter{
					ShardID:    shardID,
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "wid",
					RunID:      serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a542"),
				}).Return(nil, nil)
				mockTx.EXPECT().DeleteFromTimerInfoMaps(gomock.Any(), &sqlplugin.TimerInfoMapsFilter{
					ShardID:    shardID,
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "wid",
					RunID:      serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a542"),
				}).Return(nil, nil)
				mockTx.EXPECT().DeleteFromChildExecutionInfoMaps(gomock.Any(), &sqlplugin.ChildExecutionInfoMapsFilter{
					ShardID:    shardID,
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "wid",
					RunID:      serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a542"),
				}).Return(nil, nil)
				mockTx.EXPECT().DeleteFromRequestCancelInfoMaps(gomock.Any(), &sqlplugin.RequestCancelInfoMapsFilter{
					ShardID:    shardID,
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "wid",
					RunID:      serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a542"),
				}).Return(nil, nil)
				mockTx.EXPECT().DeleteFromSignalInfoMaps(gomock.Any(), &sqlplugin.SignalInfoMapsFilter{
					ShardID:    shardID,
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "wid",
					RunID:      serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a542"),
				}).Return(nil, nil)
				mockTx.EXPECT().DeleteFromBufferedEvents(gomock.Any(), &sqlplugin.BufferedEventsFilter{
					ShardID:    int(shardID),
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "wid",
					RunID:      serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a542"),
				}).Return(nil, nil)
				mockTx.EXPECT().DeleteFromSignalsRequestedSets(gomock.Any(), &sqlplugin.SignalsRequestedSetsFilter{
					ShardID:    shardID,
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "wid",
					RunID:      serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a542"),
				}).Return(nil, nil)
				mockTx.EXPECT().Commit().Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Error case - failed to delete from executions",
			req: &persistence.DeleteWorkflowExecutionRequest{
				DomainID:   "abdcea69-61d5-44c3-9d55-afe23505a542",
				WorkflowID: "wid",
				RunID:      "bbdcea69-61d5-44c3-9d55-afe23505a542",
			},
			mockSetup: func(mockDB *sqlplugin.MockDB, mockTx *sqlplugin.MockTx) {
				mockDB.EXPECT().GetTotalNumDBShards().Return(1)
				mockDB.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil)
				mockTx.EXPECT().DeleteFromExecutions(gomock.Any(), &sqlplugin.ExecutionsFilter{
					ShardID:    int(shardID),
					DomainID:   serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a542"),
					WorkflowID: "wid",
					RunID:      serialization.MustParseUUID("bbdcea69-61d5-44c3-9d55-afe23505a542"),
				}).Return(nil, errors.New("some error"))
				mockTx.EXPECT().IsNotFoundError(gomock.Any()).Return(true)
				mockTx.EXPECT().Rollback().Return(nil)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := sqlplugin.NewMockDB(ctrl)
			mockTx := sqlplugin.NewMockTx(ctrl)
			store, err := NewSQLExecutionStore(mockDB, nil, int(shardID), nil, nil)
			require.NoError(t, err, "failed to create execution store")

			tc.mockSetup(mockDB, mockTx)

			err = store.DeleteWorkflowExecution(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
			}
		})
	}
}

func TestTxExecuteShardLocked(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(*sqlplugin.MockDB, *sqlplugin.MockTx)
		operation string
		rangeID   int64
		fn        func(sqlplugin.Tx) error
		wantError error
	}{
		{
			name: "Success",
			mockSetup: func(mockDB *sqlplugin.MockDB, mockTx *sqlplugin.MockTx) {
				mockDB.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil)
				mockTx.EXPECT().ReadLockShards(gomock.Any(), gomock.Any()).Return(11, nil)
				mockTx.EXPECT().Commit().Return(nil)
			},
			operation: "Insert",
			rangeID:   11,
			fn:        func(sqlplugin.Tx) error { return nil },
			wantError: nil,
		},
		{
			name: "Error",
			mockSetup: func(mockDB *sqlplugin.MockDB, mockTx *sqlplugin.MockTx) {
				mockDB.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil)
				mockTx.EXPECT().ReadLockShards(gomock.Any(), gomock.Any()).Return(11, nil)
				mockTx.EXPECT().Rollback().Return(nil)
				mockDB.EXPECT().IsNotFoundError(gomock.Any()).Return(false)
				mockDB.EXPECT().IsTimeoutError(gomock.Any()).Return(false)
				mockDB.EXPECT().IsThrottlingError(gomock.Any()).Return(false)
			},
			operation: "Insert",
			rangeID:   11,
			fn:        func(sqlplugin.Tx) error { return errors.New("error") },
			wantError: &types.InternalServiceError{Message: "Insert operation failed.  Error: error"},
		},
		{
			name: "Error - shard ownership lost",
			mockSetup: func(mockDB *sqlplugin.MockDB, mockTx *sqlplugin.MockTx) {
				mockDB.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil)
				mockTx.EXPECT().ReadLockShards(gomock.Any(), gomock.Any()).Return(12, nil)
				mockTx.EXPECT().Rollback().Return(nil)
			},
			operation: "Insert",
			rangeID:   11,
			fn:        func(sqlplugin.Tx) error { return errors.New("error") },
			wantError: &persistence.ShardOwnershipLostError{ShardID: 0, Msg: "Failed to lock shard. Previous range ID: 11; new range ID: 12"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockDB := sqlplugin.NewMockDB(ctrl)
			mockTx := sqlplugin.NewMockTx(ctrl)
			tt.mockSetup(mockDB, mockTx)

			s := &sqlExecutionStore{
				shardID: 0,
				sqlStore: sqlStore{
					db:     mockDB,
					logger: testlogger.New(t),
				},
			}

			gotError := s.txExecuteShardLocked(context.Background(), 0, tt.operation, tt.rangeID, tt.fn)
			assert.Equal(t, tt.wantError, gotError)
		})
	}
}

func TestCreateWorkflowExecution(t *testing.T) {
	testCases := []struct {
		name                             string
		req                              *persistence.InternalCreateWorkflowExecutionRequest
		lockCurrentExecutionIfExistsFn   func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error)
		createOrUpdateCurrentExecutionFn func(context.Context, sqlplugin.Tx, persistence.CreateWorkflowMode, int, serialization.UUID, string, serialization.UUID, int, int, string, int64, int64) error
		applyWorkflowSnapshotTxAsNewFn   func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error
		wantErr                          bool
		want                             *persistence.CreateWorkflowExecutionResponse
		assertErr                        func(t *testing.T, err error)
	}{
		{
			name: "Success - mode brand new",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeBrandNew,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{},
				},
			},
			lockCurrentExecutionIfExistsFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error) {
				return nil, nil
			},
			createOrUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, persistence.CreateWorkflowMode, int, serialization.UUID, string, serialization.UUID, int, int, string, int64, int64) error {
				return nil
			},
			applyWorkflowSnapshotTxAsNewFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return nil
			},
			want: &persistence.CreateWorkflowExecutionResponse{},
		},
		{
			name: "Success - mode workflow ID reuse",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeWorkflowIDReuse,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{},
				},
			},
			lockCurrentExecutionIfExistsFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error) {
				return &sqlplugin.CurrentExecutionsRow{
					State: persistence.WorkflowStateCompleted,
				}, nil
			},
			createOrUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, persistence.CreateWorkflowMode, int, serialization.UUID, string, serialization.UUID, int, int, string, int64, int64) error {
				return nil
			},
			applyWorkflowSnapshotTxAsNewFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return nil
			},
			want: &persistence.CreateWorkflowExecutionResponse{},
		},
		{
			name: "Success - mode zombie",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeZombie,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateZombie,
					},
				},
			},
			lockCurrentExecutionIfExistsFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error) {
				return &sqlplugin.CurrentExecutionsRow{
					RunID: serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a54a"),
				}, nil
			},
			createOrUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, persistence.CreateWorkflowMode, int, serialization.UUID, string, serialization.UUID, int, int, string, int64, int64) error {
				return nil
			},
			applyWorkflowSnapshotTxAsNewFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return nil
			},
			want: &persistence.CreateWorkflowExecutionResponse{},
		},
		{
			name: "Error - mode state validation failed",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeZombie,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCreated,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Error - lockCurrentExecutionIfExists failed",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeBrandNew,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{},
				},
			},
			lockCurrentExecutionIfExistsFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error) {
				return nil, errors.New("some random error")
			},
			wantErr: true,
		},
		{
			name: "Error - mode brand new",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeBrandNew,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{},
				},
			},
			lockCurrentExecutionIfExistsFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error) {
				return &sqlplugin.CurrentExecutionsRow{
					CreateRequestID:  "test",
					WorkflowID:       "test",
					RunID:            serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a54a"),
					State:            persistence.WorkflowStateCreated,
					CloseStatus:      persistence.WorkflowCloseStatusNone,
					LastWriteVersion: 10,
				}, nil
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				assert.Equal(t, &persistence.WorkflowExecutionAlreadyStartedError{
					Msg:              "Workflow execution already running. WorkflowId: test",
					StartRequestID:   "test",
					RunID:            "abdcea69-61d5-44c3-9d55-afe23505a54a",
					State:            persistence.WorkflowStateCreated,
					CloseStatus:      persistence.WorkflowCloseStatusNone,
					LastWriteVersion: 10,
				}, err)
			},
		},
		{
			name: "Error - mode workflow ID reuse, version mismatch",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeWorkflowIDReuse,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{},
				},
			},
			lockCurrentExecutionIfExistsFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error) {
				return &sqlplugin.CurrentExecutionsRow{
					State:            persistence.WorkflowStateCompleted,
					LastWriteVersion: 10,
				}, nil
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				assert.Equal(t, &persistence.CurrentWorkflowConditionFailedError{
					Msg: "Workflow execution creation condition failed. WorkflowId: , LastWriteVersion: 10, PreviousLastWriteVersion: 0",
				}, err)
			},
		},
		{
			name: "Error - mode workflow ID reuse, state mismatch",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeWorkflowIDReuse,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{},
				},
			},
			lockCurrentExecutionIfExistsFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error) {
				return &sqlplugin.CurrentExecutionsRow{
					State: persistence.WorkflowStateCreated,
				}, nil
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				assert.Equal(t, &persistence.CurrentWorkflowConditionFailedError{
					Msg: "Workflow execution creation condition failed. WorkflowId: , State: 0, Expected: 2",
				}, err)
			},
		},
		{
			name: "Error - mode workflow ID reuse, run ID mismatch",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeWorkflowIDReuse,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{},
				},
			},
			lockCurrentExecutionIfExistsFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error) {
				return &sqlplugin.CurrentExecutionsRow{
					State: persistence.WorkflowStateCompleted,
					RunID: serialization.MustParseUUID("abdcea69-61d5-44c3-9d55-afe23505a54a"),
				}, nil
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				assert.Equal(t, &persistence.CurrentWorkflowConditionFailedError{
					Msg: "Workflow execution creation condition failed. WorkflowId: , RunID: abdcea69-61d5-44c3-9d55-afe23505a54a, PreviousRunID: ",
				}, err)
			},
		},
		{
			name: "Error - mode zombie, run ID match",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeZombie,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateZombie,
					},
				},
			},
			lockCurrentExecutionIfExistsFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error) {
				return &sqlplugin.CurrentExecutionsRow{}, nil
			},
			wantErr: true,
		},
		{
			name: "Error - unknown mode",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowMode(100),
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{},
				},
			},
			wantErr: true,
		},
		{
			name: "Error - createOrUpdateCurrentExecution failed",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeBrandNew,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{},
				},
			},
			lockCurrentExecutionIfExistsFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error) {
				return nil, nil
			},
			createOrUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, persistence.CreateWorkflowMode, int, serialization.UUID, string, serialization.UUID, int, int, string, int64, int64) error {
				return errors.New("some random error")
			},
			wantErr: true,
		},
		{
			name: "Error - applyWorkflowSnapshotTxAsNew failed",
			req: &persistence.InternalCreateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.CreateWorkflowModeBrandNew,
				NewWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{},
				},
			},
			lockCurrentExecutionIfExistsFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string) (*sqlplugin.CurrentExecutionsRow, error) {
				return nil, nil
			},
			createOrUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, persistence.CreateWorkflowMode, int, serialization.UUID, string, serialization.UUID, int, int, string, int64, int64) error {
				return nil
			},
			applyWorkflowSnapshotTxAsNewFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return errors.New("some random error")
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockDB := sqlplugin.NewMockDB(ctrl)
			mockDB.EXPECT().GetTotalNumDBShards().Return(1)
			s := &sqlExecutionStore{
				shardID: 0,
				sqlStore: sqlStore{
					db:     mockDB,
					logger: testlogger.New(t),
				},
				txExecuteShardLockedFn: func(_ context.Context, _ int, _ string, _ int64, fn func(sqlplugin.Tx) error) error {
					return fn(nil)
				},
				lockCurrentExecutionIfExistsFn:   tc.lockCurrentExecutionIfExistsFn,
				createOrUpdateCurrentExecutionFn: tc.createOrUpdateCurrentExecutionFn,
				applyWorkflowSnapshotTxAsNewFn:   tc.applyWorkflowSnapshotTxAsNewFn,
			}

			got, err := s.CreateWorkflowExecution(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
				if tc.assertErr != nil {
					tc.assertErr(t, err)
				}
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, got, "Unexpected result for test case")
			}
		})
	}
}

func TestUpdateWorkflowExecution(t *testing.T) {
	testCases := []struct {
		name                                   string
		req                                    *persistence.InternalUpdateWorkflowExecutionRequest
		assertNotCurrentExecutionFn            func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID) error
		assertRunIDAndUpdateCurrentExecutionFn func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error
		applyWorkflowSnapshotTxAsNewFn         func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error
		applyWorkflowMutationTxFn              func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowMutation, serialization.Parser) error
		wantErr                                bool
		assertErr                              func(t *testing.T, err error)
	}{
		{
			name: "Success - mode ignore current",
			req: &persistence.InternalUpdateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.UpdateWorkflowModeIgnoreCurrent,
				UpdateWorkflowMutation: persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{},
				},
			},
			applyWorkflowMutationTxFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowMutation, serialization.Parser) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "Success - mode bypass current",
			req: &persistence.InternalUpdateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.UpdateWorkflowModeBypassCurrent,
				UpdateWorkflowMutation: persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
			},
			assertNotCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID) error {
				return nil
			},
			applyWorkflowMutationTxFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowMutation, serialization.Parser) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "Success - mode update current, new workflow",
			req: &persistence.InternalUpdateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.UpdateWorkflowModeUpdateCurrent,
				UpdateWorkflowMutation: persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
				NewWorkflowSnapshot: &persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCreated,
					},
				},
			},
			assertRunIDAndUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error {
				return nil
			},
			applyWorkflowMutationTxFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowMutation, serialization.Parser) error {
				return nil
			},
			applyWorkflowSnapshotTxAsNewFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "Success - mode update current, no new workflow",
			req: &persistence.InternalUpdateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.UpdateWorkflowModeUpdateCurrent,
				UpdateWorkflowMutation: persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateRunning,
					},
				},
			},
			assertRunIDAndUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error {
				return nil
			},
			applyWorkflowMutationTxFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowMutation, serialization.Parser) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "Error - mode state validation failed",
			req: &persistence.InternalUpdateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.UpdateWorkflowModeUpdateCurrent,
				UpdateWorkflowMutation: persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateZombie,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Error - assertNotCurrentExecution failed",
			req: &persistence.InternalUpdateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.UpdateWorkflowModeBypassCurrent,
				UpdateWorkflowMutation: persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
			},
			assertNotCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID) error {
				return errors.New("some random error")
			},
			wantErr: true,
		},
		{
			name: "Error - domain ID mismatch",
			req: &persistence.InternalUpdateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.UpdateWorkflowModeUpdateCurrent,
				UpdateWorkflowMutation: persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						DomainID: "a8ead65c-9d0d-43a2-a6ad-dd17c99509af",
						State:    persistence.WorkflowStateCompleted,
					},
				},
				NewWorkflowSnapshot: &persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						DomainID: "c3fab112-5175-4044-a096-a32e7badd4a8",
					},
				},
			},
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				assert.Equal(t, &types.InternalServiceError{
					Message: "UpdateWorkflowExecution: cannot continue as new to another domain",
				}, err)
			},
		},
		{
			name: "Error - assertRunIDAndUpdateCurrentExecution failed",
			req: &persistence.InternalUpdateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.UpdateWorkflowModeUpdateCurrent,
				UpdateWorkflowMutation: persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
			},
			assertRunIDAndUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error {
				return errors.New("some random error")
			},
			wantErr: true,
		},
		{
			name: "Error - applyWorkflowMutationTxFn failed",
			req: &persistence.InternalUpdateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.UpdateWorkflowModeUpdateCurrent,
				UpdateWorkflowMutation: persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
			},
			assertRunIDAndUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error {
				return nil
			},
			applyWorkflowMutationTxFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowMutation, serialization.Parser) error {
				return errors.New("some random error")
			},
			wantErr: true,
		},
		{
			name: "Error - applyWorkflowSnapshotTxAsNew failed",
			req: &persistence.InternalUpdateWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.UpdateWorkflowModeUpdateCurrent,
				UpdateWorkflowMutation: persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
				NewWorkflowSnapshot: &persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCreated,
					},
				},
			},
			assertRunIDAndUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error {
				return nil
			},
			applyWorkflowMutationTxFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowMutation, serialization.Parser) error {
				return nil
			},
			applyWorkflowSnapshotTxAsNewFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return errors.New("some random error")
			},
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockDB := sqlplugin.NewMockDB(ctrl)
			mockDB.EXPECT().GetTotalNumDBShards().Return(1)
			s := &sqlExecutionStore{
				shardID: 0,
				sqlStore: sqlStore{
					db:     mockDB,
					logger: testlogger.New(t),
				},
				txExecuteShardLockedFn: func(_ context.Context, _ int, _ string, _ int64, fn func(sqlplugin.Tx) error) error {
					return fn(nil)
				},
				assertNotCurrentExecutionFn:            tc.assertNotCurrentExecutionFn,
				assertRunIDAndUpdateCurrentExecutionFn: tc.assertRunIDAndUpdateCurrentExecutionFn,
				applyWorkflowMutationTxFn:              tc.applyWorkflowMutationTxFn,
				applyWorkflowSnapshotTxAsNewFn:         tc.applyWorkflowSnapshotTxAsNewFn,
			}

			err := s.UpdateWorkflowExecution(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
				if tc.assertErr != nil {
					tc.assertErr(t, err)
				}
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
			}
		})
	}
}

func TestConflictResolveWorkflowExecution(t *testing.T) {
	testCases := []struct {
		name                                   string
		req                                    *persistence.InternalConflictResolveWorkflowExecutionRequest
		assertRunIDAndUpdateCurrentExecutionFn func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error
		assertNotCurrentExecutionFn            func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID) error
		applyWorkflowMutationTxFn              func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowMutation, serialization.Parser) error
		applyWorkflowSnapshotTxAsResetFn       func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error
		applyWorkflowSnapshotTxAsNewFn         func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error
		wantErr                                bool
		assertErr                              func(t *testing.T, err error)
	}{
		{
			name: "Success - mode bypass current",
			req: &persistence.InternalConflictResolveWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.ConflictResolveWorkflowModeBypassCurrent,
				ResetWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
			},
			assertNotCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID) error {
				return nil
			},
			applyWorkflowSnapshotTxAsResetFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "Success - mode update current, current workflow exists",
			req: &persistence.InternalConflictResolveWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.ConflictResolveWorkflowModeUpdateCurrent,
				ResetWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
				NewWorkflowSnapshot: &persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCreated,
					},
				},
				CurrentWorkflowMutation: &persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
			},
			assertRunIDAndUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error {
				return nil
			},
			applyWorkflowSnapshotTxAsResetFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return nil
			},
			applyWorkflowMutationTxFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowMutation, serialization.Parser) error {
				return nil
			},
			applyWorkflowSnapshotTxAsNewFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "Success - mode update current, no current workflow",
			req: &persistence.InternalConflictResolveWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.ConflictResolveWorkflowModeUpdateCurrent,
				ResetWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
				NewWorkflowSnapshot: &persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCreated,
					},
				},
			},
			assertRunIDAndUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error {
				return nil
			},
			applyWorkflowSnapshotTxAsResetFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return nil
			},
			applyWorkflowSnapshotTxAsNewFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "Error - mode state validation failed",
			req: &persistence.InternalConflictResolveWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.ConflictResolveWorkflowModeUpdateCurrent,
				ResetWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateZombie,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Error - assertNotCurrentExecution failed",
			req: &persistence.InternalConflictResolveWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.ConflictResolveWorkflowModeBypassCurrent,
				ResetWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
			},
			assertNotCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID) error {
				return errors.New("some random error")
			},
			wantErr: true,
		},
		{
			name: "Error - assertRunIDAndUpdateCurrentExecution failed",
			req: &persistence.InternalConflictResolveWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.ConflictResolveWorkflowModeUpdateCurrent,
				ResetWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
				NewWorkflowSnapshot: &persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCreated,
					},
				},
				CurrentWorkflowMutation: &persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
			},
			assertRunIDAndUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error {
				return errors.New("some random error")
			},
			wantErr: true,
		},
		{
			name: "Error - applyWorkflowResetSnapshotTx failed",
			req: &persistence.InternalConflictResolveWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.ConflictResolveWorkflowModeBypassCurrent,
				ResetWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
			},
			assertNotCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID) error {
				return nil
			},
			applyWorkflowSnapshotTxAsResetFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return errors.New("some random error")
			},
			wantErr: true,
		},
		{
			name: "Error - applyWorkflowMutationTxFn failed",
			req: &persistence.InternalConflictResolveWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.ConflictResolveWorkflowModeUpdateCurrent,
				ResetWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
				NewWorkflowSnapshot: &persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCreated,
					},
				},
				CurrentWorkflowMutation: &persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
			},
			assertRunIDAndUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error {
				return nil
			},
			applyWorkflowSnapshotTxAsResetFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return nil
			},
			applyWorkflowMutationTxFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowMutation, serialization.Parser) error {
				return errors.New("some random error")
			},
			wantErr: true,
		},
		{
			name: "Error - applyWorkflowSnapshotTxAsNew failed",
			req: &persistence.InternalConflictResolveWorkflowExecutionRequest{
				RangeID: 1,
				Mode:    persistence.ConflictResolveWorkflowModeUpdateCurrent,
				ResetWorkflowSnapshot: persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
				NewWorkflowSnapshot: &persistence.InternalWorkflowSnapshot{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCreated,
					},
				},
				CurrentWorkflowMutation: &persistence.InternalWorkflowMutation{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						State: persistence.WorkflowStateCompleted,
					},
				},
			},
			assertRunIDAndUpdateCurrentExecutionFn: func(context.Context, sqlplugin.Tx, int, serialization.UUID, string, serialization.UUID, serialization.UUID, string, int, int, int64, int64) error {
				return nil
			},
			applyWorkflowSnapshotTxAsResetFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return nil
			},
			applyWorkflowMutationTxFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowMutation, serialization.Parser) error {
				return nil
			},
			applyWorkflowSnapshotTxAsNewFn: func(context.Context, sqlplugin.Tx, int, *persistence.InternalWorkflowSnapshot, serialization.Parser) error {
				return errors.New("some random error")
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockDB := sqlplugin.NewMockDB(ctrl)
			mockDB.EXPECT().GetTotalNumDBShards().Return(1)
			s := &sqlExecutionStore{
				shardID: 0,
				sqlStore: sqlStore{
					db:     mockDB,
					logger: testlogger.New(t),
				},
				txExecuteShardLockedFn: func(_ context.Context, _ int, _ string, _ int64, fn func(sqlplugin.Tx) error) error {
					return fn(nil)
				},
				assertNotCurrentExecutionFn:            tc.assertNotCurrentExecutionFn,
				assertRunIDAndUpdateCurrentExecutionFn: tc.assertRunIDAndUpdateCurrentExecutionFn,
				applyWorkflowMutationTxFn:              tc.applyWorkflowMutationTxFn,
				applyWorkflowSnapshotTxAsResetFn:       tc.applyWorkflowSnapshotTxAsResetFn,
				applyWorkflowSnapshotTxAsNewFn:         tc.applyWorkflowSnapshotTxAsNewFn,
			}

			err := s.ConflictResolveWorkflowExecution(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
				if tc.assertErr != nil {
					tc.assertErr(t, err)
				}
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
			}
		})
	}
}

func TestCreateFailoverMarkerTasks(t *testing.T) {
	testCases := []struct {
		name      string
		req       *persistence.CreateFailoverMarkersRequest
		mockSetup func(*sqlplugin.MockTx, *serialization.MockParser)
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.CreateFailoverMarkersRequest{
				RangeID: 1,
				Markers: []*persistence.FailoverMarkerTask{
					{
						TaskID:              1,
						VisibilityTimestamp: time.Unix(11, 12),
						Version:             101,
						DomainID:            "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
					},
				},
			},
			mockSetup: func(tx *sqlplugin.MockTx, parser *serialization.MockParser) {
				parser.EXPECT().ReplicationTaskInfoToBlob(gomock.Any()).Return(persistence.DataBlob{
					Encoding: common.EncodingTypeThriftRW,
					Data:     []byte("test data"),
				}, nil)
				tx.EXPECT().InsertIntoReplicationTasks(gomock.Any(), []sqlplugin.ReplicationTasksRow{
					{
						ShardID:      0,
						TaskID:       1,
						Data:         []byte("test data"),
						DataEncoding: "thriftrw",
					},
				}).Return(&sqlResult{
					rowsAffected: 1,
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "Error - ReplicationTaskInfoToBlob failed",
			req: &persistence.CreateFailoverMarkersRequest{
				RangeID: 1,
				Markers: []*persistence.FailoverMarkerTask{
					{
						TaskID:              1,
						VisibilityTimestamp: time.Unix(11, 12),
						Version:             101,
						DomainID:            "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
					},
				},
			},
			mockSetup: func(tx *sqlplugin.MockTx, parser *serialization.MockParser) {
				parser.EXPECT().ReplicationTaskInfoToBlob(gomock.Any()).Return(persistence.DataBlob{}, errors.New("some random error"))
			},
			wantErr: true,
		},
		{
			name: "Error - InsertIntoReplicationTasks failed",
			req: &persistence.CreateFailoverMarkersRequest{
				RangeID: 1,
				Markers: []*persistence.FailoverMarkerTask{
					{
						TaskID:              1,
						VisibilityTimestamp: time.Unix(11, 12),
						Version:             101,
						DomainID:            "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
					},
				},
			},
			mockSetup: func(tx *sqlplugin.MockTx, parser *serialization.MockParser) {
				parser.EXPECT().ReplicationTaskInfoToBlob(gomock.Any()).Return(persistence.DataBlob{
					Encoding: common.EncodingTypeThriftRW,
					Data:     []byte("test data"),
				}, nil)
				tx.EXPECT().InsertIntoReplicationTasks(gomock.Any(), []sqlplugin.ReplicationTasksRow{
					{
						ShardID:      0,
						TaskID:       1,
						Data:         []byte("test data"),
						DataEncoding: "thriftrw",
					},
				}).Return(nil, errors.New("some random error"))
				tx.EXPECT().IsNotFoundError(gomock.Any()).Return(true)
			},
			wantErr: true,
		},
		{
			name: "Error - row affected error",
			req: &persistence.CreateFailoverMarkersRequest{
				RangeID: 1,
				Markers: []*persistence.FailoverMarkerTask{
					{
						TaskID:              1,
						VisibilityTimestamp: time.Unix(11, 12),
						Version:             101,
						DomainID:            "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
					},
				},
			},
			mockSetup: func(tx *sqlplugin.MockTx, parser *serialization.MockParser) {
				parser.EXPECT().ReplicationTaskInfoToBlob(gomock.Any()).Return(persistence.DataBlob{
					Encoding: common.EncodingTypeThriftRW,
					Data:     []byte("test data"),
				}, nil)
				tx.EXPECT().InsertIntoReplicationTasks(gomock.Any(), []sqlplugin.ReplicationTasksRow{
					{
						ShardID:      0,
						TaskID:       1,
						Data:         []byte("test data"),
						DataEncoding: "thriftrw",
					},
				}).Return(&sqlResult{
					err: errors.New("some error"),
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "Error - row affected number mismatch",
			req: &persistence.CreateFailoverMarkersRequest{
				RangeID: 1,
				Markers: []*persistence.FailoverMarkerTask{
					{
						TaskID:              1,
						VisibilityTimestamp: time.Unix(11, 12),
						Version:             101,
						DomainID:            "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
					},
				},
			},
			mockSetup: func(tx *sqlplugin.MockTx, parser *serialization.MockParser) {
				parser.EXPECT().ReplicationTaskInfoToBlob(gomock.Any()).Return(persistence.DataBlob{
					Encoding: common.EncodingTypeThriftRW,
					Data:     []byte("test data"),
				}, nil)
				tx.EXPECT().InsertIntoReplicationTasks(gomock.Any(), []sqlplugin.ReplicationTasksRow{
					{
						ShardID:      0,
						TaskID:       1,
						Data:         []byte("test data"),
						DataEncoding: "thriftrw",
					},
				}).Return(&sqlResult{
					rowsAffected: 0,
				}, nil)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := sqlplugin.NewMockDB(ctrl)
			db.EXPECT().GetTotalNumDBShards().Return(1)
			tx := sqlplugin.NewMockTx(ctrl)
			parser := serialization.NewMockParser(ctrl)
			tc.mockSetup(tx, parser)
			s := &sqlExecutionStore{
				shardID: 0,
				sqlStore: sqlStore{
					db:     db,
					logger: testlogger.New(t),
					parser: parser,
				},
				txExecuteShardLockedFn: func(_ context.Context, _ int, _ string, _ int64, fn func(sqlplugin.Tx) error) error {
					return fn(tx)
				},
			}

			err := s.CreateFailoverMarkerTasks(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
			}
		})
	}
}

func TestGetWorkflowExecution(t *testing.T) {
	testCases := []struct {
		name      string
		req       *persistence.InternalGetWorkflowExecutionRequest
		mockSetup func(*sqlplugin.MockDB, *serialization.MockParser)
		want      *persistence.InternalGetWorkflowExecutionResponse
		wantErr   bool
	}{
		{
			name: "Success case",
			req: &persistence.InternalGetWorkflowExecutionRequest{
				DomainID: "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
				Execution: types.WorkflowExecution{
					WorkflowID: "test-workflow-id",
					RunID:      "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
				},
			},
			mockSetup: func(db *sqlplugin.MockDB, parser *serialization.MockParser) {
				db.EXPECT().SelectFromExecutions(gomock.Any(), gomock.Any()).Return([]sqlplugin.ExecutionsRow{
					{
						ShardID:          0,
						DomainID:         serialization.MustParseUUID("ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d"),
						WorkflowID:       "test-workflow-id",
						RunID:            serialization.MustParseUUID("ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f"),
						NextEventID:      101,
						LastWriteVersion: 11,
						Data:             []byte("test data"),
						DataEncoding:     "thriftrw",
					},
				}, nil)
				db.EXPECT().SelectFromActivityInfoMaps(gomock.Any(), gomock.Any()).Return([]sqlplugin.ActivityInfoMapsRow{
					{
						ShardID:      0,
						DomainID:     serialization.MustParseUUID("ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d"),
						WorkflowID:   "test-workflow-id",
						RunID:        serialization.MustParseUUID("ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f"),
						ScheduleID:   101,
						Data:         []byte("test data"),
						DataEncoding: "thriftrw",
					},
				}, nil)
				db.EXPECT().SelectFromTimerInfoMaps(gomock.Any(), gomock.Any()).Return([]sqlplugin.TimerInfoMapsRow{
					{
						ShardID:      0,
						DomainID:     serialization.MustParseUUID("ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d"),
						WorkflowID:   "test-workflow-id",
						RunID:        serialization.MustParseUUID("ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f"),
						TimerID:      "101",
						Data:         []byte("test data"),
						DataEncoding: "thriftrw",
					},
				}, nil)
				db.EXPECT().SelectFromChildExecutionInfoMaps(gomock.Any(), gomock.Any()).Return([]sqlplugin.ChildExecutionInfoMapsRow{
					{
						ShardID:      0,
						DomainID:     serialization.MustParseUUID("ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d"),
						WorkflowID:   "test-workflow-id",
						RunID:        serialization.MustParseUUID("ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f"),
						InitiatedID:  101,
						Data:         []byte("test data"),
						DataEncoding: "thriftrw",
					},
				}, nil)
				db.EXPECT().SelectFromRequestCancelInfoMaps(gomock.Any(), gomock.Any()).Return([]sqlplugin.RequestCancelInfoMapsRow{
					{
						ShardID:      0,
						DomainID:     serialization.MustParseUUID("ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d"),
						WorkflowID:   "test-workflow-id",
						RunID:        serialization.MustParseUUID("ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f"),
						InitiatedID:  101,
						Data:         []byte("test data"),
						DataEncoding: "thriftrw",
					},
				}, nil)
				db.EXPECT().SelectFromSignalInfoMaps(gomock.Any(), gomock.Any()).Return([]sqlplugin.SignalInfoMapsRow{
					{
						ShardID:      0,
						DomainID:     serialization.MustParseUUID("ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d"),
						WorkflowID:   "test-workflow-id",
						RunID:        serialization.MustParseUUID("ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f"),
						InitiatedID:  101,
						Data:         []byte("test data"),
						DataEncoding: "thriftrw",
					},
				}, nil)
				db.EXPECT().SelectFromSignalsRequestedSets(gomock.Any(), gomock.Any()).Return([]sqlplugin.SignalsRequestedSetsRow{
					{
						ShardID:    0,
						DomainID:   serialization.MustParseUUID("ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d"),
						WorkflowID: "test-workflow-id",
						RunID:      serialization.MustParseUUID("ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f"),
						SignalID:   "test-signal-id",
					},
				}, nil)
				db.EXPECT().SelectFromBufferedEvents(gomock.Any(), gomock.Any()).Return([]sqlplugin.BufferedEventsRow{
					{
						ShardID:      0,
						DomainID:     serialization.MustParseUUID("ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d"),
						WorkflowID:   "test-workflow-id",
						RunID:        serialization.MustParseUUID("ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f"),
						Data:         []byte("test data"),
						DataEncoding: "thriftrw",
					},
				}, nil)
				parser.EXPECT().WorkflowExecutionInfoFromBlob(gomock.Any(), gomock.Any()).Return(&serialization.WorkflowExecutionInfo{
					ParentDomainID:                     serialization.MustParseUUID("ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d"),
					ParentWorkflowID:                   "test-parent-workflow-id",
					ParentRunID:                        serialization.MustParseUUID("ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f"),
					InitiatedID:                        101,
					CompletionEventBatchID:             common.Int64Ptr(11),
					CompletionEvent:                    []byte("test completion event"),
					CompletionEventEncoding:            "json",
					TaskList:                           "test-task-list",
					IsCron:                             true,
					WorkflowTypeName:                   "test-workflow-type",
					WorkflowTimeout:                    time.Duration(101),
					DecisionTaskTimeout:                time.Duration(102),
					ExecutionContext:                   []byte("test execution context"),
					State:                              persistence.WorkflowStateCompleted,
					CloseStatus:                        persistence.WorkflowCloseStatusCompleted,
					StartVersion:                       111,
					LastWriteEventID:                   common.Int64Ptr(11),
					LastEventTaskID:                    12,
					LastFirstEventID:                   13,
					LastProcessedEvent:                 14,
					StartTimestamp:                     time.Unix(11, 12),
					LastUpdatedTimestamp:               time.Unix(13, 14),
					DecisionVersion:                    101,
					DecisionScheduleID:                 102,
					DecisionStartedID:                  103,
					DecisionTimeout:                    time.Duration(104),
					DecisionAttempt:                    105,
					DecisionStartedTimestamp:           time.Unix(15, 16),
					DecisionScheduledTimestamp:         time.Unix(17, 18),
					CancelRequested:                    true,
					DecisionOriginalScheduledTimestamp: time.Unix(19, 20),
					CreateRequestID:                    "test-create-request-id",
					DecisionRequestID:                  "test-decision-request-id",
					CancelRequestID:                    "test-cancel-request-id",
					StickyTaskList:                     "test-sticky-task-list",
					StickyScheduleToStartTimeout:       time.Duration(106),
					RetryAttempt:                       107,
					RetryInitialInterval:               time.Duration(108),
					RetryMaximumInterval:               time.Duration(109),
					RetryMaximumAttempts:               110,
					RetryExpiration:                    time.Duration(111),
					RetryBackoffCoefficient:            111,
					RetryExpirationTimestamp:           time.Unix(23, 24),
					RetryNonRetryableErrors:            []string{"error1", "error2"},
					HasRetryPolicy:                     true,
					CronSchedule:                       "test-cron-schedule",
					EventStoreVersion:                  112,
					EventBranchToken:                   []byte("test-event-branch-token"),
					SignalCount:                        113,
					HistorySize:                        114,
					ClientLibraryVersion:               "test-client-library-version",
					ClientFeatureVersion:               "test-client-feature-version",
					ClientImpl:                         "test-client-impl",
					AutoResetPoints:                    []byte("test-auto-reset-points"),
					AutoResetPointsEncoding:            "json",
					SearchAttributes:                   map[string][]byte{"test-key": []byte("test-value")},
					Memo:                               map[string][]byte{"test-key": []byte("test-value")},
					VersionHistories:                   []byte("test-version-histories"),
					VersionHistoriesEncoding:           "json",
					FirstExecutionRunID:                serialization.MustParseUUID("ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f"),
					PartitionConfig:                    map[string]string{"test-key": "test-value"},
					Checksum:                           []byte("test-checksum"),
					ChecksumEncoding:                   "test-checksum-encoding",
				}, nil)
				parser.EXPECT().ActivityInfoFromBlob(gomock.Any(), gomock.Any()).Return(&serialization.ActivityInfo{
					Version:                  101,
					ScheduledEventBatchID:    102,
					ScheduledEvent:           []byte("test scheduled event"),
					ScheduledEventEncoding:   "json",
					ScheduledTimestamp:       time.Unix(11, 12),
					StartedID:                103,
					StartedEvent:             []byte("test started event"),
					StartedEventEncoding:     "json",
					StartedTimestamp:         time.Unix(13, 14),
					ActivityID:               "test-activity-id",
					RequestID:                "test-request-id",
					ScheduleToStartTimeout:   time.Duration(101),
					ScheduleToCloseTimeout:   time.Duration(102),
					StartToCloseTimeout:      time.Duration(103),
					HeartbeatTimeout:         time.Duration(104),
					CancelRequested:          true,
					CancelRequestID:          105,
					TimerTaskStatus:          105,
					Attempt:                  106,
					TaskList:                 "test-task-list",
					StartedIdentity:          "test-started-identity",
					HasRetryPolicy:           true,
					RetryInitialInterval:     time.Duration(107),
					RetryMaximumInterval:     time.Duration(108),
					RetryMaximumAttempts:     109,
					RetryExpirationTimestamp: time.Unix(15, 16),
					RetryBackoffCoefficient:  110,
					RetryNonRetryableErrors:  []string{"error1", "error2"},
					RetryLastFailureReason:   "test-retry-last-failure-reason",
					RetryLastWorkerIdentity:  "test-retry-last-worker-identity",
					RetryLastFailureDetails:  []byte("test-retry-last-failure-details"),
				}, nil)
				parser.EXPECT().TimerInfoFromBlob(gomock.Any(), gomock.Any()).Return(&serialization.TimerInfo{
					Version:         101,
					StartedID:       102,
					ExpiryTimestamp: time.Unix(11, 12),
					TaskID:          103,
				}, nil)
				parser.EXPECT().ChildExecutionInfoFromBlob(gomock.Any(), gomock.Any()).Return(&serialization.ChildExecutionInfo{
					Version:                101,
					InitiatedEventBatchID:  102,
					InitiatedEvent:         []byte("test initiated event"),
					InitiatedEventEncoding: "json",
					StartedID:              103,
					StartedWorkflowID:      "test-started-workflow-id",
					StartedRunID:           serialization.MustParseUUID("ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f"),
					CreateRequestID:        "test-create-request-id",
					StartedEvent:           []byte("test started event"),
					StartedEventEncoding:   "json",
					DomainID:               "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
					WorkflowTypeName:       "test-workflow-type",
					ParentClosePolicy:      101,
				}, nil)
				parser.EXPECT().RequestCancelInfoFromBlob(gomock.Any(), gomock.Any()).Return(&serialization.RequestCancelInfo{
					Version:               101,
					InitiatedEventBatchID: 102,
					CancelRequestID:       "test-cancel-request-id",
				}, nil)
				parser.EXPECT().SignalInfoFromBlob(gomock.Any(), gomock.Any()).Return(&serialization.SignalInfo{
					Version:               101,
					InitiatedEventBatchID: 102,
					Name:                  "test-signal-name",
					Input:                 []byte("test input"),
					Control:               []byte("test control"),
					RequestID:             "test-signal-request-id",
				}, nil)
			},
			want: &persistence.InternalGetWorkflowExecutionResponse{
				State: &persistence.InternalWorkflowMutableState{
					ExecutionInfo: &persistence.InternalWorkflowExecutionInfo{
						DomainID:                           "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
						WorkflowID:                         "test-workflow-id",
						RunID:                              "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
						ParentDomainID:                     "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
						ParentWorkflowID:                   "test-parent-workflow-id",
						ParentRunID:                        "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
						InitiatedID:                        101,
						CompletionEventBatchID:             11,
						CompletionEvent:                    persistence.NewDataBlob([]byte("test completion event"), common.EncodingTypeJSON),
						TaskList:                           "test-task-list",
						IsCron:                             true,
						WorkflowTypeName:                   "test-workflow-type",
						WorkflowTimeout:                    time.Duration(101),
						DecisionStartToCloseTimeout:        time.Duration(102),
						DecisionTimeout:                    time.Duration(104),
						ExecutionContext:                   []byte("test execution context"),
						State:                              persistence.WorkflowStateCompleted,
						CloseStatus:                        persistence.WorkflowCloseStatusCompleted,
						NextEventID:                        101,
						LastEventTaskID:                    12,
						LastFirstEventID:                   13,
						LastProcessedEvent:                 14,
						StartTimestamp:                     time.Unix(11, 12),
						LastUpdatedTimestamp:               time.Unix(13, 14),
						DecisionVersion:                    101,
						DecisionScheduleID:                 102,
						DecisionStartedID:                  103,
						DecisionAttempt:                    105,
						DecisionStartedTimestamp:           time.Unix(15, 16),
						DecisionScheduledTimestamp:         time.Unix(17, 18),
						CancelRequested:                    true,
						DecisionOriginalScheduledTimestamp: time.Unix(19, 20),
						CreateRequestID:                    "test-create-request-id",
						DecisionRequestID:                  "test-decision-request-id",
						CancelRequestID:                    "test-cancel-request-id",
						StickyTaskList:                     "test-sticky-task-list",
						StickyScheduleToStartTimeout:       time.Duration(106),
						HasRetryPolicy:                     true,
						CronSchedule:                       "test-cron-schedule",
						SignalCount:                        113,
						HistorySize:                        114,
						ClientLibraryVersion:               "test-client-library-version",
						ClientFeatureVersion:               "test-client-feature-version",
						ClientImpl:                         "test-client-impl",
						FirstExecutionRunID:                "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
						PartitionConfig:                    map[string]string{"test-key": "test-value"},
						AutoResetPoints:                    persistence.NewDataBlob([]byte("test-auto-reset-points"), common.EncodingTypeJSON),
						Attempt:                            107,
						InitialInterval:                    time.Duration(108),
						BackoffCoefficient:                 111,
						MaximumInterval:                    time.Duration(109),
						ExpirationTime:                     time.Unix(23, 24),
						MaximumAttempts:                    110,
						NonRetriableErrors:                 []string{"error1", "error2"},
						BranchToken:                        []byte("test-event-branch-token"),
						SearchAttributes:                   map[string][]byte{"test-key": []byte("test-value")},
						Memo:                               map[string][]byte{"test-key": []byte("test-value")},
						ExpirationInterval:                 time.Duration(111),
					},
					VersionHistories: persistence.NewDataBlob([]byte("test-version-histories"), common.EncodingTypeJSON),
					ReplicationState: &persistence.ReplicationState{
						StartVersion:     111,
						LastWriteVersion: 11,
						LastWriteEventID: 11,
					},
					ActivityInfos: map[int64]*persistence.InternalActivityInfo{
						101: {
							Version:                101,
							ScheduleID:             101,
							ScheduledEventBatchID:  102,
							ScheduledEvent:         persistence.NewDataBlob([]byte("test scheduled event"), common.EncodingTypeJSON),
							ScheduledTime:          time.Unix(11, 12),
							StartedID:              103,
							StartedTime:            time.Unix(13, 14),
							StartedEvent:           persistence.NewDataBlob([]byte("test started event"), common.EncodingTypeJSON),
							ActivityID:             "test-activity-id",
							RequestID:              "test-request-id",
							ScheduleToStartTimeout: time.Duration(101),
							ScheduleToCloseTimeout: time.Duration(102),
							StartToCloseTimeout:    time.Duration(103),
							HeartbeatTimeout:       time.Duration(104),
							CancelRequested:        true,
							CancelRequestID:        105,
							TimerTaskStatus:        105,
							Attempt:                106,
							TaskList:               "test-task-list",
							StartedIdentity:        "test-started-identity",
							HasRetryPolicy:         true,
							DomainID:               "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
							InitialInterval:        time.Duration(107),
							MaximumInterval:        time.Duration(108),
							MaximumAttempts:        109,
							ExpirationTime:         time.Unix(15, 16),
							BackoffCoefficient:     110,
							NonRetriableErrors:     []string{"error1", "error2"},
							LastFailureReason:      "test-retry-last-failure-reason",
							LastWorkerIdentity:     "test-retry-last-worker-identity",
							LastFailureDetails:     []byte("test-retry-last-failure-details"),
						},
					},
					TimerInfos: map[string]*persistence.TimerInfo{
						"101": {
							Version:    101,
							StartedID:  102,
							ExpiryTime: time.Unix(11, 12),
							TaskStatus: 103,
							TimerID:    "101",
						},
					},
					ChildExecutionInfos: map[int64]*persistence.InternalChildExecutionInfo{
						101: {
							Version:               101,
							InitiatedID:           101,
							InitiatedEvent:        persistence.NewDataBlob([]byte("test initiated event"), common.EncodingTypeJSON),
							InitiatedEventBatchID: 102,
							StartedID:             103,
							StartedEvent:          persistence.NewDataBlob([]byte("test started event"), common.EncodingTypeJSON),
							StartedWorkflowID:     "test-started-workflow-id",
							StartedRunID:          "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
							CreateRequestID:       "test-create-request-id",
							DomainID:              "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
							WorkflowTypeName:      "test-workflow-type",
							ParentClosePolicy:     101,
						},
					},
					RequestCancelInfos: map[int64]*persistence.RequestCancelInfo{
						101: {
							Version:               101,
							InitiatedID:           101,
							InitiatedEventBatchID: 102,
							CancelRequestID:       "test-cancel-request-id",
						},
					},
					SignalInfos: map[int64]*persistence.SignalInfo{
						101: {
							Version:               101,
							InitiatedID:           101,
							InitiatedEventBatchID: 102,
							SignalName:            "test-signal-name",
							Input:                 []byte("test input"),
							Control:               []byte("test control"),
							SignalRequestID:       "test-signal-request-id",
						},
					},
					SignalRequestedIDs: map[string]struct{}{
						"test-signal-id": {},
					},
					BufferedEvents: []*persistence.DataBlob{
						{
							Encoding: common.EncodingTypeThriftRW,
							Data:     []byte("test data"),
						},
					},
					ChecksumData: persistence.NewDataBlob([]byte("test-checksum"), "test-checksum-encoding"),
				},
			},
			wantErr: false,
		},
		{
			name: "Error - SelectFromExecutions failed",
			req: &persistence.InternalGetWorkflowExecutionRequest{
				DomainID: "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
				Execution: types.WorkflowExecution{
					WorkflowID: "test-workflow-id",
					RunID:      "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
				},
			},
			mockSetup: func(db *sqlplugin.MockDB, parser *serialization.MockParser) {
				db.EXPECT().SelectFromExecutions(gomock.Any(), gomock.Any()).Return(nil, errors.New("some random error"))
				db.EXPECT().SelectFromActivityInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromTimerInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromChildExecutionInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromRequestCancelInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalsRequestedSets(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromBufferedEvents(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().IsNotFoundError(gomock.Any()).Return(true).AnyTimes()
			},
			wantErr: true,
		},
		{
			name: "Error - SelectFromActivityInfoMaps failed",
			req: &persistence.InternalGetWorkflowExecutionRequest{
				DomainID: "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
				Execution: types.WorkflowExecution{
					WorkflowID: "test-workflow-id",
					RunID:      "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
				},
			},
			mockSetup: func(db *sqlplugin.MockDB, parser *serialization.MockParser) {
				db.EXPECT().SelectFromExecutions(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromActivityInfoMaps(gomock.Any(), gomock.Any()).Return(nil, errors.New("some random error"))
				db.EXPECT().SelectFromTimerInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromChildExecutionInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromRequestCancelInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalsRequestedSets(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromBufferedEvents(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().IsNotFoundError(gomock.Any()).Return(true).AnyTimes()
			},
			wantErr: true,
		},
		{
			name: "Error - SelectFromTimerInfoMaps failed",
			req: &persistence.InternalGetWorkflowExecutionRequest{
				DomainID: "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
				Execution: types.WorkflowExecution{
					WorkflowID: "test-workflow-id",
					RunID:      "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
				},
			},
			mockSetup: func(db *sqlplugin.MockDB, parser *serialization.MockParser) {
				db.EXPECT().SelectFromExecutions(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromActivityInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromTimerInfoMaps(gomock.Any(), gomock.Any()).Return(nil, errors.New("some random error"))
				db.EXPECT().SelectFromChildExecutionInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromRequestCancelInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalsRequestedSets(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromBufferedEvents(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().IsNotFoundError(gomock.Any()).Return(true).AnyTimes()
			},
			wantErr: true,
		},
		{
			name: "Error - SelectFromChildExecutionInfoMaps failed",
			req: &persistence.InternalGetWorkflowExecutionRequest{
				DomainID: "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
				Execution: types.WorkflowExecution{
					WorkflowID: "test-workflow-id",
					RunID:      "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
				},
			},
			mockSetup: func(db *sqlplugin.MockDB, parser *serialization.MockParser) {
				db.EXPECT().SelectFromExecutions(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromActivityInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromTimerInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromChildExecutionInfoMaps(gomock.Any(), gomock.Any()).Return(nil, errors.New("some random error"))
				db.EXPECT().SelectFromRequestCancelInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalsRequestedSets(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromBufferedEvents(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().IsNotFoundError(gomock.Any()).Return(true).AnyTimes()
			},
			wantErr: true,
		},
		{
			name: "Error - SelectFromRequestCancelInfoMaps failed",
			req: &persistence.InternalGetWorkflowExecutionRequest{
				DomainID: "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
				Execution: types.WorkflowExecution{
					WorkflowID: "test-workflow-id",
					RunID:      "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
				},
			},
			mockSetup: func(db *sqlplugin.MockDB, parser *serialization.MockParser) {
				db.EXPECT().SelectFromExecutions(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromActivityInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromTimerInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromChildExecutionInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromRequestCancelInfoMaps(gomock.Any(), gomock.Any()).Return(nil, errors.New("some random error"))
				db.EXPECT().SelectFromSignalInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalsRequestedSets(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromBufferedEvents(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().IsNotFoundError(gomock.Any()).Return(true).AnyTimes()
			},
			wantErr: true,
		},
		{
			name: "Error - SelectFromSignalInfoMaps failed",
			req: &persistence.InternalGetWorkflowExecutionRequest{
				DomainID: "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
				Execution: types.WorkflowExecution{
					WorkflowID: "test-workflow-id",
					RunID:      "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
				},
			},
			mockSetup: func(db *sqlplugin.MockDB, parser *serialization.MockParser) {
				db.EXPECT().SelectFromExecutions(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromActivityInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromTimerInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromChildExecutionInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromRequestCancelInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalInfoMaps(gomock.Any(), gomock.Any()).Return(nil, errors.New("some random error"))
				db.EXPECT().SelectFromSignalsRequestedSets(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromBufferedEvents(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().IsNotFoundError(gomock.Any()).Return(true).AnyTimes()
			},
			wantErr: true,
		},
		{
			name: "Error - SelectFromSignalsRequestedSets failed",
			req: &persistence.InternalGetWorkflowExecutionRequest{
				DomainID: "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
				Execution: types.WorkflowExecution{
					WorkflowID: "test-workflow-id",
					RunID:      "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
				},
			},
			mockSetup: func(db *sqlplugin.MockDB, parser *serialization.MockParser) {
				db.EXPECT().SelectFromExecutions(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromActivityInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromTimerInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromChildExecutionInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromRequestCancelInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalsRequestedSets(gomock.Any(), gomock.Any()).Return(nil, errors.New("some random error"))
				db.EXPECT().SelectFromBufferedEvents(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().IsNotFoundError(gomock.Any()).Return(true).AnyTimes()
			},
			wantErr: true,
		},
		{
			name: "Error - SelectFromBufferedEvents failed",
			req: &persistence.InternalGetWorkflowExecutionRequest{
				DomainID: "ff9c8a3f-0e4f-4d3e-a4d2-6f5f8f3f7d9d",
				Execution: types.WorkflowExecution{
					WorkflowID: "test-workflow-id",
					RunID:      "ee8d7b6e-876c-4b1e-9b6e-5e3e3c6b6b3f",
				},
			},
			mockSetup: func(db *sqlplugin.MockDB, parser *serialization.MockParser) {
				db.EXPECT().SelectFromExecutions(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromActivityInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromTimerInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromChildExecutionInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromRequestCancelInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalInfoMaps(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromSignalsRequestedSets(gomock.Any(), gomock.Any()).Return(nil, sql.ErrNoRows)
				db.EXPECT().SelectFromBufferedEvents(gomock.Any(), gomock.Any()).Return(nil, errors.New("some random error"))
				db.EXPECT().IsNotFoundError(gomock.Any()).Return(true).AnyTimes()
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := sqlplugin.NewMockDB(ctrl)
			parser := serialization.NewMockParser(ctrl)
			tc.mockSetup(db, parser)
			s := &sqlExecutionStore{
				shardID: 0,
				sqlStore: sqlStore{
					db:     db,
					logger: testlogger.New(t),
					parser: parser,
				},
			}

			resp, err := s.GetWorkflowExecution(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err, "Expected an error for test case")
			} else {
				assert.NoError(t, err, "Did not expect an error for test case")
				assert.Equal(t, tc.want, resp, "Response mismatch")
			}
		})
	}
}
