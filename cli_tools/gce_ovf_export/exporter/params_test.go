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
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestValidateAndPopulateParams(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportParams()
	params.Started = time.Date(2020, time.October, 28, 23, 24, 0, 0, time.UTC)
	params.BuildID = "abc"

	err := runValidateAndPopulateParams(params, mockCtrl)
	assert.Nil(t, err)
	assert.Equal(t, "global/networks/aNetwork", params.Network)
	assert.Equal(t, fmt.Sprintf("regions/%v/subnetworks/%v", ovfexportdomain.TestRegion, ovfexportdomain.TestSubnet), params.Subnet)
	assert.Equal(t, "gs://bucket/folder/gce-ovf-export-2020-10-28T23:24:00Z-abc", params.ScratchBucketGcsPath)
	assert.Equal(t, "gs://ovfbucket/ovfpath/", params.DestinationURI)
}

func TestValidateAndPopulateParams_BuildIDPopulated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportParams()
	params.Started = time.Date(2020, time.October, 28, 23, 24, 0, 0, time.UTC)

	err := runValidateAndPopulateParams(params, mockCtrl)
	assert.Nil(t, err)
	assert.NotEmpty(t, params.BuildID)
	assert.True(t, strings.HasPrefix(params.ScratchBucketGcsPath, "gs://bucket/folder/gce-ovf-export-2020-10-28T23:24:00Z-"))
}

func TestValidateAndPopulateParams_DefaultNetworkPopulated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	params := ovfexportdomain.GetAllInstanceExportParams()
	params.Network = ""
	params.Subnet = ""
	err := runValidateAndPopulateParams(params, mockCtrl)
	assert.Nil(t, err)
	assert.Equal(t, "global/networks/default", params.Network)
}

func runValidateAndPopulateParams(params *ovfexportdomain.OVFExportParams, mockCtrl *gomock.Controller) error {
	paramValidator := mocks.NewMockOvfExportParamValidator(mockCtrl)
	paramValidator.EXPECT().ValidateAndParseParams(params).Return(nil)
	paramPopulator := mocks.NewMockPopulator(mockCtrl)
	paramPopulator.EXPECT().PopulateMissingParameters(params.Project, params.ClientID, &params.Zone,
		&params.Region, &params.ScratchBucketGcsPath, params.DestinationURI, nil).Return(nil)
	return params.ValidateAndPopulateParams(paramValidator, paramPopulator)
}

func TestValidateAndPopulateParams_ErrorOnValidate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportParams()
	params.MachineImageName = "also-machine-image-name-which-is-invalid"
	paramValidator := mocks.NewMockOvfExportParamValidator(mockCtrl)
	paramValidator.EXPECT().ValidateAndParseParams(params).Return(fmt.Errorf("validation error"))
	err := params.ValidateAndPopulateParams(paramValidator, nil)
	assert.NotNil(t, err)
}

func TestValidateAndPopulateParams_ErrorOnPopulate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportParams()
	paramValidator := mocks.NewMockOvfExportParamValidator(mockCtrl)
	paramValidator.EXPECT().ValidateAndParseParams(params).Return(nil)

	populatorError := fmt.Errorf("populator error")
	paramPopulator := mocks.NewMockPopulator(mockCtrl)
	paramPopulator.EXPECT().PopulateMissingParameters(params.Project, params.ClientID, &params.Zone,
		&params.Region, &params.ScratchBucketGcsPath, params.DestinationURI, nil).Return(populatorError)

	err := params.ValidateAndPopulateParams(paramValidator, paramPopulator)
	assert.Equal(t, populatorError, err)
}

func TestIsInstanceExport(t *testing.T) {
	assert.True(t, ovfexportdomain.GetAllInstanceExportParams().IsInstanceExport())
	assert.False(t, ovfexportdomain.GetAllMachineImageExportParams().IsInstanceExport())
}

func TestIsMachineImageExport(t *testing.T) {
	assert.False(t, ovfexportdomain.GetAllInstanceExportParams().IsMachineImageExport())
	assert.True(t, ovfexportdomain.GetAllMachineImageExportParams().IsMachineImageExport())
}

func TestDaisyAttrs(t *testing.T) {
	params := ovfexportdomain.GetAllInstanceExportParams()
	assert.Equal(t,
		daisycommon.WorkflowAttributes{
			Project: *params.Project, Zone: params.Zone, GCSPath: params.ScratchBucketGcsPath,
			OAuth: params.Oauth, Timeout: params.Timeout, ComputeEndpoint: params.Ce,
			WorkflowDirectory: params.WorkflowDir, DisableGCSLogs: params.GcsLogsDisabled,
			DisableCloudLogs: params.CloudLogsDisabled, DisableStdoutLogs: params.StdoutLogsDisabled,
			NoExternalIP: params.NoExternalIP,
		},
		params.DaisyAttrs())
}
