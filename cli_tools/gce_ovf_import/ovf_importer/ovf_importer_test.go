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
	"io/ioutil"
	"strings"
	"testing"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/ovf"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image"
	imagemocks "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	ovfdomainmocks "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

// importTarget hold data specific to either instances or machine images.
type importTarget struct {
	name           string
	wfPath         string
	paramGenerator func() *ovfdomain.OVFImportParams
}

var (
	gmiMode = &importTarget{
		name:           "gmi",
		wfPath:         "../../../daisy_workflows/" + createGMIWorkflow,
		paramGenerator: getAllMachineImageImportParams,
	}
	instanceMode = &importTarget{
		name:           "instance",
		wfPath:         "../../../daisy_workflows/" + createInstanceWorkflow,
		paramGenerator: getAllInstanceImportParams,
	}
)

func TestSetupWorkflow_HappyCase(t *testing.T) {
	for _, mode := range []*importTarget{gmiMode, instanceMode} {
		t.Run(mode.name, func(t *testing.T) {
			runImportAndVerify(t, mode.paramGenerator(), mode)
		})
	}
}

func TestSetupWorkflow_WithUserSpecifiedMachineType(t *testing.T) {
	for _, mode := range []*importTarget{gmiMode, instanceMode} {
		t.Run(mode.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			params := mode.paramGenerator()
			params.MachineType = "e2-small2"
			testCase := mockConfiguration{
				descriptorFilenames: []string{"Ubuntu_for_Horizon71_1_1.0-disk1.vmdk"},
				fileURIs:            []string{"gs://bucket/folder/ovf/Ubuntu_for_Horizon71_1_1.0-disk1.vmdk"},
				images:              []domain.Image{image.NewImage("project", "boot-disk")},
				expectedOS:          params.OsID,
				expectImportToRun:   true,
			}
			descriptor := createOVFDescriptor(testCase.descriptorFilenames)
			worker, err := setupMocksAndRun(mockCtrl, params, mode.wfPath, descriptor, testCase)
			assert.NoError(t, err)
			daisyutils.CheckWorkflow(worker, func(wf *daisy.Workflow, err error) {
				assertMachineType(t, wf, "e2-small2")
			})
		})
	}
}

func TestSetupWorkflow_WithMachineTypeInference(t *testing.T) {
	for _, mode := range []*importTarget{gmiMode, instanceMode} {
		t.Run(mode.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			params := mode.paramGenerator()
			params.MachineType = ""
			testCase := mockConfiguration{
				descriptorFilenames: []string{"Ubuntu_for_Horizon71_1_1.0-disk1.vmdk"},
				fileURIs:            []string{"gs://bucket/folder/ovf/Ubuntu_for_Horizon71_1_1.0-disk1.vmdk"},
				images:              []domain.Image{image.NewImage("project", "boot-disk")},
				expectedOS:          params.OsID,
				expectImportToRun:   true,
			}
			descriptor := createOVFDescriptor(testCase.descriptorFilenames)
			worker, err := setupMocksAndRun(mockCtrl, params, mode.wfPath, descriptor, testCase)
			assert.NoError(t, err)
			daisyutils.CheckWorkflow(worker, func(w *daisy.Workflow, err error) {
				assertMachineType(t, w, "n1-highcpu-16")
			})
		})
	}
}

func TestDiskImport_ErrorUnpackingOVA(t *testing.T) {
	params := getAllInstanceImportParams()
	project := defaultProject
	params.Project = &project
	params.Zone = "europe-north1-b"
	params.MachineType = ""
	expectedError := errors.New("tar error")

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMockTarGcsExtractorInterface := mocks.NewMockTarGcsExtractorInterface(mockCtrl)
	mockMockTarGcsExtractorInterface.EXPECT().ExtractTarToGcs(
		"gs://ovfbucket/ovfpath/vmware.ova", params.ScratchBucketGcsPath+"/ovf").
		Return(expectedError).Times(1)

	oi := OVFImporter{workflowPath: instanceMode.wfPath,
		tarGcsExtractor: mockMockTarGcsExtractorInterface, Logger: logging.NewToolLogger("test"),
		params: params}
	err := oi.importDisksFiles()
	assert.Equal(t, expectedError, err)
}

