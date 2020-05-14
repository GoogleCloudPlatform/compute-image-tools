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
	"github.com/stretchr/testify/assert"
	computeBeta "google.golang.org/api/compute/v0.beta"
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

	if len((*w.Steps["ci"].CreateInstances).Instances[0].Instance.NetworkInterfaces[0].AccessConfigs) != 0 {
		t.Errorf("Instance AccessConfigs not empty")
	}
	if len((*w.Steps["ci"].CreateInstances).InstancesBeta[0].Instance.NetworkInterfaces[0].AccessConfigs) != 0 {
		t.Errorf("Instance AccessConfigs not empty")
	}
}

func TestUpdateWorkflowInstancesNotModifiedIfExternalIPAllowed(t *testing.T) {
	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	UpdateAllInstanceNoExternalIP(w, false)

	if len((*w.Steps["ci"].CreateInstances).Instances[0].Instance.NetworkInterfaces[0].AccessConfigs) != 1 {
		t.Errorf("Instance AccessConfigs doesn't have exactly one instance")
	}
	if len((*w.Steps["ci"].CreateInstances).InstancesBeta[0].Instance.NetworkInterfaces[0].AccessConfigs) != 1 {
		t.Errorf("Instance AccessConfigs doesn't have exactly one instance")
	}
}

func TestUpdateWorkflowInstancesNotModifiedIfNoNetworkInterfaceElement(t *testing.T) {
	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	(*w.Steps["ci"].CreateInstances).Instances[0].Instance.NetworkInterfaces = nil
	(*w.Steps["ci"].CreateInstances).InstancesBeta[0].Instance.NetworkInterfaces = nil
	UpdateAllInstanceNoExternalIP(w, true)

	if (*w.Steps["ci"].CreateInstances).Instances[0].Instance.NetworkInterfaces != nil {
		t.Errorf("Instance NetworkInterfaces should stay nil if nil before update")
	}
	if (*w.Steps["ci"].CreateInstances).InstancesBeta[0].Instance.NetworkInterfaces != nil {
		t.Errorf("Instance NetworkInterfaces should stay nil if nil before update")
	}
}

func TestRemovePrivacyLogInfoNoPrivacyInfo(t *testing.T) {
	testRemovePrivacyLogInfo(t,
		"No privacy info",
		"No privacy info",
		"Regular message should stay the same")

	testRemovePrivacyLogInfo(t,
		"[Privacy-> info",
		"[Privacy-> info",
		"Incomplete privacy info (left bracket) should stay the same")

	testRemovePrivacyLogInfo(t,
		"info <-Privacy]",
		"info <-Privacy]",
		"Incomplete privacy info (right bracket) should stay the same")
}

func TestRemovePrivacyLogInfoTranslateFailed(t *testing.T) {
	testRemovePrivacyLogInfo(t,
		"[DaisyLog] TranslateFailed: OS not detected\nFailed for project my-project",
		"TranslateFailed",
		"TranslateFailed error details should be hidden")
}

func TestRemovePrivacyLogInfoSingle(t *testing.T) {
	testRemovePrivacyLogInfo(t,
		"[Privacy-> info 1 <-Privacy] info 2",
		" info 2",
		"Privacy info (on the head) should be hidden")

	testRemovePrivacyLogInfo(t,
		"info 0 [Privacy-> info 1 <-Privacy]",
		"info 0 ",
		"Privacy info (on the tail) should be hidden")
}

func TestRemovePrivacyLogInfoMultiple(t *testing.T) {
	testRemovePrivacyLogInfo(t,
		"info 0 [Privacy-> info 1 <-Privacy] info 2 [Privacy-> info 3 <-Privacy] info 4",
		"info 0  info 2  info 4",
		"Multiple privacy info should be hidden")
}

func testRemovePrivacyLogInfo(t *testing.T, originalMessage string, expectedMessage string, onFailure string) {
	m := RemovePrivacyLogInfo(originalMessage)
	if m != expectedMessage {
		t.Errorf("%v. Expect: `%v`, actual: `%v`", onFailure, expectedMessage, m)
	}
}

func TestRemovePrivacyTagSingle(t *testing.T) {
	testRemovePrivacyTag(t,
		"[Privacy-> info 1 <-Privacy]",
		" info 1 ",
		"Single privacy tag should be removed")
}

func TestRemovePrivacyTagMultiple(t *testing.T) {
	testRemovePrivacyTag(t,
		"Error: [Privacy->abc <-Privacy] [Privacy-> xyz<-Privacy] and <-Privacy]",
		"Error: abc   xyz and ",
		"Multiple privacy tag should be removed")
}

