//  Copyright 2019 Google Inc. All Rights Reserved.
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

package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

var (
	logger                    *Logger
	serverLogEnabledPrevValue bool
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	serverLogEnabledPrevValue = serverLogEnabled
	serverLogEnabled = true
}

func shutdown() {
	serverLogEnabled = serverLogEnabledPrevValue
}

func TestLogStart(t *testing.T) {
	prepareTestLogger(t, nil, buildLogResponses(deleteRequest))

	e, r := logger.logStart()

	if r != logResult(deleteRequest) {
		t.Errorf("Unexpected logResult: %v, expect: %v", r, deleteRequest)
	}
	if e.Status != statusStart {
		t.Errorf("Unexpected Status %v, expect: %v", e.Status, statusStart)
	}
}

func TestLogSuccess(t *testing.T) {
	prepareTestLogger(t, nil, buildLogResponses(deleteRequest))
	time.Sleep(20 * time.Millisecond)

	w := literalLoggable{
		strings: map[string]string{
			importFileFormat: "vmdk",
		},
		int64s: map[string][]int64{
			targetSizeGb: {5},
			sourceSizeGb: {3, 2, 1},
		},
		traceLogs: []string{
			"serial-log1", "serial-log2",
		},
		inspectionResults: &pb.InspectionResults{
			OsCount:      1,
			UefiBootable: true,
			OsRelease: &pb.OsRelease{
				Architecture: pb.Architecture_X64,
				CliFormatted: "ubuntu-2004",
				Distro:       "ubuntu",
				MajorVersion: "20",
				MinorVersion: "04",
				DistroId:     pb.Distro_UBUNTU,
			},
		},
	}

	e, r := logger.logSuccess(w)

	if r != logResult(deleteRequest) {
		t.Errorf("Unexpected logResult: %v, expect: %v", r, deleteRequest)
	}
	if e.Status != statusSuccess {
		t.Errorf("Unexpected Status %v, expect: %v", e.Status, statusSuccess)
	}

	expectedInspectionResults := pb.InspectionResults{
		OsCount:      1,
		UefiBootable: true,
		OsRelease: &pb.OsRelease{
			Architecture: pb.Architecture_X64,
			CliFormatted: "ubuntu-2004",
			Distro:       "ubuntu",
			MajorVersion: "20",
			MinorVersion: "04",
			DistroId:     pb.Distro_UBUNTU,
		},
	}

	expectedOutputInfo := OutputInfo{
		SourcesSizeGb:     []int64{3, 2, 1},
		TargetsSizeGb:     []int64{5},
		ImportFileFormat:  "vmdk",
		SerialOutputs:     nil, // don't send serial output on success
		InspectionResults: &expectedInspectionResults,
	}

	assert.Equal(t, &expectedInspectionResults,
		e.InputParams.ImageImportParams.InspectionResults)
	assert.Equal(t, expectedOutputInfo, *e.OutputInfo)

	if e.ElapsedTimeMs < 20 {
		t.Errorf("Unexpected ElapsedTimeMs %v < %v", e.ElapsedTimeMs, 20)
	}
}

func TestLogFailure(t *testing.T) {
	prepareTestLogger(t, nil, buildLogResponses(deleteRequest))
	time.Sleep(20 * time.Millisecond)

	w := literalLoggable{
		strings: map[string]string{
			importFileFormat: "vmdk",
		},
		int64s: map[string][]int64{
			targetSizeGb: {5},
			sourceSizeGb: {3, 2, 1},
		},
		traceLogs: []string{
			"serial-log1", "serial-log2",
		},
	}
	e, r := logger.logFailure(fmt.Errorf("error - [Privacy-> sensitive <-Privacy]"), w)

	expected := OutputInfo{
		SourcesSizeGb:                    []int64{3, 2, 1},
		TargetsSizeGb:                    []int64{5},
		FailureMessage:                   "error -  sensitive ",
		FailureMessageWithoutPrivacyInfo: "error - ",
		ImportFileFormat:                 "vmdk",
		SerialOutputs:                    []string{"serial-log1", "serial-log2"},
	}
	assert.Equal(t, expected, *e.OutputInfo)

	if r != logResult(deleteRequest) {
		t.Errorf("Unexpected logResult: %v, expect: %v", r, deleteRequest)
	}
	if e.Status != statusFailure {
		t.Errorf("Unexpected Status %v, expect: %v", e.Status, statusFailure)
	}
	if e.ElapsedTimeMs < 20 {
		t.Errorf("Unexpected ElapsedTimeMs %v < %v", e.ElapsedTimeMs, 20)
	}
}

