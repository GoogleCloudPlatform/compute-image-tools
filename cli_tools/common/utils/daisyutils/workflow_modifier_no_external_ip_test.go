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

	"github.com/stretchr/testify/assert"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func Test_RemoveExternalIPModifier(t *testing.T) {
	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	(*w.Steps["ci"].CreateInstances).Instances[0].Instance.NetworkInterfaces = nil
	(*w.Steps["ci"].CreateInstances).InstancesBeta[0].Instance.NetworkInterfaces = nil
	assert.NoError(t, (&RemoveExternalIPModifier{}).Modify(w))

	assert.Nil(t, (*w.Steps["ci"].CreateInstances).Instances[0].Instance.NetworkInterfaces)
	assert.Nil(t, (*w.Steps["ci"].CreateInstances).InstancesBeta[0].Instance.NetworkInterfaces)
}

func Test_RemoveExternalIPModifier_DoesntClobberExistingConfigs(t *testing.T) {
	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	assert.NoError(t, (&RemoveExternalIPModifier{}).Modify(w))

	assert.Len(t, (*w.Steps["ci"].CreateInstances).Instances[0].Instance.NetworkInterfaces, 1)
	assert.Len(t, (*w.Steps["ci"].CreateInstances).InstancesBeta[0].Instance.NetworkInterfaces, 1)
}

func createWorkflowWithCreateInstanceNetworkAccessConfig() *daisy.Workflow {
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"ci": {
			CreateInstances: &daisy.CreateInstances{
				Instances: []*daisy.Instance{
					{
						Instance: compute.Instance{
							Disks: []*compute.AttachedDisk{{Source: "key1"}},
							NetworkInterfaces: []*compute.NetworkInterface{
								{
									Network: "n",
									AccessConfigs: []*compute.AccessConfig{
										{Type: "ONE_TO_ONE_NAT"},
									},
								},
							},
						},
					},
				},
				InstancesBeta: []*daisy.InstanceBeta{
					{
						Instance: computeBeta.Instance{
							Disks: []*computeBeta.AttachedDisk{{Source: "key1"}},
							NetworkInterfaces: []*computeBeta.NetworkInterface{
								{
									Network: "n",
									AccessConfigs: []*computeBeta.AccessConfig{
										{Type: "ONE_TO_ONE_NAT"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return w
}
