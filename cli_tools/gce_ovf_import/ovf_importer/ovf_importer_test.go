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

package ovfimporter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/ovf"
	"google.golang.org/api/compute/v1"

	computeutil "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	ovfdomainmocks "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const instanceImportWorkflowPath = "../../test_data/test_import_ovf_to_instance.wf.json"
const machineImageImportWorkflowPath = "../../test_data/test_import_ovf_to_machine_image.wf.json"

func TestSetUpInstanceWorkflowHappyPathFromOVANoExtraFlags(t *testing.T) {
	params := getAllInstanceImportParams()
	params.MachineType = ""
	params.Zone = "europe-north1-b"
	params.Region = "europe-north1"
	params.UserLabels = map[string]string{
		"userkey1": "uservalue1",
		"userkey2": "uservalue2",
	}
	params.NodeAffinities, params.NodeAffinitiesBeta, _ = computeutil.ParseNodeAffinityLabels(params.NodeAffinityLabelsFlag)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	project := *params.Project

	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListMachineTypes(project, params.Zone).
		Return(machineTypes, nil).Times(1)

	mockOvfDescriptorLoader := ovfdomainmocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	descriptor := createOVFDescriptor([]string{
		"Ubuntu_for_Horizon71_1_1.0-disk1.vmdk",
		"Ubuntu_for_Horizon71_1_1.0-disk2.vmdk",
		"Ubuntu_for_Horizon71_1_1.0-disk3.vmdk",
	})
	mockOvfDescriptorLoader.EXPECT().Load(params.ScratchBucketGcsPath+"/ovf/").Return(
		descriptor, nil)

	mockMockTarGcsExtractorInterface := mocks.NewMockTarGcsExtractorInterface(mockCtrl)
	mockMockTarGcsExtractorInterface.EXPECT().ExtractTarToGcs(
		"gs://ovfbucket/ovfpath/vmware.ova",
		params.ScratchBucketGcsPath+"/ovf").
		Return(nil).Times(1)

	oi := OVFImporter{workflowPath: instanceImportWorkflowPath,
		computeClient:       mockComputeClient,
		ovfDescriptorLoader: mockOvfDescriptorLoader, tarGcsExtractor: mockMockTarGcsExtractorInterface,
		ctx:    ctx,
		Logger: logging.NewToolLogger("test"), params: params}
	w, err := oi.setUpImportWorkflow()

	assert.Nil(t, err)
	assert.NotNil(t, w)

	w.Logger = DummyLogger{}
	oi.modifyWorkflowPreValidate(w)
	oi.modifyWorkflowPostValidate(w)
	assert.Equal(t, "n1-highcpu-16", w.Vars["machine_type"].Value)
	assert.Equal(t, project, w.Project)
	assert.Equal(t, "europe-north1-b", w.Zone)
	assert.Equal(t, params.ScratchBucketGcsPath, w.GCSPath)
	assert.Equal(t, "oAuthFilePath", w.OAuthPath)
	assert.Equal(t, "3h", w.DefaultTimeout)
	assert.Equal(t, 3+3*3, len(w.Steps))
	assert.Equal(t, "europe-north1", oi.imageLocation)

	instance := (*w.Steps["create-instance"].CreateInstances).Instances[0].Instance
	assert.Equal(t, "build123", instance.Labels["gce-ovf-import-build-id"])
	assert.Equal(t, "uservalue1", instance.Labels["userkey1"])
	assert.Equal(t, "uservalue2", instance.Labels["userkey2"])
	assert.Equal(t, false, *instance.Scheduling.AutomaticRestart)
	assert.Equal(t, 1, len(instance.Scheduling.NodeAffinities))
	assert.Equal(t, "env", instance.Scheduling.NodeAffinities[0].Key)
	assert.Equal(t, "IN", instance.Scheduling.NodeAffinities[0].Operator)
	assert.Equal(t, 2, len(instance.Scheduling.NodeAffinities[0].Values))
	assert.Equal(t, "prod", instance.Scheduling.NodeAffinities[0].Values[0])
	assert.Equal(t, "test", instance.Scheduling.NodeAffinities[0].Values[1])

	instanceBeta := (*w.Steps["create-instance"].CreateInstances).InstancesBeta[0].Instance
	assert.Equal(t, "build123", instanceBeta.Labels["gce-ovf-import-build-id"])
	assert.Equal(t, "uservalue1", instanceBeta.Labels["userkey1"])
	assert.Equal(t, "uservalue2", instanceBeta.Labels["userkey2"])
	assert.Equal(t, false, *instanceBeta.Scheduling.AutomaticRestart)
	assert.Equal(t, 1, len(instanceBeta.Scheduling.NodeAffinities))
	assert.Equal(t, "env", instanceBeta.Scheduling.NodeAffinities[0].Key)
	assert.Equal(t, "IN", instanceBeta.Scheduling.NodeAffinities[0].Operator)
	assert.Equal(t, 2, len(instanceBeta.Scheduling.NodeAffinities[0].Values))
	assert.Equal(t, "prod", instanceBeta.Scheduling.NodeAffinities[0].Values[0])
	assert.Equal(t, "test", instanceBeta.Scheduling.NodeAffinities[0].Values[1])

	assert.Equal(t, params.ScratchBucketGcsPath+"/ovf/",
		oi.gcsPathToClean)
}

