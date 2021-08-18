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

package ovfexporter

import (
	"fmt"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
)

func TestDiskExporter_HappyPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.Oauth = ""
	params.WorkflowDir = "../../../daisy_workflows/"

	project := params.Project
	region := "us-central1"

	instance := &compute.Instance{
		Disks: []*compute.AttachedDisk{
			{
				Source:     fmt.Sprintf("projects/%v/zones/us-central1-c/disks/disk-a", project),
				DeviceName: "dska",
			},
			{
				Source:     fmt.Sprintf("projects/%v/zones/us-central1-c/disks/disk-b", project),
				DeviceName: params.OvfName + "-dskb",
				//DeviceName:"dskb",
			},
		},
		Zone: params.Zone,
		Name: params.InstanceName,
	}
	disks := []*compute.Disk{
		{
			SelfLink: fmt.Sprintf("projects/%v/zones/us-central1-c/disks/disk-a", project),
			Name:     "disk-a",
			Zone:     params.Zone,
		},
		{
			SelfLink: fmt.Sprintf("projects/%v/zones/us-central1-c/disks/disk-b", project),
			Name:     "disk-b",
			Zone:     params.Zone,
		},
	}
	diskFileSizes := []int64{15, 70}
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	testGCSClient, _, err := newTestGCSClient() // used by Daisy
	if err != nil {
		t.Fail()
	}
	mockComputeClient.EXPECT().GetProject(project).Return(&compute.Project{Name: project}, nil).AnyTimes()
	mockComputeClient.EXPECT().ListZones(project).Return([]*compute.Zone{{Id: 0, Name: params.Zone}}, nil).AnyTimes()
	mockComputeClient.EXPECT().GetImageFromFamily("compute-image-tools", "debian-9-worker").Return(&compute.Image{Name: "debian-9-worker"}, nil).AnyTimes()
	mockComputeClient.EXPECT().ListDisks(project, params.Zone).Return(disks, nil).AnyTimes()
	mockComputeClient.EXPECT().ListMachineTypes(project, params.Zone).Return([]*compute.MachineType{}, nil).AnyTimes()
	mockComputeClient.EXPECT().GetMachineType(project, params.Zone, gomock.Any()).Return(&compute.MachineType{Name: "n1-highcpu-4"}, nil).AnyTimes()
	mockComputeClient.EXPECT().ListInstances(project, params.Zone).Return([]*compute.Instance{}, nil).AnyTimes()
	mockComputeClient.EXPECT().ListNetworks(project).Return([]*compute.Network{{Name: "a-network", SelfLink: params.Network}}, nil).AnyTimes()
	mockComputeClient.EXPECT().ListSubnetworks(project, region).Return([]*compute.Subnetwork{{Name: "a-subnet", Region: region, SelfLink: params.Subnet}}, nil).AnyTimes()
	mockComputeClient.EXPECT().CreateDisk(project, params.Zone, gomock.Any()).Return(nil).AnyTimes()
	mockComputeClient.EXPECT().CreateInstance(project, params.Zone, gomock.Any()).Return(nil).AnyTimes()

	for diskIndex, disk := range disks {
		exporterInstanceNamePrefix := fmt.Sprintf("inst-export-disk-%v-%v", diskIndex, instance.Disks[diskIndex].DeviceName)
		exporterInstanceDiskPrefix := fmt.Sprintf("disk-export-disk-%v-%v-", diskIndex, instance.Disks[diskIndex].DeviceName)
		mockComputeClient.EXPECT().GetSerialPortOutput(project, params.Zone, StartsWith(exporterInstanceNamePrefix), int64(1), int64(0)).Return(&compute.SerialPortOutput{Contents: "export success\n", Next: 0}, nil).AnyTimes()
		mockComputeClient.EXPECT().GetInstance(project, params.Zone, StartsWith(exporterInstanceNamePrefix)).Return(&compute.Instance{}, nil).AnyTimes()
		mockComputeClient.EXPECT().DeleteInstance(project, params.Zone, StartsWith(exporterInstanceNamePrefix)).Return(nil)
		mockComputeClient.EXPECT().DeleteDisk(project, params.Zone, StartsWith(exporterInstanceDiskPrefix)).Return(nil).AnyTimes()
		mockComputeClient.EXPECT().GetDisk(project, params.Zone, disk.Name).Return(disk, nil).AnyTimes()
		mockStorageClient.EXPECT().GetObjectAttrs(
			"ovfbucket",
			fmt.Sprintf("OVFpath/%v", diskFileName(params.OvfName, instance.Disks[diskIndex].DeviceName, params.DiskExportFormat)),
		).Return(&storage.ObjectAttrs{Size: diskFileSizes[diskIndex]}, nil).AnyTimes()
	}

	mockClientSetter := func(w *daisy.Workflow) {
		w.ComputeClient = mockComputeClient
		w.StorageClient = testGCSClient
	}
	diskExporter := &instanceDisksExporterImpl{
		computeClient: mockComputeClient,
		storageClient: mockStorageClient,
	}
	diskExporter.wfPreRunCallback = mockClientSetter

	exportedDisks, err := diskExporter.Export(instance, params)

	assert.Nil(t, err)
	assert.NotNil(t, exportedDisks)
	assert.Equal(t, len(disks), len(exportedDisks))
	for diskIndex, exportedDisk := range exportedDisks {
		assert.Equal(t, disks[diskIndex], exportedDisk.Disk)
		assert.Equal(t, instance.Disks[diskIndex], exportedDisk.AttachedDisk)
		assert.Equal(t, fmt.Sprintf("%v%v", params.DestinationDirectory, diskFileName(params.OvfName, instance.Disks[diskIndex].DeviceName, params.DiskExportFormat)), exportedDisk.GcsPath)
		assert.NotNil(t, exportedDisk.GcsFileAttrs)
		assert.Equal(t, diskFileSizes[diskIndex], exportedDisk.GcsFileAttrs.Size)
	}
}

func diskFileName(ovfName, deviceName, fileFormat string) string {
	if strings.HasPrefix(deviceName, ovfName) {
		return fmt.Sprintf("%v.%v", deviceName, fileFormat)
	}
	return fmt.Sprintf("%v-%v.%v", ovfName, deviceName, fileFormat)
}
