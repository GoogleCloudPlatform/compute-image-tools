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

package upgrader

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
)

func TestValidateParams(t *testing.T) {
	type testCase struct {
		testName        string
		u               *upgrader
		expectedError   string
		expectedTimeout string
	}

	var u *upgrader
	var tcs []testCase

	tcs = append(tcs, testCase{"Normal case", initTest(), "", DefaultTimeout})

	u = initTest()
	u.ClientID = ""
	tcs = append(tcs, testCase{"No client id", u, "The flag -client-id must be provided", DefaultTimeout})

	u = initTest()
	u.SourceOS = "android"
	tcs = append(tcs, testCase{"validateOSVersion failure", u,
		"Flag -source-os value 'android' unsupported. Please choose a supported version from {windows-2008r2}.", DefaultTimeout})

	u = initTest()
	u.Instance = "bad/url"
	tcs = append(tcs, testCase{"validateAndDeriveInstanceURI failure", u,
		"Please provide the instance flag either with the name of the instance or in the form of 'projects/<project>/zones/<zone>/instances/<instance>', not bad/url", DefaultTimeout})

	u = initTest()
	u.Instance = daisy.GetInstanceURI(testProject, testZone, testInstanceNoLicense)
	tcs = append(tcs, testCase{"validateAndDeriveInstance failure", u,
		"Can only upgrade GCE instance with projects/windows-cloud/global/licenses/windows-server-2008-r2-dc license attached", DefaultTimeout})

	u = initTest()
	u.Timeout = "1m"
	tcs = append(tcs, testCase{"override timeout", u, "", "1m"})

	for _, tc := range tcs {
		u = tc.u
		err := u.validateAndDeriveParams()
		if tc.expectedError != "" {
			assert.EqualErrorf(t, err, tc.expectedError, "[test name: %v] Unexpected error.", tc.testName)
		} else {
			assert.NoError(t, err, "[test name: %v] Unexpected error.", tc.testName)
		}
		if err != nil {
			continue
		}

		assert.Equalf(t, tc.expectedTimeout, u.Timeout, "[test name: %v] Unexpected Timeout.", tc.testName)
		assert.NotEmptyf(t, u.machineImageBackupName, "[test name: %v] Unexpected machineImageBackupName.", tc.testName)
		assert.NotEmptyf(t, u.osDiskSnapshotName, "[test name: %v] Unexpected osDiskSnapshotName.", tc.testName)
		assert.NotEmptyf(t, u.newOSDiskName, "[test name: %v] Unexpected newOSDiskName.", tc.testName)
		assert.NotEmptyf(t, u.installMediaDiskName, "[test name: %v] Unexpected installMediaDiskName.", tc.testName)
		assert.Equalf(t, testProject, *u.ProjectPtr, "[test name: %v] Unexpected ProjectPtr value.", tc.testName)
	}
}

func TestValidateOSVersion(t *testing.T) {
	type testCase struct {
		testName      string
		sourceOS      string
		targetOS      string
		expectedError string
	}

	tcs := []testCase{
		{"Unsupported source OS", "windows-2008", "windows-2008r2", "Flag -source-os value 'windows-2008' unsupported. Please choose a supported version from {windows-2008r2}."},
		{"Unsupported target OS", "windows-2008r2", "windows-2012", "Flag -target-os value 'windows-2012' unsupported. Please choose a supported version from {windows-2012r2}."},
		{"Source OS not provided", "", versionWindows2012r2, "Flag -source-os must be provided. Please choose a supported version from {windows-2008r2}."},
		{"Target OS not provided", versionWindows2008r2, "", "Flag -target-os must be provided. Please choose a supported version from {windows-2012r2}."},
	}
	for supportedSourceOS, supportedTargetOS := range supportedSourceOSVersions {
		tcs = append(tcs, testCase{
			fmt.Sprintf("From %v to %v", supportedSourceOS, supportedTargetOS),
			supportedSourceOS,
			supportedTargetOS,
			"",
		})
	}

	for _, tc := range tcs {
		err := validateOSVersion(tc.sourceOS, tc.targetOS)
		if tc.expectedError != "" {
			assert.EqualErrorf(t, err, tc.expectedError, "[test name: %v]", tc.testName)
		} else {
			assert.NoError(t, err, "[test name: %v]", tc.testName)
		}
	}
}