func TestSetUpMachineImageWorkflowHappyPathFromOVANoExtraFlags(t *testing.T) {
	params := getAllMachineImageImportParams()
	params.MachineType = ""
	params.Zone = "europe-north1-b"
	params.Region = "europe-north1"
	params.UserLabels = map[string]string{
		"userkey1": "uservalue1",
		"userkey2": "uservalue2",
	}
	params.NodeAffinities, params.NodeAffinitiesBeta, _ = computeutil.ParseNodeAffinityLabels(params.NodeAffinityLabelsFlag)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	project := *params.Project

	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListMachineTypes(project, params.Zone).
		Return(machineTypes, nil).Times(1)

	mockOvfDescriptorLoader := ovfdomainmocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	descriptor := createOVFDescriptor([]string{
		"Ubuntu_for_Horizon71_1_1.0-disk1.vmdk",
		"Ubuntu_for_Horizon71_1_1.0-disk2.vmdk",
		"Ubuntu_for_Horizon71_1_1.0-disk3.vmdk",
	})
	mockOvfDescriptorLoader.EXPECT().Load(params.ScratchBucketGcsPath+"/ovf/").Return(
		descriptor, nil)

	mockMockTarGcsExtractorInterface := mocks.NewMockTarGcsExtractorInterface(mockCtrl)
	mockMockTarGcsExtractorInterface.EXPECT().ExtractTarToGcs(
		"gs://ovfbucket/ovfpath/vmware.ova",
		params.ScratchBucketGcsPath+"/ovf").
		Return(nil).Times(1)

	oi := OVFImporter{workflowPath: machineImageImportWorkflowPath,
		computeClient:       mockComputeClient,
		ovfDescriptorLoader: mockOvfDescriptorLoader, tarGcsExtractor: mockMockTarGcsExtractorInterface,
		ctx:    ctx,
		Logger: logging.NewToolLogger("test"), params: params}
	w, err := oi.setUpImportWorkflow()

	assert.Nil(t, err)
	assert.NotNil(t, w)

	w.Logger = DummyLogger{}
	oi.modifyWorkflowPreValidate(w)
	oi.modifyWorkflowPostValidate(w)
	assert.Equal(t, "n1-highcpu-16", w.Vars["machine_type"].Value)
	assert.Equal(t, project, w.Project)
	assert.Equal(t, "europe-north1-b", w.Zone)
	assert.Equal(t, params.ScratchBucketGcsPath, w.GCSPath)
	assert.Equal(t, "oAuthFilePath", w.OAuthPath)
	assert.Equal(t, "3h", w.DefaultTimeout)
	assert.Equal(t, 4+3*3, len(w.Steps))
	assert.Equal(t, "europe-north1", oi.imageLocation)

	instance := (*w.Steps["create-instance"].CreateInstances).Instances[0].Instance
	assert.Equal(t, "build123", instance.Labels["gce-ovf-import-build-id"])
	assert.Equal(t, "uservalue1", instance.Labels["userkey1"])
	assert.Equal(t, "uservalue2", instance.Labels["userkey2"])
	assert.Equal(t, false, *instance.Scheduling.AutomaticRestart)
	assert.Equal(t, 1, len(instance.Scheduling.NodeAffinities))
	assert.Equal(t, "env", instance.Scheduling.NodeAffinities[0].Key)
	assert.Equal(t, "IN", instance.Scheduling.NodeAffinities[0].Operator)
	assert.Equal(t, 2, len(instance.Scheduling.NodeAffinities[0].Values))
	assert.Equal(t, "prod", instance.Scheduling.NodeAffinities[0].Values[0])
	assert.Equal(t, "test", instance.Scheduling.NodeAffinities[0].Values[1])

	instanceBeta := (*w.Steps["create-instance"].CreateInstances).InstancesBeta[0].Instance
	assert.Equal(t, "build123", instanceBeta.Labels["gce-ovf-import-build-id"])
	assert.Equal(t, "uservalue1", instanceBeta.Labels["userkey1"])
	assert.Equal(t, "uservalue2", instanceBeta.Labels["userkey2"])
	assert.Equal(t, false, *instanceBeta.Scheduling.AutomaticRestart)
	assert.Equal(t, 1, len(instanceBeta.Scheduling.NodeAffinities))
	assert.Equal(t, "env", instanceBeta.Scheduling.NodeAffinities[0].Key)
	assert.Equal(t, "IN", instanceBeta.Scheduling.NodeAffinities[0].Operator)
	assert.Equal(t, 2, len(instanceBeta.Scheduling.NodeAffinities[0].Values))
	assert.Equal(t, "prod", instanceBeta.Scheduling.NodeAffinities[0].Values[0])
	assert.Equal(t, "test", instanceBeta.Scheduling.NodeAffinities[0].Values[1])

	assert.Equal(t, params.ScratchBucketGcsPath+"/ovf/",
		oi.gcsPathToClean)

	machineImage := (*w.Steps["create-machine-image"].CreateMachineImages)[0].MachineImage
	assert.Equal(t, "us-west2", machineImage.StorageLocations[0])
}

