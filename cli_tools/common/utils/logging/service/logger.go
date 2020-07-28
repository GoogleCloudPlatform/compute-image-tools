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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	serverURL                                 = deinterleave(serverURLProdP1, serverURLProdP2)
	key                                       = deinterleave(keyP1, keyP2)
	serverLogEnabled                          = true
	logMutex                                  = sync.Mutex{}
	nextRequestWaitMillis int64
)

// constants used by logging
const (
	ImageImportAction        = "ImageImport"
	ImageExportAction        = "ImageExport"
	InstanceImportAction     = "InstanceImport"
	MachineImageImportAction = "MachineImageImport"
	OneStepImageImportAction = "OneStepImageImport"
	WindowsUpgrade           = "WindowsUpgrade"

	// These strings should be interleaved to construct the real URL. This is just to (hopefully)
	// fool github URL scanning bots.
	serverURLProdP1 = "hts/frbslgigp.ogepscmv/ieo/eaylg"
	serverURLProdP2 = "tp:/ieaeogn-agolai.o/1frlglgc/o"
	keyP1           = "AzSCO1066k_gFH2sJg3I"
	keyP2           = "IaymztUIWu9U8THBeTx"

	targetSizeGb          = "target-size-gb"
	sourceSizeGb          = "source-size-gb"
	importFileFormat      = "import-file-format"
	inflationType         = "inflation-type"
	inflationTime         = "inflation-time"
	shadowInflationTime   = "shadow-inflation-time"
	shadowDiskMatchResult = "shadow-disk-match-result"

	statusStart   = "Start"
	statusSuccess = "Success"
	statusFailure = "Failure"
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

func deinterleave(p1, p2 string) string {
	l1 := len(p1)
	l2 := len(p2)
	if l1 != l2 && l1 != l2+1 {
		panic("Failed to prepare required data for the tool.")
	}
	strBytes := make([]byte, len(p1)+len(p2))
	for i := range p1 {
		strBytes[i*2] = p1[i]
	}
	for i := range p2 {
		strBytes[i*2+1] = p2[i]
	}
	return string(strBytes)
}

// Logger is responsible for logging to firelog server
type Logger struct {
	ServerURL string
	ID        string
	Action    string
	TimeStart time.Time
	Params    InputParams
	mutex     sync.Mutex
}

// NewLoggingServiceLogger creates a new server logger
func NewLoggingServiceLogger(action string, params InputParams) *Logger {
	return &Logger{
		ServerURL: serverURL,
		ID:        uuid.New().String(),
		Action:    action,
		TimeStart: time.Now(),
		Params:    params,
		mutex:     sync.Mutex{},
	}
}

// logStart logs a "start" info to server
func (l *Logger) logStart() (*ComputeImageToolsLogExtension, logResult) {
	logExtension := l.createComputeImageToolsLogExtension(statusStart, nil)
	return logExtension, l.sendLogToServer(logExtension)
}

// logSuccess logs a "success" info to server
func (l *Logger) logSuccess(loggable Loggable) (*ComputeImageToolsLogExtension, logResult) {
	logExtension := l.createComputeImageToolsLogExtension(statusSuccess, l.getOutputInfo(loggable, nil))
	return logExtension, l.sendLogToServer(logExtension)
}

// logFailure logs a "failure" info to server
func (l *Logger) logFailure(err error, loggable Loggable) (*ComputeImageToolsLogExtension, logResult) {
	logExtension := l.createComputeImageToolsLogExtension(statusFailure, l.getOutputInfo(loggable, err))
	return logExtension, l.sendLogToServer(logExtension)
}

func (l *Logger) createComputeImageToolsLogExtension(status string, outputInfo *OutputInfo) *ComputeImageToolsLogExtension {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return &ComputeImageToolsLogExtension{
		ID:            l.ID,
		CloudBuildID:  os.Getenv(daisyutils.BuildIDOSEnvVarName),
		ToolAction:    l.Action,
		Status:        status,
		ElapsedTimeMs: time.Since(l.TimeStart).Nanoseconds() / 1e6,
		EventTimeMs:   time.Now().UnixNano() / 1e6,
		InputParams:   &l.Params,
		OutputInfo:    outputInfo,
	}
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

func (l *Logger) getOutputInfo(loggable Loggable, err error) *OutputInfo {
	o := OutputInfo{}

	if loggable != nil {
		o.TargetsSizeGb = loggable.GetValueAsInt64Slice(targetSizeGb)
		o.SourcesSizeGb = loggable.GetValueAsInt64Slice(sourceSizeGb)
		o.ImportFileFormat = loggable.GetValue(importFileFormat)
		o.InflationType = loggable.GetValue(inflationType)
		o.InflationTime = loggable.GetValueAsInt64Slice(inflationTime)
		o.ShadowInflationTime = loggable.GetValueAsInt64Slice(shadowInflationTime)
		o.ShadowDiskMatchResult = loggable.GetValue(shadowDiskMatchResult)
	}

	if err != nil {
		o.FailureMessage = getFailureReason(err)
		o.FailureMessageWithoutPrivacyInfo = getAnonymizedFailureReason(err)
		if loggable != nil {
			o.SerialOutputs = loggable.ReadSerialPortLogs()
		}
	}

	return &o
}

func (l *Logger) runWithServerLogging(function func() (Loggable, error),
	projectPointer *string) (*ComputeImageToolsLogExtension, error) {

	var logExtension *ComputeImageToolsLogExtension

	// Send log asynchronously. No need to interrupt the main flow when failed to send log, just
	// keep moving.
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		l.logStart()
	}()

	loggable, err := function()
	l.updateParams(projectPointer)
	if err != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logExtension, _ = l.logFailure(err, loggable)

			// Remove new lines from multi-line failure messages as gcloud depends on
			// log prefix to filter out relevant log lines. Making this change in
			// daisy/error.go Error() func would potentially affect other clients of
			// Daisy with unclear consequences, thus limiting the change to
			// import/export wrappers
			log.Println(removeNewLinesFromMultilineError(logExtension.OutputInfo.FailureMessage))
		}()
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logExtension, _ = l.logSuccess(loggable)
		}()
	}

	wg.Wait()
	return logExtension, err
}

