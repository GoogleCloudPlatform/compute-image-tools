//  Copyright 2021 Google Inc. All Rights Reserved.
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

package daisyutils

import (
	"testing"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/stretchr/testify/assert"
)

func Test_ApplyEnvToWorkflow(t *testing.T) {
	env := EnvironmentSettings{
		Project:         "lucky-lemur",
		Zone:            "us-west1-c",
		GCSPath:         "new-path",
		OAuth:           "new-oauth",
		Timeout:         "new-timeout",
		ComputeEndpoint: "new-endpoint",
	}
	original := &daisy.Workflow{
		Project:         "original-project",
		Zone:            "original-zone",
		GCSPath:         "original-path",
		OAuthPath:       "original-oauth",
		DefaultTimeout:  "original-timeout",
		ComputeEndpoint: "original-endpoint",
	}
	assert.NoError(t, (&ApplyEnvToWorkflow{env}).PreRunHook(original))
	expected := &daisy.Workflow{
		Project:         "lucky-lemur",
		Zone:            "us-west1-c",
		GCSPath:         "new-path",
		OAuthPath:       "new-oauth",
		DefaultTimeout:  "new-timeout",
		ComputeEndpoint: "new-endpoint",
	}
	assert.Equal(t, original, expected)
}
