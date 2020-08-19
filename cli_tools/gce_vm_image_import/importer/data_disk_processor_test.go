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
	"testing"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
)

func Test_ImageIncludesTrackingLabelAndLicense(t *testing.T) {
	trackingLicense := "projects/compute-image-tools/global/licenses/virtual-disk-import"
	trackingLabelKey := "gce-image-import"
	trackingLabelValue := "true"

	mockClient := mockComputeClient{expectedProject: "project-1234", t: t}

	processor := newDataDiskProcessor(
		persistentDisk{uri: "global/projects/pid/pd/id"},
		&mockClient,
		"project-1234",
		map[string]string{"user-key": "user-value"},
		"northamerica",
		"description-content",
		"family-name",
		"image-name")

	_, err := processor.process(persistentDisk{})
	assert.NoError(t, err)
	assert.Equal(t, 1, mockClient.invocations)
	assert.Equal(t, compute.Image{
		SourceDisk:       "global/projects/pid/pd/id",
		Labels:           map[string]string{trackingLabelKey: trackingLabelValue, "user-key": "user-value"},
		StorageLocations: []string{"northamerica"},
		Description:      "description-content",
		Family:           "family-name",
		Name:             "image-name",
		Licenses:         []string{trackingLicense},
	}, mockClient.actualImage, "Processor should add tracking license and tracking label.")
}

type mockComputeClient struct {
	daisyCompute.Client
	expectedProject string
	actualImage     compute.Image
	t               *testing.T
	invocations     int
}

func (m *mockComputeClient) CreateImage(project string, i *compute.Image) error {
	assert.Equal(m.t, m.expectedProject, project)
	m.actualImage = *i
	m.invocations++
	return nil
}
