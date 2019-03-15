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
//  limitations under the License.

package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/test"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/ovf"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
)

func TestSetUpWorkflowHappyPathFromOVANoExtraFlags(t *testing.T) {
	cliArgs := getAllCliArgs()
	defer testutils.ClearStringFlag(cliArgs, "project", &project)()
	defer testutils.ClearStringFlag(cliArgs, "zone", &zone)()
	defer testutils.ClearStringFlag(cliArgs, "machine-type", &machineType)()
	defer testutils.ClearStringFlag(cliArgs, "scratch-bucket-gcs-path", &scratchBucketGcsPath)()

	defer testutils.BackupOsArgs()()
	testutils.BuildOsArgs(cliArgs)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true).AnyTimes()
	mockMetadataGce.EXPECT().ProjectID().Return("gceProject", nil)
	mockMetadataGce.EXPECT().Zone().Return("europe-north1-b", nil)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	createdScratchBucketName := "gceproject-ovf-import-bkt-europe-north1"
	mockStorageClient.EXPECT().CreateBucket(createdScratchBucketName, "gceProject",
		&storage.BucketAttrs{
			Name:     createdScratchBucketName,
			Location: "europe-north1",
		}).Return(nil)

	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListMachineTypes("gceProject", "europe-north1-b").
		Return(machineTypes, nil).Times(1)

	mockOvfDescriptorLoader := mocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load(
		fmt.Sprintf("gs://%v/ovf-import-build123/ovf/", createdScratchBucketName)).Return(
		createOVFDescriptor(), nil)

	mockMockTarGcsExtractorInterface := mocks.NewMockTarGcsExtractorInterface(mockCtrl)
	mockMockTarGcsExtractorInterface.EXPECT().ExtractTarToGcs(
		"gs://ovfbucket/ovfpath/vmware.ova",
		fmt.Sprintf("gs://%v/ovf-import-build123/ovf", createdScratchBucketName)).
		Return(nil).Times(1)

	someBucketAttrs := &storage.BucketAttrs{
		Name:     "some-bucket",
		Location: "us-west2",
	}
	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	mockBucketIterator.EXPECT().Next().Return(someBucketAttrs, nil)
	mockBucketIterator.EXPECT().Next().Return(nil, iterator.Done)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(ctx, mockStorageClient, "gceProject").
		Return(mockBucketIterator)

	oi := OVFImporter{mgce: mockMetadataGce, workflowPath: "../test_data/test_import_ovf.wf.json",
		storageClient: mockStorageClient, computeClient: mockComputeClient, buildID: "build123",
		ovfDescriptorLoader: mockOvfDescriptorLoader, tarGcsExtractor: mockMockTarGcsExtractorInterface,
		ctx: ctx, bucketIteratorCreator: mockBucketIteratorCreator,
		logger: logging.NewLogger("test")}
	w, err := oi.setUpImportWorkflow()

	assert.Nil(t, err)
	assert.NotNil(t, w)

	oi.modifyWorkflowPreValidate(w)
	oi.modifyWorkflowPostValidate(w)
	assert.Equal(t, "n1-highcpu-16", w.Vars["machine_type"].Value)
	assert.Equal(t, "gceProject", w.Project)
	assert.Equal(t, "europe-north1-b", w.Zone)
	assert.Equal(t, fmt.Sprintf("gs://%v/", createdScratchBucketName), w.GCSPath)
	assert.Equal(t, "oAuthFilePath", w.OAuthPath)
	assert.Equal(t, "3h", w.DefaultTimeout)
	assert.Equal(t, 3+3*3, len(w.Steps))
	assert.Equal(t, "build123", (*w.Steps["create-instance"].CreateInstances)[0].
		Instance.Labels["gce-ovf-import-build-id"])
	assert.Equal(t, "uservalue1", (*w.Steps["create-instance"].CreateInstances)[0].
		Instance.Labels["userkey1"])
	assert.Equal(t, "uservalue2", (*w.Steps["create-instance"].CreateInstances)[0].
		Instance.Labels["userkey2"])
	assert.Equal(t, fmt.Sprintf("gs://%v/ovf-import-build123/ovf/", createdScratchBucketName),
		oi.gcsPathToClean)
}

