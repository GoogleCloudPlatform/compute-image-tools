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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPopulate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportParams()
	params.Started = time.Date(2020, time.October, 28, 23, 24, 0, 0, time.UTC)
	params.BuildID = "abc"

	err := runPopulateParams(params, mockCtrl)
	assert.Nil(t, err)
	assert.Equal(t, "global/networks/aNetwork", params.Network)
	assert.Equal(t, fmt.Sprintf("regions/%v/subnetworks/%v", ovfexportdomain.TestRegion, ovfexportdomain.TestSubnet), params.Subnet)
	assert.Equal(t, "gs://bucket/folder/gce-ovf-export-2020-10-28T23:24:00Z-abc", params.ScratchBucketGcsPath)
	assert.Equal(t, "gs://ovfbucket/ovfpath/", params.DestinationURI)
}

func TestPopulate_BuildIDPopulated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportParams()
	params.Started = time.Date(2020, time.October, 28, 23, 24, 0, 0, time.UTC)

	err := runPopulateParams(params, mockCtrl)
	assert.Nil(t, err)
	assert.NotEmpty(t, params.BuildID)
	assert.True(t, strings.HasPrefix(params.ScratchBucketGcsPath, "gs://bucket/folder/gce-ovf-export-2020-10-28T23:24:00Z-"))
}

func TestPopulate_DefaultNetworkPopulated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	params := ovfexportdomain.GetAllInstanceExportParams()
	params.Network = ""
	params.Subnet = ""
	err := runPopulateParams(params, mockCtrl)
	assert.Nil(t, err)
	assert.Equal(t, "global/networks/default", params.Network)
}

func TestPopulate_ErrorOnSuperPopulatorError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	params := ovfexportdomain.GetAllInstanceExportParams()
	superPopulatorErr := fmt.Errorf("super populator error")
	paramPopulator := mocks.NewMockPopulator(mockCtrl)
	paramPopulator.EXPECT().PopulateMissingParameters(params.Project, params.ClientID, &params.Zone,
		&params.Region, &params.ScratchBucketGcsPath, params.DestinationURI, nil).Return(superPopulatorErr)
	ovfExporParamPopulator := ovfExportParamPopulatorImpl{Populator: paramPopulator}
	err := ovfExporParamPopulator.Populate(params)
	assert.Equal(t, superPopulatorErr, err)
}

func runPopulateParams(params *ovfexportdomain.OVFExportParams, mockCtrl *gomock.Controller) error {
	paramPopulator := mocks.NewMockPopulator(mockCtrl)
	paramPopulator.EXPECT().PopulateMissingParameters(params.Project, params.ClientID, &params.Zone,
		&params.Region, &params.ScratchBucketGcsPath, params.DestinationURI, nil).Return(nil)
	ovfExporParamPopulator := ovfExportParamPopulatorImpl{Populator: paramPopulator}
	return ovfExporParamPopulator.Populate(params)
}
