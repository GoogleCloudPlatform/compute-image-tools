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
	"testing"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

func TestCleaner(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.Oauth = ""
	project := params.Project
	region := "us-central1"

	instance := &compute.Instance{
		Disks: []*compute.AttachedDisk{
			{
				Source:     fmt.Sprintf("projects/%v/zones/us-central1-c/disks/disk-a", project),
				DeviceName: "disk-a-dev",
				Mode:       "READ_WRITE",
			},
			{
				Source:     fmt.Sprintf("projects/%v/zones/us-central1-c/disks/disk-b", project),
				DeviceName: "disk-b-dev",
				Mode:       "READ_WRITE",
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
	mockComputeClient := mocks.NewMockClient(mockCtrl)
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
	mockComputeClient.EXPECT().ListInstances(project, params.Zone).Return([]*compute.Instance{instance}, nil).AnyTimes()
	mockComputeClient.EXPECT().ListNetworks(project).Return([]*compute.Network{{Name: "a-network", SelfLink: params.Network}}, nil).AnyTimes()
	mockComputeClient.EXPECT().ListSubnetworks(project, region).Return([]*compute.Subnetwork{{Name: "a-subnet", Region: region, SelfLink: params.Subnet}}, nil).AnyTimes()
	mockComputeClient.EXPECT().GetInstance(project, params.Zone, params.InstanceName).Return(instance, nil).AnyTimes()
	mockComputeClient.EXPECT().StartInstance(project, params.Zone, params.InstanceName).Return(nil)

	for diskIndex := range disks {
		mockComputeClient.EXPECT().AttachDisk(project, params.Zone, params.InstanceName, instance.Disks[diskIndex]).Return(nil)
	}

	mockClientSetter := func(w *daisy.Workflow) {
		w.ComputeClient = mockComputeClient
		w.StorageClient = testGCSClient
	}
	instanceExportCleaner := &instanceExportCleanerImpl{}
	instanceExportCleaner.wfPreRunCallback = mockClientSetter

	err = instanceExportCleaner.Clean(instance, params)
	assert.Nil(t, err)
}