func Test_InstanceImport_SetupWorkflow_HappyCase_PreGA(t *testing.T) {
	wfPath := "../../../daisy_workflows/" + createInstanceWorkflow
	params := getAllInstanceImportParams()
	params.ReleaseTrack = domain.Beta
	createMachineImage := false
	verifyModuleImport(t, wfPath, params, createMachineImage)
}

func Test_MachineImageImport_SetupWorkflow_HappyCase_PreGA(t *testing.T) {
	wfPath := "../../../daisy_workflows/" + createGMIWorkflow
	params := getAllMachineImageImportParams()
	params.ReleaseTrack = domain.Beta
	createMachineImage := true
	verifyModuleImport(t, wfPath, params, createMachineImage)
}

func verifyModuleImport(t *testing.T, wfPath string, params *domain.OVFImportParams, createMachineImage bool) {
	testCase := ModuleImportTestCase{
		descriptorFilenames: []string{
			"Ubuntu_for_Horizon71_1_1.0-disk1.vmdk",
			"Ubuntu_for_Horizon71_1_1.0-disk2.vmdk",
			"Ubuntu_for_Horizon71_1_1.0-disk3.vmdk",
		},
		fileURIs: []string{
			"gs://bucket/folder/ovf/Ubuntu_for_Horizon71_1_1.0-disk1.vmdk",
			"gs://bucket/folder/ovf/Ubuntu_for_Horizon71_1_1.0-disk2.vmdk",
			"gs://bucket/folder/ovf/Ubuntu_for_Horizon71_1_1.0-disk3.vmdk",
		},
		imageURIs: []string{
			"images/uri/boot-disk",
			"images/uri/data-disk-1",
			"images/uri/data-disk-2",
		},
	}

	descriptor := createOVFDescriptor(testCase.descriptorFilenames)
	w := runImportWithModules(t, params, wfPath, descriptor, testCase)

	// Workflow validation
	assert.Equal(t, *params.Project, w.Project)
	assert.Equal(t, params.Timeout, w.DefaultTimeout)
	assert.Equal(t, params.Zone, w.Zone)
	assert.Equal(t, params.Oauth, w.OAuthPath)
	assert.Equal(t, params.Ce, w.ComputeEndpoint)
	assert.Equal(t, params.ScratchBucketGcsPath, w.GCSPath)
	if createMachineImage {
		// Creating the machine image adds two steps:
		//  1. Stop the instance
		//  2. Create the GMI
		//
		// (It also updates the cleanup step to delete the instance.)
		assert.Len(t, w.Steps, 4)
	} else {
		assert.Len(t, w.Steps, 2)
	}
	assert.Len(t, w.Steps["create-instance"].CreateInstances.Instances, 1, "Expect one instance created")
	if createMachineImage {
		assert.Len(t, *w.Steps["create-machine-image"].CreateMachineImages, 1, "Expect one GMI created")
	}

	instance := w.Steps["create-instance"].CreateInstances.Instances[0]
	cleanup := w.Steps["cleanup"].DeleteResources

	// Boot Disk
	bootDisk := instance.Disks[0]
	checkDaisyVariable(t, w, "boot_disk_image_uri", testCase.imageURIs[0], bootDisk.InitializeParams.SourceImage)
	assert.True(t, bootDisk.AutoDelete, "Delete boot disk when instance is deleted.")
	assert.True(t, bootDisk.Boot, "Boot disk is configured to boot.")
	assert.Contains(t, cleanup.Images, "${boot_disk_image_uri}", "Delete the boot disk image after instance creation.")

	// Data Disks
	assert.Len(t, instance.Disks, len(descriptor.Disk.Disks))
	assert.Len(t, cleanup.Images, len(testCase.imageURIs))
	for i, diskURI := range testCase.imageURIs[1:] {
		dataDisk := instance.Disks[i+1]
		assert.Equal(t, diskURI, dataDisk.InitializeParams.SourceImage, "Include data disk on final instance.")
		assert.Regexp(t, "^[a-z].*", dataDisk.InitializeParams.DiskName, "Disk name should start with letter.")
		assert.True(t, dataDisk.AutoDelete, "Delete the disk when the instance is deleted.")
		assert.False(t, dataDisk.Boot, "Data disk are not configured to boot.")
		assert.Contains(t, cleanup.Images, testCase.imageURIs[i+1], "Delete the data disk image after instance creation.")
	}

	// Instance
	assert.Equal(t, params.CanIPForward, instance.CanIpForward)
	assert.Equal(t, params.DeletionProtection, instance.DeletionProtection)
	if params.NoRestartOnFailure {
		assert.False(t, *instance.Scheduling.AutomaticRestart)
	} else {
		assert.Nil(t, instance.Scheduling.AutomaticRestart)
	}
	assert.Equal(t, params.NodeAffinities, instance.Scheduling.NodeAffinities)
	assert.Equal(t, params.Hostname, instance.Hostname)
	checkDaisyVariable(t, w, "description", params.Description, instance.Description)
	checkDaisyVariable(t, w, "machine_type", params.MachineType, instance.MachineType)
	assert.True(t, instance.ExactName, "Use the instance name provided by the user.")
	assert.True(t, instance.NoCleanup, "Retain the instance after daisy runs.")
	expectedLabels := map[string]string{
		"gce-ovf-import-build-id": params.BuildID,
		"gce-ovf-import-tmp":      "true",
	}
	for k, v := range params.UserLabels {
		expectedLabels[k] = v
	}
	assert.Equal(t, expectedLabels, instance.Labels)

	// Network
	assert.Len(t, instance.NetworkInterfaces, 1, "Expect one network to be created")
	networkInterface := instance.NetworkInterfaces[0]
	checkDaisyVariable(t, w, "network", params.Network, networkInterface.Network)
	checkDaisyVariable(t, w, "subnet", params.Subnet, networkInterface.Subnetwork)
	checkDaisyVariable(t, w, "private_network_ip", params.PrivateNetworkIP, networkInterface.NetworkIP)
	if params.NoExternalIP {
		assert.Len(t, networkInterface.AccessConfigs, 0, "No access config when disabling external IP")
	} else {
		assert.Len(t, networkInterface.AccessConfigs, 1, "Expect one access config to create external IP")
		accessConfig := networkInterface.AccessConfigs[0]
		checkDaisyVariable(t, w, "network_tier", params.NetworkTier, accessConfig.NetworkTier)
	}

	// Cleanup
	if createMachineImage {
		machineImage := []*daisy.MachineImage(*w.Steps["create-machine-image"].CreateMachineImages)[0]
		checkDaisyVariable(t, w, "machine_image_name", params.MachineImageName, machineImage.Name)
		assert.Equal(t, instance.Name, machineImage.SourceInstance)
		checkDaisyVariable(t, w, "description", params.Description, machineImage.Description)
		assert.True(t, machineImage.ExactName)
		assert.True(t, machineImage.NoCleanup)
		assert.Equal(t, []string{instance.Name}, cleanup.Instances)
	}
}