func TestValidateInstance(t *testing.T) {
	initTest()

	type testCase struct {
		testName             string
		instance             string
		expectedURIError     string
		expectedError        string
		inputProject         string
		inputZone            string
		expectedProject      string
		expectedZone         string
		expectedInstanceName string
		mgce                 domain.MetadataGCEInterface
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return(testProject2, nil)

	mockMetadataGceFail := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGceFail.EXPECT().OnGCE().Return(false)

	tcs := []testCase{
		{
			"Normal case without original startup script",
			daisy.GetInstanceURI(testProject, testZone, testInstance),
			"",
			"",
			"",
			"",
			testProject, testZone, testInstance,
			mockMetadataGce,
		},
		{
			"Normal case with original startup script",
			daisy.GetInstanceURI(testProject, testZone, testInstanceWithStartupScript),
			"",
			"",
			"",
			"",
			testProject, testZone, testInstanceWithStartupScript,
			mockMetadataGce,
		},
		{
			"Normal case with existing startup script backup",
			daisy.GetInstanceURI(testProject, testZone, testInstanceWithExistingStartupScriptBackup),
			"",
			"",
			"",
			"",
			testProject, testZone, testInstanceWithExistingStartupScriptBackup,
			mockMetadataGce,
		},
		{
			"No disk error",
			daisy.GetInstanceURI(testProject, testZone, testInstanceNoDisk),
			"",
			"No disks attached to the instance.",
			"",
			"",
			testProject, testZone, testInstanceNoDisk,
			mockMetadataGce,
		},
		{
			"License error",
			daisy.GetInstanceURI(testProject, testZone, testInstanceNoLicense),
			"",
			"Can only upgrade GCE instance with projects/windows-cloud/global/licenses/windows-server-2008-r2-dc license attached",
			"",
			"",
			testProject, testZone, testInstanceNoLicense,
			mockMetadataGce,
		},
		{
			"OS disk error",
			daisy.GetInstanceURI(testProject, testZone, testInstanceNoBootDisk),
			"",
			"The instance has no boot disk.",
			"",
			"",
			testProject, testZone, testInstanceNoBootDisk,
			mockMetadataGce,
		},
		{
			"Instance doesn't exist",
			daisy.GetInstanceURI(testProject, testZone, DNE),
			"",
			"Failed to get instance: googleapi: got HTTP response code 404 with body: ",
			"",
			"",
			testProject, testZone, DNE,
			mockMetadataGce,
		},
		{
			"Bad instance flag error",
			"bad/url",
			"Please provide the instance flag either with the name of the instance or in the form of 'projects/<project>/zones/<zone>/instances/<instance>', not bad/url",
			"",
			"",
			"",
			testProject, testZone, "bad/url",
			mockMetadataGce,
		},
		{
			"No instance flag",
			"",
			"Flag -instance must be provided",
			"",
			"",
			"",
			testProject, testZone, "",
			mockMetadataGce,
		},
		{
			"Instance name without project",
			testInstance,
			"project cannot be determined because build is not running on GCE",
			"",
			"",
			testZone2,
			"", testZone2, testInstance,
			mockMetadataGceFail,
		},
		{
			"Instance name with fallback project (on GCE)",
			testInstance,
			"",
			"",
			"",
			testZone2,
			testProject2, testZone2, testInstance,
			mockMetadataGce,
		},
		{
			"Instance name without input zone",
			testInstance,
			"--zone must be provided when --instance is not a URI with zone info.",
			"",
			testProject2,
			"",
			testProject2, testZone2, testInstance,
			mockMetadataGce,
		},
		{
			"Instance name with input project and zone",
			testInstance,
			"",
			"",
			testProject2,
			testZone2,
			testProject2, testZone2, testInstance,
			mockMetadataGce,
		},
		{
			"Override input project and zone",
			daisy.GetInstanceURI(testProject, testZone, testInstance),
			"",
			"",
			testProject2,
			testZone2,
			testProject, testZone, testInstance,
			mockMetadataGce,
		},
	}

	originalMGCE := mgce
	defer func() {
		mgce = originalMGCE
	}()

	for _, tc := range tcs {
		derivedVars := derivedVars{}
		mgce = tc.mgce

		err := validateAndDeriveInstanceURI(tc.instance, &tc.inputProject, tc.inputZone, &derivedVars)
		if tc.expectedURIError != "" {
			assert.EqualErrorf(t, err, tc.expectedURIError, "[test name: %v] Unexpected error from validateAndDeriveInstanceURI.", tc.testName)
			continue
		} else {
			assert.NoErrorf(t, err, "[test name: %v] Unexpected error from validateAndDeriveInstanceURI.", tc.testName)
			if err != nil {
				continue
			}
		}
		if !instanceURLRgx.Match([]byte(derivedVars.instanceURI)) {
			t.Errorf("[%v]: Expect correct derivedVars.instanceURI format error but it's bad format %v.", tc.testName, derivedVars.instanceURI)
			continue
		}

		if tc.expectedProject != derivedVars.instanceProject || tc.expectedZone != derivedVars.instanceZone || tc.expectedInstanceName != derivedVars.instanceName {
			t.Errorf("[%v]: Unexpected breakdown of instance URI. Actual project, zone, instanceName are %v, %v, %v while expect %v, %v, %v.",
				tc.testName, derivedVars.instanceProject, derivedVars.instanceZone, derivedVars.instanceName,
				tc.expectedProject, tc.expectedZone, tc.expectedInstanceName)
		}
		expectedURI := daisy.GetInstanceURI(tc.expectedProject, tc.expectedZone, tc.expectedInstanceName)
		if expectedURI != derivedVars.instanceURI {
			t.Errorf("[%v]: Unexpected instance URI. Actual: %v, while expect: %v.",
				tc.testName, derivedVars.instanceURI, expectedURI)
		}

		err = validateAndDeriveInstance(&derivedVars, testSourceOS)
		if tc.expectedError == "" {
			if err != nil {
				t.Errorf("[%v]: Unexpected error: %v", tc.testName, err)
			} else {
				if derivedVars.instanceName == testInstance {
					assert.Nil(t, derivedVars.originalWindowsStartupScriptURL, "[test name: %v] Unexpected derivedVars.originalWindowsStartupScriptURL.", tc.testName)
				} else if derivedVars.instanceName == testInstanceWithStartupScript ||
					derivedVars.instanceName == testInstanceWithExistingStartupScriptBackup {
					if derivedVars.originalWindowsStartupScriptURL == nil || *derivedVars.originalWindowsStartupScriptURL != testOriginalStartupScript {
						t.Errorf("[%v]: Unexpected originalWindowsStartupScriptURL: %v, expect: %v", tc.testName, derivedVars.originalWindowsStartupScriptURL, testOriginalStartupScript)
					}
				}
			}
		} else {
			assert.EqualErrorf(t, err, tc.expectedError, "[test name: %v]: Unexpected error from validateAndDeriveInstance.", tc.testName)
		}
	}
}

func TestValidateOSDisk(t *testing.T) {
	initTest()

	type testCase struct {
		testName      string
		osDisk        *compute.AttachedDisk
		expectedError string
	}

	tcs := []testCase{
		{
			"Disk exists",
			&compute.AttachedDisk{Source: testDiskURI, DeviceName: testDiskDeviceName,
				AutoDelete: testDiskAutoDelete, Boot: true},
			"",
		},
		{
			"Disk not exist",
			&compute.AttachedDisk{Source: daisy.GetDiskURI(testProject, testZone, DNE),
				DeviceName: testDiskDeviceName, AutoDelete: testDiskAutoDelete, Boot: true},
			"Failed to get OS disk info: googleapi: got HTTP response code 404 with body: ",
		},
		{
			"Disk not boot",
			&compute.AttachedDisk{Source: testDiskURI, DeviceName: testDiskDeviceName,
				AutoDelete: testDiskAutoDelete, Boot: false},
			"The instance has no boot disk.",
		},
	}

	for _, tc := range tcs {
		derivedVars := derivedVars{}
		err := validateAndDeriveOSDisk(tc.osDisk, &derivedVars)
		if tc.expectedError == "" {
			if err != nil {
				t.Errorf("[%v]: Unexpected error: %v", tc.testName, err)
			} else {
				assert.Equalf(t, testDiskURI, derivedVars.osDiskURI, "[%v]: Unexpected derivedVars.osDiskURI", tc.testName)
				assert.Equalf(t, testDiskDeviceName, derivedVars.osDiskDeviceName, "[%v]: Unexpected derivedVars.osDiskDeviceName", tc.testName)
				assert.Equalf(t, testDiskAutoDelete, derivedVars.osDiskAutoDelete, "[%v]: Unexpected derivedVars.osDiskAutoDelete", tc.testName)
				assert.Equalf(t, testDiskType, derivedVars.osDiskType, "[%v]: Unexpected derivedVars.osDiskType", tc.testName)
			}
		} else {
			assert.EqualErrorf(t, err, tc.expectedError, "[test name: %v] Unexpected error.", tc.testName)
		}
	}
}

func TestValidateLicense(t *testing.T) {
	type testCase struct {
		testName      string
		osDisk        *compute.AttachedDisk
		expectedError string
	}

	tcs := []testCase{
		{
			"No license",
			&compute.AttachedDisk{},
			"Can only upgrade GCE instance with projects/windows-cloud/global/licenses/windows-server-2008-r2-dc license attached",
		},
		{
			"No expected license",
			&compute.AttachedDisk{
				Licenses: []string{
					"random-license",
				}},
			"Can only upgrade GCE instance with projects/windows-cloud/global/licenses/windows-server-2008-r2-dc license attached",
		},
		{
			"Expected license",
			&compute.AttachedDisk{
				Licenses: []string{
					expectedCurrentLicense[testSourceOS],
				}},
			"",
		},
		{
			"Expected license with some other license",
			&compute.AttachedDisk{
				Licenses: []string{
					"random-1",
					expectedCurrentLicense[testSourceOS],
					"random-2",
				}},
			"",
		},
		{
			"Upgraded",
			&compute.AttachedDisk{
				Licenses: []string{
					expectedCurrentLicense[testSourceOS],
					licenseToAdd[testSourceOS],
				}},
			"The GCE instance is with projects/windows-cloud/global/licenses/windows-server-2012-r2-dc-in-place-upgrade license attached, which means it either has been upgraded or has started an upgrade in the past.",
		},
	}

	for _, tc := range tcs {
		err := validateLicense(tc.osDisk, testSourceOS)
		if tc.expectedError != "" {
			assert.EqualErrorf(t, err, tc.expectedError, "[test name: %v]", tc.testName)
		} else {
			assert.NoError(t, err, "[test name: %v]", tc.testName)
		}
	}
}