func TestSetUpWorkflowHappyPathFromOVAExistingScratchBucketProjectZoneAsFlags(t *testing.T) {
	cliArgs := getAllCliArgs()
	defer testutils.SetStringP(&project, "aProject")()
	defer testutils.SetStringP(&zone, "europe-west2-b")()
	defer testutils.ClearStringFlag(cliArgs, "machine-type", &machineType)()

	defer testutils.BackupOsArgs()()
	testutils.BuildOsArgs(cliArgs)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListMachineTypes("aProject", "europe-west2-b").
		Return(machineTypes, nil).Times(1)

	mockOvfDescriptorLoader := mocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load("gs://bucket/folder/ovf-import-build123/ovf/").Return(
		createOVFDescriptor(), nil)

	mockMockTarGcsExtractorInterface := mocks.NewMockTarGcsExtractorInterface(mockCtrl)
	mockMockTarGcsExtractorInterface.EXPECT().ExtractTarToGcs(
		"gs://ovfbucket/ovfpath/vmware.ova", "gs://bucket/folder/ovf-import-build123/ovf").
		Return(nil).Times(1)

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	mockZoneValidator.EXPECT().
		ZoneValid("aProject", "europe-west2-b").Return(nil)

	oi := OVFImporter{mgce: mockMetadataGce, workflowPath: "../test_data/test_import_ovf.wf.json",
		storageClient: mockStorageClient, computeClient: mockComputeClient, buildID: "build123",
		ovfDescriptorLoader: mockOvfDescriptorLoader, tarGcsExtractor: mockMockTarGcsExtractorInterface,
		logger: logging.NewLogger("test"), zoneValidator: mockZoneValidator}
	w, err := oi.setUpImportWorkflow()

	assert.Nil(t, err)
	assert.NotNil(t, w)

	oi.modifyWorkflowPreValidate(w)
	oi.modifyWorkflowPostValidate(w)
	assert.Equal(t, "n1-highcpu-16", w.Vars["machine_type"].Value)
	assert.Equal(t, "aProject", w.Project)
	assert.Equal(t, "europe-west2-b", w.Zone)
	assert.Equal(t, "gs://bucket/folder", w.GCSPath)
	assert.Equal(t, "oAuthFilePath", w.OAuthPath)
	assert.Equal(t, "3h", w.DefaultTimeout)
	assert.Equal(t, 3+3*3, len(w.Steps))
	assert.Equal(t, "build123", (*w.Steps["create-instance"].CreateInstances)[0].
		Instance.Labels["gce-ovf-import-build-id"])
	assert.Equal(t, "uservalue1", (*w.Steps["create-instance"].CreateInstances)[0].
		Instance.Labels["userkey1"])
	assert.Equal(t, "uservalue2", (*w.Steps["create-instance"].CreateInstances)[0].
		Instance.Labels["userkey2"])
	assert.Equal(t, "gs://bucket/folder/ovf-import-build123/ovf/", oi.gcsPathToClean)
}

func TestSetUpWorkflowPopulateMissingParametersError(t *testing.T) {
	cliArgs := getAllCliArgs()
	defer testutils.ClearStringFlag(cliArgs, "project", &project)()
	defer testutils.BackupOsArgs()()
	testutils.BuildOsArgs(cliArgs)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).AnyTimes()

	oi := OVFImporter{mgce: mockMetadataGce, logger: logging.NewLogger("test")}
	w, err := oi.setUpImportWorkflow()

	assert.NotNil(t, err)
	assert.Nil(t, w)
}

func TestSetUpWorkflowPopulateFlagValidationFailed(t *testing.T) {
	cliArgs := getAllCliArgs()
	defer testutils.ClearStringFlag(cliArgs, instanceNameFlagKey, &instanceNames)()
	defer testutils.BackupOsArgs()()
	testutils.BuildOsArgs(cliArgs)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).AnyTimes()

	oi := OVFImporter{mgce: mockMetadataGce, logger: logging.NewLogger("test")}
	w, err := oi.setUpImportWorkflow()

	assert.NotNil(t, err)
	assert.Nil(t, w)
}