// checkDaisyVariable ensures that a variable is declared, the desired value is injected, and that it's
// being used at a location within the workflow.
func checkDaisyVariable(t *testing.T, w *daisy.Workflow, declaredVariableName string, expectedValue string, expectedLocationInTemplate string) {
	assert.Equal(t, expectedValue, w.Vars[declaredVariableName].Value)
	assert.Equal(t, fmt.Sprintf("${%s}", declaredVariableName), expectedLocationInTemplate)
}

type ModuleImportTestCase struct {
	descriptorFilenames []string
	fileURIs            []string
	imageURIs           []string
}

func runImportWithModules(t *testing.T, params *domain.OVFImportParams, wfPath string, descriptor *ovf.Envelope, testCase ModuleImportTestCase) *daisy.Workflow {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockOvfDescriptorLoader := ovfdomainmocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load(params.ScratchBucketGcsPath+"/ovf/").Return(
		descriptor, nil)
	mockMockTarGcsExtractorInterface := mocks.NewMockTarGcsExtractorInterface(mockCtrl)
	mockMockTarGcsExtractorInterface.EXPECT().ExtractTarToGcs(
		params.OvfOvaGcsPath, params.ScratchBucketGcsPath+"/ovf").
		Return(nil).Times(1)
	mockMultiDiskImporter := ovfdomainmocks.NewMockMultiImageImporterInterface(mockCtrl)
	mockMultiDiskImporter.EXPECT().ImportAll(
		gomock.Any(),
		gomock.Any(),
		testCase.fileURIs).Return(testCase.imageURIs, nil)
	oi := OVFImporter{ctx: context.Background(), workflowPath: wfPath, multiImageImporter: mockMultiDiskImporter,
		storageClient: mockStorageClient, computeClient: mockComputeClient,
		ovfDescriptorLoader: mockOvfDescriptorLoader, tarGcsExtractor: mockMockTarGcsExtractorInterface,
		Logger: logging.NewToolLogger("test"), params: params}
	w, err := oi.setUpImportWorkflow()

	assert.NoError(t, err)
	assert.NotNil(t, w)

	w.Logger = DummyLogger{}

	oi.modifyWorkflowPreValidate(w)
	oi.modifyWorkflowPostValidate(w)
	return w
}

