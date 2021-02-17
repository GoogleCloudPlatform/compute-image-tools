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

package import_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/cli"
)

func Test_Import_UsesNetworkAndSubnetFromArgs(t *testing.T) {
	t.Parallel()
	// This project doesn't have a 'default' network. Import will fail if a worker
	// isn't configured to use the custom network and subnet.
	project := "compute-image-test-custom-vpc"
	network := "projects/compute-image-test-custom-vpc/global/networks/unrestricted-egress"
	subnet := "regions/us-central1/subnetworks/unrestricted-egress"
	imageName := "i" + uuid.New().String()
	err := cli.Main([]string{
		"-image_name", imageName,
		"-client_id", "test",
		"-source_image", "projects/compute-image-tools-test/global/images/debian-9-translate",
		"-project", project,
		"-zone", "us-central1-a",
		"-network", network,
		"-subnet", subnet,
	}, logging.NewToolLogger("[test]"), "../../../daisy_workflows")

	assert.NoError(t, err)
}
