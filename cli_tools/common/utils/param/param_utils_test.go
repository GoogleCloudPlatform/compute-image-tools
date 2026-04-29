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

package param

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

func TestGetRegion(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   error
	}{
		{"us-central1-c", "us-central1", nil},
		{"europe-north1-a", "europe-north1", nil},
		{"europe", "", fmt.Errorf("%v is not a valid zone", "europe")},
		{"", "", fmt.Errorf("zone is empty. Can't determine region")},
	}

	for _, test := range tests {
		zone := &test.input
		got, err := paramhelper.GetRegion(*zone)
		if test.want != got {
			t.Errorf("%v != %v", test.want, got)
		} else if err != test.err && test.err.Error() != err.Error() {
			t.Errorf("%v != %v", test.err, err)
		}
	}
}

func TestPopulateRegion(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   error
	}{
		{"us-central1-c", "us-central1", nil},
		{"europe", "", fmt.Errorf("%v is not a valid zone", "europe")},
		{"", "", fmt.Errorf("zone is empty. Can't determine region")},
	}

	for _, test := range tests {
		zone := &test.input
		regionInit := ""
		region := &regionInit
		err := PopulateRegion(region, *zone)
		if err != test.err && test.err.Error() != err.Error() {
			t.Errorf("%v != %v", test.err, err)
		} else if region != nil && test.want != *region {
			t.Errorf("%v != %v", test.want, *region)
		}
	}
}

func TestPopulateProjectIfMissingProjectPopulatedFromGCE(t *testing.T) {
	project := ""
	expectedProject := "gce_project"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return(expectedProject, nil)

	err := PopulateProjectIfMissing(mockMetadataGce, &project)

	assert.Nil(t, err)
	assert.Equal(t, expectedProject, project)
}

func TestPopulateProjectIfMissingProjectNotOnGCE(t *testing.T) {
	project := ""
	expectedProject := ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false)

	err := PopulateProjectIfMissing(mockMetadataGce, &project)

	assert.NotNil(t, err)
	assert.Equal(t, expectedProject, project)
}

func TestPopulateProjectIfNotMissingProject(t *testing.T) {
	project := "aProject"
	expectedProject := "aProject"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)

	err := PopulateProjectIfMissing(mockMetadataGce, &project)

	assert.Nil(t, err)
	assert.Equal(t, expectedProject, project)
}

func TestPopulateProjectIfMissingProjectWithErrorRetrievingFromGCE(t *testing.T) {
	project := ""
	expectedProject := ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return("", daisy.Errf("gce error"))

	err := PopulateProjectIfMissing(mockMetadataGce, &project)

	assert.NotNil(t, err)
	assert.Equal(t, expectedProject, project)
}

func TestGetImageResourcePath_HappyCase(t *testing.T) {
	assert.Equal(t, "projects/proj-12/global/images/ubuntu-19", GetImageResourcePath("proj-12", "ubuntu-19"))
}

func TestGetImageResourcePath_PanicWhenInvalidImageName(t *testing.T) {
	assert.Panics(t, func() {
		GetImageResourcePath("proj-12", "projects/proj-12/global/images/ubuntu-19")
	})
}

func TestGetImageResourcePath_PanicWhenInvalidProjectID(t *testing.T) {
	assert.Panics(t, func() {
		GetImageResourcePath("", "ubuntu-19")
	})
}

func TestGetGlobalResourcePathFromNameOnly(t *testing.T) {
	var n = GetGlobalResourcePath("networks", "aNetwork")
	assert.Equal(t, "global/networks/aNetwork", n)
}

func TestGetGlobalResourcePathFromRelativeURL(t *testing.T) {
	var n = GetGlobalResourcePath("networks", "x/blabla")
	assert.Equal(t, "x/blabla", n)
}

func TestGetGlobalResourcePathFromFullURL(t *testing.T) {
	var n = GetGlobalResourcePath("networks", "https://www.googleapis.com/compute/v1/x/blabla")
	assert.Equal(t, "x/blabla", n)
}

func TestGetRegionalResourcePathFromNameOnly(t *testing.T) {
	var n = GetRegionalResourcePath("aRegion", "subnetworks", "aSubnetwork")
	assert.Equal(t, "regions/aRegion/subnetworks/aSubnetwork", n)
}

func TestGetRegionalResourcePathFromRelativeURL(t *testing.T) {
	var n = GetRegionalResourcePath("aRegion", "subnetworks", "x/blabla")
	assert.Equal(t, "x/blabla", n)
}

func TestGetRegionalResourcePathFromFullURL(t *testing.T) {
	var n = GetRegionalResourcePath("aRegion", "subnetworks", "https://www.googleapis.com/compute/v1/x/blabla")
	assert.Equal(t, "x/blabla", n)
}
