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
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
	"google.golang.org/api/compute/v1"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	mock_disk "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_DefaultPlanner_Plan_SkipInspectionWhenCustomWorkflowExists(t *testing.T) {
	// Using an uninitialized inspector ensures there will be a nil dereference if inspection tries to run.
	var inspector disk.Inspector
	customWorkflow := "workflow/path"
	processPlanner := newProcessPlanner(ImportArguments{CustomWorkflow: customWorkflow}, inspector)
	actualPlan, err := processPlanner.plan(persistentDisk{})
	assert.NoError(t, err)

	expectedPlan := &processingPlan{translationWorkflowPath: customWorkflow}
	assert.Equal(t, expectedPlan, actualPlan)
}

func Test_DefaultPlanner_Plan_InspectionFailures(t *testing.T) {
	pd := persistentDisk{uri: "disk/uri"}
	inspectionError := errors.New("inspection failed")
	for _, tt := range []struct {
		name                 string
		args                 ImportArguments
		expectErrorToContain string
		expectedResults      *processingPlan
	}{
		{
			name: "Succeed when inspection fails but OS is provided.",
			args: ImportArguments{
				OS:          "debian-8",
				Inspect:     true,
				WorkflowDir: "workflowroot",
			},
			expectedResults: &processingPlan{
				requiredLicenses:        []string{"projects/debian-cloud/global/licenses/debian-8-jessie"},
				translationWorkflowPath: "workflowroot/image_import/debian/translate_debian_8.wf.json",
			},
		},
		{
			name: "Succeed when inspection fails but OS and uefi_compatible is provided.",
			args: ImportArguments{
				OS:             "windows-2012r2",
				Inspect:        true,
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
			args: ImportArguments{
				Inspect:     true,
				WorkflowDir: "workflowroot",
			},
			expectErrorToContain: "Please re-import with the operating system specified",
		},
		{
			name: "Fail when inspection fails and specified OS is not supported",
			args: ImportArguments{
				OS:          "kali-rolling",
				Inspect:     true,
				WorkflowDir: "workflowroot",
			},
			expectErrorToContain: "kali-rolling.*is invalid",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockInspector := mock_disk.NewMockInspector(mockCtrl)
			mockInspector.EXPECT().Inspect(pd.uri, true).Return(nil, inspectionError)
			processPlanner := newProcessPlanner(tt.args, mockInspector)
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
		args                 ImportArguments
		inspectionResults    *pb.InspectionResults
		expectErrorToContain string
		expectedResults      *processingPlan
	}{
		{
			name: "Use provided OS, even when inspection passes.",
			args: ImportArguments{
				OS:          "debian-8",
				Inspect:     true,
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
			},
		},
		{
			name: "Support BYOL for inspection results",
			args: ImportArguments{
				Inspect:     true,
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
			},
		},
		{
			name: "Fail when BYOL is specified, but detected OS doesn't support it.",
			args: ImportArguments{
				Inspect:     true,
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
			args: ImportArguments{
				Inspect:     true,
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
			args: ImportArguments{
				Inspect:     true,
				WorkflowDir: "workflowroot",
			},
			inspectionResults: &pb.InspectionResults{
				OsCount: 0,
			},
			expectErrorToContain: "lease re-import with the operating system specified",
		},
		{
			name: "Use provided UEFI argument, even when inspection shows UEFI is not supported.",
			args: ImportArguments{
				OS:             "debian-8",
				Inspect:        true,
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
			args: ImportArguments{
				OS:          "debian-8",
				Inspect:     true,
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
			mockInspector.EXPECT().Inspect(pd.uri, true).Return(tt.inspectionResults, nil)
			processPlanner := newProcessPlanner(tt.args, mockInspector)
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