func TestSetUpWorkflowErrorUnpackingOVA(t *testing.T) {
	cliArgs := getAllCliArgs()
	defer testutils.SetStringP(&project, "aProject")()
	defer testutils.SetStringP(&zone, "europe-north1-b")()
	defer testutils.ClearStringFlag(cliArgs, "machine-type", &machineType)()

	defer testutils.BackupOsArgs()()
	testutils.BuildOsArgs(cliArgs)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true).AnyTimes()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	mockMockTarGcsExtractorInterface := mocks.NewMockTarGcsExtractorInterface(mockCtrl)
	mockMockTarGcsExtractorInterface.EXPECT().ExtractTarToGcs(
		"gs://ovfbucket/ovfpath/vmware.ova", "gs://bucket/folder/ovf-import-build123/ovf").
		Return(errors.New("tar error")).Times(1)

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	mockZoneValidator.EXPECT().
		ZoneValid("aProject", "europe-north1-b").Return(nil)

	oi := OVFImporter{mgce: mockMetadataGce, workflowPath: "../test_data/test_import_ovf.wf.json",
		storageClient: mockStorageClient, buildID: "build123",
		tarGcsExtractor: mockMockTarGcsExtractorInterface, logger: logging.NewLogger("test"),
		zoneValidator: mockZoneValidator}
	w, err := oi.setUpImportWorkflow()

	assert.NotNil(t, err)
	assert.Nil(t, w)
}

func TestSetUpWorkflowErrorLoadingDescriptor(t *testing.T) {
	cliArgs := getAllCliArgs()
	defer testutils.SetStringP(&project, "aProject")()
	defer testutils.SetStringP(&zone, "europe-north1-b")()
	defer testutils.SetStringP(&ovfOvaGcsPath, "gs://ovfbucket/ovffolder/")()

	defer testutils.ClearStringFlag(cliArgs, "machine-type", &machineType)()

	defer testutils.BackupOsArgs()()
	testutils.BuildOsArgs(cliArgs)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).AnyTimes()

	mockOvfDescriptorLoader := mocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load("gs://ovfbucket/ovffolder/").Return(
		nil, errors.New("ovf desc error"))

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	mockZoneValidator.EXPECT().
		ZoneValid("aProject", "europe-north1-b").Return(nil)

	oi := OVFImporter{mgce: mockMetadataGce, workflowPath: "../test_data/test_import_ovf.wf.json",
		buildID: "build123", ovfDescriptorLoader: mockOvfDescriptorLoader,
		logger: logging.NewLogger("test"), zoneValidator: mockZoneValidator}
	w, err := oi.setUpImportWorkflow()

	assert.NotNil(t, err)
	assert.Nil(t, w)
	assert.Equal(t, "", oi.gcsPathToClean)
}

func TestCleanUp(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().DeleteGcsPath("aPath")
	mockStorageClient.EXPECT().Close()

	oi := OVFImporter{storageClient: mockStorageClient, gcsPathToClean: "aPath",
		logger: logging.NewLogger("test")}
	oi.CleanUp()
}

func TestFlagsInstanceNameNotProvided(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	defer testutils.ClearStringFlag(cliArgs, instanceNameFlagKey, &instanceNames)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, t)
}

func TestFlagsOvfGcsPathFlagKeyNotProvided(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	defer testutils.ClearStringFlag(cliArgs, ovfGcsPathFlagKey, &ovfOvaGcsPath)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, t)
}

func TestFlagsOvfGcsPathFlagNotValid(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	cliArgs[ovfGcsPathFlagKey] = "NOT_GCS_PATH"
	buildOsArgsAndAssertErrorOnValidate(cliArgs, t)
}

func TestFlagsClientIdNotProvided(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	defer testutils.ClearStringFlag(cliArgs, clientIDFlagKey, &clientID)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, t)
}

func TestFlagsOsIDNotProvided(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	defer testutils.ClearStringFlag(cliArgs, "os", &osID)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, t)
}

func TestFlagsOsIDInvalid(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	cliArgs["os"] = "INVALID_OS"
	buildOsArgsAndAssertErrorOnValidate(cliArgs, t)
}

func TestFlagsLabelsInvalid(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	cliArgs["labels"] = "NOT_VALID_LABEL_DEFINITION"
	buildOsArgsAndAssertErrorOnValidate(cliArgs, t)
}

func TestFlagsNetworkInterfaceSetWhenPrivateNetworkIPSet(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	cliArgs["network-interface"] = "aInterface"
	defer testutils.ClearStringFlag(cliArgs, "network", &network)()
	defer testutils.ClearStringFlag(cliArgs, "subnet", &subnet)()
	defer testutils.ClearStringFlag(cliArgs, "network-tier", &networkTier)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, t)
}

