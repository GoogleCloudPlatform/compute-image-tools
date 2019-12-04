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
//  limitations under the License

package daisyovfutils

import (
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
)

func TestAddDiskImportSteps(t *testing.T) {

	w := daisy.New()
	w.Vars["instance_name"] = daisy.Var{Value: "an_instance"}
	w.DefaultTimeout = "180m"

	diskInfos := []ovfutils.DiskInfo{
		{FilePath: "gs://abucket/apath/disk1.vmdk", SizeInGB: 20},
		{FilePath: "gs://abucket/apath/disk2.vmdk", SizeInGB: 1},
	}

	w.Steps = map[string]*daisy.Step{
		"create-boot-disk": {
			CreateDisks: &daisy.CreateDisks{
				{
					Disk: compute.Disk{
						Name:        "instance-boot-disk",
						SourceImage: "source_image",
						Type:        "pd-ssd",
					},
				},
			},
		},
		"create-instance": {
			CreateInstances: &daisy.CreateInstances{
				{
					Instance: compute.Instance{
						Disks:  []*compute.AttachedDisk{{Source: "boot_disk", Boot: true}},
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Instance: compute.Instance{
						Disks: []*compute.AttachedDisk{{Source: "key2"}},
					},
				},
			},
		},
	}
	w.Dependencies = map[string][]string{
		"create-instance": {"create-boot-disk"},
	}

	AddDiskImportSteps(w, diskInfos)

	assert.NotNil(t, w)
	assert.Equal(t, 2+4*len(diskInfos), len(w.Steps))

	assert.NotNil(t, w.Steps["setup-data-disk-1"])
	assert.NotNil(t, w.Steps["create-data-disk-import-instance-1"])
	assert.NotNil(t, w.Steps["wait-for-data-disk-1-signal"])
	assert.NotNil(t, w.Steps["delete-data-disk-1-import-instance"])

	assert.NotNil(t, w.Steps["setup-data-disk-2"])
	assert.NotNil(t, w.Steps["create-data-disk-import-instance-2"])
	assert.NotNil(t, w.Steps["wait-for-data-disk-2-signal"])
	assert.NotNil(t, w.Steps["delete-data-disk-2-import-instance"])

	assert.Nil(t, w.Dependencies["setup-data-disk-1"])
	assert.Equal(t, []string{"setup-data-disk-1"}, w.Dependencies["create-data-disk-import-instance-1"])
	assert.Equal(t, []string{"create-data-disk-import-instance-1"}, w.Dependencies["wait-for-data-disk-1-signal"])
	assert.Equal(t, []string{"wait-for-data-disk-1-signal"}, w.Dependencies["delete-data-disk-1-import-instance"])

	assert.Nil(t, w.Dependencies["setup-data-disk-2"])
	assert.Equal(t, []string{"setup-data-disk-2"}, w.Dependencies["create-data-disk-import-instance-2"])
	assert.Equal(t, []string{"create-data-disk-import-instance-2"}, w.Dependencies["wait-for-data-disk-2-signal"])
	assert.Equal(t, []string{"wait-for-data-disk-2-signal"}, w.Dependencies["delete-data-disk-2-import-instance"])

	assert.Equal(t,
		[]string{"create-boot-disk", "delete-data-disk-1-import-instance", "delete-data-disk-2-import-instance"},
		w.Dependencies["create-instance"])

	assert.Equal(t, "10", (*w.Steps["setup-data-disk-1"].CreateDisks)[0].SizeGb)
	assert.Equal(t, "10", (*w.Steps["setup-data-disk-1"].CreateDisks)[1].SizeGb)

	assert.Equal(t, "10", (*w.Steps["setup-data-disk-2"].CreateDisks)[0].SizeGb)
	assert.Equal(t, "10", (*w.Steps["setup-data-disk-2"].CreateDisks)[1].SizeGb)

	assert.Equal(t,
		[]*compute.AttachedDisk{
			{Source: "boot_disk", Boot: true},
			{Source: "an_instance-data-disk-1", AutoDelete: true},
			{Source: "an_instance-data-disk-2", AutoDelete: true},
		},
		(*w.Steps[createInstanceStepName].CreateInstances)[0].Disks)

	assert.Equal(t, diskInfos[0].FilePath, getMetadataValue((*w.Steps["create-data-disk-import-instance-1"].CreateInstances)[0].Instance.Metadata, "source_disk_file"))
	assert.Equal(t, diskInfos[1].FilePath, getMetadataValue((*w.Steps["create-data-disk-import-instance-2"].CreateInstances)[0].Instance.Metadata, "source_disk_file"))

	assert.Equal(t, w.DefaultTimeout, w.Steps["setup-data-disk-1"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["create-data-disk-import-instance-1"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["wait-for-data-disk-1-signal"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["delete-data-disk-1-import-instance"].Timeout)

	assert.Equal(t, w.DefaultTimeout, w.Steps["setup-data-disk-2"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["create-data-disk-import-instance-2"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["wait-for-data-disk-2-signal"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["delete-data-disk-2-import-instance"].Timeout)
}

func getMetadataValue(metadata *compute.Metadata, key string) string {
	for _, metadataItem := range metadata.Items {
		if metadataItem.Key == key {
			return *metadataItem.Value
		}
	}
	return ""
}
