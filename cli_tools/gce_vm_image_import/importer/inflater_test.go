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

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
)

func TestCreateDaisyInflater_Image_HappyCase(t *testing.T) {
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source:      imageSource{uri: "projects/test/uri/image"},
		Zone:        "us-west1-b",
		ExecutionID: "1234",
	})

	assert.Equal(t, "zones/us-west1-b/disks/disk-1234", inflater.inflatedDiskURI)
	assert.Equal(t, "projects/test/uri/image", inflater.wf.Vars["source_image"].Value)
	inflatedDisk := getDisk(inflater.wf, 0)
	assert.Contains(t, inflatedDisk.Licenses,
		"projects/compute-image-tools/global/licenses/virtual-disk-import")
}

func TestCreateDaisyInflater_Image_Windows(t *testing.T) {
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source: imageSource{uri: "image/uri"},
		OS:     "windows-2019",
	})

	assert.Contains(t, getDisk(inflater.wf, 0).GuestOsFeatures, &compute.GuestOsFeature{
		Type: "WINDOWS",
	})
}

func TestCreateDaisyInflater_Image_NotWindows(t *testing.T) {
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source: imageSource{uri: "image/uri"},
		OS:     "ubunt-1804",
	})

	assert.NotContains(t, getDisk(inflater.wf, 0).GuestOsFeatures, &compute.GuestOsFeature{
		Type: "WINDOWS",
	})
}

func TestCreateDaisyInflater_File_HappyCase(t *testing.T) {
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
	})

	assert.Equal(t, "zones/us-west1-c/disks/disk-1234", inflater.inflatedDiskURI)
	assert.Equal(t, "gs://bucket/vmdk", inflater.wf.Vars["source_disk_file"].Value)
	assert.Equal(t, "projects/subnet/subnet", inflater.wf.Vars["import_subnet"].Value)
	assert.Equal(t, "projects/network/network", inflater.wf.Vars["import_network"].Value)

	network := getWorkerNetwork(t, inflater.wf)
	assert.Nil(t, network.AccessConfigs, "AccessConfigs must be nil to allow ExternalIP to be allocated.")
}

func TestCreateDaisyInflater_File_NoExternalIP(t *testing.T) {
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		NoExternalIP: true,
	})

	network := getWorkerNetwork(t, inflater.wf)
	assert.NotNil(t, network.AccessConfigs, "To disable external IPs, AccessConfigs must be non-nil.")
}

func TestCreateDaisyInflater_File_Windows(t *testing.T) {
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source: fileSource{gcsPath: "gs://bucket/vmdk"},
		OS:     "windows-2019",
	})

	inflatedDisk := getDisk(inflater.wf, 1)
	assert.Contains(t, inflatedDisk.GuestOsFeatures, &compute.GuestOsFeature{
		Type: "WINDOWS",
	})
}

func TestCreateDaisyInflater_File_NotWindows(t *testing.T) {
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source: fileSource{gcsPath: "gs://bucket/vmdk"},
		OS:     "ubuntu-1804",
	})

	inflatedDisk := getDisk(inflater.wf, 1)
	assert.NotContains(t, inflatedDisk.GuestOsFeatures, &compute.GuestOsFeature{
		Type: "WINDOWS",
	})
}

func TestInflaterCancel(t *testing.T) {
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source: fileSource{gcsPath: "gs://bucket/vmdk"},
		OS:     "ubuntu-1804",
	})

	inflater.cancel("timed-out")

	inflater.wf.IterateWorkflowSteps(func(step *daisy.Step) {
		if step.CreateDisks != nil {
			for _, disk := range *step.CreateDisks {
				assert.False(t, disk.NoCleanup)
			}
		}
	})

	_, channelOpen := <-inflater.wf.Cancel
	assert.False(t, channelOpen, "inflater.wf.Cancel should be closed on timeout")
}

func createDaisyInflaterSafe(t *testing.T, args ImportArguments) *daisyInflater {
	args.WorkflowDir = "testdata/image_import"
	inflater, err := createDaisyInflater(args)
	assert.NoError(t, err)
	realInflater, ok := inflater.(*daisyInflater)
	assert.True(t, ok)
	return realInflater
}

func getWorkerNetwork(t *testing.T, workflow *daisy.Workflow) *compute.NetworkInterface {
	for _, step := range workflow.Steps {
		if step.CreateInstances != nil {
			instances := step.CreateInstances.Instances
			assert.Len(t, instances, 1)
			network := instances[0].NetworkInterfaces
			assert.Len(t, network, 1)
			return network[0]
		}
	}
	panic("expected create instance step with single network")
}
