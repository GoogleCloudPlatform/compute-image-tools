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

package logging

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/google/uuid"
	"github.com/minio/highwayhash"
)

var (
	httpClient            httpClientInterface = &http.Client{Timeout: 5 * time.Second}
	serverURL                                 = serverURLProd
	serverLogEnabled                          = false
	logMutex                                  = sync.Mutex{}
	nextRequestWaitMillis int64
)

// constants used by logging
const (
	serverURLProd        = "https://firebaselogging-pa.googleapis.com/v1/firelog/legacy/log"
	key                  = "Ix3TgeJBsH2TH8FUg9_ukW6I6U0t1zOmCySazIA"
	ImageImportAction    = "ImageImport"
	ImageExportAction    = "ImageExport"
	InstanceImportAction = "InstanceImport"
	targetSizeGb         = "target-size-gb"
	sourceSizeGb         = "source-size-gb"
	statusStart          = "Start"
	statusSuccess        = "Success"
	statusFailure        = "Failure"
)

type logResult string

const (
	failedOnCreateRequest          logResult = "FailedOnCreateRequest"
	failedOnCreateRequestJSON      logResult = "FailedOnCreateRequestJSON"
	failedToParseResponse          logResult = "FailedToParseResponse"
	failedAfterRetry               logResult = "FailedAfterRetry"
	failedOnUndefinedResponse      logResult = "FailedOnUndefinedResponse"
	failedOnMissingResponseDetails logResult = "FailedOnMissingResponseDetails"
	serverLogDisabled              logResult = "ServerLogDisabled"
)

type httpClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

func reverse(str string) string {
	l := len(str)
	revStr := make([]byte, l)
	for i := 0; i <= l/2; i++ {
		revStr[i], revStr[l-1-i] = str[l-1-i], str[i]
	}
	return string(revStr)
}

// FirelogLoggerInterface is server logger abstraction
type FirelogLoggerInterface interface {
	logStart()
	logSuccess(w *daisy.Workflow)
	logFailure(err error, w *daisy.Workflow)
}

// FirelogLogger is responsible for logging to firelog server
type FirelogLogger struct {
	ServerURL string
	ID        string
	Action    string
	Params    InputParams
}

// NewFirelogLogger creates a new server logger
func NewFirelogLogger(action string, params InputParams) *FirelogLogger {
	return &FirelogLogger{
		ServerURL: serverURL,
		ID:        uuid.New().String(),
		Action:    action,
		Params:    params,
	}
}

// logStart logs a "start" info to server
func (l *FirelogLogger) logStart() (*ComputeImageToolsLogExtension, logResult) {
	logEvent := &ComputeImageToolsLogExtension{
		ID:           l.ID,
		CloudBuildID: os.Getenv(daisyutils.BuildIDOSEnv),
		ToolAction:   l.Action,
		Status:       statusStart,
		InputParams:  &l.Params,
	}

	return logEvent, l.sendLogToServer(logEvent)
}

// logSuccess logs a "success" info to server
func (l *FirelogLogger) logSuccess(w *daisy.Workflow) (*ComputeImageToolsLogExtension, logResult) {
	logEvent := &ComputeImageToolsLogExtension{
		ID:           l.ID,
		CloudBuildID: os.Getenv(daisyutils.BuildIDOSEnv),
		ToolAction:   l.Action,
		Status:       statusSuccess,
		InputParams:  &l.Params,
		OutputInfo:   getOutputInfo(w, nil),
	}

	return logEvent, l.sendLogToServer(logEvent)
}

// logFailure logs a "failure" info to server
func (l *FirelogLogger) logFailure(err error, w *daisy.Workflow) (*ComputeImageToolsLogExtension, logResult) {
	logEvent := &ComputeImageToolsLogExtension{
		ID:           l.ID,
		CloudBuildID: os.Getenv(daisyutils.BuildIDOSEnv),
		ToolAction:   l.Action,
		Status:       statusFailure,
		InputParams:  &l.Params,
		OutputInfo:   getOutputInfo(w, err),
	}

	return logEvent, l.sendLogToServer(logEvent)
}

func getFailureReason(err error) string {
	return daisyutils.RemovePrivacyLogTag(err.Error())
}

func getAnonymizedFailureReason(err error) string {
	derr := daisy.ToDError(err)
	if derr == nil {
		return ""
	}
	anonymizedErrs := []string{}
	for _, m := range derr.AnonymizedErrs() {
		anonymizedErrs = append(anonymizedErrs, daisyutils.RemovePrivacyLogInfo(m))
	}
	return strings.Join(anonymizedErrs, "\n")
}

