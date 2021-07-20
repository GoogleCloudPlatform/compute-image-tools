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
	"strings"

	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

// NetworkResolver standardizes and validates network and subnet fields. It follows the
// rules from the `networkInterfaces[].network` section of instances.insert:
//    https://cloud.google.com/compute/docs/reference/rest/v1/instances/insert
type NetworkResolver interface {
	// Resolve returns the URI representation of network and subnet
	// within a given region.
	//
	// There are two goals:
	//
	//    a. Explicitly use the 'default' network only when
	//       network is omitted and subnet is empty.
	//    b. Convert bare identifiers to URIs.
	Resolve(originalNetwork, originalSubnet, region, project string) (network, subnet string, err error)
}

// NewNetworkResolver returns a NetworkResolver implementation that uses the Compute API.
func NewNetworkResolver(client daisyCompute.Client) NetworkResolver {
	return &computeNetworkResolver{client}
}

// computeNetworkResolver uses the Compute API to implement NetworkResolver.
type computeNetworkResolver struct {
	client daisyCompute.Client
}

func (r *computeNetworkResolver) Resolve(
	originalNetwork, originalSubnet, region, project string) (network, subnet string, err error) {

	// 1. Segment the user's input into component fields such as network name, subnet name, project, and region.
	// If the URI in originalNetwork or originalSubnet didn't specify project or region, then backfill
	// those fields using region and project.
	networkFields, subnetFields, err := parseNetworkAndSubnet(originalNetwork, originalSubnet, region, project)
	if err != nil {
		return "", "", err
	}

	if networkFields.String() == "" && subnetFields.String() == "" {
		return "", "", nil
	}

	// 2. Query the Compute API to check whether the network and subnet exist.
	var subnetResponse *compute.Subnetwork
	var networkResponse *compute.Network
	if subnetFields.String() != "" {
		subnetResponse, err = r.client.GetSubnetwork(subnetFields.Project, subnetFields.Region, subnetFields.Name)
		if err != nil {
			return "", "", daisy.Errf("Validation of subnetwork %q failed: %s", subnetFields, err)
		}
	}
	if networkFields.String() != "" {
		networkResponse, err = r.client.GetNetwork(networkFields.Project, networkFields.Name)
		if err != nil {
			return "", "", daisy.Errf("Validation of network %q failed: %s", networkFields, err)
		}
	}

	// 3. Check whether the subnet's network matches the user's specified network.
	if subnetResponse != nil && networkResponse != nil && subnetResponse.Network != networkResponse.SelfLink {
		return "", "", daisy.Errf("Network %q does not contain subnet %q", networkFields, subnetFields)
	}
	return networkFields.String(), subnetFields.String(), err
}

// parseNetworkAndSubnet parses the user's values into structs and backfills
// missing fields based on other values provided by the user.
func parseNetworkAndSubnet(originalNetwork, originalSubnet, region, project string) (
	*paramhelper.NetworkResource, *paramhelper.SubnetResource, error) {

	networkFields, err := paramhelper.SplitNetworkResource(strings.TrimSpace(originalNetwork))
	if err != nil {
		return nil, nil, err
	}
	subnetFields, err := paramhelper.SplitSubnetResource(strings.TrimSpace(originalSubnet))
	if err != nil {
		return nil, nil, err
	}
	if networkFields.String() == "" && subnetFields.String() == "" {
		return &paramhelper.NetworkResource{
				Name:    "default",
				Project: project,
			}, &paramhelper.SubnetResource{
				Name:    "default",
				Project: project,
				Region:  region,
			}, nil
	}
	if networkFields.String() != "" {
		if networkFields.Project == "" {
			networkFields.Project = project
		}
	}
	if subnetFields.String() != "" {
		if subnetFields.Project == "" {
			subnetFields.Project = project
		}
		if subnetFields.Region == "" {
			subnetFields.Region = region
		}
	}
	return networkFields, subnetFields, nil
}
