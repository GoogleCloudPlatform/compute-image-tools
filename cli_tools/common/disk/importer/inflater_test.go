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
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
)

// gcloud expects log lines to start with the substring "[import". Daisy
// constructs the log prefix using the workflow's name.
func TestCreateDaisyInflater_SetsWorkflowNameToGcloudPrefix(t *testing.T) {
	inflater := createDaisyInflaterForImageSafe(t, ImportArguments{
		Source: imageSource{uri: "projects/test/uri/image"},
	})
	assert.Equal(t, inflater.wf.Name, "import-image")
}

func TestCreateDaisyInflater_Image_HappyCase(t *testing.T) {
	inflater := createDaisyInflaterForImageSafe(t, ImportArguments{
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
	inflater := createDaisyInflaterForImageSafe(t, ImportArguments{
		Source: imageSource{uri: "image/uri"},
		OS:     "windows-2019",
	})

	assert.Contains(t, getDisk(inflater.wf, 0).GuestOsFeatures, &compute.GuestOsFeature{
		Type: "WINDOWS",
	})
}

func TestCreateDaisyInflater_Image_NotWindows(t *testing.T) {
	inflater := createDaisyInflaterForImageSafe(t, ImportArguments{
		Source: imageSource{uri: "image/uri"},
		OS:     "ubuntu-1804",
	})

	assert.NotContains(t, getDisk(inflater.wf, 0).GuestOsFeatures, &compute.GuestOsFeature{
		Type: "WINDOWS",
	})
}

func TestCreateDaisyInflater_Image_UEFI(t *testing.T) {
	inflater := createDaisyInflaterForImageSafe(t, ImportArguments{
		Source:         imageSource{uri: "image/uri"},
		OS:             "ubuntu-1804",
		UefiCompatible: true,
	})

	assert.Contains(t, getDisk(inflater.wf, 0).GuestOsFeatures, &compute.GuestOsFeature{
		Type: "UEFI_COMPATIBLE",
	})
}

func TestCreateDaisyInflater_Image_NotUEFI(t *testing.T) {
	inflater := createDaisyInflaterForImageSafe(t, ImportArguments{
		Source:         imageSource{uri: "image/uri"},
		OS:             "ubuntu-1804",
		UefiCompatible: false,
	})

	assert.NotContains(t, getDisk(inflater.wf, 0).GuestOsFeatures, &compute.GuestOsFeature{
		Type: "UEFI_COMPATIBLE",
	})
}

func TestCreateDaisyInflater_File_HappyCase(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source:       source,
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
	}, mockInspector{
		t:                 t,
		expectedReference: source.gcsPath,
		errorToReturn:     nil,
		metaToReturn:      imagefile.Metadata{},
	})

	assert.Equal(t, "zones/us-west1-c/disks/disk-1234", inflater.inflatedDiskURI)
	assert.Equal(t, "gs://bucket/vmdk", inflater.wf.Vars["source_disk_file"].Value)
	assert.Equal(t, "projects/subnet/subnet", inflater.wf.Vars["import_subnet"].Value)
	assert.Equal(t, "projects/network/network", inflater.wf.Vars["import_network"].Value)

	network := getWorkerNetwork(t, inflater.wf)
	assert.Nil(t, network.AccessConfigs, "AccessConfigs must be nil to allow ExternalIP to be allocated.")
}

func TestCreateDaisyInflater_File_NoExternalIP(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source:       source,
		NoExternalIP: true,
	}, mockInspector{
		t:                 t,
		expectedReference: source.gcsPath,
		errorToReturn:     nil,
		metaToReturn:      imagefile.Metadata{},
	})

	network := getWorkerNetwork(t, inflater.wf)
	assert.NotNil(t, network.AccessConfigs, "To disable external IPs, AccessConfigs must be non-nil.")
}

func TestCreateDaisyInflater_File_UsesFallbackSizes_WhenInspectionFails(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source:       source,
		NoExternalIP: true,
	}, mockInspector{
		t:                 t,
		expectedReference: source.gcsPath,
		errorToReturn:     errors.New("inspection failed"),
		metaToReturn:      imagefile.Metadata{},
	})

	// The 10GB defaults are hardcoded in inflate_file.wf.json.
	assert.Equal(t, "10", inflater.wf.Vars["scratch_disk_size_gb"].Value)
	assert.Equal(t, "10", inflater.wf.Vars["inflated_disk_size_gb"].Value)
}

