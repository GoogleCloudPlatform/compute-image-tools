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

package daisycommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func Test_NewDaisyWorker_AppliesEnvVariablesToWorkflow(t *testing.T) {
	wf := daisy.New()
	wf.Project = "old-project"
	wf.Zone = "old-zone"
	wf.GCSPath = "old-gcs-path"
	wf.OAuthPath = "old-oauth-path"
	wf.DefaultTimeout = "old-timeout"
	wf.ComputeEndpoint = "old-endpoint"
	env := EnvironmentSettings{
		Project:         "lucky-lemur",
		Zone:            "us-west1-c",
		GCSPath:         "new-path",
		OAuth:           "new-oauth",
		Timeout:         "new-timeout",
		ComputeEndpoint: "new-endpoint",
	}
	NewDaisyWorker(wf, env, logging.NewToolLogger("test"))
	assert.Equal(t, env.Project, wf.Project)
	assert.Equal(t, env.Zone, wf.Zone)
	assert.Equal(t, env.GCSPath, wf.GCSPath)
	assert.Equal(t, env.OAuth, wf.OAuthPath)
	assert.Equal(t, env.Timeout, wf.DefaultTimeout)
	assert.Equal(t, env.ComputeEndpoint, wf.ComputeEndpoint)
}

func Test_NewDaisyWorker_AppliesWorkerCustomizations(t *testing.T) {
	wf := daisy.New()
	env := EnvironmentSettings{
		Network:               "network",
		Subnet:                "subnet",
		ComputeServiceAccount: "compute-service-account",
	}
	NewDaisyWorker(wf, env, logging.NewToolLogger("test"))
	assert.Equal(t, env.Network, wf.Vars["network"].Value)
	assert.Equal(t, env.Subnet, wf.Vars["subnet"].Value)
	assert.Equal(t, env.ComputeServiceAccount, wf.Vars["compute_service_account"].Value)
}

func Test_NewDaisyWorker_UpdatesAllInstanceNoExternalIP(t *testing.T) {
	wf := daisy.New()
	step, err := wf.NewStep("worker-instance")
	if err != nil {
		t.Fatal(err)
	}
	networkInterface := &compute.NetworkInterface{
		AccessConfigs: nil,
	}
	step.CreateInstances = &daisy.CreateInstances{
		Instances: []*daisy.Instance{{
			Instance: compute.Instance{
				NetworkInterfaces: []*compute.NetworkInterface{networkInterface},
			},
		}},
	}
	assert.Nil(t, networkInterface.AccessConfigs)
	NewDaisyWorker(wf, EnvironmentSettings{NoExternalIP: true}, logging.NewToolLogger("test"))
	assert.NotNil(t, networkInterface.AccessConfigs)
	assert.Empty(t, networkInterface.AccessConfigs)
}