func TestFlagsNetworkInterfaceSetWhenNetworkSet(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	cliArgs["network-interface"] = "aInterface"
	defer testutils.ClearStringFlag(cliArgs, "private-network-ip", &privateNetworkIP)()
	defer testutils.ClearStringFlag(cliArgs, "subnet", &subnet)()
	defer testutils.ClearStringFlag(cliArgs, "network-tier", &networkTier)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, t)
}

func TestFlagsNetworkInterfaceSetWhenSubnetSet(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	cliArgs["network-interface"] = "aInterface"
	defer testutils.ClearStringFlag(cliArgs, "private-network-ip", &privateNetworkIP)()
	defer testutils.ClearStringFlag(cliArgs, "network", &network)()
	defer testutils.ClearStringFlag(cliArgs, "network-tier", &networkTier)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, t)
}

func TestFlagsNetworkInterfaceSetWhenNetworkTierSet(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	cliArgs["network-interface"] = "aInterface"
	defer testutils.ClearStringFlag(cliArgs, "private-network-ip", &privateNetworkIP)()
	defer testutils.ClearStringFlag(cliArgs, "network", &network)()
	defer testutils.ClearStringFlag(cliArgs, "subnet", &subnet)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, t)
}

func TestFlagsAllValid(t *testing.T) {
	defer testutils.BackupOsArgs()()
	testutils.BuildOsArgs(getAllCliArgs())
	assert.Nil(t, validateAndParseFlags())
}

func TestBuildDaisyVarsFromDisk(t *testing.T) {
	testutils.BuildOsArgs(getAllCliArgs())
	assert.Nil(t, validateAndParseFlags())

	defer testutils.SetStringP(&region, "aRegion")()

	varMap := buildDaisyVars("translateworkflow.wf.json", "gs://abucket/apath/bootdisk.vmdk", "n1-standard-2")

	assert.Equal(t, "instance1", varMap["instance_name"])
	assert.Equal(t, "translateworkflow.wf.json", varMap["translate_workflow"])
	assert.Equal(t, strconv.FormatBool(false), varMap["install_gce_packages"])
	assert.Equal(t, "gs://abucket/apath/bootdisk.vmdk", varMap["boot_disk_file"])
	assert.Equal(t, "global/networks/aNetwork", varMap["network"])
	assert.Equal(t, "regions/aRegion/subnetworks/aSubnet", varMap["subnet"])
	assert.Equal(t, "n1-standard-2", varMap["machine_type"])
	assert.Equal(t, "aDescription", varMap["description"])
	assert.Equal(t, "10.0.0.1", varMap["private_network_ip"])
	assert.Equal(t, "PREMIUM", varMap["network_tier"])

	assert.Equal(t, len(varMap), 10)
}

func TestPopulateMissingParametersProjectZoneRegionFromGCE(t *testing.T) {
	defer testutils.SetStringP(&project, "")()
	defer testutils.SetStringP(&zone, "")()
	defer testutils.SetStringP(&region, "")()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true).AnyTimes()
	mockMetadataGce.EXPECT().ProjectID().Return("aProject", nil)
	mockMetadataGce.EXPECT().Zone().Return("europe-north1-b", nil)

	oi := OVFImporter{mgce: mockMetadataGce, logger: logging.NewLogger("test")}
	err := oi.populateMissingParameters()

	assert.Nil(t, err)
	assert.Equal(t, "aProject", *project)
	assert.Equal(t, "europe-north1-b", *zone)
	assert.Equal(t, "europe-north1", *region)
}

func TestPopulateMissingParametersNotChangedIfDefinedAndOnGCE(t *testing.T) {
	defer testutils.SetStringP(&project, "aProject123")()
	defer testutils.SetStringP(&zone, "aZone123")()
	defer testutils.SetStringP(&region, "aRegion123")()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true).AnyTimes()
	mockMetadataGce.EXPECT().ProjectID().Return("aProject", nil).AnyTimes()
	mockMetadataGce.EXPECT().Zone().Return("europe-north1-b", nil).AnyTimes()

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	mockZoneValidator.EXPECT().
		ZoneValid("aProject123", "aZone123").Return(nil)

	oi := OVFImporter{mgce: mockMetadataGce, logger: logging.NewLogger("test"),
		zoneValidator: mockZoneValidator}
	err := oi.populateMissingParameters()

	assert.Nil(t, err)
	assert.Equal(t, "aProject123", *project)
	assert.Equal(t, "aZone123", *zone)
	assert.Equal(t, "aRegion123", *region)
}

