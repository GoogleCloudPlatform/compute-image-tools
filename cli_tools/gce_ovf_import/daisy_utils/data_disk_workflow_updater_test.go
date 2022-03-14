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
	"fmt"
	"testing"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	ovfutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_utils"
)

func TestAddDiskImportSteps(t *testing.T) {
	w := createBaseImportWorkflow("an_instance")
	diskInfos := []ovfutils.DiskInfo{
		{FilePath: "gs://abucket/apath/disk1.vmdk", SizeInGB: 20},
		{FilePath: "gs://abucket/apath/disk2.vmdk", SizeInGB: 1},
	}

	AddDiskImportSteps(w, diskInfos)

	assert.NotNil(t, w)

	// 2 for create boot disk step and create instance from the base workflow
	// 4 for each disk (setup-data-disk, create-data-disk-import-instance,
	// wait-for-data-disk-signal and delete-data-disk-import-instance)
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

	assert.Equal(t, 3, len(*w.Steps["setup-data-disk-1"].CreateDisks))
	assert.Equal(t, "10", (*w.Steps["setup-data-disk-1"].CreateDisks)[0].SizeGb)
	assert.Equal(t, "10", (*w.Steps["setup-data-disk-1"].CreateDisks)[1].SizeGb)
	assert.Equal(t, "10", (*w.Steps["setup-data-disk-1"].CreateDisks)[2].SizeGb)

	assert.Equal(t, 3, len(*w.Steps["setup-data-disk-2"].CreateDisks))
	assert.Equal(t, "10", (*w.Steps["setup-data-disk-2"].CreateDisks)[0].SizeGb)
	assert.Equal(t, "10", (*w.Steps["setup-data-disk-2"].CreateDisks)[1].SizeGb)
	assert.Equal(t, "10", (*w.Steps["setup-data-disk-2"].CreateDisks)[2].SizeGb)

	assert.Equal(t,
		[]*compute.AttachedDisk{
			{Source: "boot_disk", Boot: true},
			{Source: "an_instance-1", AutoDelete: true},
			{Source: "an_instance-2", AutoDelete: true},
		},
		(*w.Steps[createInstanceStepName].CreateInstances).Instances[0].Disks)

	assert.Equal(t, diskInfos[0].FilePath, getMetadataValue((*w.Steps["create-data-disk-import-instance-1"].CreateInstances).Instances[0].Instance.Metadata, "source_disk_file"))
	assert.Equal(t, diskInfos[1].FilePath, getMetadataValue((*w.Steps["create-data-disk-import-instance-2"].CreateInstances).Instances[0].Instance.Metadata, "source_disk_file"))

	assert.Equal(t, w.DefaultTimeout, w.Steps["setup-data-disk-1"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["create-data-disk-import-instance-1"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["wait-for-data-disk-1-signal"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["delete-data-disk-1-import-instance"].Timeout)

	assert.Equal(t, w.DefaultTimeout, w.Steps["setup-data-disk-2"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["create-data-disk-import-instance-2"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["wait-for-data-disk-2-signal"].Timeout)
	assert.Equal(t, w.DefaultTimeout, w.Steps["delete-data-disk-2-import-instance"].Timeout)

	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-1"].CreateDisks)[0].Name))
	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-1"].CreateDisks)[1].Name))
	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-1"].CreateDisks)[2].Name))

	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-2"].CreateDisks)[0].Name))
	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-2"].CreateDisks)[1].Name))
	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-2"].CreateDisks)[2].Name))
}

func TestAddDiskImportStepsDiskNamesValidWhenInstanceNameLong(t *testing.T) {
	w := createBaseImportWorkflow("a-very-long-instance-name-that-is-at-the-limit-of-allowed-leng")

	diskInfos := []ovfutils.DiskInfo{
		{FilePath: "gs://abucket/apath/disk1.vmdk", SizeInGB: 20},
		{FilePath: "gs://abucket/apath/disk2.vmdk", SizeInGB: 1},
	}
	AddDiskImportSteps(w, diskInfos)

	assert.NotNil(t, w)
	assert.Equal(t, 2+4*len(diskInfos), len(w.Steps))

	assert.NotNil(t, w.Steps["setup-data-disk-1"])
	assert.NotNil(t, w.Steps["setup-data-disk-2"])

	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-1"].CreateDisks)[0].Name))
	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-1"].CreateDisks)[1].Name))
	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-1"].CreateDisks)[2].Name))

	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-2"].CreateDisks)[0].Name))
	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-2"].CreateDisks)[1].Name))
	assert.NoError(t, validation.ValidateRfc1035Label((*w.Steps["setup-data-disk-2"].CreateDisks)[2].Name))
}

func TestCreateDisksOnInstance(t *testing.T) {
	wfPath := "../../../daisy_workflows/ovf_import/create_instance.wf.json"
	wf, err := daisy.NewFromFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	imageURIs := []string{
		"uri-1",
		"uri-2",
	}
	createInstanceStep := wf.Steps["create-instance"].CreateInstances.Instances[0]
	CreateDisksOnInstance(createInstanceStep, "test-instance", imageURIs)
	for i, expectedDiskName := range []string{
		"test-instance-1",
		"test-instance-2",
	} {
		// Offset by one since template includes the bootdisk as the first element in disk lists.
		dataDiskIndex := i + 1
		dataDisk := createInstanceStep.Disks[dataDiskIndex]
		expectedSourceURI := imageURIs[i]
		assert.True(t, dataDisk.AutoDelete)
		assert.Equal(t, expectedDiskName, dataDisk.InitializeParams.DiskName)
		assert.Equal(t, expectedSourceURI, dataDisk.InitializeParams.SourceImage)
		assert.Equal(t, "pd-ssd", dataDisk.InitializeParams.DiskType)
	}
}

func TestAppendDisksToInstance(t *testing.T) {
	wfPath := "../../../daisy_workflows/ovf_import/create_instance.wf.json"
	wf, err := daisy.NewFromFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}

	disks := []domain.Disk{}

	for i := 0; i < 2; i++ {
		diskName := fmt.Sprintf("disk-name%d", i+1)
		disk, err := disk.NewDisk("project", "zone", diskName)
		assert.NoError(t, err)
		disks = append(disks, disk)
	}
	createInstanceStep := wf.Steps["create-instance"].CreateInstances.Instances[0]
	AppendDisksToInstance(createInstanceStep, disks)
	for i, disk := range disks {
		// Offset by one since template includes the bootdisk as the first element in disk lists.
		dataDiskIndex := i + 1
		dataDisk := createInstanceStep.Disks[dataDiskIndex]
		expectedSourceURI := disk.GetURI()
		assert.True(t, dataDisk.AutoDelete)
		assert.Equal(t, expectedSourceURI, dataDisk.Source)
	}
}

func getMetadataValue(metadata *compute.Metadata, key string) string {
	for _, metadataItem := range metadata.Items {
		if metadataItem.Key == key {
			return *metadataItem.Value
		}
	}
	return ""
}

func createBaseImportWorkflow(instanceName string) *daisy.Workflow {
	w := daisy.New()
	w.Vars["instance_name"] = daisy.Var{Value: instanceName}
	w.DefaultTimeout = "180m"

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
				Instances: []*daisy.Instance{
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
		},
	}
	w.Dependencies = map[string][]string{
		"create-instance": {"create-boot-disk"},
	}
	return w
}