func getOutputInfo(w *daisy.Workflow, err error) *OutputInfo {
	o := OutputInfo{
		TargetSizeGb: getInt64Value(w.GetSerialConsoleOutputValue(targetSizeGb)),
		SourceSizeGb: getInt64Value(w.GetSerialConsoleOutputValue(sourceSizeGb)),
	}

	if err != nil {
		o.FailureMessage = getFailureReason(err)
		o.FailureMessageWithoutPrivacyInfo = getAnonymizedFailureReason(err)
	}

	return &o
}

func getInt64Value(s string) int64 {
	i, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0
	}
	return i
}

func (l *FirelogLogger) runWithServerLogging(function func() (*daisy.Workflow, error)) *ComputeImageToolsLogExtension {
	var event *ComputeImageToolsLogExtension

	// Send log asynchronously. No need to interrupt the main flow when failed to send log, just
	// keep moving.
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		l.logStart()
	}()

	if w, err := function(); err != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			event, _ = l.logFailure(err, w)
			log.Println(event.OutputInfo.FailureMessage)
		}()
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			event, _ = l.logSuccess(w)
		}()
	}

	wg.Wait()
	return event
}

// RunWithServerLogging runs the function with server logging
func RunWithServerLogging(action string, params InputParams, function func() (*daisy.Workflow, error)) {
	l := NewFirelogLogger(action, params)
	l.runWithServerLogging(function)
}

func (l *FirelogLogger) sendLogToServer(logEvent *ComputeImageToolsLogExtension) logResult {
	return l.sendLogToServerWithRetry(logEvent, 3)
}

func (l *FirelogLogger) sendLogToServerWithRetry(logEvent *ComputeImageToolsLogExtension, maxRetry int) logResult {
	logMutex.Lock()
	defer logMutex.Unlock()

	for i := 0; i < maxRetry; i++ {
		// Before sending a new request, wait for a while if server asked to do so
		if nextRequestWaitMillis > 0 {
			nextRequestWaitMillis = 0
			time.Sleep(time.Duration(nextRequestWaitMillis) * time.Millisecond)
		}

		logRequestJSON, err := constructLogRequest(logEvent)
		fmt.Println(string(logRequestJSON))
		if err != nil {
			fmt.Println("Failed to log to server: failed to prepare json log data.")
			return failedOnCreateRequestJSON
		}

		if !serverLogEnabled {
			return serverLogDisabled
		}

		req, err := http.NewRequest("POST", l.ServerURL, bytes.NewBuffer(logRequestJSON))
		if err != nil {
			fmt.Println("Failed to create log request: ", err)
			return failedOnCreateRequest
		}

		req.Header.Set("Content-type", "application/json")
		req.Header.Set("X-Goog-Api-Key", reverse(key))
		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Println("Failed to log to server: ", err)
			continue
		}

		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		var lr LogResponse
		if err = json.Unmarshal(body, &lr); err != nil {
			fmt.Println("Failed to parse log response: ", err, "\nResponse: ", string(body))
			return failedToParseResponse
		}

		// Honor "NextRequestWaitMillis" from server for traffic control. However, wait no more than 5s to prevent a long
		// stuck
		if lr.NextRequestWaitMillis > 0 {
			nextRequestWaitMillis = lr.NextRequestWaitMillis
			if nextRequestWaitMillis > 5000 {
				nextRequestWaitMillis = 5000
			}
		}

		// Honor "ResponseAction" from server to control retrying
		if len(lr.LogResponseDetails) > 0 {
			action := lr.LogResponseDetails[0].ResponseAction
			if action == DeleteRequest || action == ResponseActionUnknown {
				// Log success or unknown status, just return
				return logResult(action)
			} else if action == RetryRequestLater {
				// Retry as server asked
				continue
			}
			// Return if client failed to receive a defined response action
			fmt.Println("Failed to log to server: undefined response action: ", action)
			return failedOnUndefinedResponse
		}

		// Return if client failed to receive response details from server
		fmt.Println("Failed to log to server: missing response details")
		return failedOnMissingResponseDetails
	}

	fmt.Println("Failed to log to server after retrying")
	return failedAfterRetry
}

// Hash a given string for obfuscation
func Hash(s string) string {
	hash, _ := highwayhash.New([]byte("compute-image-tools-obfuscate-01"))
	return hex.EncodeToString(hash.Sum([]byte(s)))
}
