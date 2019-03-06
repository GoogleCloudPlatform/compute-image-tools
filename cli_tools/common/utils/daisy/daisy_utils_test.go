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

package daisyutils

import (
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
	"reflect"
	"testing"
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
	result := GetTranslateWorkflowPath(&input)
	if result != "ubuntu/translate_ubuntu_1604.wf.json" {
		t.Errorf("expected `%v`, got `%v`",
			"ubuntu/translate_ubuntu_1604.wf.json", result)
	}
}

func TestGetTranslateWorkflowPathInvalid(t *testing.T) {
	input := "not-an-OS"
	result := GetTranslateWorkflowPath(&input)
	if result != "" {
		t.Errorf("expected empty result, got `%v`", result)
	}
}

func TestParseWorkflows(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	path := "../../../../daisy/test_data/test.wf.json"
	varMap := map[string]string{"key1": "var1", "key2": "var2"}
	project := "project"
	zone := "zone"
	gcsPath := "gcspath"
	oauth := "oauthpath"
	dTimeout := "10m"
	endpoint := "endpoint"
	w, err := ParseWorkflow(mocks.NewMockMetadataGCEInterface(mockCtrl), path, varMap, project, zone,
		gcsPath, oauth, dTimeout, endpoint, true, true, true)
	if err != nil {
		t.Fatal(err)
	}

	assertWorkflow(t, w, project, zone, gcsPath, oauth, dTimeout, endpoint, varMap)
}

func TestParseWorkflowsProjectAndZoneFromMetadataGCE(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	path := "../../../../daisy/test_data/test_no_project_no_zone.wf.json"
	varMap := map[string]string{"key1": "var1", "key2": "var2"}
	project := ""
	zone := ""
	gcsPath := "gcspath"
	oauth := "oauthpath"
	dTimeout := "10m"
	endpoint := "endpoint"

	projectFromMetadata := "project_from_metadata"
	zoneFromMetadata := "zone_from_metadata"

	mockMetadataGCE := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGCE.EXPECT().OnGCE().Return(true).AnyTimes()
	mockMetadataGCE.EXPECT().ProjectID().Return(projectFromMetadata, nil)
	mockMetadataGCE.EXPECT().Zone().Return(zoneFromMetadata, nil)

	w, err := ParseWorkflow(mockMetadataGCE, path, varMap, project, zone,
		gcsPath, oauth, dTimeout, endpoint, true, true, true)
	if err != nil {
		t.Fatal(err)
	}

	assertWorkflow(t, w, projectFromMetadata, zoneFromMetadata, gcsPath, oauth, dTimeout, endpoint, varMap)
}

func TestParseWorkflowsReturnsErrorWhenMetadataProjectIDReturnsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	path := "../../../../daisy/test_data/test_no_project_no_zone.wf.json"
	varMap := map[string]string{"key1": "var1", "key2": "var2"}
	project := ""
	zone := ""
	gcsPath := "gcspath"
	oauth := "oauthpath"
	dTimeout := "10m"
	endpoint := "endpoint"

	mockMetadataGCE := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGCE.EXPECT().OnGCE().Return(true).AnyTimes()
	mockMetadataGCE.EXPECT().ProjectID().Return("", fmt.Errorf("projectID error"))

	w, err := ParseWorkflow(mockMetadataGCE, path, varMap, project, zone,
		gcsPath, oauth, dTimeout, endpoint, true, true, true)
	assert.NotNil(t, err)
	assert.Nil(t, w)
}

func TestParseWorkflowsReturnsErrorWhenMetadataZoneReturnsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	path := "../../../../daisy/test_data/test_no_project_no_zone.wf.json"
	varMap := map[string]string{"key1": "var1", "key2": "var2"}
	project := ""
	zone := ""
	gcsPath := "gcspath"
	oauth := "oauthpath"
	dTimeout := "10m"
	endpoint := "endpoint"

	projectFromMetadata := "project_from_metadata"

	mockMetadataGCE := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGCE.EXPECT().OnGCE().Return(true).AnyTimes()
	mockMetadataGCE.EXPECT().ProjectID().Return(projectFromMetadata, nil)
	mockMetadataGCE.EXPECT().Zone().Return("", fmt.Errorf("zone error"))

	w, err := ParseWorkflow(mockMetadataGCE, path, varMap, project, zone,
		gcsPath, oauth, dTimeout, endpoint, true, true, true)
	assert.NotNil(t, err)
	assert.Nil(t, w)
}

func TestParseWorkflowsInvalidPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	varMap := map[string]string{"key1": "var1", "key2": "var2"}
	project := "project"
	zone := "zone"
	gcsPath := "gcspath"
	oauth := "oauthpath"
	dTimeout := "10m"
	endpoint := "endpoint"
	w, err := ParseWorkflow(mocks.NewMockMetadataGCEInterface(mockCtrl), "NOT_VALID_PATH",
		varMap, project, zone, gcsPath, oauth, dTimeout, endpoint, true, true, true)
	assert.Nil(t, w)
	assert.NotNil(t, err)
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

func assertWorkflow(t *testing.T, w *daisy.Workflow, project string, zone string, gcsPath string,
	oauth string, dTimeout string, endpoint string, varMap map[string]string) {
	tests := []struct {
		want, got interface{}
	}{
		{w.Project, project},
		{w.Zone, zone},
		{w.GCSPath, gcsPath},
		{w.OAuthPath, oauth},
		{w.DefaultTimeout, dTimeout},
		{w.ComputeEndpoint, endpoint},
	}
	for _, tt := range tests {
		if tt.want != tt.got {
			t.Errorf("%v != %v", tt.want, tt.got)
		}
	}
	if reflect.DeepEqual(w.Vars, varMap) {
		t.Errorf("unexpected vars, want: %v, got: %v", varMap, w.Vars)
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
