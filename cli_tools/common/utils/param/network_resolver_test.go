//  Copyright 2021 Google Inc. All Rights Reserved.
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

package param

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

func TestNetworkResolver_PassingCases(t *testing.T) {
	type getNetworkComputeParams struct{ project, name string }
	type getSubnetComputeParams struct{ project, region, name string }
	tests := []struct {
		name                            string
		originalNetwork, originalSubnet string
		expectedNetwork, expectedSubnet string
		expectedGetNetworkParams        *getNetworkComputeParams
		expectedGetSubnetParams         *getSubnetComputeParams
	}{
		{
			name:                     "expand names to URIs",
			originalNetwork:          "network-id",
			originalSubnet:           "subnet-id",
			expectedNetwork:          "projects/project-id/global/networks/network-id",
			expectedSubnet:           "projects/project-id/regions/region-id/subnetworks/subnet-id",
			expectedGetNetworkParams: &getNetworkComputeParams{"project-id", "network-id"},
			expectedGetSubnetParams:  &getSubnetComputeParams{"project-id", "region-id", "subnet-id"},
		}, {
			name:                     "use default for network when both empty",
			originalNetwork:          "",
			originalSubnet:           "",
			expectedNetwork:          "projects/project-id/global/networks/default",
			expectedSubnet:           "",
			expectedGetNetworkParams: &getNetworkComputeParams{"project-id", "default"},
			expectedGetSubnetParams:  &getSubnetComputeParams{"project-id", "region-id", "default"},
		}, {
			name:                    "leave network empty when subnet populated",
			originalNetwork:         "",
			originalSubnet:          "subnet-id",
			expectedNetwork:         "",
			expectedSubnet:          "projects/project-id/regions/region-id/subnetworks/subnet-id",
			expectedGetSubnetParams: &getSubnetComputeParams{"project-id", "region-id", "subnet-id"},
		}, {
			name:                     "leave subnet empty when network populated",
			originalNetwork:          "network-id",
			originalSubnet:           "",
			expectedNetwork:          "projects/project-id/global/networks/network-id",
			expectedSubnet:           "",
			expectedGetNetworkParams: &getNetworkComputeParams{"project-id", "network-id"},
		}, {
			name:                     "call API using project and region from original URIs",
			originalNetwork:          "projects/uri-project-id/global/networks/network-id",
			originalSubnet:           "projects/uri-project-id/regions/uri-region-id/subnetworks/subnet-id",
			expectedNetwork:          "projects/uri-project-id/global/networks/network-id",
			expectedSubnet:           "projects/uri-project-id/regions/uri-region-id/subnetworks/subnet-id",
			expectedGetNetworkParams: &getNetworkComputeParams{"uri-project-id", "network-id"},
			expectedGetSubnetParams:  &getSubnetComputeParams{"uri-project-id", "uri-region-id", "subnet-id"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockComputeClient := mocks.NewMockClient(mockCtrl)

			if tt.expectedGetSubnetParams != nil {
				mockComputeClient.EXPECT().GetSubnetwork(tt.expectedGetSubnetParams.project, tt.expectedGetSubnetParams.region, tt.expectedGetSubnetParams.name).Return(&compute.Subnetwork{
					Network: tt.expectedNetwork,
				}, nil)
			}
			if tt.expectedGetNetworkParams != nil {
				mockComputeClient.EXPECT().GetNetwork(tt.expectedGetNetworkParams.project, tt.expectedGetNetworkParams.name).Return(&compute.Network{
					SelfLink: tt.expectedNetwork,
				}, nil)
			}

			n := NewNetworkResolver(mockComputeClient)
			actualNetwork, actualSubnet, err := n.Resolve(
				tt.originalNetwork, tt.originalSubnet, "region-id", "project-id")
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedNetwork, actualNetwork)
			assert.Equal(t, tt.expectedSubnet, actualSubnet)
		})
	}
}

func TestNetworkResolver_FailWhenNetworkDoesntContainSubnet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetSubnetwork("project", "region", "subnet").Return(&compute.Subnetwork{
		Network: "other-network",
	}, nil)
	mockComputeClient.EXPECT().GetNetwork("project", "network").Return(&compute.Network{
		SelfLink: "network",
	}, nil)

	n := NewNetworkResolver(mockComputeClient)
	_, _, err := n.Resolve(
		"network", "subnet", "region", "project")
	assert.EqualError(t, err, "Network \"projects/project/global/networks/network\" "+
		"does not contain subnet \"projects/project/regions/region/subnetworks/subnet\"")
}

func TestNetworkResolver_FailWhenNetworkLookupFails(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetNetwork("project", "network").Return(nil, errors.New("failed to lookup network"))

	n := NewNetworkResolver(mockComputeClient)
	_, _, err := n.Resolve(
		"network", "", "region", "project")
	assert.EqualError(t, err, "Validation of network \"projects/project/global/networks/network\" failed: failed to lookup network")
}

func TestNetworkResolver_FailWhenSubnetLookupFails(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)

	mockComputeClient.EXPECT().GetSubnetwork("project", "region", "subnet").Return(nil, errors.New("failed to lookup subnet"))

	n := NewNetworkResolver(mockComputeClient)
	_, _, err := n.Resolve(
		"", "subnet", "region", "project")
	assert.EqualError(t, err, "Validation of subnetwork \"projects/project/regions/region/subnetworks/subnet\" failed: failed to lookup subnet")
}
