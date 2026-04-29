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
	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

// RemoveExternalIPHook is a WorkflowHook that updates CreateInstances in a
// daisy workflow such that they won't be created with an external IP address.
//
// For more info on external IPs, see the `--no-address` flag:
//
//	https://cloud.google.com/sdk/gcloud/reference/compute/instances/create
type RemoveExternalIPHook struct{}

// PreRunHook updates the CreateInstances steps so that they won't have an external IP.
func (t *RemoveExternalIPHook) PreRunHook(wf *daisy.Workflow) error {
	wf.IterateWorkflowSteps(func(step *daisy.Step) {
		if step.CreateInstances != nil {
			for _, instance := range step.CreateInstances.Instances {
				if instance.Instance.NetworkInterfaces == nil {
					continue
				}
				for _, networkInterface := range instance.Instance.NetworkInterfaces {
					networkInterface.AccessConfigs = []*compute.AccessConfig{}
				}
			}
			for _, instance := range step.CreateInstances.InstancesBeta {
				if instance.Instance.NetworkInterfaces == nil {
					continue
				}
				for _, networkInterface := range instance.Instance.NetworkInterfaces {
					networkInterface.AccessConfigs = []*computeBeta.AccessConfig{}
				}
			}

		}
	})
	return nil
}
