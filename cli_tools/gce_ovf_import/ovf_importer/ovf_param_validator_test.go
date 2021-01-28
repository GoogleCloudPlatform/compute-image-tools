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
	"errors"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

const (
	defaultProject = "project-name"
	defaultZone    = "us-west2-1"
	defaultBuildID = "build123"
)

func TestMachineImageStorageLocationProvidedForInstanceImport(t *testing.T) {
	params := getAllInstanceImportParams()
	params.MachineImageStorageLocation = "us-west2"
	assertErrorOnValidate(t, params, "-machine-image-storage-location can't be provided when importing an instance")
}

func Test_ValidateAndParseParams_Fail_WhenRegionCantBeFoundFromZone(t *testing.T) {
	params := getAllInstanceImportParams()
	params.Zone = "uscentral1"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	mockZoneValidator.EXPECT().
		ZoneValid(*params.Project, params.Zone).Return(nil)

	err := (&ParamValidatorAndPopulator{zoneValidator: mockZoneValidator}).ValidateAndPopulate(params)
	assert.Contains(t, err.Error(), "uscentral1 is not a valid zone")
}

func Test_ValidateAndParseParams_Fail_WhenZoneMissingAndLookupFails(t *testing.T) {
	params := getAllInstanceImportParams()
	params.Zone = ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true).AnyTimes()
	mockMetadataGce.EXPECT().Zone().Return("", errors.New("zone not found"))
	mockMetadataGce.EXPECT().ProjectID().Return(defaultProject, nil).AnyTimes()

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	if params.Zone != "" {
		mockZoneValidator.EXPECT().
			ZoneValid(defaultProject, defaultZone).Return(nil)
	}

	err := (&ParamValidatorAndPopulator{metadataClient: mockMetadataGce, zoneValidator: mockZoneValidator}).ValidateAndPopulate(params)
	assert.Contains(t, err.Error(), "can't infer zone: zone not found")
}

func Test_ValidateAndParseParams_Fail_WhenZoneMissingAndNotOnGCE(t *testing.T) {
	params := getAllInstanceImportParams()
	params.Zone = ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).AnyTimes()
	mockMetadataGce.EXPECT().ProjectID().Return(defaultProject, nil).AnyTimes()

	err := (&ParamValidatorAndPopulator{metadataClient: mockMetadataGce}).ValidateAndPopulate(params)
	assert.Contains(t, err.Error(), "zone cannot be determined because build is not running on Google Compute Engine")
}

func Test_ValidateAndParseParams_Fail_WhenZoneFailsValidation(t *testing.T) {
	params := getAllInstanceImportParams()
	params.Zone = "zzz-east"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	mockZoneValidator.EXPECT().
		ZoneValid(*params.Project, params.Zone).Return(errors.New("unrecognized zone"))

	err := (&ParamValidatorAndPopulator{zoneValidator: mockZoneValidator}).ValidateAndPopulate(params)
	assert.Contains(t, err.Error(), "unrecognized zone")
}

func Test_ValidateAndParseParams_GenerateBucketName_WhenNotProvided(t *testing.T) {
	projectName := "test-google"
	params := getAllInstanceImportParams()
	params.Region = "us-west2"
	params.Zone = "us-west2-a"
	params.ScratchBucketGcsPath = ""
	params.Project = &projectName
	expectedBucketName := "test-elgoog-ovf-import-bkt-us-west2"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	mockZoneValidator.EXPECT().
		ZoneValid(projectName, params.Zone).Return(nil)

	someBucketAttrs := &storage.BucketAttrs{
		Name:     expectedBucketName,
		Location: "us-west2",
	}
	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	mockBucketIterator.EXPECT().Next().Return(someBucketAttrs, nil)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().CreateBucketIterator(gomock.Any(), gomock.Any(), *params.Project).Return(mockBucketIterator)

	mockStorage := mocks.NewMockStorageClientInterface(mockCtrl)
	err := (&ParamValidatorAndPopulator{
		zoneValidator:         mockZoneValidator,
		bucketIteratorCreator: mockBucketIteratorCreator,
		logger:                logging.NewToolLogger("test"),
		storageClient:         mockStorage,
	}).ValidateAndPopulate(params)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("gs://%s/%s", expectedBucketName, params.BuildID), params.ScratchBucketGcsPath)
}

