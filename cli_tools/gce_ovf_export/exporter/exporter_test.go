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
	commondisk "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
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
			GcsFileAttrs: &storage.ObjectAttrs{Size: 90 * bytesPerGB},
		},
	}
)

func TestRun_HappyPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportParams()
	instance := &compute.Instance{}

	inspectionResults := commondisk.InspectionResult{Architecture: "x86", Distro: "ubuntu", Major: "16", Minor: "4"}

	mockLogger := mocks.NewMockLoggerInterface(mockCtrl)
	mockLogger.EXPECT().Log(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(*params.Project, params.Zone, params.InstanceName).Return(instance, nil)

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	mockInstanceExportPreparer := mocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Return(nil)
	mockInstanceExportPreparer.EXPECT().TraceLogs().Return([]string{"preparer trace log"})

	mockInstanceDisksExporter := mocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInstanceDisksExporter.EXPECT().Export(instance, params).Return(exportedDisks, nil)
	mockInstanceDisksExporter.EXPECT().TraceLogs().Return([]string{"disk exporter trace log"})

	mockInspector := mocks.NewMockInspector(mockCtrl)
	mockInspector.EXPECT().Inspect(
		fmt.Sprintf("projects/%v/zones/%v/disks/%v", *params.Project, params.Zone, "bootdisk"), true).Return(inspectionResults, nil)
	mockInspector.EXPECT().TraceLogs().Return([]string{"inspector trace log"})

	mockOvfDescriptorGenerator := mocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfDescriptorGenerator.EXPECT().GenerateAndWriteOVFDescriptor(instance, exportedDisks, "ovfbucket", "OVFpath", &inspectionResults).Return(nil)

	mockOvfManifestGenerator := mocks.NewMockOvfManifestGenerator(mockCtrl)
	mockOvfManifestGenerator.EXPECT().GenerateAndWriteToGCS(params.DestinationURI, params.InstanceName).Return(nil)

	mockInstanceExportCleaner := mocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)
	mockInstanceExportCleaner.EXPECT().TraceLogs().Return([]string{"cleaner trace log"})
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
	loggable, err := exporter.Run(context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, loggable)

	serialLogs := loggable.ReadSerialPortLogs()
	assert.Equal(t, []string{"preparer trace log", "disk exporter trace log", "inspector trace log", "cleaner trace log"}, serialLogs)
	assert.Equal(t, []int64{10, 20, 300}, loggable.GetValueAsInt64Slice("source-size-gb"))
	assert.Equal(t, []int64{3, 7, 90}, loggable.GetValueAsInt64Slice("target-size-gb"))
}

