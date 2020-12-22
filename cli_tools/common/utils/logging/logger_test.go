//  Copyright 2020 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package logging

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pbtesting"
)

var (
	dateTime = time.Date(2009, 11, 10, 23, 10, 15, 0, time.UTC)
)

func Test_RedirectGlobalLogsToUser_CapturesStandardLog(t *testing.T) {
	// Ensure that previous log settings are overwritten.
	log.SetPrefix("old-prefix")
	log.SetFlags(log.LstdFlags)
	log.SetOutput(os.Stderr)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().User("hello world\n")
	RedirectGlobalLogsToUser(mockLogger)
	log.Print("hello world")
}

func Test_DefaultToolLogger_User_FormatsLikeDaisy(t *testing.T) {
	fromDaisy := (&daisy.LogEntry{
		LocalTimestamp: dateTime,
		WorkflowName:   "image-import",
		Message:        "msg",
	}).String()

	logger, written := setupTestLogger("[image-import]", dateTime)
	logger.User("msg")
	fromToolLogger := written.String()

	assert.Equal(t, fromDaisy, fromToolLogger)
}

func Test_DefaultToolLogger_User_Prefixes(t *testing.T) {
	// Using a colon after the prefix follows the pattern of Daisy's
	// standard logger. See daisy.LogEntry.String
	type test struct {
		name          string
		userPrefix    string
		expectWritten string
	}

	tests := []test{
		{
			name:          "add colon after prefix",
			userPrefix:    "import-image",
			expectWritten: "import-image: 2009-11-10T23:10:15Z message\n",
		},
		{
			name:          "don't add extra colon",
			userPrefix:    "import-image:",
			expectWritten: "import-image: 2009-11-10T23:10:15Z message\n",
		},
		{
			name:          "no colon when empty prefix",
			userPrefix:    "",
			expectWritten: "2009-11-10T23:10:15Z message\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, written := setupTestLogger(tt.userPrefix, dateTime)
			logger.User("message")
			actual := written.String()
			assert.Equal(t, tt.expectWritten, actual)
		})
	}
}

func Test_DefaultToolLogger_UserAndDebugInterleave(t *testing.T) {
	type test struct {
		name         string
		logCalls     func(writer Logger)
		expectedLogs string
	}

	tests := []test{
		{
			name:         "no logs when empty write",
			logCalls:     func(writer Logger) {},
			expectedLogs: "",
		},
		{
			name: "prepend specified user prefix",
			logCalls: func(writer Logger) {
				writer.User("hello user")
			},
			expectedLogs: "[image-import]: 2009-11-10T23:10:15Z hello user\n",
		},
		{
			name: "prepend a debug prefix",
			logCalls: func(writer Logger) {
				writer.Debug("hello debug")
			},
			expectedLogs: "[debug]: 2009-11-10T23:10:15Z hello debug\n",
		},
		{
			name: "maintain order when multiple writes",
			logCalls: func(writer Logger) {
				writer.Debug("hello debug1")
				writer.User("hello user1")
				writer.User("hello user2")
				writer.Debug("hello debug2")
			},
			expectedLogs: "[debug]: 2009-11-10T23:10:15Z hello debug1\n" +
				"[image-import]: 2009-11-10T23:10:15Z hello user1\n" +
				"[image-import]: 2009-11-10T23:10:15Z hello user2\n" +
				"[debug]: 2009-11-10T23:10:15Z hello debug2\n",
		},
		{
			name: "don't add extra newlines",
			logCalls: func(writer Logger) {
				writer.Debug("hello debug1\n")
				writer.User("hello user1\n")
			},
			expectedLogs: "[debug]: 2009-11-10T23:10:15Z hello debug1\n" +
				"[image-import]: 2009-11-10T23:10:15Z hello user1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, written := setupTestLogger("[image-import]", dateTime)
			tt.logCalls(logger)
			assert.Equal(t, tt.expectedLogs, written.String())
			if tt.expectedLogs == "" {
				assert.Nil(t, logger.ReadOutputInfo().SerialOutputs,
					"Only append SerialLog when there were logs written.")
			} else {
				assert.Equal(t, []string{tt.expectedLogs}, logger.ReadOutputInfo().SerialOutputs,
					"Create a single SerialOutput member containing all debug and user logs.")
			}
		})
	}
}