func Test_ValidateAndParseParams_CreateScratchBucket_WhenGeneratedDoesntExist(t *testing.T) {
	projectName := "goog-test"
	params := getAllInstanceImportParams()
	params.Region = "us-west2"
	params.Zone = "us-west2-a"
	params.ScratchBucketGcsPath = ""
	params.Project = &projectName
	expectedBucketName := "ggoo-test-ovf-import-bkt-us-west2"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	mockZoneValidator.EXPECT().
		ZoneValid(projectName, params.Zone).Return(nil)

	someBucketAttrs := &storage.BucketAttrs{
		Name:     "other-bucket",
		Location: "us-west2",
	}
	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	mockBucketIterator.EXPECT().Next().Return(someBucketAttrs, nil)
	mockBucketIterator.EXPECT().Next().Return(nil, iterator.Done)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().CreateBucketIterator(gomock.Any(), gomock.Any(), *params.Project).Return(mockBucketIterator)

	mockStorage := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorage.EXPECT().CreateBucket(expectedBucketName, projectName, &storage.BucketAttrs{Name: expectedBucketName, Location: params.Region})
	err := (&ParamValidatorAndPopulator{
		zoneValidator:         mockZoneValidator,
		bucketIteratorCreator: mockBucketIteratorCreator,
		logger:                logging.NewToolLogger("test"),
		storageClient:         mockStorage,
	}).ValidateAndPopulate(params)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("gs://%s/%s", expectedBucketName, params.BuildID), params.ScratchBucketGcsPath)
}

func Test_ValidateAndParseParams_ErrorMessages(t *testing.T) {
	type testCase struct {
		name                 string
		expectErrorToContain string
		paramModifier        func(params *domain.OVFImportParams)
	}

	cases := []testCase{
		{
			name: "One of {InstanceNames, MachineImageNames} is required",
			paramModifier: func(params *domain.OVFImportParams) {
				params.InstanceNames = ""
				params.MachineImageName = ""
			},
			expectErrorToContain: "Either the flag -instance-names or -machine-image-name must be provided",
		},
		{
			name: "Only one of {InstanceNames, MachineImageNames} allowed",
			paramModifier: func(params *domain.OVFImportParams) {
				params.InstanceNames = "a"
				params.MachineImageName = "a"
			},
			expectErrorToContain: "-instance-names and -machine-image-name can't be provided at the same time",
		},
		{
			name: "hostname is validated for syntax",
			paramModifier: func(params *domain.OVFImportParams) {
				params.Hostname = "host|name"
			},
			expectErrorToContain: "The flag `hostname` must conform to RFC 1035 requirements for valid hostnames",
		},
		{
			name: "hostname is validated for length",
			paramModifier: func(params *domain.OVFImportParams) {
				params.Hostname = "host.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain"
			},
			expectErrorToContain: "The flag `hostname` must conform to RFC 1035 requirements for valid hostnames",
		},
		{
			name: "require ClientID",
			paramModifier: func(params *domain.OVFImportParams) {
				params.ClientID = ""
			},
			expectErrorToContain: "flag -client-id must be provided",
		}, {
			name: "labels must be parseable",
			paramModifier: func(params *domain.OVFImportParams) {
				params.Labels = "NOT_VALID"
			},
			expectErrorToContain: "failed to parse key-value pair",
		}, {
			name: "Don't allow invalid timeout",
			paramModifier: func(params *domain.OVFImportParams) {
				params.Timeout = "300"
			},
			expectErrorToContain: "Error parsing timeout `300`",
		}, {
			name: "require OvfOvaGcsPath",
			paramModifier: func(params *domain.OVFImportParams) {
				params.OvfOvaGcsPath = ""
			},
			expectErrorToContain: "The flag -ovf-gcs-path must be provided",
		}, {
			name: "validate OvfOvaGcsPath",
			paramModifier: func(params *domain.OVFImportParams) {
				params.OvfOvaGcsPath = "%%%%%"
			},
			expectErrorToContain: "ovf-gcs-path should be a path to OVF or OVA package in Cloud Storage",
		}, {
			name: "validate ReleaseTrack",
			paramModifier: func(params *domain.OVFImportParams) {
				params.ReleaseTrack = "garbage"
			},
			expectErrorToContain: "invalid value for release-track flag",
		},
	}

	for _, importType := range []string{"instance", "gmi"} {
		for _, tt := range cases {
			t.Run(fmt.Sprintf("%s %s", importType, tt.name), func(t *testing.T) {
				var params *domain.OVFImportParams
				if importType == "instance" {
					params = getAllInstanceImportParams()
				} else {
					params = getAllMachineImageImportParams()
				}
				tt.paramModifier(params)
				assertErrorOnValidate(t, params, tt.expectErrorToContain)
			})
		}
	}
}

