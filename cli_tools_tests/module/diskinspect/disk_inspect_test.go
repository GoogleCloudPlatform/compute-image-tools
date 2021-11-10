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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/common/runtime"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

const (
	workflowDir = "../../../daisy_workflows"
)

func createEnvironment() daisyutils.EnvironmentSettings {
	return daisyutils.EnvironmentSettings{
		Zone:              runtime.GetConfig("GOOGLE_CLOUD_ZONE", "compute/zone"),
		Project:           runtime.GetConfig("GOOGLE_CLOUD_PROJECT", "project"),
		WorkflowDirectory: workflowDir,
		Tool: daisyutils.Tool{
			HumanReadableName: "module-tests",
			ResourceLabelName: "module-tests",
		},
		ExecutionID: path.RandString(5),
	}
}

func TestInspectDisk(t *testing.T) {
	t.Parallel()

	client, err := daisycompute.NewClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		caseName string
		imageURI string
		expected *pb.InspectionResults
	}{
		{
			imageURI: "projects/opensuse-cloud/global/images/opensuse-leap-15-3-v20211011",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "opensuse-15",
					Distro:       "opensuse",
					MajorVersion: "15",
					MinorVersion: "3",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_OPENSUSE,
				},
				BiosBootable: true,
				UefiBootable: true,
				OsCount:      1,
			},
		}, {
			imageURI: "projects/suse-sap-cloud/global/images/sles-15-sp3-sap-v20210812",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "sles-sap-15",
					Distro:       "sles-sap",
					MajorVersion: "15",
					MinorVersion: "3",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_SLES_SAP,
				},
				BiosBootable: true,
				UefiBootable: true,
				OsCount:      1,
			},
		},

		// UEFI
		{
			caseName: "UEFI inspection test for GPT UEFI",
			imageURI: "projects/gce-uefi-images/global/images/rhel-7-v20200403",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "rhel-7",
					Distro:       "rhel",
					MajorVersion: "7",
					MinorVersion: "8",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_RHEL,
				},
				UefiBootable: true,
				OsCount:      1,
			},
		}, {
			caseName: "UEFI inspection test for MBR-only",
			imageURI: "projects/debian-cloud/global/images/debian-9-stretch-v20211028",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "debian-9",
					Distro:       "debian",
					MajorVersion: "9",
					MinorVersion: "13",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_DEBIAN,
				},
				OsCount: 1,
			},
		}, {
			caseName: "UEFI inspection test for GPT UEFI - windows",
			imageURI: "projects/gce-uefi-images/global/images/windows-server-2019-dc-core-v20200609",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-2019",
					Distro:       "windows",
					MajorVersion: "2019",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_WINDOWS,
				},
				UefiBootable: true,
				OsCount:      1,
			},
		}, {
			caseName: "UEFI inspection test for GPT UEFI with BIOS boot",
			imageURI: "projects/gce-uefi-images/global/images/ubuntu-1804-bionic-v20200317",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "ubuntu-1804",
					Distro:       "ubuntu",
					MajorVersion: "18",
					MinorVersion: "04",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_UBUNTU,
				},
				BiosBootable: true,
				UefiBootable: true,
				OsCount:      1,
			},
		}, {
			caseName: "UEFI inspection test for GPT UEFI with hybrid MBR",
			imageURI: "projects/compute-image-tools-test/global/images/image-ubuntu-2004-hybrid-mbr",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "ubuntu-2004",
					Distro:       "ubuntu",
					MajorVersion: "20",
					MinorVersion: "04",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_UBUNTU,
				},
				BiosBootable: true,
				UefiBootable: true,
				OsCount:      1,
			},
		}, {
			caseName: "UEFI inspection test for MBR-only UEFI",
			imageURI: "projects/compute-image-tools-test/global/images/image-uefi-mbr-only",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "ubuntu-1604",
					Distro:       "ubuntu",
					MajorVersion: "16",
					MinorVersion: "04",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_UBUNTU,
				},
				UefiBootable: true,
				OsCount:      1,
			},
		},

		// Windows Server
		{
			imageURI: "projects/windows-cloud/global/images/windows-server-2008-r2-dc-v20200114",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-2008r2",
					Distro:       "windows",
					MajorVersion: "2008",
					MinorVersion: "r2",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_WINDOWS,
				},
				OsCount: 1,
			},
		}, {
			imageURI: "projects/compute-image-tools-test/global/images/windows-2012-r2-vmware-import",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-2012r2",
					Distro:       "windows",
					MajorVersion: "2012",
					MinorVersion: "r2",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_WINDOWS,
				},
				OsCount: 1,
			},
		}, {
			imageURI: "projects/compute-image-tools-test/global/images/windows-2016-import",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-2016",
					Distro:       "windows",
					MajorVersion: "2016",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_WINDOWS,
				},
				OsCount: 1,
			},
		}, {
			imageURI: "projects/compute-image-tools-test/global/images/windows-2019",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-2019",
					Distro:       "windows",
					MajorVersion: "2019",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_WINDOWS,
				},
				OsCount: 1,
			},
		},

		// Windows Desktop
		{
			imageURI: "projects/compute-image-tools-test/global/images/windows-7-ent-x86-nodrivers",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-7-x86",
					Distro:       "windows",
					MajorVersion: "7",
					Architecture: pb.Architecture_X86,
					DistroId:     pb.Distro_WINDOWS,
				},
				OsCount: 1,
			},
		}, {
			imageURI: "projects/compute-image-tools-test/global/images/windows-7-import",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-7-x64",
					Distro:       "windows",
					MajorVersion: "7",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_WINDOWS,
				},
				OsCount: 1,
			},
		}, {
			imageURI: "projects/compute-image-tools-test/global/images/windows-8-1-ent-x86-nodrivers",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-8-x86",
					Distro:       "windows",
					MajorVersion: "8",
					MinorVersion: "1",
					Architecture: pb.Architecture_X86,
					DistroId:     pb.Distro_WINDOWS,
				},
				OsCount: 1,
			},
		}, {
			imageURI: "projects/compute-image-tools-test/global/images/windows-8-1-x64",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-8-x64",
					Distro:       "windows",
					MajorVersion: "8",
					MinorVersion: "1",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_WINDOWS,
				},
				OsCount: 1,
			},
		}, {
			imageURI: "projects/compute-image-tools-test/global/images/windows-10-1909-ent-x86-nodrivers",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-10-x86",
					Distro:       "windows",
					MajorVersion: "10",
					Architecture: pb.Architecture_X86,
					DistroId:     pb.Distro_WINDOWS,
				},
				OsCount: 1,
			},
		}, {
			imageURI: "projects/compute-image-tools-test/global/images/windows-10-1709-import",
			expected: &pb.InspectionResults{
				OsRelease: &pb.OsRelease{
					CliFormatted: "windows-10-x64",
					Distro:       "windows",
					MajorVersion: "10",
					Architecture: pb.Architecture_X64,
					DistroId:     pb.Distro_WINDOWS,
				},
				OsCount: 1,
			},
		},
	} {
		// Without this, each parallel test will reference the last tt instance.
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		currentTest := tt
		name := currentTest.caseName
		if name == "" {
			name = currentTest.imageURI
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := createEnvironment()
			inspector, err := disk.NewInspector(env, logging.NewToolLogger(t.Name()))
			if err != nil {
				t.Fatal(err)
			}

			diskURI := createDisk(t, client, env, currentTest.imageURI)

			actual, err := inspector.Inspect(diskURI)
			assert.NoError(t, err)
			actual.ElapsedTimeMs = 0
			if diff := cmp.Diff(currentTest.expected, actual, protocmp.Transform()); diff != "" {
				t.Errorf("unexpected difference:\n%v", diff)
			}
			deleteDisk(t, client, env, diskURI)
		})
	}
}