func TestDiskImport_ErrorLoadingDescriptor(t *testing.T) {
	params := getAllInstanceImportParams()
	project := defaultProject
	params.Project = &project
	params.Zone = "europe-north1-b"
	params.OvfOvaGcsPath = "gs://ovfbucket/ovffolder/"
	params.MachineType = ""
	expectedError := errors.New("ovf desc error")

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockOvfDescriptorLoader := ovfdomainmocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load("gs://ovfbucket/ovffolder/").Return(
		nil, expectedError)

	oi := OVFImporter{workflowPath: instanceMode.wfPath,
		ovfDescriptorLoader: mockOvfDescriptorLoader,
		Logger:              logging.NewToolLogger("test"), params: params}
	err := oi.importDisksFiles()

	assert.Equal(t, expectedError, err)
	assert.Equal(t, "", oi.gcsPathToClean)
}

func TestBootDiskImageImport_buildBootDiskImageImportRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().NewReader().Return(ioutil.NopCloser(strings.NewReader("file content")), nil)
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockStorageObject)
	params := getAllInstanceImportParams()
	project := defaultProject
	params.Project = &project
	params.OvfOvaGcsPath = "gs://ovfbucket/ovffolder/"
	params.BYOL = true
	params.OsID = ""
	imageName := "img-1"

	fileURIs := []string{"gs://bucket/disk1.vmdk", "gs://bucket/disk2.vmdk"}

	disk1, disk1Err := disk.NewDisk("project1", "zone-1", "disk1")
	disk2, disk2Err := disk.NewDisk("project1", "zone-1", "disk2")
	assert.NoError(t, disk1Err)
	assert.NoError(t, disk2Err)

	oi := OVFImporter{
		workflowPath:  instanceMode.wfPath,
		Logger:        logging.NewToolLogger("test"),
		params:        params,
		disks:         []domain.Disk{disk1, disk2},
		storageClient: mockStorageClient,
	}

	request, err := oi.buildBootDiskImageImportRequest(imageName, fileURIs[0])
	assert.NoError(t, err)
	assert.Equal(t, request.ImageName, imageName)
	assert.Equal(t, request.ExecutionID, imageName)
	assert.Equal(t, request.CloudLogsDisabled, params.CloudLogsDisabled)
	assert.Equal(t, request.ComputeEndpoint, params.Ce)
	assert.Equal(t, request.ComputeServiceAccount, params.ComputeServiceAccount)
	assert.Equal(t, request.WorkflowDir, params.WorkflowDir)
	assert.Equal(t, request.GcsLogsDisabled, params.GcsLogsDisabled)
	assert.Equal(t, request.Network, params.Network)
	assert.Equal(t, request.NoExternalIP, params.NoExternalIP)
	assert.Equal(t, request.NoGuestEnvironment, params.NoGuestEnvironment)
	assert.Equal(t, request.Oauth, params.Oauth)
	assert.Equal(t, request.Project, *params.Project)
	assert.Equal(t, request.ScratchBucketGcsPath, path.JoinURL(params.ScratchBucketGcsPath, imageName))
	assert.Equal(t, request.Source.Path(), fileURIs[0])
	assert.Equal(t, request.StdoutLogsDisabled, params.StdoutLogsDisabled)
	assert.Equal(t, request.Subnet, params.Subnet)
	assert.Equal(t, request.Tool, params.GetTool())
	assert.Equal(t, request.UefiCompatible, params.UefiCompatible)
	assert.Equal(t, request.Zone, params.Zone)
	assert.Equal(t, request.DataDisks, oi.disks)
	assert.True(t, request.BYOL)
}