func TestPopulateMissingParametersProjectEmptyNotOnGCE(t *testing.T) {
	defer testutils.SetStringP(&project, "")()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).AnyTimes()

	oi := OVFImporter{mgce: mockMetadataGce, logger: logging.NewLogger("test")}
	err := oi.populateMissingParameters()
	assert.NotNil(t, err)
}

func TestPopulateMissingParametersErrorRetrievingProjectIDFromGCE(t *testing.T) {
	defer testutils.SetStringP(&project, "")()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true).AnyTimes()
	mockMetadataGce.EXPECT().ProjectID().Return("", errors.New("err"))

	oi := OVFImporter{mgce: mockMetadataGce, logger: logging.NewLogger("test")}
	err := oi.populateMissingParameters()
	assert.NotNil(t, err)
}

func TestPopulateMissingParametersZoneEmptyNotOnGCE(t *testing.T) {
	defer testutils.SetStringP(&project, "aProject")()
	defer testutils.SetStringP(&zone, "")()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).AnyTimes()

	oi := OVFImporter{mgce: mockMetadataGce, logger: logging.NewLogger("test")}
	err := oi.populateMissingParameters()

	assert.NotNil(t, err)
	assert.Equal(t, "aProject", *project)
}

func TestPopulateMissingParametersErrorRetrievingZone(t *testing.T) {
	defer testutils.SetStringP(&project, "aProject")()
	defer testutils.SetStringP(&zone, "")()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true).AnyTimes()
	mockMetadataGce.EXPECT().Zone().Return("", errors.New("err"))

	oi := OVFImporter{mgce: mockMetadataGce}
	err := oi.populateMissingParameters()

	assert.NotNil(t, err)
	assert.Equal(t, "aProject", *project)
}

func TestPopulateMissingParametersEmptyZoneReturnedFromGCE(t *testing.T) {
	defer testutils.SetStringP(&project, "aProject")()
	defer testutils.SetStringP(&zone, "")()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true).AnyTimes()
	mockMetadataGce.EXPECT().Zone().Return("", nil)

	oi := OVFImporter{mgce: mockMetadataGce, logger: logging.NewLogger("test")}
	err := oi.populateMissingParameters()

	assert.NotNil(t, err)
	assert.Equal(t, "aProject", *project)
}

func TestPopulateMissingParametersInvalidZone(t *testing.T) {
	defer testutils.SetStringP(&project, "aProject")()
	defer testutils.SetStringP(&zone, "europe-north1-b")()
	defer testutils.SetStringP(&region, "")()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	mockZoneValidator.EXPECT().
		ZoneValid("aProject", "europe-north1-b").Return(fmt.Errorf("error"))

	oi := OVFImporter{logger: logging.NewLogger("test"), zoneValidator: mockZoneValidator}
	err := oi.populateMissingParameters()

	assert.NotNil(t, err)
	assert.Equal(t, "europe-north1-b", *zone)
	assert.Equal(t, "aProject", *project)
}

func buildOsArgsAndAssertErrorOnValidate(cliArgs map[string]interface{}, t *testing.T) {
	testutils.BuildOsArgs(cliArgs)
	assert.NotNil(t, validateAndParseFlags())
}

func createControllerItem(instanceID string, resourceType uint16) ovf.ResourceAllocationSettingData {
	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:   instanceID,
			ResourceType: &resourceType,
		},
	}
}

func createDiskItem(instanceID string, addressOnParent string,
	elementName string, hostResource string, parent string) ovf.ResourceAllocationSettingData {
	diskType := uint16(17)
	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:      instanceID,
			ResourceType:    &diskType,
			AddressOnParent: &addressOnParent,
			ElementName:     elementName,
			HostResource:    []string{hostResource},
			Parent:          &parent,
		},
	}
}

