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

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
)

func TestCreateDaisyInflater_Image_HappyCase(t *testing.T) {
	inflater := createDaisyInflaterSafe(t, ImportArguments{
		Source:      imageSource{uri: "projects/test/uri/image"},
		Zone:        "us-west1-b",
		ExecutionID: "1234",
	})

	assert.Equal(t, "zones/us-west1-b/disks/disk-1234", inflater.uri)
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
		Source:      fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:      "projects/subnet/subnet",
		Network:     "projects/network/network",
		Zone:        "us-west1-c",
		ExecutionID: "1234",
	})

	assert.Equal(t, "zones/us-west1-c/disks/disk-1234", inflater.uri)
	assert.Equal(t, "gs://bucket/vmdk", inflater.wf.Vars["source_disk_file"].Value)
	assert.Equal(t, "projects/subnet/subnet", inflater.wf.Vars["import_subnet"].Value)
	assert.Equal(t, "projects/network/network", inflater.wf.Vars["import_network"].Value)
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

func createDaisyInflaterSafe(t *testing.T, spec ImportArguments) daisyInflater {
	inflater, err := createDaisyInflater(spec, "testdata/image_import")
	assert.NoError(t, err)
	realInflater, ok := inflater.(daisyInflater)
	assert.True(t, ok)
	return realInflater
}