func TestRunWithServerLoggingSuccess(t *testing.T) {
	prepareTestLogger(t, nil, buildLogResponses(deleteRequest, deleteRequest))

	logExtension, _ := logger.runWithServerLogging(
		func() (Loggable, error) {
			return literalLoggable{}, nil
		}, nil)
	if logExtension.Status != statusSuccess {
		t.Errorf("Unexpected Status: %v, expect: %v", logExtension.Status, statusSuccess)
	}
}

func TestRunWithServerLoggingFailed(t *testing.T) {
	prepareTestLogger(t, nil, buildLogResponses(deleteRequest, deleteRequest))

	logExtension, _ := logger.runWithServerLogging(
		func() (Loggable, error) {
			return literalLoggable{}, fmt.Errorf("test msg - failure by purpose")
		}, nil)
	if logExtension.Status != statusFailure {
		t.Errorf("Unexpected Status: %v, expect: %v", logExtension.Status, statusFailure)
	}
}

func TestRunWithServerLogging_LogsFailure_WhenApplicationPanics(t *testing.T) {
	prepareTestLogger(t, nil, buildLogResponses(deleteRequest, deleteRequest))

	panicMessage := "client code panic"
	logExtension, err := logger.runWithServerLogging(
		func() (Loggable, error) {
			panic(panicMessage)
		}, nil)
	assert.EqualError(t, err, "A fatal error has occurred. Please submit an issue at https://github.com/GoogleCloudPlatform/compute-image-tools/issues")
	assert.Equal(t, statusFailure, logExtension.Status)

	// Include stacktrace and panic message in serial outputs.
	assertContainsSubstring(t, "stacktrace", logExtension.OutputInfo.SerialOutputs)
	assertContainsSubstring(t, panicMessage, logExtension.OutputInfo.SerialOutputs)
}

func assertContainsSubstring(t *testing.T, sub string, arr []string) {
	for _, s := range arr {
		if strings.Contains(s, sub) {
			return
		}
	}
	t.Errorf("Substring %q not found in %v", sub, arr)
}

func TestRunWithServerLoggingSuccessWithUpdatedProject(t *testing.T) {
	prepareTestLogger(t, nil, buildLogResponses(deleteRequest, deleteRequest))

	project := "dummy-project"
	logExtension, _ := logger.runWithServerLogging(
		func() (Loggable, error) {
			return literalLoggable{}, nil
		}, &project)
	if logExtension.Status != statusSuccess {
		t.Errorf("Unexpected Status: %v, expect: %v", logExtension.Status, statusSuccess)
	}
	if logExtension.InputParams.ImageImportParams.Project != project {
		t.Errorf("Unexpected Updated Project: %v, expect: %v",
			logExtension.InputParams.ImageImportParams.Project, project)
	}
}

func TestSendLogToServerSuccess(t *testing.T) {
	testSendLogToServerWithResponses(t, logResult(deleteRequest), buildLogResponses(deleteRequest))
}

func TestSendLogToServerResponseActionUnknown(t *testing.T) {
	testSendLogToServerWithResponses(t, logResult(responseActionUnknown), buildLogResponses(responseActionUnknown))
}

func TestSendLogToServerSuccessAfterRetry(t *testing.T) {
	testSendLogToServerWithResponses(t, logResult(deleteRequest), buildLogResponses(retryRequestLater, retryRequestLater, deleteRequest))
}

func TestSendLogToServerFailedOnCreateRequest(t *testing.T) {
	backupServerURL := serverURL
	serverURL = "%%bad-url"
	defer func() { serverURL = backupServerURL }()
	testSendLogToServerWithResponses(t, failedOnCreateRequest, nil)
}

func TestSendLogToServerFailedOnCreateRequestJSON(t *testing.T) {
	prepareTestLogger(t, nil, nil)
	r := logger.sendLogToServer(nil)
	if r != logResult(failedOnCreateRequestJSON) {
		t.Errorf("Unexpected Status: %v, expect: %v", r, failedOnCreateRequestJSON)
	}
}