func TestSetUpWork_OSIDs(t *testing.T) {
	tests := []struct {
		name                 string
		osTypeFromDescriptor string
		osIDFromUser         string
		expectedOSID         string
		expectedError        string
	}{
		{
			name:         "Use osID from user when specified and descriptor osID empty",
			osIDFromUser: "ubuntu-1804",
			expectedOSID: "ubuntu-1804",
		},
		{
			name:                 "Use osID from user, even when descriptor present",
			osTypeFromDescriptor: "rhel7_64Guest",
			osIDFromUser:         "ubuntu-1804",
			expectedOSID:         "ubuntu-1804",
		},
		{
			name:                 "Use osID from descriptor when descriptor valid and osID not specified by user",
			osTypeFromDescriptor: "rhel7_64Guest",
			expectedOSID:         "rhel-7",
		},
		{
			name:          "Error when osID from user is invalid",
			osIDFromUser:  "os-id-that-isnt-valid",
			expectedError: "os `os-id-that-isnt-valid` is invalid",
		},
		{
			name:          "Error when osID from user not supported for import",
			osIDFromUser:  "ubuntu-1004",
			expectedError: "os `ubuntu-1004` is invalid",
		},
		{
			name:                 "Use OS detection when osID not specified by user and osID from descriptor is ambiguous",
			osTypeFromDescriptor: "ubuntu64Guest",
			expectedOSID:         "",
		},
		{
			name:                 "Use OS detection when osID not specified by user and osID from descriptor is invalid",
			osTypeFromDescriptor: "os-type-that-isnt-valid",
			expectedOSID:         "",
		},
		{
			name:         "Error when osID not specified by user or descriptor",
			expectedOSID: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			descriptorFilenames := []string{"Ubuntu_for_Horizon71_1_1.0-disk1.vmdk"}
			params := getAllInstanceImportParams()
			params.OsID = tc.osIDFromUser
			descriptor := createOVFDescriptor(descriptorFilenames)
			if tc.osTypeFromDescriptor != "" {
				descriptor.VirtualSystem.OperatingSystem = []ovf.OperatingSystemSection{{OSType: &tc.osTypeFromDescriptor}}
			}
			worker, err := setupMocksAndRun(mockCtrl, params, instanceMode.wfPath, descriptor, mockConfiguration{
				descriptorFilenames: descriptorFilenames,
				fileURIs:            []string{"gs://bucket/folder/ovf/Ubuntu_for_Horizon71_1_1.0-disk1.vmdk"},
				images:              []domain.Image{image.NewImage("project", "boot-disk")},
				expectedOS:          tc.expectedOSID,
				expectImportToRun:   tc.expectedError == "",
			})
			if tc.expectedError == "" {
				assert.NoError(t, err)
				assert.NotNil(t, worker)
			} else {
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, worker)
			}
		})
	}
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

func TestGetImportWorkflowPath(t *testing.T) {
	for _, mode := range []*importTarget{gmiMode, instanceMode} {
		t.Run(mode.name, func(t *testing.T) {
			params := mode.paramGenerator()
			params.WorkflowDir = "workflow/dir"
			wfPath := getImportWorkflowPath(params)
			if mode == gmiMode {
				assert.Equal(t, "workflow/dir/ovf_import/create_gmi.wf.json", wfPath)
			} else {
				assert.Equal(t, "workflow/dir/ovf_import/create_instance.wf.json", wfPath)
			}
		})
	}
}

func TestBuildDaisyVars_NetworkAndSubnets(t *testing.T) {
	tests := []struct {
		network      string
		subnet       string
		expectedVars map[string]string
	}{
		{
			network: "",
			subnet:  "",
		},
		{
			network: "private-net",
			subnet:  "",
			expectedVars: map[string]string{
				"network": "private-net",
			},
		},
		{
			network: "",
			subnet:  "private-sub",
			expectedVars: map[string]string{
				"network": "",
				"subnet":  "private-sub",
			},
		},
		{
			network: "private-net",
			subnet:  "private-sub",
			expectedVars: map[string]string{
				"network": "private-net",
				"subnet":  "private-sub",
			},
		},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("net=%q,sub=%q", tc.network, tc.subnet), func(t *testing.T) {
			params := getAllInstanceImportParams()
			params.Network = tc.network
			params.Subnet = tc.subnet
			actualParams := (&OVFImporter{params: params}).buildDaisyVars(image.NewImage(*params.Project, "image-name"), "")
			for _, key := range []string{"network", "subnet"} {
				if val, found := tc.expectedVars[key]; found {
					assert.Equal(t, val, actualParams[key])
				} else {
					assert.NotContains(t, actualParams, key)
				}
			}
		})
	}
}

