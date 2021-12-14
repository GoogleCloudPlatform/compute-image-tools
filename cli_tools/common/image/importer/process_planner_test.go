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

package importer

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	mock_disk "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/distro"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

func Test_DefaultPlanner_Plan_SkipInspectionWhenCustomWorkflowExists(t *testing.T) {
	// Using an uninitialized inspector ensures there will be a nil dereference if inspection tries to run.
	var inspector disk.Inspector
	customWorkflow := "workflow/path"
	processPlanner := newProcessPlanner(ImageImportRequest{CustomWorkflow: customWorkflow}, inspector, logging.NewToolLogger("test"))
	actualPlan, err := processPlanner.plan(persistentDisk{})
	assert.NoError(t, err)

	expectedPlan := &processingPlan{translationWorkflowPath: customWorkflow}
	assert.Equal(t, expectedPlan, actualPlan)
}

func Test_DefaultPlanner_Plan_LogWarningWithoutErrorWhenUefiIsSpecifiedButNotDetected(t *testing.T) {
	pd := persistentDisk{uri: "disk/uri"}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockInspector := mock_disk.NewMockInspector(mockCtrl)
	inspectionResults := &pb.InspectionResults{
		OsRelease: &pb.OsRelease{
			Distro:       "ubuntu",
			MajorVersion: "16",
			MinorVersion: "4",
		},
		UefiBootable: false,
	}

	mockInspector.EXPECT().Inspect(pd.uri).Return(inspectionResults, nil)

	mockLogger := mocks.NewMockLogger(mockCtrl)
	// Preserving the order of the calls to EXPECT() is important otherwise AnyTimes() will catch
	// all the calls to the mockLogger and the test will fail.
	mockLogger.EXPECT().User("UEFI booting was specified, but we could not detect a UEFI bootloader. " +
		"Specifying an incorrect boot type can increase load times, or lead to boot failures.")
	mockLogger.EXPECT().User(gomock.Any()).AnyTimes()

	processPlanner := newProcessPlanner(ImageImportRequest{
		UefiCompatible: true,
		OS:             "ubuntu-1604",
	}, mockInspector, mockLogger)
	_, actualError := processPlanner.plan(pd)
	assert.NoError(t, actualError)
}

func Test_DefaultPlanner_Plan_InspectionFailures(t *testing.T) {
	pd := persistentDisk{uri: "disk/uri"}
	inspectionError := errors.New("inspection failed")
	for _, tt := range []struct {
		name                 string
		request              ImageImportRequest
		expectErrorToContain string
		expectedResults      *processingPlan
	}{
		{
			name: "Succeed when inspection fails but OS is provided.",
			request: ImageImportRequest{
				OS:          "debian-8",
				WorkflowDir: "workflowroot",
			},
			expectedResults: &processingPlan{
				requiredLicenses:        []string{"projects/debian-cloud/global/licenses/debian-8-jessie"},
				translationWorkflowPath: "workflowroot/image_import/debian/translate_debian_8.wf.json",
			},
		},
		{
			name: "Succeed when inspection fails but OS and uefi_compatible is provided.",
			request: ImageImportRequest{
				OS:             "windows-2012r2",
				UefiCompatible: true,
				WorkflowDir:    "workflowroot",
			},
			expectedResults: &processingPlan{
				requiredLicenses:        []string{"projects/windows-cloud/global/licenses/windows-server-2012-r2-dc"},
				translationWorkflowPath: "workflowroot/image_import/windows/translate_windows_2012_r2.wf.json",
				requiredFeatures:        []*compute.GuestOsFeature{{Type: "WINDOWS"}, {Type: "UEFI_COMPATIBLE"}},
			},
		},
		{
			name: "Fail when inspection fails and OS is not provided.",
			request: ImageImportRequest{
				WorkflowDir: "workflowroot",
			},
			expectErrorToContain: "Please re-import with the operating system specified",
		},
		{
			name: "Fail when inspection fails and specified OS is not supported",
			request: ImageImportRequest{
				OS:          "kali-rolling",
				WorkflowDir: "workflowroot",
			},
			expectErrorToContain: "kali-rolling.*is invalid",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockInspector := mock_disk.NewMockInspector(mockCtrl)
			mockInspector.EXPECT().Inspect(pd.uri).Return(nil, inspectionError)
			processPlanner := newProcessPlanner(tt.request, mockInspector, logging.NewToolLogger("test"))
			actualResults, actualError := processPlanner.plan(pd)
			if tt.expectErrorToContain == "" {
				assert.NoError(t, actualError)
			} else {
				assert.Error(t, actualError)
				assert.Regexp(t, tt.expectErrorToContain, actualError.Error())
			}
			assert.Equal(t, tt.expectedResults, actualResults)
		})
	}
}

