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
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
	"testing"
)

func TestAddDiskImportSteps(t *testing.T) {

	w := daisy.New()
	diskInfos := []ovfutils.DiskInfo{
		{"gs://abucket/apath/disk1.vmdk", 20},
		{"gs://abucket/apath/disk2.vmdk", 1},
	}

	w.Steps = map[string]*daisy.Step{
		"create-boot-disk": {
			CreateDisks: &daisy.CreateDisks{
				{
					Disk: compute.Disk{
						Name:        "${instance_name}-boot-disk",
						SourceImage: "${boot_image_name}",
						Type:        "pd-ssd",
					},
				},
			},
		},
		"create-instance": {
			CreateInstances: &daisy.CreateInstances{
				{
					Instance: compute.Instance{
						Disks:  []*compute.AttachedDisk{{Source: "key1"}},
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
	assert.Equal(t, 2+3*len(diskInfos), len(w.Steps))

	assert.Equal(t, []string{"create-boot-disk"}, w.Dependencies["setup-data-disk-1"])
	assert.Equal(t, []string{"setup-data-disk-1"}, w.Dependencies["create-data-disk-import-instance-1"])
	assert.Equal(t, []string{"create-data-disk-import-instance-1"}, w.Dependencies["wait-for-data-disk-1-signal"])
	assert.Equal(t, []string{"wait-for-data-disk-1-signal"}, w.Dependencies["setup-data-disk-2"])
	assert.Equal(t, []string{"setup-data-disk-2"}, w.Dependencies["create-data-disk-import-instance-2"])
	assert.Equal(t, []string{"create-data-disk-import-instance-2"}, w.Dependencies["wait-for-data-disk-2-signal"])
	assert.Equal(t, []string{"wait-for-data-disk-2-signal"}, w.Dependencies["create-instance"])

	//TODO: further assertion to validate the workflow if deemed necessary

}
