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
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	mock_disk "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	ovfexportmocks "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

var (
	exportedDisks = []*ovfexportdomain.ExportedDisk{
		{
			Disk:         &compute.Disk{Name: "bootdisk", SizeGb: 10},
			AttachedDisk: &compute.AttachedDisk{Boot: true},
			GcsFileAttrs: &storage.ObjectAttrs{Size: 3 * bytesPerGB},
		},
		{
			Disk:         &compute.Disk{Name: "datadisk1", SizeGb: 20},
			AttachedDisk: &compute.AttachedDisk{Boot: false},
			GcsFileAttrs: &storage.ObjectAttrs{Size: 7 * bytesPerGB},
		},
		{
			Disk:         &compute.Disk{Name: "datadisk2", SizeGb: 300},
			AttachedDisk: &compute.AttachedDisk{Boot: false},
			GcsFileAttrs: &storage.ObjectAttrs{Size: 90*bytesPerGB + 1},
		},
	}
)

func TestRun_HappyPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	instance := &compute.Instance{}

	inspectionResults := &pb.InspectionResults{
		OsRelease: &pb.OsRelease{
			Architecture: pb.Architecture_X86,
			Distro:       "ubuntu",
			MajorVersion: "16",
			MinorVersion: "4",
		},
	}

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().User(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(params.Project, params.Zone, params.InstanceName).Return(instance, nil)

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	mockInstanceExportPreparer := ovfexportmocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Return(nil)

	mockInstanceDisksExporter := ovfexportmocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInstanceDisksExporter.EXPECT().Export(instance, params).Return(exportedDisks, nil)

	mockInspector := mock_disk.NewMockInspector(mockCtrl)
	mockInspector.EXPECT().Inspect(
		fmt.Sprintf("projects/%v/zones/%v/disks/%v", params.Project, params.Zone, "bootdisk")).Return(inspectionResults, nil)

	mockOvfDescriptorGenerator := ovfexportmocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfDescriptorGenerator.EXPECT().GenerateAndWriteOVFDescriptor(instance, exportedDisks, "ovfbucket", "OVFpath/", params.OvfName+".ovf", inspectionResults).Return(nil)

	mockOvfManifestGenerator := ovfexportmocks.NewMockOvfManifestGenerator(mockCtrl)
	mockOvfManifestGenerator.EXPECT().GenerateAndWriteToGCS(params.DestinationDirectory, params.OvfName+".mf").Return(nil)

	mockInstanceExportCleaner := ovfexportmocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)
	mockStorageClient.EXPECT().Close().Return(nil)

	exporter := &OVFExporter{
		storageClient:          mockStorageClient,
		computeClient:          mockComputeClient,
		mgce:                   mockMetadataGce,
		bucketIteratorCreator:  mockBucketIteratorCreator,
		Logger:                 mockLogger,
		params:                 params,
		loggableBuilder:        service.NewOvfExportLoggableBuilder(),
		ovfDescriptorGenerator: mockOvfDescriptorGenerator,
		manifestFileGenerator:  mockOvfManifestGenerator,
		inspector:              mockInspector,
		instanceDisksExporter:  mockInstanceDisksExporter,
		instanceExportPreparer: mockInstanceExportPreparer,
		instanceExportCleaner:  mockInstanceExportCleaner,
	}
	err := exporter.Run(context.Background())
	assert.Nil(t, err)
}

