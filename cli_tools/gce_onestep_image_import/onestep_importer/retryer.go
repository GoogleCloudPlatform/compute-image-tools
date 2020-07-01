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

package importer

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
)

// retryer implements the AWS SDK's request.Retryer interface.
// It wraps the SDK's built in DefaultRetryer with custom retry delay durations.
type retryer struct {
	defaultRetryer client.DefaultRetryer
}

var retryDurations = []time.Duration{
	time.Second * 1,
	time.Second * 2,
	time.Second * 4,
	time.Second * 8,
	time.Second * 8,
}

const maxRetryTimes = 5

// ShouldRetry returns if the failed request is retryable.
func (retryer retryer) ShouldRetry(r *request.Request) bool {
	return retryer.defaultRetryer.ShouldRetry(r)
}

// MaxRetries is the number of times a request may be retried before failing.
func (retryer retryer) MaxRetries() int {
	return maxRetryTimes
}

// RetryRules return the retry delay that should be used by the SDK before
// making another request attempt for the failed request.
func (retryer retryer) RetryRules(r *request.Request) time.Duration {
	ret := retryDurations[r.RetryCount-1]
	return ret
}