func testRemovePrivacyTag(t *testing.T, originalMessage string, expectedMessage string, onFailure string) {
	m := RemovePrivacyLogTag(originalMessage)
	if m != expectedMessage {
		t.Errorf("%v. Expect: `%v`, actual: `%v`", onFailure, expectedMessage, m)
	}
}

func TestUpdateToUEFICompatible(t *testing.T) {
	w := createWorkflowWithCreateDiskImageAndIncludeWorkflow()

	UpdateToUEFICompatible(w)

	assert.Equal(t, 1, len((*w.Steps["cd"].CreateDisks)[0].GuestOsFeatures))
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["cd"].CreateDisks)[0].GuestOsFeatures[0].Type)

	assert.Equal(t, 0, len((*w.Steps["cd"].CreateDisks)[1].GuestOsFeatures))

	assert.Equal(t, 1, len((*w.Steps["iw"].IncludeWorkflow.Workflow.Steps["iw-cd"].CreateDisks)[0].GuestOsFeatures))
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["iw"].IncludeWorkflow.Workflow.Steps["iw-cd"].CreateDisks)[0].GuestOsFeatures[0].Type)

	assert.Equal(t, 1, len((*w.Steps["cimg"].CreateImages).Images[0].GuestOsFeatures))
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["cimg"].CreateImages).Images[0].GuestOsFeatures[0])
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["cimg"].CreateImages).Images[0].Image.GuestOsFeatures[0].Type)

	assert.Equal(t, 1, len((*w.Steps["cimg"].CreateImages).ImagesBeta[0].GuestOsFeatures))
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["cimg"].CreateImages).ImagesBeta[0].GuestOsFeatures[0])
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["cimg"].CreateImages).ImagesBeta[0].Image.GuestOsFeatures[0].Type)
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

func createWorkflowWithCreateDiskImageAndIncludeWorkflow() *daisy.Workflow {
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"cd": {
			CreateDisks: &daisy.CreateDisks{
				{
					Disk: compute.Disk{},
				},
				{
					Disk: compute.Disk{
						SourceImage: "projects/compute-image-tools/global/images/family/debian-9-worker",
					},
				},
			},
		},
		"cimg": {
			CreateImages: &daisy.CreateImages{
				Images: []*daisy.Image{
					{
						Image: compute.Image{
							Name: "final-image-1",
						},
					},
				},
				ImagesBeta: []*daisy.ImageBeta{
					{
						Image: computeBeta.Image{
							Name: "final-image-1",
						},
					},
				},
			},
		},
		"cins": {
			CreateInstances: &daisy.CreateInstances{
				Instances: []*daisy.Instance{
					{
						Instance: compute.Instance{
							Disks: []*compute.AttachedDisk{{Source: "key1"}},
						},
					},
				},
				InstancesBeta: []*daisy.InstanceBeta{
					{
						Instance: computeBeta.Instance{
							Disks: []*computeBeta.AttachedDisk{{Source: "key1"}},
						},
					},
				},
			},
		},
		"iw": {
			IncludeWorkflow: &daisy.IncludeWorkflow{
				Workflow: &daisy.Workflow{
					Steps: map[string]*daisy.Step{
						"iw-cd": {
							CreateDisks: &daisy.CreateDisks{
								{
									Disk: compute.Disk{},
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

func TestGetResourceRealName(t *testing.T) {
	type testCase struct {
		testName         string
		resourceName     string
		expectedRealName string
	}

	tcs := []testCase{
		{"simple resource name", "resname", "resname"},
		{"URI", "path/resname", "resname"},
		{"longer URI", "https://resource/path/resname", "resname"},
	}

	for _, tc := range tcs {
		realName := GetResourceRealName(tc.resourceName)
		if realName != tc.expectedRealName {
			t.Errorf("[%v]: Expected real name '%v' != actrual real name '%v'", tc.testName, tc.expectedRealName, realName)
		}
	}
}

func TestGetDeviceURI(t *testing.T) {
	uri := GetDeviceURI("p", "z", "d")
	expectedURI := "projects/p/zones/z/devices/d"
	if uri != expectedURI {
		t.Errorf("URI '%v' doesn't match expected '%v'", uri, expectedURI)
	}
}

func TestGetDiskURI(t *testing.T) {
	uri := GetDiskURI("p", "z", "d")
	expectedURI := "projects/p/zones/z/disks/d"
	if uri != expectedURI {
		t.Errorf("URI '%v' doesn't match expected '%v'", uri, expectedURI)
	}
}

func TestGetInstanceURI(t *testing.T) {
	uri := GetInstanceURI("p", "z", "i")
	expectedURI := "projects/p/zones/z/instances/i"
	if uri != expectedURI {
		t.Errorf("URI '%v' doesn't match expected '%v'", uri, expectedURI)
	}
}
