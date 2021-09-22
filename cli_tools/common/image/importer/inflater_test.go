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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func TestCreateFallbackInflater_File(t *testing.T) {
	//Test the creation of a fallback inflater, which primarily uses API inflater
	//and uses Daisy inflater as a fallback.
	//TODO: remove SkipNow once inflater is switched to the fallback variant (not shadow)
	t.SkipNow()

	inflater, err := newInflater(ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, nil, &storage.Client{}, mockInspector{
		t:                 t,
		expectedReference: "gs://bucket/vmdk",
		errorToReturn:     nil,
		metaToReturn:      imagefile.Metadata{},
	}, nil)
	assert.NoError(t, err)
	facade, ok := inflater.(*inflaterFacade)
	assert.True(t, ok)

	daisyInflater, ok := facade.daisyInflater.(*daisyInflater)
	assert.True(t, ok)
	assert.Equal(t, "zones/us-west1-c/disks/disk-1234", daisyInflater.inflatedDiskURI)
	daisyutils.CheckWorkflow(daisyInflater.worker, func(wf *daisy.Workflow) {
		assert.Equal(t, "gs://bucket/vmdk", wf.Vars["source_disk_file"].Value)
		assert.Equal(t, "projects/subnet/subnet", wf.Vars["import_subnet"].Value)
		assert.Equal(t, "projects/network/network", wf.Vars["import_network"].Value)

		network := getWorkerNetwork(t, wf)
		assert.Nil(t, network.AccessConfigs, "AccessConfigs must be nil to allow ExternalIP to be allocated.")
	})

	apiInflater, ok := facade.apiInflater.(*apiInflater)
	assert.True(t, ok)
	assert.NotContains(t, apiInflater.guestOsFeatures,
		&compute.GuestOsFeature{Type: "UEFI_COMPATIBLE"})
}

func TestCreateShadowTestInflater_File(t *testing.T) {
	//Test the creation of a shadow test inflater, which primarily uses Daisy
	//inflater while API inflater is used only to verify its output against Daisy
	//inflater
	//TODO: remove/disable this test once API inflater is the default (fallback mode)

	inflater, err := newInflater(ImageImportRequest{
		Source:      fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:      "projects/subnet/subnet",
		Network:     "projects/network/network",
		Zone:        "us-west1-c",
		ExecutionID: "1234",
		Tool: daisyutils.Tool{
			ResourceLabelName: "image-import",
		},
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, nil, &storage.Client{}, mockInspector{
		t:                 t,
		expectedReference: "gs://bucket/vmdk",
		errorToReturn:     nil,
		metaToReturn:      imagefile.Metadata{},
	}, nil)
	assert.NoError(t, err)
	facade, ok := inflater.(*shadowTestInflaterFacade)
	assert.True(t, ok)

	daisyInflater, ok := facade.mainInflater.(*daisyInflater)
	assert.True(t, ok)
	daisyutils.CheckWorkflow(daisyInflater.worker, func(wf *daisy.Workflow) {
		assert.Equal(t, "zones/us-west1-c/disks/disk-1234", daisyInflater.inflatedDiskURI)
		assert.Equal(t, "gs://bucket/vmdk", wf.Vars["source_disk_file"].Value)
		assert.Equal(t, "projects/subnet/subnet", wf.Vars["import_subnet"].Value)
		assert.Equal(t, "projects/network/network", wf.Vars["import_network"].Value)

		network := getWorkerNetwork(t, wf)
		assert.Nil(t, network.AccessConfigs, "AccessConfigs must be nil to allow ExternalIP to be allocated.")
	})

	apiInflater, ok := facade.shadowInflater.(*apiInflater)
	assert.True(t, ok)
	assert.NotContains(t, apiInflater.guestOsFeatures,
		&compute.GuestOsFeature{Type: "UEFI_COMPATIBLE"})
}

func TestCreateInflater_Image(t *testing.T) {
	inflater, err := newInflater(ImageImportRequest{
		Source:      imageSource{uri: "projects/test/uri/image"},
		Zone:        "us-west1-b",
		ExecutionID: "1234",
		WorkflowDir: daisyWorkflows,
		Tool: daisyutils.Tool{
			ResourceLabelName: "image-import",
		},
	}, nil, &storage.Client{}, nil, nil)
	assert.NoError(t, err)
	realInflater, ok := inflater.(*daisyInflater)
	assert.True(t, ok)
	daisyutils.CheckWorkflow(realInflater.worker, func(wf *daisy.Workflow) {
		assert.Equal(t, "zones/us-west1-b/disks/disk-1234", realInflater.inflatedDiskURI)
		assert.Equal(t, "projects/test/uri/image", wf.Vars["source_image"].Value)
		inflatedDisk := getDisk(wf, 0)
		assert.Contains(t, inflatedDisk.Licenses,
			"projects/compute-image-tools/global/licenses/virtual-disk-import")
	})

}

func TestInflaterFacade_SuccessOnApiInflater(t *testing.T) {
	facade := inflaterFacade{
		apiInflater: &mockInflater{
			pd: persistentDisk{
				uri: "disk1",
			},
		},
		daisyInflater: &mockInflater{
			pd: persistentDisk{
				uri: "disk2",
			},
		},
		logger: logging.NewToolLogger(t.Name()),
	}

	pd, _, err := facade.Inflate()
	assert.NoError(t, err)
	assert.Equal(t, "disk1", pd.uri)
}

func TestInflaterFacade_FailedOnApiInflater(t *testing.T) {
	apiError := fmt.Errorf("any failure")
	facade := inflaterFacade{
		apiInflater: &mockInflater{
			err: apiError,
		},
		daisyInflater: &mockInflater{
			pd: persistentDisk{
				uri: "disk2",
			},
		},
		logger: logging.NewToolLogger(t.Name()),
	}

	_, _, err := facade.Inflate()
	assert.Equal(t, apiError, err)
}

func TestInflaterFacade_SuccessOnDaisyInflater(t *testing.T) {
	apiError := fmt.Errorf("failed on INVALID_IMAGE_FILE")
	facade := inflaterFacade{
		apiInflater: &mockInflater{
			err: apiError,
		},
		daisyInflater: &mockInflater{
			pd: persistentDisk{
				uri: "disk2",
			},
		},
		logger: logging.NewToolLogger(t.Name()),
	}

	pd, _, err := facade.Inflate()
	assert.NoError(t, err)
	assert.Equal(t, "disk2", pd.uri)
}

func TestInflaterFacade_FailedOnDaisyInflater(t *testing.T) {
	apiError := fmt.Errorf("failed on INVALID_IMAGE_FILE")
	daisyError := fmt.Errorf("failed on daisy")
	facade := inflaterFacade{
		apiInflater: &mockInflater{
			err: apiError,
		},
		daisyInflater: &mockInflater{
			err: daisyError,
		},
		logger: logging.NewToolLogger(t.Name()),
	}

	_, _, err := facade.Inflate()
	assert.Equal(t, daisyError, err)
}
