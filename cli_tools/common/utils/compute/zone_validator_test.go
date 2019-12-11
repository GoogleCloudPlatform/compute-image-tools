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
//  limitations under the License

package compute

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
)

func TestZoneValid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	zones := []*compute.Zone{{Name: "zone1"}, {Name: "zone2"}}
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListZones("aProject").Return(zones, nil)

	zv := ZoneValidator{ComputeClient: mockComputeClient}
	err := zv.ZoneValid("aProject", "zone2")
	assert.Nil(t, err)
}

func TestZoneInvalid(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	zones := []*compute.Zone{{Name: "zone1"}, {Name: "zone2"}}
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListZones("aProject").Return(zones, nil)

	zv := ZoneValidator{ComputeClient: mockComputeClient}
	err := zv.ZoneValid("aProject", "zone3")
	assert.NotNil(t, err)
}

func TestZoneErrorRetrievingZones(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListZones("aProject").Return(nil, fmt.Errorf("error"))

	zv := ZoneValidator{ComputeClient: mockComputeClient}
	err := zv.ZoneValid("aProject", "zone1")
	assert.NotNil(t, err)
}