func TestRun_DontRunDiskExporterIfPreparerTimedOut(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.Timeout = 100 * time.Millisecond
	instance := &compute.Instance{}

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().User(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(params.Project, params.Zone, params.InstanceName).Return(instance, nil)
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	preparerCancelChan := make(chan bool)
	mockInstanceExportPreparer := ovfexportmocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Do(
		func(instance *compute.Instance, params *ovfexportdomain.OVFExportArgs) {
			sleepStep(preparerCancelChan)
		}).Return(nil)
	mockInstanceExportPreparer.EXPECT().Cancel("timed-out").Do(func(_ string) { preparerCancelChan <- true }).Return(true)

	mockInstanceDisksExporter := ovfexportmocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInspector := mock_disk.NewMockInspector(mockCtrl)
	mockOvfDescriptorGenerator := ovfexportmocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfManifestGenerator := ovfexportmocks.NewMockOvfManifestGenerator(mockCtrl)

	mockInstanceExportCleaner := ovfexportmocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)

	exporter := &OVFExporter{
		storageClient:          mockStorageClient,
		computeClient:          mockComputeClient,
		mgce:                   mockMetadataGce,
		bucketIteratorCreator:  mockBucketIteratorCreator,
		Logger:                 mockLogger,
		params:                 params,
		loggableBuilder:        service.NewOvfExportLoggableBuilder(),
		ovfDescriptorGenerator: mockOvfDescriptorGenerator,
		manifestFileGenerator:  mockOvfManifestGenerator,
		inspector:              mockInspector,
		instanceDisksExporter:  mockInstanceDisksExporter,
		instanceExportPreparer: mockInstanceExportPreparer,
		instanceExportCleaner:  mockInstanceExportCleaner,
	}
	runAndAssertTimeout(t, exporter)

}

func TestRun_DontRunInspectorIfDiskExporterTimedOut(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.Timeout = 100 * time.Millisecond
	instance := &compute.Instance{}

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().User(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(params.Project, params.Zone, params.InstanceName).Return(instance, nil)
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	mockInstanceExportPreparer := ovfexportmocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Return(nil)

	diskExporterCancelChan := make(chan bool)
	mockInstanceDisksExporter := ovfexportmocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInstanceDisksExporter.EXPECT().Export(instance, params).Do(
		func(instance *compute.Instance, params *ovfexportdomain.OVFExportArgs) {
			sleepStep(diskExporterCancelChan)
		}).Return(nil, nil)
	mockInstanceDisksExporter.EXPECT().Cancel("timed-out").Do(func(_ string) { diskExporterCancelChan <- true }).Return(true)

	mockInspector := mock_disk.NewMockInspector(mockCtrl)
	mockOvfDescriptorGenerator := ovfexportmocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfManifestGenerator := ovfexportmocks.NewMockOvfManifestGenerator(mockCtrl)

	mockInstanceExportCleaner := ovfexportmocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)

	exporter := &OVFExporter{
		storageClient:          mockStorageClient,
		computeClient:          mockComputeClient,
		mgce:                   mockMetadataGce,
		bucketIteratorCreator:  mockBucketIteratorCreator,
		Logger:                 mockLogger,
		params:                 params,
		loggableBuilder:        service.NewOvfExportLoggableBuilder(),
		ovfDescriptorGenerator: mockOvfDescriptorGenerator,
		manifestFileGenerator:  mockOvfManifestGenerator,
		inspector:              mockInspector,
		instanceDisksExporter:  mockInstanceDisksExporter,
		instanceExportPreparer: mockInstanceExportPreparer,
		instanceExportCleaner:  mockInstanceExportCleaner,
	}
	runAndAssertTimeout(t, exporter)
}

func TestRun_DontRunDescriptorGeneratorIfInspectorTimedOut(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.Timeout = 100 * time.Millisecond
	instance := &compute.Instance{}

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().User(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(params.Project, params.Zone, params.InstanceName).Return(instance, nil)
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	mockInstanceExportPreparer := ovfexportmocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Return(nil)

	mockInstanceDisksExporter := ovfexportmocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInstanceDisksExporter.EXPECT().Export(instance, params).Return(exportedDisks, nil)

	inspectorCancelChan := make(chan bool)
	mockInspector := mock_disk.NewMockInspector(mockCtrl)
	mockInspector.EXPECT().Inspect(
		fmt.Sprintf("projects/%v/zones/%v/disks/%v", params.Project, params.Zone, "bootdisk")).Do(func(reference string) { sleepStep(inspectorCancelChan) }).Return(&pb.InspectionResults{}, nil)
	mockInspector.EXPECT().Cancel("timed-out").Do(func(_ string) { inspectorCancelChan <- true }).Return(true)

	mockOvfDescriptorGenerator := ovfexportmocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfManifestGenerator := ovfexportmocks.NewMockOvfManifestGenerator(mockCtrl)

	mockInstanceExportCleaner := ovfexportmocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)

	exporter := &OVFExporter{
		storageClient:          mockStorageClient,
		computeClient:          mockComputeClient,
		mgce:                   mockMetadataGce,
		bucketIteratorCreator:  mockBucketIteratorCreator,
		Logger:                 mockLogger,
		params:                 params,
		loggableBuilder:        service.NewOvfExportLoggableBuilder(),
		ovfDescriptorGenerator: mockOvfDescriptorGenerator,
		manifestFileGenerator:  mockOvfManifestGenerator,
		inspector:              mockInspector,
		instanceDisksExporter:  mockInstanceDisksExporter,
		instanceExportPreparer: mockInstanceExportPreparer,
		instanceExportCleaner:  mockInstanceExportCleaner,
	}
	runAndAssertTimeout(t, exporter)
}