func TestRun_DontRunDiskExporterIfPreparerTimedOut(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportParams()
	params.Timeout = 100 * time.Millisecond
	instance := &compute.Instance{}

	mockLogger := mocks.NewMockLoggerInterface(mockCtrl)
	mockLogger.EXPECT().Log(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(*params.Project, params.Zone, params.InstanceName).Return(instance, nil)
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	preparerCancelChan := make(chan bool)
	mockInstanceExportPreparer := mocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Do(
		func(instance *compute.Instance, params *ovfexportdomain.OVFExportParams) {
			sleepStep(preparerCancelChan)
		}).Return(nil)
	mockInstanceExportPreparer.EXPECT().Cancel("timed-out").Do(func(_ string) { preparerCancelChan <- true }).Return(true)
	mockInstanceExportPreparer.EXPECT().TraceLogs().Return([]string{"preparer trace log"})

	mockInstanceDisksExporter := mocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInspector := mocks.NewMockInspector(mockCtrl)
	mockOvfDescriptorGenerator := mocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfManifestGenerator := mocks.NewMockOvfManifestGenerator(mockCtrl)

	mockInstanceExportCleaner := mocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)
	mockInstanceExportCleaner.EXPECT().TraceLogs().Return([]string{"cleaner trace log"})

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

	params := ovfexportdomain.GetAllInstanceExportParams()
	params.Timeout = 100 * time.Millisecond
	instance := &compute.Instance{}

	mockLogger := mocks.NewMockLoggerInterface(mockCtrl)
	mockLogger.EXPECT().Log(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(*params.Project, params.Zone, params.InstanceName).Return(instance, nil)
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	mockInstanceExportPreparer := mocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Return(nil)
	mockInstanceExportPreparer.EXPECT().TraceLogs().Return([]string{"preparer trace log"})

	diskExporterCancelChan := make(chan bool)
	mockInstanceDisksExporter := mocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInstanceDisksExporter.EXPECT().Export(instance, params).Do(
		func(instance *compute.Instance, params *ovfexportdomain.OVFExportParams) {
			sleepStep(diskExporterCancelChan)
		}).Return(nil, nil)
	mockInstanceDisksExporter.EXPECT().Cancel("timed-out").Do(func(_ string) { diskExporterCancelChan <- true }).Return(true)
	mockInstanceDisksExporter.EXPECT().TraceLogs().Return([]string{"disk exporter trace log"})

	mockInspector := mocks.NewMockInspector(mockCtrl)
	mockOvfDescriptorGenerator := mocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfManifestGenerator := mocks.NewMockOvfManifestGenerator(mockCtrl)

	mockInstanceExportCleaner := mocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)
	mockInstanceExportCleaner.EXPECT().TraceLogs().Return([]string{"cleaner trace log"})

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

	params := ovfexportdomain.GetAllInstanceExportParams()
	params.Timeout = 100 * time.Millisecond
	instance := &compute.Instance{}

	mockLogger := mocks.NewMockLoggerInterface(mockCtrl)
	mockLogger.EXPECT().Log(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(*params.Project, params.Zone, params.InstanceName).Return(instance, nil)
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	mockInstanceExportPreparer := mocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Return(nil)
	mockInstanceExportPreparer.EXPECT().TraceLogs().Return([]string{"preparer trace log"})

	mockInstanceDisksExporter := mocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInstanceDisksExporter.EXPECT().Export(instance, params).Return(exportedDisks, nil)
	mockInstanceDisksExporter.EXPECT().TraceLogs().Return([]string{"disk exporter trace log"})

	inspectorCancelChan := make(chan bool)
	mockInspector := mocks.NewMockInspector(mockCtrl)
	mockInspector.EXPECT().Inspect(
		fmt.Sprintf("projects/%v/zones/%v/disks/%v", *params.Project, params.Zone, "bootdisk"), true).Do(func(reference string, inspectOS bool) { sleepStep(inspectorCancelChan) }).Return(commondisk.InspectionResult{}, nil)
	mockInspector.EXPECT().Cancel("timed-out").Do(func(_ string) { inspectorCancelChan <- true }).Return(true)
	mockInspector.EXPECT().TraceLogs().Return([]string{"inspector trace log"})

	mockOvfDescriptorGenerator := mocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfManifestGenerator := mocks.NewMockOvfManifestGenerator(mockCtrl)

	mockInstanceExportCleaner := mocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)
	mockInstanceExportCleaner.EXPECT().TraceLogs().Return([]string{"cleaner trace log"})

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

	params := ovfexportdomain.GetAllInstanceExportParams()
	params.Timeout = 100 * time.Millisecond
	instance := &compute.Instance{}
	inspectionResults := commondisk.InspectionResult{Architecture: "x86", Distro: "ubuntu", Major: "16", Minor: "4"}

	mockLogger := mocks.NewMockLoggerInterface(mockCtrl)
	mockLogger.EXPECT().Log(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(*params.Project, params.Zone, params.InstanceName).Return(instance, nil)
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	mockInstanceExportPreparer := mocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Return(nil)
	mockInstanceExportPreparer.EXPECT().TraceLogs().Return([]string{"preparer trace log"})

	mockInstanceDisksExporter := mocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInstanceDisksExporter.EXPECT().Export(instance, params).Return(exportedDisks, nil)
	mockInstanceDisksExporter.EXPECT().TraceLogs().Return([]string{"disk exporter trace log"})

	mockInspector := mocks.NewMockInspector(mockCtrl)
	mockInspector.EXPECT().Inspect(
		fmt.Sprintf("projects/%v/zones/%v/disks/%v", *params.Project, params.Zone, "bootdisk"), true).Return(inspectionResults, nil)
	mockInspector.EXPECT().TraceLogs().Return([]string{"inspector trace log"})

	descriptorGeneratorCancelChan := make(chan bool)
	mockOvfDescriptorGenerator := mocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfDescriptorGenerator.EXPECT().GenerateAndWriteOVFDescriptor(
		instance, exportedDisks, "ovfbucket", "OVFpath", &inspectionResults).Do(
		func(_ *compute.Instance, _ []*ovfexportdomain.ExportedDisk, _, _ string, _ *commondisk.InspectionResult) {
			sleepStep(descriptorGeneratorCancelChan)
		}).Return(nil)
	mockOvfDescriptorGenerator.EXPECT().Cancel("timed-out").Do(func(_ string) { descriptorGeneratorCancelChan <- true }).Return(true)

	mockOvfManifestGenerator := mocks.NewMockOvfManifestGenerator(mockCtrl)

	mockInstanceExportCleaner := mocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)
	mockInstanceExportCleaner.EXPECT().TraceLogs().Return([]string{"cleaner trace log"})

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

	params := ovfexportdomain.GetAllInstanceExportParams()
	params.Timeout = 100 * time.Millisecond
	instance := &compute.Instance{}
	inspectionResults := commondisk.InspectionResult{Architecture: "x86", Distro: "ubuntu", Major: "16", Minor: "4"}

	mockLogger := mocks.NewMockLoggerInterface(mockCtrl)
	mockLogger.EXPECT().Log(gomock.Any()).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil)

	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetInstance(*params.Project, params.Zone, params.InstanceName).Return(instance, nil)
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)

	mockInstanceExportPreparer := mocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportPreparer.EXPECT().Prepare(instance, params).Return(nil)
	mockInstanceExportPreparer.EXPECT().TraceLogs().Return([]string{"preparer trace log"})

	mockInstanceDisksExporter := mocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInstanceDisksExporter.EXPECT().Export(instance, params).Return(exportedDisks, nil)
	mockInstanceDisksExporter.EXPECT().TraceLogs().Return([]string{"disk exporter trace log"})

	mockInspector := mocks.NewMockInspector(mockCtrl)
	mockInspector.EXPECT().Inspect(
		fmt.Sprintf("projects/%v/zones/%v/disks/%v", *params.Project, params.Zone, "bootdisk"), true).Return(inspectionResults, nil)
	mockInspector.EXPECT().TraceLogs().Return([]string{"inspector trace log"})

	mockOvfDescriptorGenerator := mocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfDescriptorGenerator.EXPECT().GenerateAndWriteOVFDescriptor(
		instance, exportedDisks, "ovfbucket", "OVFpath", &inspectionResults).Return(nil)

	manifestGeneratorCancelChan := make(chan bool)
	mockOvfManifestGenerator := mocks.NewMockOvfManifestGenerator(mockCtrl)
	mockOvfManifestGenerator.EXPECT().GenerateAndWriteToGCS(params.DestinationURI, params.InstanceName).Do(
		func(_, _ string) { sleepStep(manifestGeneratorCancelChan) }).Return(nil)
	mockOvfManifestGenerator.EXPECT().Cancel("timed-out").Do(
		func(_ string) { manifestGeneratorCancelChan <- true }).Return(true)

	mockInstanceExportCleaner := mocks.NewMockInstanceExportCleaner(mockCtrl)
	mockInstanceExportCleaner.EXPECT().Clean(instance, params).Return(nil)
	mockInstanceExportCleaner.EXPECT().TraceLogs().Return([]string{"cleaner trace log"})

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
	loggable, err := exporter.Run(context.Background())
	duration := time.Since(start)

	assert.NotNil(t, loggable)
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

	params := ovfexportdomain.GetAllInstanceExportParams()
	paramValidator := mocks.NewMockOvfExportParamValidator(mockCtrl)
	paramValidator.EXPECT().ValidateAndParseParams(params).Return(nil)
	paramPopulator := mocks.NewMockOvfExportParamPopulator(mockCtrl)
	paramPopulator.EXPECT().Populate(params).Return(nil)
	err := ValidateAndPopulateParams(params, paramValidator, paramPopulator)
	assert.Nil(t, err)
}

func TestValidateAndPopulateParams_ErrorOnValidate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportParams()
	params.MachineImageName = "also-machine-image-name-which-is-invalid"
	paramValidator := mocks.NewMockOvfExportParamValidator(mockCtrl)
	paramValidator.EXPECT().ValidateAndParseParams(params).Return(fmt.Errorf("validation error"))
	err := ValidateAndPopulateParams(params, paramValidator, nil)
	assert.NotNil(t, err)
}

func TestValidateAndPopulateParams_ErrorOnPopulate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportParams()
	paramValidator := mocks.NewMockOvfExportParamValidator(mockCtrl)
	paramValidator.EXPECT().ValidateAndParseParams(params).Return(nil)

	populatorError := fmt.Errorf("populator error")
	paramPopulator := mocks.NewMockOvfExportParamPopulator(mockCtrl)
	paramPopulator.EXPECT().Populate(params).Return(populatorError)

	err := ValidateAndPopulateParams(params, paramValidator, paramPopulator)
	assert.Equal(t, populatorError, err)
}
