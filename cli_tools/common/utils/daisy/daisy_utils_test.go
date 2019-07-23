//  Copyright 2018 Google Inc. All Rights Reserved.
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

package daisy

import (
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

func TestValidateOsValid(t *testing.T) {
	err := ValidateOS("ubuntu-1604")
	if err != nil {
		t.Errorf("expected nil error, got `%v`", err)
	}
}

func TestValidateOsInvalid(t *testing.T) {
	err := ValidateOS("not-an-OS")
	if err == nil {
		t.Errorf("expected non-nil error")
	}
}

func TestGetTranslateWorkflowPathValid(t *testing.T) {
	input := "ubuntu-1604"
	result := GetTranslateWorkflowPath(input)
	if result != "ubuntu/translate_ubuntu_1604.wf.json" {
		t.Errorf("expected `%v`, got `%v`",
			"ubuntu/translate_ubuntu_1604.wf.json", result)
	}
}

func TestGetTranslateWorkflowPathInvalid(t *testing.T) {
	input := "not-an-OS"
	result := GetTranslateWorkflowPath(input)
	if result != "" {
		t.Errorf("expected empty result, got `%v`", result)
	}
}

func TestUpdateWorkflowInstancesConfiguredForNoExternalIP(t *testing.T) {
	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	UpdateAllInstanceNoExternalIP(w, true)

	if len((*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces[0].AccessConfigs) != 0 {
		t.Errorf("Instance AccessConfigs not empty")
	}
}

func TestUpdateWorkflowInstancesNotModifiedIfExternalIPAllowed(t *testing.T) {
	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	UpdateAllInstanceNoExternalIP(w, false)

	if len((*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces[0].AccessConfigs) != 1 {
		t.Errorf("Instance AccessConfigs doesn't have exactly one instance")
	}
}

func TestUpdateWorkflowInstancesNotModifiedIfNoNetworkInterfaceElement(t *testing.T) {
	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	(*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces = nil
	UpdateAllInstanceNoExternalIP(w, true)

	if (*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces != nil {
		t.Errorf("Instance NetworkInterfaces should stay nil if nil before update")
	}
}

func createWorkflowWithCreateInstanceNetworkAccessConfig() *daisy.Workflow {
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"ci": {
			CreateInstances: &daisy.CreateInstances{
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
		},
	}
	return w
}
