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

package importer

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func TestDaisyInflater_Inflate_ReadsDiskStatsFromWorker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorker := mocks.NewMockDaisyWorker(ctrl)
	inflater := daisyInflater{
		mockWorker, map[string]string{}, nil, "/disk/uri", logging.NewToolLogger("test"),
	}
	mockWorker.EXPECT().RunAndReadSerialValues(inflater.vars, targetSizeGBKey,
		sourceSizeGBKey, importFileFormatKey, diskChecksumKey).DoAndReturn(
		func(vars map[string]string, keys ...string) (map[string]string, error) {
			// Guarantee that the workflow executes for at least 1ms
			time.Sleep(time.Millisecond)
			return map[string]string{
				targetSizeGBKey:     "100",
				sourceSizeGBKey:     "200",
				importFileFormatKey: "vhd",
				diskChecksumKey:     "9abc",
			}, nil
		})
	pDisk, shadowFields, e := inflater.Inflate()
	assert.Nil(t, e)
	assert.Equal(t, persistentDisk{uri: "/disk/uri", sizeGb: 100, sourceGb: 200, sourceType: "vhd"}, pDisk)
	assert.Greater(t, shadowFields.inflationTime.Milliseconds(), int64(0), "inflation time should be greater than 0")
	shadowFields.inflationTime = 0
	assert.Equal(t, inflationInfo{checksum: "9abc", inflationType: "qemu"}, shadowFields)
}

func TestDaisyInflater_Inflate_IncludesDiskStatsOnError(t *testing.T) {
	expectedError := errors.New("failure to inflate disk")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorker := mocks.NewMockDaisyWorker(ctrl)
	inflater := daisyInflater{
		mockWorker, map[string]string{}, nil, "/disk/uri", logging.NewToolLogger("test"),
	}
	mockWorker.EXPECT().RunAndReadSerialValues(inflater.vars, targetSizeGBKey,
		sourceSizeGBKey, importFileFormatKey, diskChecksumKey).Return(map[string]string{
		targetSizeGBKey:     "50",
		sourceSizeGBKey:     "12",
		importFileFormatKey: "qcow2",
		diskChecksumKey:     "9abc",
	}, expectedError)
	pDisk, shadowFields, actualError := inflater.Inflate()
	assert.Equal(t, expectedError, actualError)
	assert.Equal(t, persistentDisk{uri: "/disk/uri", sizeGb: 50, sourceGb: 12, sourceType: "qcow2"}, pDisk)
	shadowFields.inflationTime = 0
	assert.Equal(t, inflationInfo{checksum: "9abc", inflationType: "qemu"}, shadowFields)
}

// gcloud expects log lines to start with the substring "[import". Daisy
// constructs the log prefix using the workflow's name.
func TestCreateDaisyInflater_SetsWorkflowNameToGcloudPrefix(t *testing.T) {
	inflater := createDaisyInflaterForImageSafe(t, ImageImportRequest{
		Source:             imageSource{uri: "projects/test/uri/image"},
		DaisyLogLinePrefix: "disk-1",
	})
	daisyutils.CheckEnvironment(inflater.worker, func(env daisyutils.EnvironmentSettings) {
		assert.Equal(t, "disk-1-inflate", env.DaisyLogLinePrefix)
	})
}

func TestCreateDaisyInflater_Image_HappyCase(t *testing.T) {
	inflater := createDaisyInflaterForImageSafe(t, ImageImportRequest{
		Source:      imageSource{uri: "projects/test/uri/image"},
		Zone:        "us-west1-b",
		ExecutionID: "1234",
	})

	assert.Equal(t, "zones/us-west1-b/disks/disk-1234", inflater.inflatedDiskURI)
	assert.Equal(t, "projects/test/uri/image", inflater.vars["source_image"])
	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		assert.Contains(t, getDisk(wf, 0).Licenses,
			"projects/compute-image-tools/global/licenses/virtual-disk-import")
	})
}

func TestCreateDaisyInflater_Image_Windows(t *testing.T) {
	inflater := createDaisyInflaterForImageSafe(t, ImageImportRequest{
		Source: imageSource{uri: "image/uri"},
		OS:     "windows-2019",
	})
	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		assert.Contains(t, getDisk(wf, 0).GuestOsFeatures, &compute.GuestOsFeature{
			Type: "WINDOWS",
		})
	})
}

func TestCreateDaisyInflater_Image_NotWindows(t *testing.T) {
	inflater := createDaisyInflaterForImageSafe(t, ImageImportRequest{
		Source: imageSource{uri: "image/uri"},
		OS:     "ubuntu-1804",
	})
	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		assert.NotContains(t, getDisk(wf, 0).GuestOsFeatures, &compute.GuestOsFeature{
			Type: "WINDOWS",
		})
	})
}

func TestCreateDaisyInflater_Image_UEFI(t *testing.T) {
	inflater := createDaisyInflaterForImageSafe(t, ImageImportRequest{
		Source:         imageSource{uri: "image/uri"},
		OS:             "ubuntu-1804",
		UefiCompatible: true,
	})
	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		assert.Contains(t, getDisk(wf, 0).GuestOsFeatures, &compute.GuestOsFeature{
			Type: "UEFI_COMPATIBLE",
		})
	})
}