func TestSetUpWorkflowErrorUnpackingOVA(t *testing.T) {
	params := getAllInstanceImportParams()
	project := defaultProject
	params.Project = &project
	params.Zone = "europe-north1-b"
	params.MachineType = ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMockTarGcsExtractorInterface := mocks.NewMockTarGcsExtractorInterface(mockCtrl)
	mockMockTarGcsExtractorInterface.EXPECT().ExtractTarToGcs(
		"gs://ovfbucket/ovfpath/vmware.ova", params.ScratchBucketGcsPath+"/ovf").
		Return(errors.New("tar error")).Times(1)

	oi := OVFImporter{workflowPath: instanceImportWorkflowPath,
		tarGcsExtractor: mockMockTarGcsExtractorInterface, Logger: logging.NewToolLogger("test"),
		params: params}
	w, err := oi.setUpImportWorkflow()

	assert.NotNil(t, err)
	assert.Nil(t, w)
}

func TestSetUpWorkflowErrorLoadingDescriptor(t *testing.T) {
	params := getAllInstanceImportParams()
	project := defaultProject
	params.Project = &project
	params.Zone = "europe-north1-b"
	params.OvfOvaGcsPath = "gs://ovfbucket/ovffolder/"
	params.MachineType = ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockOvfDescriptorLoader := ovfdomainmocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load("gs://ovfbucket/ovffolder/").Return(
		nil, errors.New("ovf desc error"))

	oi := OVFImporter{workflowPath: instanceImportWorkflowPath,
		ovfDescriptorLoader: mockOvfDescriptorLoader,
		Logger:              logging.NewToolLogger("test"), params: params}
	w, err := oi.setUpImportWorkflow()

	assert.NotNil(t, err)
	assert.Nil(t, w)
	assert.Equal(t, "", oi.gcsPathToClean)
}

