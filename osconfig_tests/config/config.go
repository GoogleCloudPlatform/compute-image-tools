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

package config

import (
	"fmt"
	"time"
)

var (
	// TODO: allow this to be configurable through flag to test against staging
	prodEndpoint           = "osconfig.googleapis.com:443"
	oauthDefault           = ""
	bucketDefault          = "osconfig-agent-end2end-tests"
	logsPathDefault        = "logs"
	logPushIntervalDefault = 3 * time.Second

	testConfig *config
)

type config struct {
	svcendpoint, oauthPath, logBucket, logsPath string
	logPushInterval                             time.Duration
}

func SetConfig() {
	testConfig = &config{
		svcendpoint:     prodEndpoint,
		oauthPath:       oauthDefault,
		logBucket:       bucketDefault,
		logsPath:        fmt.Sprintf("%s-%s", logsPathDefault, time.Now().Format("2006-01-02-15:04:05")),
		logPushInterval: logPushIntervalDefault,
	}
}

// SvcEndpoint returns the svcEndpoint
func SvcEndpoint() string {
	return testConfig.svcendpoint
}

// OauthPath returns the oauthPath file path
func OauthPath() string {
	return testConfig.oauthPath
}

// LogBucket returns the oauthPath file path
func LogBucket() string {
	return testConfig.logBucket
}

// LogsPath returns the oauthPath file path
func LogsPath() string {
	return testConfig.logsPath
}

// LogPushInterval returns the interval at which the serial console logs are written to GCS
func LogPushInterval() time.Duration {
	return testConfig.logPushInterval
}
