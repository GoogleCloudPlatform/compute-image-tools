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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/stretchr/testify/assert"
)

func TestCreateInflater_File(t *testing.T) {
	inflater, err := createInflater(ImportArguments{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  "testdata",
	},
		nil,
		storage.Client{},
		mockInspector{
			t:                 t,
			expectedReference: "gs://bucket/vmdk",
			errorToReturn:     nil,
			metaToReturn:      imagefile.Metadata{},
		})
	assert.NoError(t, err)
	realInflater, ok := inflater.(*inflaterFacade)
	assert.True(t, ok)

	mainInflater, ok := realInflater.mainInflater.(*daisyInflater)
	assert.True(t, ok)
	assert.Equal(t, "zones/us-west1-c/disks/disk-1234", mainInflater.inflatedDiskURI)
	assert.Equal(t, "gs://bucket/vmdk", mainInflater.wf.Vars["source_disk_file"].Value)
	assert.Equal(t, "projects/subnet/subnet", mainInflater.wf.Vars["import_subnet"].Value)
	assert.Equal(t, "projects/network/network", mainInflater.wf.Vars["import_network"].Value)

	network := getWorkerNetwork(t, mainInflater.wf)
	assert.Nil(t, network.AccessConfigs, "AccessConfigs must be nil to allow ExternalIP to be allocated.")

	_, ok = realInflater.shadowInflater.(*apiInflater)
	assert.True(t, ok)
}

func TestCreateInflater_Image(t *testing.T) {
	inflater, err := createInflater(ImportArguments{
		Source:      imageSource{uri: "projects/test/uri/image"},
		Zone:        "us-west1-b",
		ExecutionID: "1234",
		WorkflowDir: "testdata",
	}, nil, storage.Client{}, nil)
	assert.NoError(t, err)
	realInflater, ok := inflater.(*daisyInflater)
	assert.True(t, ok)
	assert.Equal(t, "zones/us-west1-b/disks/disk-1234", realInflater.inflatedDiskURI)
	assert.Equal(t, "projects/test/uri/image", realInflater.wf.Vars["source_image"].Value)
	inflatedDisk := getDisk(realInflater.wf, 0)
	assert.Contains(t, inflatedDisk.Licenses,
		"projects/compute-image-tools/global/licenses/virtual-disk-import")
}
