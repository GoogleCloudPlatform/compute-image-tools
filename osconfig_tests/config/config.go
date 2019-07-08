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
	"flag"
	"fmt"
	"time"
)

var (
	// TODO: allow this to be configurable through flag to test against staging
	prodEndpoint           = "osconfig.googleapis.com:443"
	oauthDefault           = flag.String("local_oauth", "", "path to service creds file")
	agentRepo              = flag.String("agent_repo", "unstable", "repo to pull agent from (unstable, staging, or stable)")
	bucketDefault          = "osconfig-agent-end2end-tests"
	logPushIntervalDefault = 3 * time.Second
	logsPath               = fmt.Sprintf("logs-%s", time.Now().Format("2006-01-02-15:04:05"))
)

// AgentRepo returns the agentRepo
func AgentRepo() string {
	return *agentRepo
}

// SvcEndpoint returns the svcEndpoint
func SvcEndpoint() string {
	return prodEndpoint
}

// OauthPath returns the oauthPath file path
func OauthPath() string {
	return *oauthDefault
}

// LogBucket returns the oauthPath file path
func LogBucket() string {
	return bucketDefault
}

// LogsPath returns the oauthPath file path
func LogsPath() string {
	return logsPath
}

// LogPushInterval returns the interval at which the serial console logs are written to GCS
func LogPushInterval() time.Duration {
	return logPushIntervalDefault
}
