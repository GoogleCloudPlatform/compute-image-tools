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

package upgrader

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

// DNE represents do-not-exist resource.
const DNE = "dne"

var (
	testWf                                      = "test-wf"
	testProject                                 = "test-project"
	testProject2                                = "test-project2"
	testZone                                    = "test-zone"
	testZone2                                   = "test-zone2"
	testRegion                                  = "test-zo"
	testDisk                                    = "test-disk"
	testDiskDeviceName                          = "test-disk-device-name"
	testDiskAutoDelete                          = true
	testDiskType                                = "pd-ssd"
	testDiskTypeURI                             = fmt.Sprintf("projects/%v/zones/%v/diskTypes/%v", testProject, testZone, testDiskType)
	testDiskURI                                 = daisy.GetDiskURI(testProject, testZone, testDisk)
	testForwardingRule                          = "test-forwarding-rule"
	testFirewallRule                            = "test-firewall-rule"
	testImage                                   = "test-image"
	testMachineImage                            = "test-machine-image"
	testSnapshot                                = "test-snapshot"
	testInstance                                = "test-instance"
	testInstanceNoDisk                          = "test-instance-no-disk"
	testInstanceNoBootDisk                      = "test-instance-no-boot-disk"
	testInstanceNoLicense                       = "test-instance-no-license"
	testInstanceWithStartupScript               = "test-instance-with-startup-script"
	testInstanceWithExistingStartupScriptBackup = "test-instance-with-existing-startup-script-backup"
	testMachineType                             = "test-machine-type"
	testLicense                                 = "test-license"
	testNetwork                                 = "test-network"
	testSubnetwork                              = "test-subnetwork"
	testTargetInstance                          = "test-target-instance"
	testFamily                                  = "test-family"
	testGCSPath                                 = "gs://test-bucket"
	testGCSObjs                                 []string
	testGCSObjsMx                               = sync.Mutex{}
	testSourceOS                                = versionWindows2008r2
	testTargetOS                                = versionWindows2012r2
	testOriginalStartupScript                   = "original"
)

func initTest() *upgrader {
	computeClient = newTestGCEClient()
	return &upgrader{
		InputParams: &InputParams{
			ClientID:               "test",
			Instance:               daisy.GetInstanceURI(testProject, testZone, testInstance),
			SkipMachineImageBackup: false,
			AutoRollback:           false,
			SourceOS:               "windows-2008r2",
			TargetOS:               "windows-2012r2",
			ProjectPtr:             new(string),
			Timeout:                "",
			ScratchBucketGcsPath:   "",
			Oauth:                  "",
			Ce:                     "",
			GcsLogsDisabled:        false,
			CloudLogsDisabled:      false,
			StdoutLogsDisabled:     false,
			CurrentExecutablePath:  ".",
		},

		ctx: context.Background(),
	}
}

func newTestGCEClient() *daisyCompute.TestClient {
	_, c, _ := daisyCompute.NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.String(), "serialPort?alt=json&port=1") {
			fmt.Fprintln(w, `{"Contents":"failsuccess","Start":"0"}`)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), "serialPort?alt=json&port=2") {
			fmt.Fprintln(w, `{"Contents":"successfail","Start":"0"}`)
		} else {
			fmt.Fprintln(w, `{"Status":"DONE","SelfLink":"link"}`)
		}
	}))

	c.GetInstanceFn = func(project, zone, name string) (*compute.Instance, error) {
		if project == DNE || zone == DNE || name == DNE {
			return nil, &googleapi.Error{Code: http.StatusNotFound}
		}
		if name == testInstanceNoDisk {
			return &compute.Instance{}, nil
		}
		if name == testInstanceNoBootDisk {
			return &compute.Instance{
				Name: name,
				Disks: []*compute.AttachedDisk{{DeviceName: testDisk, Source: testDiskURI, Boot: false,
					Licenses: []string{
						expectedCurrentLicense[testSourceOS],
					}}}}, nil
		}
		if name == testInstanceNoLicense {
			return &compute.Instance{
				Name:  name,
				Disks: []*compute.AttachedDisk{{DeviceName: testDisk, Source: testDiskURI, Boot: true}}}, nil
		}
		if name == testInstanceWithStartupScript {
			return &compute.Instance{
				Name: name,
				Disks: []*compute.AttachedDisk{{
					DeviceName: testDisk,
					Source:     testDiskURI,
					Boot:       true,
					Licenses: []string{
						expectedCurrentLicense[testSourceOS],
					},
				}},
				Metadata: &compute.Metadata{
					Items: []*compute.MetadataItems{
						{
							Key:   metadataKeyWindowsStartupScriptURL,
							Value: &testOriginalStartupScript,
						},
					},
				},
			}, nil
		}
		if name == testInstanceWithExistingStartupScriptBackup {
			return &compute.Instance{
				Name: name,
				Disks: []*compute.AttachedDisk{{
					DeviceName: testDisk,
					Source:     testDiskURI,
					Boot:       true,
					Licenses: []string{
						expectedCurrentLicense[testSourceOS],
					},
				}},
				Metadata: &compute.Metadata{
					Items: []*compute.MetadataItems{
						{
							Key:   metadataKeyWindowsStartupScriptURL,
							Value: &testOriginalStartupScript,
						},
						{
							Key:   metadataKeyWindowsStartupScriptURLBackup,
							Value: &testOriginalStartupScript,
						},
					},
				},
			}, nil
		}
		return &compute.Instance{
			Name: name,
			Disks: []*compute.AttachedDisk{{
				DeviceName: testDisk,
				Source:     testDiskURI,
				Boot:       true,
				Licenses: []string{
					expectedCurrentLicense[testSourceOS],
				},
			}}}, nil
	}

	c.GetDiskFn = func(project, zone, name string) (disk *compute.Disk, e error) {
		if project == DNE || zone == DNE || name == DNE {
			return nil, &googleapi.Error{Code: http.StatusNotFound}
		}
		return &compute.Disk{
			Type: testDiskTypeURI,
		}, nil
	}

	return c
}