func TestGetOvfGcsPath(t *testing.T) {
	tests := []struct {
		name          string
		tmpGcsPath    string
		ovfPath       string
		expectExtract bool
		expectedPath  string
		expectedError error
		expectCleanup bool
	}{
		{
			name:         "return directory when user gives descriptor",
			tmpGcsPath:   "gs://tmp-bucket/",
			ovfPath:      "gs://res-bucket/path/to/descriptor.ovf",
			expectedPath: "gs://res-bucket/path/to/",
		}, {
			name:          "extract to tmp when user gives archive",
			tmpGcsPath:    "gs://tmp-bucket/",
			ovfPath:       "gs://res-bucket/archive.ova",
			expectedPath:  "gs://tmp-bucket/ovf",
			expectExtract: true,
			expectCleanup: true,
		}, {
			name:          "error when extraction fails",
			tmpGcsPath:    "gs://tmp-bucket/",
			ovfPath:       "gs://res-bucket/archive.ova",
			expectedPath:  "gs://tmp-bucket/ovf",
			expectedError: errors.New("extraction error"),
			expectExtract: true,
			expectCleanup: true,
		}, {
			name:         "return directory when user gives directory",
			tmpGcsPath:   "gs://tmp-bucket/",
			ovfPath:      "gs://res-bucket/directory/",
			expectedPath: "gs://res-bucket/directory/",
		},
		{
			name:         "return directory when user gives non-OVA and non-OVF file",
			tmpGcsPath:   "gs://tmp-bucket/",
			ovfPath:      "gs://res-bucket/file-like",
			expectedPath: "gs://res-bucket/file-like/",
		},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			params := getAllInstanceImportParams()
			params.OvfOvaGcsPath = tc.ovfPath

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockExtractor := mocks.NewMockTarGcsExtractorInterface(mockCtrl)
			if tc.expectExtract {
				mockExtractor.EXPECT().ExtractTarToGcs(tc.ovfPath, tc.expectedPath).Return(tc.expectedError)
			}
			oi := &OVFImporter{
				Logger:          logging.NewToolLogger("test"),
				params:          params,
				tarGcsExtractor: mockExtractor,
			}
			actualPath, actualCleanup, actualErr := oi.getOvfGcsPath(tc.tmpGcsPath)
			assert.Equal(t, path.ToDirectoryURL(tc.expectedPath), actualPath, "always return path with trailing slash")
			assert.Equal(t, tc.expectCleanup, actualCleanup)
			assert.Equal(t, tc.expectedError, actualErr)
		})
	}
}

func TestToWorkingDir(t *testing.T) {
	tests := []struct {
		dir            string
		executablePath string
		expectedResult string
	}{
		{
			dir:            "/absolute",
			executablePath: "/not/used",
			expectedResult: "/absolute",
		},
		{
			dir:            "../workflows/daisy.wf.json",
			executablePath: "/opt/bin/",
			expectedResult: "/opt/workflows/daisy.wf.json",
		},
		{
			dir:            "./workflows/daisy.wf.json",
			executablePath: "/opt/bin/",
			expectedResult: "/opt/bin/workflows/daisy.wf.json",
		},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("executable=%q,dir=%q", tc.executablePath, tc.dir), func(t *testing.T) {
			params := getAllInstanceImportParams()
			params.CurrentExecutablePath = tc.executablePath
			assert.Equal(t, tc.expectedResult, toWorkingDir(tc.dir, params))
		})
	}
}