func createOVFDescriptor() *ovf.Envelope {
	virtualHardware := ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("5", 6),
			createDiskItem("7", "1", "disk1",
				"ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0",
				"ovf:/disk/vmdisk1", "5"),
			createDiskItem("8", "2", "disk2",
				"ovf:/disk/vmdisk3", "5"),
			createCPUItem("11", 16),
			createMemoryItem("12", 4096),
		},
	}
	diskCapacityAllocationUnits := "byte * 2^30"
	fileRef1 := "file1"
	fileRef2 := "file2"
	fileRef3 := "file3"
	ovfDescriptor := &ovf.Envelope{
		Disk: &ovf.DiskSection{Disks: []ovf.VirtualDiskDesc{
			{Capacity: "20", CapacityAllocationUnits: &diskCapacityAllocationUnits, DiskID: "vmdisk1", FileRef: &fileRef1},
			{Capacity: "1", CapacityAllocationUnits: &diskCapacityAllocationUnits, DiskID: "vmdisk2", FileRef: &fileRef2},
			{Capacity: "5", CapacityAllocationUnits: &diskCapacityAllocationUnits, DiskID: "vmdisk3", FileRef: &fileRef3},
		}},
		References: []ovf.File{
			{Href: "Ubuntu_for_Horizon71_1_1.0-disk1.vmdk", ID: "file1", Size: 1151322112},
			{Href: "Ubuntu_for_Horizon71_1_1.0-disk2.vmdk", ID: "file2", Size: 68096},
			{Href: "Ubuntu_for_Horizon71_1_1.0-disk3.vmdk", ID: "file3", Size: 68096},
		},
		VirtualSystem: &ovf.VirtualSystem{
			VirtualHardware: []ovf.VirtualHardwareSection{virtualHardware},
		},
	}
	return ovfDescriptor
}

func createCPUItem(instanceID string, quantity uint) ovf.ResourceAllocationSettingData {
	resourceType := uint16(3)
	mhz := "hertz * 10^6"
	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:      instanceID,
			ResourceType:    &resourceType,
			VirtualQuantity: &quantity,
			AllocationUnits: &mhz,
		},
	}
}

func createMemoryItem(instanceID string, quantityMB uint) ovf.ResourceAllocationSettingData {
	resourceType := uint16(4)
	mb := "byte * 2^20"

	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:      instanceID,
			ResourceType:    &resourceType,
			VirtualQuantity: &quantityMB,
			AllocationUnits: &mb,
		},
	}
}

func getAllCliArgs() map[string]interface{} {
	return map[string]interface{}{
		instanceNameFlagKey:             "instance1",
		clientIDFlagKey:                 "aClient",
		ovfGcsPathFlagKey:               "gs://ovfbucket/ovfpath/vmware.ova",
		"no-guest-environment":          true,
		"can-ip-forward":                true,
		"deletion-protection":           true,
		"description":                   "aDescription",
		"labels":                        "userkey1=uservalue1,userkey2=uservalue2",
		"machine-type":                  "n1-standard-2",
		"network":                       "aNetwork",
		"network-interface":             "",
		"subnet":                        "aSubnet",
		"network-tier":                  "PREMIUM",
		"private-network-ip":            "10.0.0.1",
		"no-external-ip":                true,
		"no-restart-on-failure":         true,
		"os":                            "ubuntu-1404",
		"shielded-integrity-monitoring": true,
		"shielded-secure-boot":          true,
		"shielded-vtpm":                 true,
		"tags":                          "tag1=val1",
		"zone":                          "us-central1-c",
		"boot-disk-kms-key":             "aKey",
		"boot-disk-kms-keyring":         "aKeyring",
		"boot-disk-kms-location":        "aKmsLocation",
		"boot-disk-kms-project":         "aKmsProject",
		"timeout":                       "3h",
		"project":                       "aProject",
		"scratch-bucket-gcs-path":       "gs://bucket/folder",
		"oauth":                         "oAuthFilePath",
		"compute-endpoint-override":     "us-east1-c",
		"disable-gcs-logging":           true,
		"disable-cloud-logging":         true,
		"disable-stdout-logging":        true,
	}
}

var machineTypes = []*compute.MachineType{
	{
		GuestCpus:                    1,
		Id:                           2000,
		IsSharedCpu:                  true,
		MaximumPersistentDisks:       16,
		MaximumPersistentDisksSizeGb: 3072,
		MemoryMb:                     1740,
		Name:                         "g1-small",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    16,
		Id:                           4016,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     14746,
		Name:                         "n1-highcpu-16",
		Zone:                         "us-east1-b",
	},
}