func TestCreateDaisyInflater_Image_NotUEFI(t *testing.T) {
	inflater := createDaisyInflaterForImageSafe(t, ImageImportRequest{
		Source:         imageSource{uri: "image/uri"},
		OS:             "ubuntu-1804",
		UefiCompatible: false,
	})
	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		assert.NotContains(t, getDisk(wf, 0).GuestOsFeatures, &compute.GuestOsFeature{
			Type: "UEFI_COMPATIBLE",
		})
	})
}

func TestCreateDaisyInflater_File_HappyCase(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImageImportRequest{
		Source:       source,
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
	}, imagefile.Metadata{})
	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		assert.Equal(t, "zones/us-west1-c/disks/disk-1234", inflater.inflatedDiskURI)
		assert.Equal(t, "gs://bucket/vmdk", wf.Vars["source_disk_file"].Value)
		assert.Equal(t, "projects/subnet/subnet", wf.Vars["import_subnet"].Value)
		assert.Equal(t, "projects/network/network", wf.Vars["import_network"].Value)
		assert.Equal(t, "default", wf.Vars["compute_service_account"].Value)

		network := getWorkerNetwork(t, wf)
		assert.Nil(t, network.AccessConfigs, "AccessConfigs must be nil to allow ExternalIP to be allocated.")
	})
}

func TestCreateDaisyInflater_File_ComputeServiceAcount(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImageImportRequest{
		Source:                source,
		ComputeServiceAccount: "csa",
	}, imagefile.Metadata{})
	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		assert.Equal(t, "csa", wf.Vars["compute_service_account"].Value)
	})
}

func TestCreateDaisyInflater_File_NoExternalIP(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImageImportRequest{
		Source:       source,
		NoExternalIP: true,
	}, imagefile.Metadata{})
	daisyutils.CheckEnvironment(inflater.worker, func(env daisyutils.EnvironmentSettings) {
		assert.True(t, env.NoExternalIP)
	})
}

func TestCreateDaisyInflater_File_UsesFallbackSizes_WhenInspectionFails(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImageImportRequest{
		Source:       source,
		NoExternalIP: true,
	}, imagefile.Metadata{})
	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		// The 10GB defaults are hardcoded in inflate_file.wf.json.
		assert.Equal(t, "10", wf.Vars["scratch_disk_size_gb"].Value)
		assert.Equal(t, "10", wf.Vars["inflated_disk_size_gb"].Value)
	})
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
			inflater := createDaisyInflaterSafe(t, ImageImportRequest{
				Source:       source,
				NoExternalIP: true,
			}, imagefile.Metadata{
				VirtualSizeGB:  tt.virtualSize,
				PhysicalSizeGB: tt.physicalSize,
			})
			daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
				assert.Equal(t, tt.expectedInflated, wf.Vars["inflated_disk_size_gb"].Value)
				assert.Equal(t, tt.expectedScratch, wf.Vars["scratch_disk_size_gb"].Value)
			})
		})
	}
}

func TestCreateDaisyInflater_File_Windows(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImageImportRequest{
		Source: source,
		OS:     "windows-2019",
	}, imagefile.Metadata{})
	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		inflatedDisk := getDisk(wf, 1)
		assert.Contains(t, inflatedDisk.GuestOsFeatures, &compute.GuestOsFeature{
			Type: "WINDOWS",
		})
	})
}

func TestCreateDaisyInflater_File_NotWindows(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImageImportRequest{
		Source: source,
		OS:     "ubuntu-1804",
	}, imagefile.Metadata{})

	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		inflatedDisk := getDisk(wf, 1)
		assert.NotContains(t, inflatedDisk.GuestOsFeatures, &compute.GuestOsFeature{
			Type: "WINDOWS",
		})
	})
}

func TestCreateDaisyInflater_File_UEFI(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImageImportRequest{
		Source:         source,
		OS:             "ubuntu-1804",
		UefiCompatible: true,
	}, imagefile.Metadata{})

	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		inflatedDisk := getDisk(wf, 1)
		assert.Contains(t, inflatedDisk.GuestOsFeatures, &compute.GuestOsFeature{
			Type: "UEFI_COMPATIBLE",
		})
	})
}

func TestCreateDaisyInflater_File_NotUEFI(t *testing.T) {
	source := fileSource{gcsPath: "gs://bucket/vmdk"}
	inflater := createDaisyInflaterSafe(t, ImageImportRequest{
		Source:         source,
		OS:             "ubuntu-1804",
		UefiCompatible: false,
	}, imagefile.Metadata{})

	daisyutils.CheckWorkflow(inflater.worker, func(wf *daisy.Workflow, err error) {
		inflatedDisk := getDisk(wf, 1)
		assert.NotContains(t, inflatedDisk.GuestOsFeatures, &compute.GuestOsFeature{
			Type: "UEFI_COMPATIBLE",
		})
	})
}

func createDaisyInflaterSafe(t *testing.T, request ImageImportRequest,
	fileMetadata imagefile.Metadata) *daisyInflater {

	if request.ExecutionID == "" {
		request.ExecutionID = "build-id"
	}

	if request.Tool.ResourceLabelName == "" {
		request.Tool.ResourceLabelName = "image-import"
	}

	request.WorkflowDir = "../../../../daisy_workflows"
	daisyInflater, err := newDaisyInflater(request, fileMetadata, logging.NewToolLogger("test"))
	assert.NoError(t, err)
	return daisyInflater
}

func createDaisyInflaterForImageSafe(t *testing.T, request ImageImportRequest) *daisyInflater {
	return createDaisyInflaterSafe(t, request, imagefile.Metadata{})
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
