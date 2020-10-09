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

package diskinspect

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/common/runtime"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

const (
	defaultNetwork = ""
	defaultSubnet  = ""
	workflowDir    = "../../../daisy_workflows"
)

var (
	zone = runtime.GetConfig("GOOGLE_CLOUD_ZONE", "compute/zone")
)

func TestInspectDisk(t *testing.T) {
	t.Parallel()

	project := runtime.GetConfig("GOOGLE_CLOUD_PROJECT", "project")

	client, err := daisycompute.NewClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		imageURI string
		expected disk.InspectionResult
	}{
		{
			"projects/opensuse-cloud/global/images/opensuse-leap-15-2-v20200702",
			disk.InspectionResult{
				Architecture: "x64",
				Distro:       "opensuse",
				Major:        "15",
				Minor:        "2",
			},
		}, {
			"projects/suse-sap-cloud/global/images/sles-15-sp1-sap-v20200803",
			disk.InspectionResult{
				Architecture: "x64",
				Distro:       "sles-sap",
				Major:        "15",
				Minor:        "1",
			},
		}, {
			"projects/compute-image-tools-test/global/images/windows-7-ent-x86-nodrivers",
			disk.InspectionResult{
				Architecture: "x86",
				Distro:       "windows",
				Major:        "6",
				Minor:        "1",
			},
		},
	} {
		// Without this, each parallel test will reference the last tt instance.
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		currentTest := tt
		t.Run(currentTest.imageURI, func(t *testing.T) {
			t.Parallel()
			inspector, err := disk.NewInspector(daisycommon.WorkflowAttributes{
				Project:           project,
				Zone:              zone,
				WorkflowDirectory: workflowDir,
			}, defaultNetwork, defaultSubnet)
			if err != nil {
				t.Fatal(err)
			}

			diskURI := createDisk(t, client, project, currentTest.imageURI)

			actual, err := inspector.Inspect(diskURI, true)
			assert.NoError(t, err)
			// Manual formatting for two reasons:
			//  1. go-junit-report doesn't have good support for testify/assert:
			//     https://github.com/jstemmer/go-junit-report/issues/47
			//  2. Align the equals to simplify comparing the structs. For example:
			//
			//     inspection_test.go:72:
			//        expected = ...
			//          actual = ...
			if currentTest.expected != actual {
				t.Errorf("\nexpected = %#v"+
					"\n  actual = %#v", currentTest.expected, actual)
			}
			deleteDisk(t, client, project, diskURI)
		})
	}
}

func TestInspectionWorksWithNonDefaultNetwork(t *testing.T) {
	t.Parallel()

	project := "compute-image-test-custom-vpc"
	client, err := daisycompute.NewClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	network := "projects/compute-image-test-custom-vpc/global/networks/unrestricted-egress"
	region, err := paramhelper.GetRegion(zone)
	if err != nil {
		t.Fatal(err)
	}
	subnet := fmt.Sprintf("projects/compute-image-test-custom-vpc/regions/%s/subnetworks/unrestricted-egress", region)

	for _, tt := range []struct {
		caseName string
		network  string
		subnet   string
	}{
		{"network and subnet", network, subnet},
		{"network only", network, ""},
		{"subnet only", "", subnet},
	} {
		currentTest := tt
		t.Run(currentTest.caseName, func(t *testing.T) {
			t.Parallel()
			t.Logf("Network=%s, Subnet=%s, project=%s", network, subnet, project)
			inspector, err := disk.NewInspector(daisycommon.WorkflowAttributes{
				Project:           "compute-image-test-custom-vpc",
				Zone:              zone,
				WorkflowDirectory: workflowDir,
			}, currentTest.network, currentTest.subnet)
			if err != nil {
				t.Fatal(err)
			}

			diskURI := createDisk(t, client, project, "projects/opensuse-cloud/global/images/opensuse-leap-15-2-v20200702")
			defer deleteDisk(t, client, project, diskURI)
			actual, err := inspector.Inspect(diskURI, true)
			if err != nil {
				t.Fatalf("Inspect failed: %v", err)
			}
			if actual.Distro != "opensuse" {
				t.Errorf("expected=opensuse, actual=%s", actual.Distro)
			}
		})
	}
}

func createDisk(t *testing.T, client daisycompute.Client, project, srcImage string) string {
	name := "d" + uuid.New().String()
	err := client.CreateDisk(project, zone, &compute.Disk{
		Name:        name,
		SourceImage: srcImage,
	})
	if err != nil {
		t.Fatal(err)
	}
	diskURI := fmt.Sprintf("projects/%s/zones/%s/disks/%s", project, zone, name)
	t.Logf("created disk: %s", diskURI)
	return diskURI
}

func deleteDisk(t *testing.T, client daisycompute.Client, project, diskURI string) {
	name := diskURI[strings.LastIndex(diskURI, "/")+1:]
	err := client.DeleteDisk(project, zone, name)
	if err != nil {
		t.Logf("Failed to delete disk: %v", err)
	} else {
		t.Logf("Deleted disk")
	}
}