func TestSetUpWorkOSIdFromOVFDescriptor(t *testing.T) {
	params := getAllInstanceImportParams()
	params.OsID = ""
	params.OvfOvaGcsPath = "gs://ovfbucket/ovffolder/"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	w, err := setupMocksForOSIdTesting(mockCtrl, "rhel7_64Guest", params)

	assert.Nil(t, err)
	assert.NotNil(t, w)
	assert.Equal(t, "../image_import/enterprise_linux/translate_rhel_7_licensed.wf.json", w.Vars["translate_workflow"].Value)
}

func TestSetUpWorkOSIdFromDescriptorInvalidAndOSFlagNotSpecified(t *testing.T) {
	params := getAllInstanceImportParams()
	params.OsID = ""
	params.OvfOvaGcsPath = "gs://ovfbucket/ovffolder/"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	w, err := setupMocksForOSIdTesting(mockCtrl, "no-OS-ID", params)

	assert.Nil(t, w)
	assert.NotNil(t, err)
}

func TestSetUpWorkOSIdFromDescriptorNonDeterministicAndOSFlagNotSpecified(t *testing.T) {
	params := getAllInstanceImportParams()
	params.OsID = ""
	params.OvfOvaGcsPath = "gs://ovfbucket/ovffolder/"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	w, err := setupMocksForOSIdTesting(mockCtrl, "ubuntu64Guest", params)

	assert.Nil(t, w)
	assert.NotNil(t, err)
}

func TestSetUpWorkOSFlagInvalid(t *testing.T) {
	params := getAllInstanceImportParams()
	params.OsID = "not-OS-ID"
	params.OvfOvaGcsPath = "gs://ovfbucket/ovffolder/"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	w, err := setupMocksForOSIdTesting(mockCtrl, "", params)

	assert.Nil(t, w)
	assert.NotNil(t, err)
}