func TestRun_DontRunManifestGeneratorIfDescriptorGeneratorTimedOut(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.Timeout = 100 * time.Millisecond
	instance := &compute.Instance{}
	inspectionResults := &pb.InspectionResults{
		OsRelease: &pb.OsRelease{
			Architecture: pb.Architecture_X86,
			Distro:       "ubuntu",
			MajorVersion: "16",
			MinorVersion: "4",
		},
	}
	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().User(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(params.Project, params.Zone, params.InstanceName).Return(instance, nil)
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	mockInstanceExportPreparer := ovfexportmocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Return(nil)

	mockInstanceDisksExporter := ovfexportmocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInstanceDisksExporter.EXPECT().Export(instance, params).Return(exportedDisks, nil)

	mockInspector := mock_disk.NewMockInspector(mockCtrl)
	mockInspector.EXPECT().Inspect(
		fmt.Sprintf("projects/%v/zones/%v/disks/%v", params.Project, params.Zone, "bootdisk")).Return(inspectionResults, nil)

	descriptorGeneratorCancelChan := make(chan bool)
	mockOvfDescriptorGenerator := ovfexportmocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfDescriptorGenerator.EXPECT().GenerateAndWriteOVFDescriptor(
		instance, exportedDisks, "ovfbucket", "OVFpath/", params.OvfName+".ovf", inspectionResults).Do(
		func(_ *compute.Instance, _ []*ovfexportdomain.ExportedDisk, _, _, _ string, _ *pb.InspectionResults) {
			sleepStep(descriptorGeneratorCancelChan)
		}).Return(nil)
	mockOvfDescriptorGenerator.EXPECT().Cancel("timed-out").Do(func(_ string) { descriptorGeneratorCancelChan <- true }).Return(true)

	mockOvfManifestGenerator := ovfexportmocks.NewMockOvfManifestGenerator(mockCtrl)

	mockInstanceExportCleaner := ovfexportmocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)

	exporter := &OVFExporter{
		storageClient:          mockStorageClient,
		computeClient:          mockComputeClient,
		mgce:                   mockMetadataGce,
		bucketIteratorCreator:  mockBucketIteratorCreator,
		Logger:                 mockLogger,
		params:                 params,
		loggableBuilder:        service.NewOvfExportLoggableBuilder(),
		ovfDescriptorGenerator: mockOvfDescriptorGenerator,
		manifestFileGenerator:  mockOvfManifestGenerator,
		inspector:              mockInspector,
		instanceDisksExporter:  mockInstanceDisksExporter,
		instanceExportPreparer: mockInstanceExportPreparer,
		instanceExportCleaner:  mockInstanceExportCleaner,
	}
	runAndAssertTimeout(t, exporter)
}

