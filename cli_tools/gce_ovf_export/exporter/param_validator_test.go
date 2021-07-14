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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

var validReleaseTracks = []string{"ga", "beta", "alpha"}

func TestInstanceNameAndMachineImageNameProvidedAtTheSameTime(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.MachineImageName = "machine-image-name"
	assertErrorOnValidate(t, params, createDefaultParamValidator(mockCtrl, false))
}

func TestInstanceExportFlagsInstanceNameNotProvided(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.InstanceName = ""
	assertErrorOnValidate(t, params, createDefaultParamValidator(mockCtrl, false))
}

func TestInstanceExportFlagsOvfGcsPathFlagKeyNotProvided(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.DestinationURI = ""
	assertErrorOnValidate(t, params, createDefaultParamValidator(mockCtrl, false))
}

func TestInstanceExportFlagsOvfGcsPathFlagNotValid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.DestinationURI = "NOT_GCS_PATH"
	assertErrorOnValidate(t, params, createDefaultParamValidator(mockCtrl, false))
}

func TestInstanceExportZoneInvalid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.Zone = "not-a-zone"

	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	zoneError := fmt.Errorf("invalid zone")
	mockZoneValidator.EXPECT().ZoneValid(ovfexportdomain.TestProject, params.Zone).Return(zoneError)
	validator := &ovfExportParamValidatorImpl{
		validReleaseTracks: validReleaseTracks,
		zoneValidator:      mockZoneValidator,
	}

	assert.Equal(t, zoneError, validator.ValidateAndParseParams(params))
}

func TestInstanceExportFlagsAllowsClientIdNotProvided(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	validator := createDefaultParamValidator(mockCtrl, true)
	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.ClientID = ""
	assert.Nil(t, validator.ValidateAndParseParams(params))
}

func TestInstanceExportFlagsInvalidReleaseTrack(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.ReleaseTrack = "not-a-release-track"
	assertErrorOnValidate(t, params, createDefaultParamValidator(mockCtrl, false))
}

func TestInstanceExportFlagsAllValid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	validator := createDefaultParamValidator(mockCtrl, true)
	assert.Nil(t, validator.ValidateAndParseParams(ovfexportdomain.GetAllInstanceExportArgs()))
}

func TestInstanceExportFlagsAllValidBucketOnlyPathTrailingSlash(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	validator := createDefaultParamValidator(mockCtrl, true)

	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.DestinationURI = "gs://bucket_name/"
	assert.Nil(t, validator.ValidateAndParseParams(ovfexportdomain.GetAllInstanceExportArgs()))
}

func TestInstanceExportFlagsAllValidBucketOnlyPathNoTrailingSlash(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	validator := createDefaultParamValidator(mockCtrl, true)

	params := ovfexportdomain.GetAllInstanceExportArgs()
	params.DestinationURI = "gs://bucket_name"
	assert.Nil(t, validator.ValidateAndParseParams(ovfexportdomain.GetAllInstanceExportArgs()))
}

func TestMachineImageExportFlagsAllValid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	validator := createDefaultParamValidator(mockCtrl, true)
	assert.Nil(t, validator.ValidateAndParseParams(ovfexportdomain.GetAllMachineImageExportArgs()))
}

func TestMachineImageExportFlagsMachineImageNameNotProvided(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	validator := createDefaultParamValidator(mockCtrl, false)
	params := ovfexportdomain.GetAllMachineImageExportArgs()
	params.MachineImageName = ""
	assertErrorOnValidate(t, params, validator)
}

func TestMachineImageExportFlagsOvfGcsPathFlagKeyNotProvided(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	validator := createDefaultParamValidator(mockCtrl, false)
	params := ovfexportdomain.GetAllMachineImageExportArgs()
	params.DestinationURI = ""
	assertErrorOnValidate(t, params, validator)
}

func TestMachineImageExportFlagsOvfGcsPathFlagNotValid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	validator := createDefaultParamValidator(mockCtrl, false)
	params := ovfexportdomain.GetAllMachineImageExportArgs()
	params.DestinationURI = "NOT_GCS_PATH"
	assertErrorOnValidate(t, params, validator)
}

func TestMachineImageExportFlagsAllowsClientIdNotProvided(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	validator := createDefaultParamValidator(mockCtrl, true)
	params := ovfexportdomain.GetAllMachineImageExportArgs()
	params.ClientID = ""
	assert.Nil(t, validator.ValidateAndParseParams(params))
}

func TestMachineImageExportFlagsAllValidBucketOnlyPathTrailingSlash(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	validator := createDefaultParamValidator(mockCtrl, true)
	params := ovfexportdomain.GetAllMachineImageExportArgs()
	params.DestinationURI = "gs://bucket_name/"
	assert.Nil(t, validator.ValidateAndParseParams(ovfexportdomain.GetAllMachineImageExportArgs()))
}

func TestMachineImageExportFlagsAllValidBucketOnlyPathNoTrailingSlash(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	validator := createDefaultParamValidator(mockCtrl, true)
	params := ovfexportdomain.GetAllMachineImageExportArgs()
	params.DestinationURI = "gs://bucket_name"
	assert.Nil(t, validator.ValidateAndParseParams(ovfexportdomain.GetAllMachineImageExportArgs()))
}

func assertErrorOnValidate(t *testing.T, params *ovfexportdomain.OVFExportArgs, validator *ovfExportParamValidatorImpl) {
	assert.NotNil(t, validator.ValidateAndParseParams(params))
}

func createDefaultParamValidator(mockCtrl *gomock.Controller, validateZone bool) *ovfExportParamValidatorImpl {
	mockZoneValidator := mocks.NewMockZoneValidatorInterface(mockCtrl)
	if validateZone {
		mockZoneValidator.EXPECT().ZoneValid(ovfexportdomain.TestProject, ovfexportdomain.TestZone).Return(nil)
	}
	validator := &ovfExportParamValidatorImpl{
		validReleaseTracks: validReleaseTracks,
		zoneValidator:      mockZoneValidator,
	}
	return validator
}