func runImportAndVerify(t *testing.T, params *ovfdomain.OVFImportParams, mode *importTarget) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	instanceName := "instance1"
	if mode == gmiMode {
		instanceName = "machineimage1"
	}

	disk1, disk1Err := disk.NewDisk("project-name", params.Zone, fmt.Sprintf("disk-%s-0", instanceName))
	disk2, disk2Err := disk.NewDisk("project-name", params.Zone, fmt.Sprintf("disk-%s-1", instanceName))
	assert.NoError(t, disk1Err)
	assert.NoError(t, disk2Err)

	testCase := mockConfiguration{
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
		images: []domain.Image{
			image.NewImage("project-name", fmt.Sprintf("%s-boot-image", instanceName)),
		},
		disks:             []domain.Disk{disk1, disk2},
		expectedOS:        params.OsID,
		expectImportToRun: true,
	}

	descriptor := createOVFDescriptor(testCase.descriptorFilenames)
	worker, err := setupMocksAndRun(mockCtrl, params, mode.wfPath, descriptor, testCase)
	if err != nil {
		t.Fatal(err)
	}
	daisyutils.CheckEnvironment(worker, func(env daisyutils.EnvironmentSettings) {
		assert.Equal(t, params.NoExternalIP, env.NoExternalIP)
	})
	daisyutils.CheckResourceLabeler(worker, func(rl *daisyutils.ResourceLabeler) {
		assert.Equal(t, params.BuildID, rl.BuildID)
		assert.Equal(t, params.UserLabels, rl.UserLabels)
		assert.Equal(t, "gce-ovf-import-build-id", rl.BuildIDLabelKey)
		assert.Equal(t, params.Region, rl.ImageLocation)
		assert.Equal(t, "gce-ovf-import", rl.InstanceLabelKeyRetriever(params.InstanceNames))
		assert.Equal(t, "gce-ovf-import-tmp", rl.DiskLabelKeyRetriever(nil))
		assert.Equal(t, "gce-ovf-import-tmp", rl.ImageLabelKeyRetriever("imgname"))
	})
	daisyutils.CheckWorkflow(worker, func(w *daisy.Workflow, err error) {
		// Workflow validation
		assert.Equal(t, *params.Project, w.Project)
		assert.Equal(t, params.Timeout, w.DefaultTimeout)
		assert.Equal(t, params.Zone, w.Zone)
		assert.Equal(t, params.Oauth, w.OAuthPath)
		assert.Equal(t, params.Ce, w.ComputeEndpoint)
		assert.Equal(t, params.ScratchBucketGcsPath, w.GCSPath)
		if mode == gmiMode {
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
		if mode == gmiMode {
			assert.Len(t, *w.Steps["create-machine-image"].CreateMachineImages, 1, "Expect one GMI created")
		}

		instance := w.Steps["create-instance"].CreateInstances.Instances[0]
		cleanup := w.Steps["cleanup"].DeleteResources

		// Boot Disk
		bootDisk := instance.Disks[0]
		checkDaisyVariable(t, w, "boot_disk_image_uri", testCase.images[0].GetURI(), bootDisk.InitializeParams.SourceImage)
		assert.True(t, bootDisk.AutoDelete, "Delete boot disk when instance is deleted.")
		assert.True(t, bootDisk.Boot, "Boot disk is configured to boot.")
		assert.Contains(t, cleanup.Images, "${boot_disk_image_uri}", "Delete the boot disk image after instance creation.")
		assert.Len(t, cleanup.Images, len(testCase.images))

		// Data Disks
		assert.Len(t, instance.Disks, len(descriptor.Disk.Disks))
		for i, disk := range instance.Disks[1:] {
			assert.Equal(t, disk.Source, testCase.disks[i].GetURI())
			assert.True(t, disk.AutoDelete, "Delete the disk when the instance is deleted.")
			assert.False(t, disk.Boot, "Data disk are not configured to boot.")
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

		// Network
		assert.Len(t, instance.NetworkInterfaces, 1, "Expect one network to be created")
		networkInterface := instance.NetworkInterfaces[0]
		checkDaisyVariable(t, w, "network", params.Network, networkInterface.Network)
		checkDaisyVariable(t, w, "subnet", params.Subnet, networkInterface.Subnetwork)
		checkDaisyVariable(t, w, "private_network_ip", params.PrivateNetworkIP, networkInterface.NetworkIP)

		// Cleanup
		if mode == gmiMode {
			machineImage := []*daisy.MachineImage(*w.Steps["create-machine-image"].CreateMachineImages)[0]
			checkDaisyVariable(t, w, "machine_image_name", params.MachineImageName, machineImage.Name)
			assert.Equal(t, instance.Name, machineImage.SourceInstance)
			checkDaisyVariable(t, w, "description", params.Description, machineImage.Description)
			assert.True(t, machineImage.ExactName)
			assert.True(t, machineImage.NoCleanup)
			assert.Equal(t, []string{instance.Name}, cleanup.Instances)
		}
	})
}

// checkDaisyVariable ensures that a variable is declared, the desired value is injected, and that it's
// being used at a location within the workflow.
func checkDaisyVariable(t *testing.T, w *daisy.Workflow, declaredVariableName string, expectedValue string, expectedLocationInTemplate string) {
	assert.Equal(t, expectedValue, w.Vars[declaredVariableName].Value)
	assert.Equal(t, fmt.Sprintf("${%s}", declaredVariableName), expectedLocationInTemplate)
}

type mockConfiguration struct {
	expectedOS          string
	expectImportToRun   bool
	descriptorFilenames []string
	fileURIs            []string
	images              []domain.Image
	disks               []domain.Disk
}

func setupMocksAndRun(mockCtrl *gomock.Controller, params *ovfdomain.OVFImportParams, wfPath string, descriptor *ovf.Envelope, mockConfig mockConfiguration) (daisyutils.DaisyWorker, error) {
	params.InstanceNames = strings.ToLower(strings.TrimSpace(params.InstanceNames))
	params.MachineImageName = strings.ToLower(strings.TrimSpace(params.MachineImageName))
	expectedParams := *params
	expectedParams.OsID = mockConfig.expectedOS
	expectedParams.Deadline = params.Deadline.Add(-1 * instanceConstructionTime)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)

	mockComputeClient.EXPECT().CreateDisk(*params.Project, params.Zone, gomock.Any()).Return(nil).AnyTimes()

	var imageName string
	if params.InstanceNames == "" {
		imageName = params.MachineImageName
	} else {
		imageName = params.InstanceNames
	}
	imageName += "-boot-image"

	if params.MachineType == "" {
		mockComputeClient.EXPECT().ListMachineTypes(*params.Project, params.Zone).Return(machineTypes, nil)
	}
	mockOvfDescriptorLoader := ovfdomainmocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load(params.ScratchBucketGcsPath+"/ovf/").Return(
		descriptor, nil)
	mockMockTarGcsExtractorInterface := mocks.NewMockTarGcsExtractorInterface(mockCtrl)
	mockMockTarGcsExtractorInterface.EXPECT().ExtractTarToGcs(
		params.OvfOvaGcsPath, params.ScratchBucketGcsPath+"/ovf").
		Return(nil).Times(1)

	mockImporter := imagemocks.NewMockImporter(mockCtrl)
	mockMultiDiskImporter := ovfdomainmocks.NewMockMultiDiskImporterInterface(mockCtrl)

	oi := OVFImporter{ctx: context.Background(), workflowPath: wfPath,
		storageClient: mockStorageClient, computeClient: mockComputeClient,
		ovfDescriptorLoader: mockOvfDescriptorLoader, tarGcsExtractor: mockMockTarGcsExtractorInterface,
		imageImporter:     mockImporter,
		multiDiskImporter: mockMultiDiskImporter,
		Logger:            logging.NewToolLogger("test"), params: params}

	if mockConfig.expectImportToRun {
		mockImporter.EXPECT().Run(oi.ctx)
		mockStorageObject.EXPECT().NewReader().Return(ioutil.NopCloser(strings.NewReader("file content")), nil)
		mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockStorageObject)
		if len(mockConfig.descriptorFilenames) > 1 {
			mockMultiDiskImporter.EXPECT().Import(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockConfig.disks, nil)
		}
	}

	err := oi.importDisksFiles()
	if err != nil {
		return nil, err
	}
	return oi.createWorkerForFinalInstance(), nil
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

func assertMachineType(t *testing.T, w *daisy.Workflow, expectedType string) {
	checkDaisyVariable(t, w, "machine_type", expectedType, w.Steps["create-instance"].CreateInstances.Instances[0].MachineType)
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