func TestSendLogToServerLogDisabled(t *testing.T) {
	serverLogEnabled = false
	defer func() { serverLogEnabled = true }()

	testSendLogToServerWithResponses(t, serverLogDisabled, buildLogResponses(deleteRequest))
}

func TestSendLogToServerFailedToParseResponse(t *testing.T) {
	prepareTestLoggerWithJSONLogResponse(t, nil, []string{"bad-json"})
	r := logger.sendLogToServer(buildComputeImageToolsLogExtension())
	if r != logResult(failedToParseResponse) {
		t.Errorf("Unexpected Status: %v, expect: %v", r, failedToParseResponse)
	}
}

func TestSendLogToServerFailedOnUndefinedResponse(t *testing.T) {
	testSendLogToServerWithResponses(t, failedOnUndefinedResponse, buildLogResponses("UndefinedResponseForTest"))
}

func TestSendLogToServerFailedOnMissingResponseDetails(t *testing.T) {
	testSendLogToServerWithResponses(t, failedOnMissingResponseDetails, []logResponse{{}})
}

func TestSendLogToServerFailedAfterRetry(t *testing.T) {
	testSendLogToServerWithResponses(t, failedAfterRetry, buildLogResponses(retryRequestLater, retryRequestLater, retryRequestLater, deleteRequest))
}

func testSendLogToServerWithResponses(t *testing.T, expectedLogResult logResult, resps []logResponse) {
	prepareTestLogger(t, nil, resps)
	r := logger.sendLogToServer(buildComputeImageToolsLogExtension())
	if r != logResult(expectedLogResult) {
		t.Errorf("Unexpected Status: %v, expect: %v", r, expectedLogResult)
	}
}

func TestRemoveNewLinesFromMultilineErrorSingleLine(t *testing.T) {
	testRemoveNewLinesFromMultilineError(t, "Single line", "Single line")
}

func TestRemoveNewLinesFromMultilineErrorMultiLine(t *testing.T) {
	testRemoveNewLinesFromMultilineError(t, "Header:\nFirst line\nSecond line", "Header: First line; Second line")
}

func TestRemoveNewLinesFromMultilineErrorEmptyString(t *testing.T) {
	testRemoveNewLinesFromMultilineError(t, "", "")
}

func testRemoveNewLinesFromMultilineError(t *testing.T, input string, expected string) {
	result := removeNewLinesFromMultilineError(input)
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func prepareTestLogger(t *testing.T, err error, resps []logResponse) {
	var lrs []string
	for _, resp := range resps {
		bytes, _ := json.Marshal(resp)
		lrs = append(lrs, string(bytes))
	}

	prepareTestLoggerWithJSONLogResponse(t, err, lrs)
}

func prepareTestLoggerWithJSONLogResponse(t *testing.T, err error, lrs []string) {
	httpClient = &MockHTTPClient{
		t:   t,
		lrs: lrs,
		err: err,
	}

	logger = NewLoggingServiceLogger(ImageImportAction, InputParams{
		ImageImportParams: &ImageImportParams{
			CommonParams: &CommonParams{
				ClientID: "test-client",
			},
		},
	})
}

func buildComputeImageToolsLogExtension() *ComputeImageToolsLogExtension {
	logExtension := &ComputeImageToolsLogExtension{
		ID:           "dummy-id",
		CloudBuildID: "dummy-cloud-build-id",
		ToolAction:   ImageImportAction,
		Status:       statusStart,
		InputParams: &InputParams{
			ImageImportParams: &ImageImportParams{
				CommonParams: &CommonParams{
					Project:           "dummy-project",
					ObfuscatedProject: Hash("dummy-project"),
				},
				SourceImage: "dummy-image",
			},
		},
	}
	return logExtension
}

func buildLogResponses(actions ...responseAction) []logResponse {
	var lrs []logResponse
	for _, a := range actions {
		lrs = append(lrs, logResponse{
			NextRequestWaitMillis: 100,
			LogResponseDetails: []logResponseDetails{
				{
					ResponseAction: a,
				},
			},
		})
	}
	return lrs
}

type MockHTTPClient struct {
	t     *testing.T
	lrs   []string
	index int
	err   error
}

func (c *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c.index >= len(c.lrs) {
		c.t.Fatal("Exceeds time of prepared mock calls")
	}

	bodyReader := io.NopCloser(strings.NewReader(c.lrs[c.index]))
	resp := http.Response{
		Body: bodyReader,
	}

	c.index++

	return &resp, c.err
}