func Test_DefaultPlanner_Plan_InspectionSucceeds(t *testing.T) {
	pd := persistentDisk{uri: "disk/uri"}
	for _, tt := range []struct {
		name                 string
		request              ImageImportRequest
		inspectionResults    *pb.InspectionResults
		expectErrorToContain string
		expectedResults      *processingPlan
	}{
		{
			name: "Use provided OS, even when inspection passes.",
			request: ImageImportRequest{
				OS:          "debian-8",
				WorkflowDir: "workflowroot",
			},
			inspectionResults: &pb.InspectionResults{
				OsCount: 1,
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-10",
				},
			},
			expectedResults: &processingPlan{
				requiredLicenses:        []string{"projects/debian-cloud/global/licenses/debian-8-jessie"},
				translationWorkflowPath: "workflowroot/image_import/debian/translate_debian_8.wf.json",
				detectedOs:              distro.FromGcloudOSArgumentMustParse("windows-10"),
			},
		},
		{
			name: "Support BYOL for inspection results",
			request: ImageImportRequest{
				BYOL:        true,
				WorkflowDir: "workflowroot",
			},
			inspectionResults: &pb.InspectionResults{
				OsCount: 1,
				OsRelease: &pb.OsRelease{
					CliFormatted: "rhel-8",
				},
			},
			expectedResults: &processingPlan{
				requiredLicenses:        []string{"projects/rhel-cloud/global/licenses/rhel-8-byos"},
				translationWorkflowPath: "workflowroot/image_import/enterprise_linux/translate_rhel_8_byol.wf.json",
				detectedOs:              distro.FromGcloudOSArgumentMustParse("rhel-8"),
			},
		},
		{
			name: "Fail when BYOL is specified, but detected OS doesn't support it.",
			request: ImageImportRequest{
				BYOL:        true,
				WorkflowDir: "workflowroot",
			},
			inspectionResults: &pb.InspectionResults{
				OsCount: 1,
				OsRelease: &pb.OsRelease{
					CliFormatted: "debian-8",
				},
			},
			expectErrorToContain: "debian-8-byol.*is invalid",
		},
		{
			name: "Fail when inspected OS is not supported",
			request: ImageImportRequest{
				WorkflowDir: "workflowroot",
			},
			inspectionResults: &pb.InspectionResults{
				OsCount: 1,
				OsRelease: &pb.OsRelease{
					CliFormatted: "kali-rolling",
				},
			},
			expectErrorToContain: "kali-rolling.*is invalid",
		},
		{
			name: "Fail when inspection succeeds, but doesn't find an OS, and OS is not provided.",
			request: ImageImportRequest{
				WorkflowDir: "workflowroot",
			},
			inspectionResults: &pb.InspectionResults{
				OsCount: 0,
			},
			expectErrorToContain: "lease re-import with the operating system specified",
		},
		{
			name: "Use provided UEFI argument, even when inspection shows UEFI is not supported.",
			request: ImageImportRequest{
				OS:             "debian-8",
				UefiCompatible: true,
				WorkflowDir:    "workflowroot",
			},
			inspectionResults: &pb.InspectionResults{
				UefiBootable: false,
			},
			expectedResults: &processingPlan{
				requiredLicenses:        []string{"projects/debian-cloud/global/licenses/debian-8-jessie"},
				translationWorkflowPath: "workflowroot/image_import/debian/translate_debian_8.wf.json",
				requiredFeatures:        []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
			},
		},
		{
			name: "Don't use UEFI when disk is GPT and can boot with BIOS or UEFI.",
			request: ImageImportRequest{
				OS:          "debian-8",
				WorkflowDir: "workflowroot",
			},
			inspectionResults: &pb.InspectionResults{
				UefiBootable: true,
				BiosBootable: true,
			},
			expectedResults: &processingPlan{
				requiredLicenses:        []string{"projects/debian-cloud/global/licenses/debian-8-jessie"},
				translationWorkflowPath: "workflowroot/image_import/debian/translate_debian_8.wf.json",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockInspector := mock_disk.NewMockInspector(mockCtrl)
			mockInspector.EXPECT().Inspect(pd.uri).Return(tt.inspectionResults, nil)
			processPlanner := newProcessPlanner(tt.request, mockInspector, logging.NewToolLogger("test"))
			actualResults, actualError := processPlanner.plan(pd)
			if tt.expectErrorToContain == "" {
				assert.NoError(t, actualError)
			} else {
				assert.Error(t, actualError)
				assert.Regexp(t, tt.expectErrorToContain, actualError.Error())
			}
			assert.Equal(t, tt.expectedResults, actualResults)
		})
	}
}