func TestRun_TimeOutOnManifestGeneratorTimingOut(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.Timeout = 100 * time.Millisecond
	instance := &compute.Instance{}
	inspectionResults := &pb.InspectionResults{
		OsRelease: &pb.OsRelease{
			Architecture: pb.Architecture_X86,
			Distro:       "ubuntu",
			MajorVersion: "16",
			MinorVersion: "4",
		},
	}
	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().User(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(params.Project, params.Zone, params.InstanceName).Return(instance, nil)
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	mockInstanceExportPreparer := ovfexportmocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Return(nil)

	mockInstanceDisksExporter := ovfexportmocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInstanceDisksExporter.EXPECT().Export(instance, params).Return(exportedDisks, nil)

	mockInspector := mock_disk.NewMockInspector(mockCtrl)
	mockInspector.EXPECT().Inspect(
		fmt.Sprintf("projects/%v/zones/%v/disks/%v", params.Project, params.Zone, "bootdisk")).Return(inspectionResults, nil)

	mockOvfDescriptorGenerator := ovfexportmocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfDescriptorGenerator.EXPECT().GenerateAndWriteOVFDescriptor(
		instance, exportedDisks, "ovfbucket", "OVFpath/", params.OvfName+".ovf", inspectionResults).Return(nil)

	manifestGeneratorCancelChan := make(chan bool)
	mockOvfManifestGenerator := ovfexportmocks.NewMockOvfManifestGenerator(mockCtrl)
	mockOvfManifestGenerator.EXPECT().GenerateAndWriteToGCS(params.DestinationDirectory, params.OvfName+".mf").Do(
		func(_, _ string) { sleepStep(manifestGeneratorCancelChan) }).Return(nil)
	mockOvfManifestGenerator.EXPECT().Cancel("timed-out").Do(
		func(_ string) { manifestGeneratorCancelChan <- true }).Return(true)

	mockInstanceExportCleaner := ovfexportmocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)

	exporter := &OVFExporter{
		storageClient:          mockStorageClient,
		computeClient:          mockComputeClient,
		mgce:                   mockMetadataGce,
		bucketIteratorCreator:  mockBucketIteratorCreator,
		Logger:                 mockLogger,
		params:                 params,
		loggableBuilder:        service.NewOvfExportLoggableBuilder(),
		ovfDescriptorGenerator: mockOvfDescriptorGenerator,
		manifestFileGenerator:  mockOvfManifestGenerator,
		inspector:              mockInspector,
		instanceDisksExporter:  mockInstanceDisksExporter,
		instanceExportPreparer: mockInstanceExportPreparer,
		instanceExportCleaner:  mockInstanceExportCleaner,
	}
	runAndAssertTimeout(t, exporter)
}

func runAndAssertTimeout(t *testing.T, exporter *OVFExporter) {
	start := time.Now()
	err := exporter.Run(context.Background())
	duration := time.Since(start)

	assert.NotNil(t, err)
	// to ensure disk exporter got interrupted and didn't run the full 5 seconds
	assert.True(t, duration < time.Duration(1)*time.Second)
}

func sleepStep(cancelChan chan bool) {
	select {
	case <-cancelChan:
		break
	case <-time.After(5 * time.Second):
		break
	}
}

func TestValidateAndPopulateParams(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	paramValidator := ovfexportmocks.NewMockOvfExportParamValidator(mockCtrl)
	paramValidator.EXPECT().ValidateAndParseParams(params).Return(nil)
	paramPopulator := ovfexportmocks.NewMockOvfExportParamPopulator(mockCtrl)
	paramPopulator.EXPECT().Populate(params).Return(nil)
	err := validateAndPopulateParams(params, paramValidator, paramPopulator)
	assert.Nil(t, err)
}

func TestValidateAndPopulateParams_ErrorOnValidate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.MachineImageName = "also-machine-image-name-which-is-invalid"
	paramValidator := ovfexportmocks.NewMockOvfExportParamValidator(mockCtrl)
	paramValidator.EXPECT().ValidateAndParseParams(params).Return(fmt.Errorf("validation error"))
	err := validateAndPopulateParams(params, paramValidator, nil)
	assert.NotNil(t, err)
}

func TestValidateAndPopulateParams_ErrorOnPopulate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	paramValidator := ovfexportmocks.NewMockOvfExportParamValidator(mockCtrl)
	paramValidator.EXPECT().ValidateAndParseParams(params).Return(nil)

	populatorError := fmt.Errorf("populator error")
	paramPopulator := ovfexportmocks.NewMockOvfExportParamPopulator(mockCtrl)
	paramPopulator.EXPECT().Populate(params).Return(populatorError)

	err := validateAndPopulateParams(params, paramValidator, paramPopulator)
	assert.Equal(t, populatorError, err)
}