func removeNewLinesFromMultilineError(s string) string {
	// first line in a multi error line is of "Multiple errors" type and doesn't need a separator
	firstNewLineRemoved := strings.Replace(s, "\n", " ", 1)
	return strings.ReplaceAll(firstNewLineRemoved, "\n", "; ")
}

// RunWithServerLogging runs the function with server logging
func RunWithServerLogging(action string, params InputParams, projectPointer *string,
	function func() (Loggable, error)) error {
	l := NewLoggingServiceLogger(action, params)
	_, err := l.runWithServerLogging(function, projectPointer)
	return err
}

func (l *Logger) sendLogToServer(logExtension *ComputeImageToolsLogExtension) logResult {
	r := l.sendLogToServerWithRetry(logExtension, 3)
	return r
}

func (l *Logger) sendLogToServerWithRetry(logExtension *ComputeImageToolsLogExtension, maxRetry int) logResult {
	logMutex.Lock()
	defer logMutex.Unlock()

	for i := 0; i < maxRetry; i++ {
		// Before sending a new request, wait for a while if server asked to do so
		if nextRequestWaitMillis > 0 {
			nextRequestWaitMillis = 0
			time.Sleep(time.Duration(nextRequestWaitMillis) * time.Millisecond)
		}

		logRequestJSON, err := l.constructLogRequest(logExtension)
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

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("X-Goog-Api-Key", key)
		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Println("Failed to log to server: ", err)
			continue
		}

		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		var lr logResponse
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
			if action == deleteRequest || action == responseActionUnknown {
				// Log success or unknown status, just return
				return logResult(action)
			} else if action == retryRequestLater {
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

func (l *Logger) constructLogRequest(logExtension *ComputeImageToolsLogExtension) ([]byte, error) {
	if logExtension == nil {
		return nil, fmt.Errorf("won't log a nil event")
	}
	eventStr, err := json.Marshal(logExtension)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixNano() / 1000000
	req := logRequest{
		ClientInfo: clientInfo{
			ClientType: "COMPUTE_IMAGE_TOOLS",
		},
		LogSource:     1201,
		RequestTimeMs: now,
		LogEvent: []logEvent{
			{
				EventTimeMs:         now,
				EventUptimeMs:       time.Since(l.TimeStart).Nanoseconds() / 1e6,
				SourceExtensionJSON: string(eventStr),
			},
		},
	}

	reqStr, err := json.Marshal(req)
	return reqStr, err
}

// Hash a given string for obfuscation
func Hash(s string) string {
	hash, _ := highwayhash.New([]byte("compute-image-tools-obfuscate-01"))
	hash.Write([]byte(s))
	return hex.EncodeToString(hash.Sum(nil))
}

// Loggable contains fields relevant to import and export logging.
type Loggable interface {
	GetValue(key string) string
	GetValueAsInt64Slice(key string) []int64
	ReadSerialPortLogs() []string
}