func TestInspectDisk_NoOSResults_WhenDistroUnrecognized(t *testing.T) {
	t.Parallel()

	image := "projects/compute-image-tools-test/global/images/manjaro"
	expected := &pb.InspectionResults{}

	assertInspectionSucceeds(t, image, createEnvironment(), expected)
}

func TestInspectDisk_NoOSResults_WhenDiskEmpty(t *testing.T) {
	t.Parallel()

	image := "projects/compute-image-tools-test/global/images/blank-10g"
	expected := &pb.InspectionResults{}

	assertInspectionSucceeds(t, image, createEnvironment(), expected)
}

func TestInspectDisk_SupportsNoExternalIP(t *testing.T) {
	t.Parallel()

	client, err := daisycompute.NewClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	env := daisyutils.EnvironmentSettings{
		Project:           "gce-guest-no-external-ip-3afc2",
		Zone:              "us-west1-a",
		WorkflowDirectory: workflowDir,
		NoExternalIP:      true,
		Network:           "projects/gce-guest-no-external-ip-3afc2/global/networks/nat",
		Subnet:            "projects/gce-guest-no-external-ip-3afc2/regions/us-west1/subnetworks/nat",
		Tool: daisyutils.Tool{
			HumanReadableName: "module-tests",
			ResourceLabelName: "module-tests",
		},
		ExecutionID: path.RandString(5),
	}
	inspector, err := disk.NewInspector(env, logging.NewToolLogger(t.Name()))
	if err != nil {
		t.Fatal(err)
	}

	diskURI := createDisk(t, client, env, "projects/debian-cloud/global/images/debian-9-stretch-v20200714")
	defer deleteDisk(t, client, env, diskURI)

	actual, err := inspector.Inspect(diskURI)
	assert.NoError(t, err)
	actual.ElapsedTimeMs = 0
	expected := &pb.InspectionResults{
		OsCount: 1,
		OsRelease: &pb.OsRelease{
			CliFormatted: "debian-9",
			Distro:       "debian",
			MajorVersion: "9",
			MinorVersion: "12",
			Architecture: pb.Architecture_X64,
			DistroId:     pb.Distro_DEBIAN,
		},
	}
	if diff := cmp.Diff(expected, actual, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}
}