func Test_ValidateAndParseParams_SuccessfulCases(t *testing.T) {
	type testCase struct {
		name string
		// paramModifier allows test cases to customize parameters prior to validation.
		paramModifier func(params *domain.OVFImportParams)
		// checkResult is called after validation to determine whether the case passed.
		checkResult func(t *testing.T, params *domain.OVFImportParams)
	}

	cases := []testCase{
		{
			name: "replace empty network with default when subnet empty",
			paramModifier: func(params *domain.OVFImportParams) {
				params.Network = ""
				params.Subnet = ""
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, "global/networks/default", params.Network)
				assert.Equal(t, "", params.Subnet)
			},
		}, {
			name: "keep empty network when subnet specified",
			paramModifier: func(params *domain.OVFImportParams) {
				params.Network = ""
				params.Subnet = "secure-sub"
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, "", params.Network)
				assert.Equal(t, "regions/us-west2/subnetworks/secure-sub", params.Subnet)
			},
		}, {
			name: "make network and subnet into URI when specified",
			paramModifier: func(params *domain.OVFImportParams) {
				params.Network = "secure"
				params.Subnet = "secure-sub"
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, "global/networks/secure", params.Network)
				assert.Equal(t, "regions/us-west2/subnetworks/secure-sub", params.Subnet)
			},
		}, {
			name: "populate zone when missing",
			paramModifier: func(params *domain.OVFImportParams) {
				params.Zone = ""
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, defaultZone, params.Zone)
			},
		}, {
			name: "populate project when missing",
			paramModifier: func(params *domain.OVFImportParams) {
				empty := ""
				params.Project = &empty
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, defaultProject, *params.Project)
			},
		}, {
			name: "populate region when missing",
			paramModifier: func(params *domain.OVFImportParams) {
				params.Region = ""
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, "us-west2", params.Region)
			},
		}, {
			name: "Trailing slash on OvfOvaGcsPath may be omitted",
			paramModifier: func(params *domain.OVFImportParams) {
				params.OvfOvaGcsPath = "gs://bucket"
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, "gs://bucket", params.OvfOvaGcsPath)
			},
		}, {
			name: "OvfOvaGcsPath may be included",
			paramModifier: func(params *domain.OVFImportParams) {
				params.OvfOvaGcsPath = "gs://bucket/"
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, "gs://bucket/", params.OvfOvaGcsPath)
			},
		}, {
			name: "Parse node affinities",
			paramModifier: func(params *domain.OVFImportParams) {
				params.NodeAffinityLabelsFlag = []string{"env,IN,prod,test"}
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Len(t, params.NodeAffinities, 1)
				assert.Len(t, params.NodeAffinitiesBeta, 1)
			},
		}, {
			name: "Parse labels",
			paramModifier: func(params *domain.OVFImportParams) {
				params.Labels = "env=test,region=us"
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, map[string]string{"env": "test", "region": "us"}, params.UserLabels)
			},
		}, {
			name: "Convert empty release track to GA",
			paramModifier: func(params *domain.OVFImportParams) {
				params.ReleaseTrack = ""
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, domain.GA, params.ReleaseTrack)
			},
		}, {
			name: "Init build ID if empty",
			paramModifier: func(params *domain.OVFImportParams) {
				params.BuildID = ""
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.NotEmpty(t, params.BuildID)
			},
		}, {
			name: "Keep build ID if set",
			paramModifier: func(params *domain.OVFImportParams) {
				params.BuildID = "abcd"
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, "abcd", params.BuildID)
			},
		}, {
			name: "Look for buildID in environment",
			paramModifier: func(params *domain.OVFImportParams) {
				err := os.Setenv("BUILD_ID", "xyz")
				if err != nil {
					t.Fatal(err)
				}
				params.BuildID = ""
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				assert.Equal(t, "xyz", params.BuildID)
				err := os.Unsetenv("BUILD_ID")
				if err != nil {
					t.Fatal(err)
				}
			},
		}, {
			name: "Calculate deadline from timeout",
			paramModifier: func(params *domain.OVFImportParams) {
				params.Timeout = "20h"
			},
			checkResult: func(t *testing.T, params *domain.OVFImportParams) {
				now := time.Now()
				expectedDeadline := now.Add(20 * time.Hour).Unix()
				actualDeadline := params.Deadline.Unix()

				diff := int(math.Abs(float64(expectedDeadline - actualDeadline)))
				assert.LessOrEqual(t, diff, 100)
			},
		},
	}

	for _, importType := range []string{"instance", "gmi"} {
		for _, tt := range cases {
			t.Run(fmt.Sprintf("%s %s", importType, tt.name), func(t *testing.T) {
				var params *domain.OVFImportParams
				if importType == "instance" {
					params = getAllInstanceImportParams()
				} else {
					params = getAllMachineImageImportParams()
				}
				tt.paramModifier(params)
				assertNoErrorOnValidate(t, params)
				tt.checkResult(t, params)
			})
		}
	}
}