func TestCleanUp(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().DeleteGcsPath("aPath")
	mockStorageClient.EXPECT().Close()

	oi := OVFImporter{storageClient: mockStorageClient, gcsPathToClean: "aPath",
		Logger: logging.NewToolLogger("test")}
	oi.CleanUp()
}

func TestHandleTimeoutSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().User("Timeout 0s exceeded, stopping workflow \"wf\"")

	params := getAllInstanceImportParams()
	params.Timeout = "0s"
	params.Deadline = time.Now()

	oi := OVFImporter{Logger: mockLogger, params: params}
	w := daisy.New()
	w.Name = "wf"
	oi.handleTimeout(w)

	_, channelOpen := <-w.Cancel
	assert.False(t, channelOpen, "w.Cancel should be closed on timeout")
}

func setupMocksForOSIdTesting(mockCtrl *gomock.Controller, osType string,
	params *domain.OVFImportParams) (*daisy.Workflow, error) {

	mockOvfDescriptorLoader := ovfdomainmocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)

	descriptor := createOVFDescriptor([]string{
		"Ubuntu_for_Horizon71_1_1.0-disk1.vmdk",
		"Ubuntu_for_Horizon71_1_1.0-disk2.vmdk",
		"Ubuntu_for_Horizon71_1_1.0-disk3.vmdk",
	})
	if osType != "" {
		descriptor.VirtualSystem.OperatingSystem = []ovf.OperatingSystemSection{{OSType: &osType}}
	}
	mockOvfDescriptorLoader.EXPECT().Load("gs://ovfbucket/ovffolder/").Return(
		descriptor, nil)

	oi := OVFImporter{workflowPath: instanceImportWorkflowPath,
		ovfDescriptorLoader: mockOvfDescriptorLoader, Logger: logging.NewToolLogger("test"),
		params: params}
	return oi.setUpImportWorkflow()
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

func createOVFDescriptor(vmdkNames []string) *ovf.Envelope {
	virtualHardware := ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("5", 6),
			createCPUItem("11", 16),
			createMemoryItem("12", 4096),
		},
	}

	diskCapacityAllocationUnits := "byte * 2^30"
	ovfDescriptor := &ovf.Envelope{
		Disk:       &ovf.DiskSection{Disks: []ovf.VirtualDiskDesc{}},
		References: []ovf.File{},
	}
	for i := 0; i < len(vmdkNames); i++ {
		uri := fmt.Sprintf("ovf:/disk/vmdisk%d", i+1)
		elementName := fmt.Sprintf("disk%d", i)
		addressOnParent := fmt.Sprintf("%d", i)
		resourceID := fmt.Sprintf("%d", i+6)
		virtualHardware.Item = append(virtualHardware.Item,
			createDiskItem(resourceID, addressOnParent, elementName, uri, "5"),
		)

		fileRef := fmt.Sprintf("file%d", i+1)
		ovfDescriptor.Disk.Disks = append(ovfDescriptor.Disk.Disks,
			ovf.VirtualDiskDesc{
				Capacity:                "20",
				CapacityAllocationUnits: &diskCapacityAllocationUnits,
				DiskID:                  fmt.Sprintf("vmdisk%d", i+1),
				FileRef:                 &fileRef,
			},
		)
		ovfDescriptor.References = append(ovfDescriptor.References,
			ovf.File{
				Href: vmdkNames[i],
				ID:   fileRef,
				Size: 1151322112,
			},
		)
	}
	ovfDescriptor.VirtualSystem = &ovf.VirtualSystem{
		VirtualHardware: []ovf.VirtualHardwareSection{virtualHardware},
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

type DummyLogger struct{}

func (dl DummyLogger) WriteLogEntry(e *daisy.LogEntry)                                          {}
func (dl DummyLogger) WriteSerialPortLogs(w *daisy.Workflow, instance string, buf bytes.Buffer) {}
func (dl DummyLogger) Flush()                                                                   {}
func (dl DummyLogger) ReadSerialPortLogs() []string                                             { return nil }
