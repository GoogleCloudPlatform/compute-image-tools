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
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

var (
	project = getConfig("GOOGLE_CLOUD_PROJECT", "project")
	zone    = getConfig("GOOGLE_CLOUD_ZONE", "compute/zone")

	wfAttrs = daisycommon.WorkflowAttributes{
		Project:           project,
		Zone:              zone,
		WorkflowDirectory: "../../daisy_workflows",
	}
)

func TestBootInspect(t *testing.T) {

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
		tt := tt
		t.Run(tt.imageURI, func(t *testing.T) {
			t.Parallel()
			inspector, err := disk.NewInspector(wfAttrs)
			if err != nil {
				t.Fatal(err)
			}

			diskURI := createDisk(t, client, tt.imageURI)

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
			//
			// Also, since this is consumed by go-junit-report,
			if tt.expected != actual {
				t.Errorf("\nexpected = %#v"+
					"\n  actual = %#v", tt.expected, actual)
			}
			deleteDisk(t, client, diskURI)
		})
	}
}

func createDisk(t *testing.T, client daisycompute.Client, srcImage string) string {
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

func deleteDisk(t *testing.T, client daisycompute.Client, diskURI string) {
	name := diskURI[strings.LastIndex(diskURI, "/")+1:]
	err := client.DeleteDisk(project, zone, name)
	if err != nil {
		t.Logf("Failed to delete disk: %v", err)
	} else {
		t.Logf("Deleted disk")
	}
}

func getConfig(envKey, gcloudConfig string) string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}

	out, err := exec.Command("gcloud", "config", "get-value", gcloudConfig).Output()
	if err != nil {
		log.Panicf("Environment variable $%s is required", envKey)
	}
	return strings.TrimSpace(string(out))
}