func TestCreateDaisyInflater_File_SetsSizesFromInspectedFile(t *testing.T) {
	tests := []struct {
		physicalSize     int64
		virtualSize      int64
		expectedInflated string
		expectedScratch  string
	}{
		{
			virtualSize:      1,
			expectedInflated: "10",
			expectedScratch:  "10",
		},
		{
			physicalSize:     8,
			virtualSize:      9,
			expectedInflated: "10",
			expectedScratch:  "10",
		},
		{
			physicalSize:     1008,
			virtualSize:      1024,
			expectedInflated: "1024",
			expectedScratch:  "1109",
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			source := fileSource{gcsPath: "gs://bucket/vmdk"}
			inflater := createDaisyInflaterSafe(t, ImportArguments{
				Source:       source,
				NoExternalIP: true,
			}, mockInspector{
				t:                 t,
				expectedReference: source.gcsPath,
				metaToReturn: imagefile.Metadata{
					VirtualSizeGB:  tt.virtualSize,
					PhysicalSizeGB: tt.physicalSize,
				},
			})

			assert.Equal(t, tt.expectedInflated, inflater.wf.Vars["inflated_disk_size_gb"].Value)
			assert.Equal(t, tt.expectedScratch, inflater.wf.Vars["scratch_disk_size_gb"].Value)
		})
	}
}

func TestCreateDaisyInflater_File_Windows(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source: source,
		OS:     "windows-2019",
	}, mockInspector{
		t:                 t,
		expectedReference: source.gcsPath,
		errorToReturn:     nil,
		metaToReturn:      imagefile.Metadata{},
	})

	inflatedDisk := getDisk(inflater.wf, 1)
	assert.Contains(t, inflatedDisk.GuestOsFeatures, &compute.GuestOsFeature{
		Type: "WINDOWS",
	})
}

func TestCreateDaisyInflater_File_NotWindows(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source: source,
		OS:     "ubuntu-1804",
	}, mockInspector{
		t:                 t,
		expectedReference: source.gcsPath,
		errorToReturn:     nil,
		metaToReturn:      imagefile.Metadata{},
	})

	inflatedDisk := getDisk(inflater.wf, 1)
	assert.NotContains(t, inflatedDisk.GuestOsFeatures, &compute.GuestOsFeature{
		Type: "WINDOWS",
	})
}

func TestCreateDaisyInflater_File_UEFI(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source:         source,
		OS:             "ubuntu-1804",
		UefiCompatible: true,
	}, mockInspector{
		t:                 t,
		expectedReference: source.gcsPath,
		errorToReturn:     nil,
		metaToReturn:      imagefile.Metadata{},
	})

	inflatedDisk := getDisk(inflater.wf, 1)
	assert.Contains(t, inflatedDisk.GuestOsFeatures, &compute.GuestOsFeature{
		Type: "UEFI_COMPATIBLE",
	})
}

func TestCreateDaisyInflater_File_NotUEFI(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source:         source,
		OS:             "ubuntu-1804",
		UefiCompatible: false,
	}, mockInspector{
		t:                 t,
		expectedReference: source.gcsPath,
		errorToReturn:     nil,
		metaToReturn:      imagefile.Metadata{},
	})

	inflatedDisk := getDisk(inflater.wf, 1)
	assert.NotContains(t, inflatedDisk.GuestOsFeatures, &compute.GuestOsFeature{
		Type: "UEFI_COMPATIBLE",
	})
}

func createDaisyInflaterSafe(t *testing.T, args ImportArguments,
	inspector imagefile.Inspector) *daisyInflater {
	args.WorkflowDir = "../../../../daisy_workflows"
	inflater, err := NewDaisyInflater(args, inspector)
	assert.NoError(t, err)
	realInflater, ok := inflater.(*daisyInflater)
	assert.True(t, ok)
	return realInflater
}

func createDaisyInflaterForImageSafe(t *testing.T, args ImportArguments) *daisyInflater {
	return createDaisyInflaterSafe(t, args, nil)
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

type mockInspector struct {
	t                 *testing.T
	expectedReference string
	errorToReturn     error
	metaToReturn      imagefile.Metadata
}

func (m mockInspector) Inspect(
	ctx context.Context, reference string) (imagefile.Metadata, error) {
	assert.Equal(m.t, m.expectedReference, reference)
	return m.metaToReturn, m.errorToReturn
}