func Test_DefaultToolLogger_Metric_MergesNestedStruct(t *testing.T) {
	logger := NewToolLogger("[user]")
	logger.Metric(&pb.OutputInfo{IsUefiDetected: true})
	logger.Metric(&pb.OutputInfo{InspectionResults: &pb.InspectionResults{
		ErrorWhen: pb.InspectionResults_INTERPRETING_INSPECTION_RESULTS}})
	expected := &pb.OutputInfo{
		IsUefiDetected: true,
		InspectionResults: &pb.InspectionResults{
			ErrorWhen: pb.InspectionResults_INTERPRETING_INSPECTION_RESULTS,
		},
	}
	pbtesting.AssertEqual(t, expected, logger.ReadOutputInfo())
}

func Test_DefaultToolLogger_Metric_AppendsSlices(t *testing.T) {
	logger := NewToolLogger("[user]")
	logger.Metric(&pb.OutputInfo{InflationTimeMs: []int64{30}})
	logger.Metric(&pb.OutputInfo{InflationTimeMs: []int64{40}})
	expected := &pb.OutputInfo{InflationTimeMs: []int64{30, 40}}

	pbtesting.AssertEqual(t, expected, logger.ReadOutputInfo())
}

func Test_DefaultToolLogger_Metric_DoesntClobberSingleValuesWithDefaultValues(t *testing.T) {
	logger := NewToolLogger("[user]")
	logger.Metric(&pb.OutputInfo{IsUefiDetected: true})
	logger.Metric(&pb.OutputInfo{InflationType: "api"})
	expected := &pb.OutputInfo{IsUefiDetected: true, InflationType: "api"}
	pbtesting.AssertEqual(t, expected, logger.ReadOutputInfo())
}

func TestToolLogger_ReadOutputInfo_ClearsState(t *testing.T) {
	logger, _ := setupTestLogger("[user]", dateTime)

	// 1. Use the logger; on first read, OutputInfo should contain all buffered information.
	logger.Metric(&pb.OutputInfo{IsUefiDetected: true})
	logger.User("hi")
	logger.Trace("trace")
	firstRead := logger.ReadOutputInfo()
	pbtesting.AssertEqual(t, &pb.OutputInfo{
		IsUefiDetected: true,
		SerialOutputs:  []string{"[user]: 2009-11-10T23:10:15Z hi\n", "trace"},
	}, firstRead)

	// 2. On second read, the buffers should be cleared.
	secondRead := logger.ReadOutputInfo()
	pbtesting.AssertEqual(t, &pb.OutputInfo{}, secondRead)

	// 3. Use the logger again and verify that the new information is kept.
	logger.Metric(&pb.OutputInfo{InflationType: "daisy"})
	logger.User("hi 1")
	logger.Trace("trace 1")

	thirdRead := logger.ReadOutputInfo()
	pbtesting.AssertEqual(t, &pb.OutputInfo{
		InflationType: "daisy",
		SerialOutputs: []string{"[user]: 2009-11-10T23:10:15Z hi 1\n", "trace 1"},
	}, thirdRead)
}

func setupTestLogger(userPrefix string, now time.Time) (ToolLogger, fmt.Stringer) {
	var buffer bytes.Buffer
	realLogger := NewToolLogger(userPrefix).(*defaultToolLogger)
	realLogger.timeProvider = func() time.Time {
		return now
	}
	realLogger.output.SetOutput(&buffer)
	return realLogger, &buffer
}