func TestInstanceImportFlagsAllValid(t *testing.T) {
	assertNoErrorOnValidate(t, getAllInstanceImportParams())
}

func TestMachineImageImportFlagsAllValid(t *testing.T) {
	assertNoErrorOnValidate(t, getAllMachineImageImportParams())
}

func runValidateAndParseParams(t *testing.T, params *domain.OVFImportParams) error {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true).AnyTimes()
	mockMetadataGce.EXPECT().Zone().Return(defaultZone, nil).AnyTimes()
	mockMetadataGce.EXPECT().ProjectID().Return(defaultProject, nil).AnyTimes()

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	if params.Zone != "" {
		mockZoneValidator.EXPECT().
			ZoneValid(defaultProject, defaultZone).Return(nil)
	}

	err := (&ParamValidatorAndPopulator{metadataClient: mockMetadataGce, zoneValidator: mockZoneValidator}).ValidateAndPopulate(params)
	return err
}

func assertNoErrorOnValidate(t *testing.T, params *domain.OVFImportParams) {
	t.Helper()
	err := runValidateAndParseParams(t, params)
	if err != nil {
		t.Fatal(err)
	}
}

func assertErrorOnValidate(t *testing.T, params *domain.OVFImportParams, expectErrorToContain string) {
	t.Helper()
	err := runValidateAndParseParams(t, params)
	if err == nil {
		t.Fatal("expected error")
	}

	assert.Regexp(t, expectErrorToContain, err)
}

func getAllInstanceImportParams() *domain.OVFImportParams {
	project := defaultProject
	return &domain.OVFImportParams{
		BuildID:                     defaultBuildID,
		InstanceNames:               "instance1",
		ClientID:                    "aClient",
		OvfOvaGcsPath:               "gs://ovfbucket/ovfpath/vmware.ova",
		NoGuestEnvironment:          true,
		CanIPForward:                true,
		DeletionProtection:          true,
		Description:                 "aDescription",
		Labels:                      "userkey1=uservalue1,userkey2=uservalue2",
		MachineType:                 "n1-standard-2",
		Network:                     "aNetwork",
		Subnet:                      "aSubnet",
		NetworkTier:                 "PREMIUM",
		PrivateNetworkIP:            "10.0.0.1",
		NoExternalIP:                true,
		NoRestartOnFailure:          true,
		OsID:                        "ubuntu-1404",
		ShieldedIntegrityMonitoring: true,
		ShieldedSecureBoot:          true,
		ShieldedVtpm:                true,
		Tags:                        "tag1=val1",
		Zone:                        defaultZone,
		BootDiskKmskey:              "aKey",
		BootDiskKmsKeyring:          "aKeyring",
		BootDiskKmsLocation:         "aKmsLocation",
		BootDiskKmsProject:          "aKmsProject",
		Timeout:                     "3h",
		Deadline:                    time.Now().Add(time.Hour * 3),
		Project:                     &project,
		ScratchBucketGcsPath:        "gs://bucket/folder",
		Oauth:                       "oAuthFilePath",
		Ce:                          "us-east1-c",
		GcsLogsDisabled:             true,
		CloudLogsDisabled:           true,
		StdoutLogsDisabled:          true,
		NodeAffinityLabelsFlag:      []string{"env,IN,prod,test"},
		Hostname:                    "a-host.a-domain",
	}
}

func getAllMachineImageImportParams() *domain.OVFImportParams {
	params := getAllInstanceImportParams()
	params.InstanceNames = ""
	params.MachineImageName = "machineImage1"
	params.MachineImageStorageLocation = "us-west2"
	return params
}