func TestInspectionDisk_SupportsNonDefaultNetwork(t *testing.T) {
	t.Parallel()

	project := "compute-image-test-custom-vpc"
	zone := "us-central1-a"
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
			t.Logf("Network=%s, Subnet=%s, project=%s", currentTest.network, currentTest.subnet, project)
			env := daisyutils.EnvironmentSettings{
				Project:           "compute-image-test-custom-vpc",
				Zone:              zone,
				WorkflowDirectory: workflowDir,
				Network:           currentTest.network,
				Subnet:            currentTest.subnet,
				Tool: daisyutils.Tool{
					HumanReadableName: "module-tests",
					ResourceLabelName: "module-tests",
				},
				ExecutionID: path.RandString(5),
			}
			inspector, err := disk.NewInspector(env, logging.NewToolLogger(t.Name()))
			if err != nil {
				t.Fatal(err)
			}

			diskURI := createDisk(t, client, env, "projects/opensuse-cloud/global/images/opensuse-leap-15-2-v20200702")
			defer deleteDisk(t, client, env, diskURI)
			actual, err := inspector.Inspect(diskURI)
			if err != nil {
				t.Fatalf("Inspect failed: %v", err)
			}
			if actual.GetOsRelease().GetDistro() != "opensuse" {
				t.Errorf("expected=opensuse, actual=%s", actual.GetOsRelease().GetDistro())
			}
		})
	}
}

func createDisk(t *testing.T, client daisycompute.Client, env daisyutils.EnvironmentSettings, srcImage string) string {
	name := "d" + uuid.New().String()
	err := client.CreateDisk(env.Project, env.Zone, &compute.Disk{
		Name:        name,
		SourceImage: srcImage,
	})
	if err != nil {
		t.Fatal(err)
	}
	diskURI := fmt.Sprintf("projects/%s/zones/%s/disks/%s", env.Project, env.Zone, name)
	t.Logf("created disk: %s", diskURI)
	return diskURI
}

func deleteDisk(t *testing.T, client daisycompute.Client, env daisyutils.EnvironmentSettings, diskURI string) {
	name := diskURI[strings.LastIndex(diskURI, "/")+1:]
	err := client.DeleteDisk(env.Project, env.Zone, name)
	if err != nil {
		t.Logf("Failed to delete disk: %v", err)
	} else {
		t.Logf("Deleted disk")
	}
}

func assertInspectionSucceeds(t *testing.T, image string, env daisyutils.EnvironmentSettings, expected *pb.InspectionResults) {
	client, inspector := makeClientAndInspector(t, env)

	diskURI := createDisk(t, client, env, image)
	defer deleteDisk(t, client, env, diskURI)
	actual, err := inspector.Inspect(diskURI)
	assert.NoError(t, err)
	actual.ElapsedTimeMs = 0
	if diff := cmp.Diff(expected, actual, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}
}

func makeClientAndInspector(t *testing.T, env daisyutils.EnvironmentSettings) (daisycompute.Client, disk.Inspector) {
	client, err := daisycompute.NewClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	inspector, err := disk.NewInspector(env, logging.NewToolLogger(t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	return client, inspector
}
