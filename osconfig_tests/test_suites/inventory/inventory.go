//  Copyright 2018 Google Inc. All Rights Reserved.
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

package inventory

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path"
	"regexp"
	"sync"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	apiBeta "google.golang.org/api/compute/v0.beta"
	api "google.golang.org/api/compute/v1"
)

const (
	testSuiteName = "InventoryTests"

	// TODO: Should these be configurable via flags?
	testProject = "compute-image-test-pool-001"
	testZone    = "us-central1-c"
)

type inventoryTestSetup struct {
	image       string
	name        string
	packageType []string
	shortName   string
	startup     *api.MetadataItems
}

// TestSuite is a InventoryTests test suite.
func TestSuite(ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite, logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp) {
	defer tswg.Done()

	if testSuiteRegex != nil && !testSuiteRegex.MatchString(testSuiteName) {
		return
	}

	testSuite := junitxml.NewTestSuite(testSuiteName)
	defer testSuite.Finish(testSuites)

	logger.Printf("Running TestSuite %q", testSuite.Name)

	testSetup := []*inventoryTestSetup{
		// Windows images.
		&inventoryTestSetup{
			image:       "projects/windows-cloud/global/images/family/windows-2008-r2",
			packageType: []string{"googet", "wua", "qfe"},
			shortName:   "windows",
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-cmd",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&inventoryTestSetup{
			image:       "projects/windows-cloud/global/images/family/windows-2012-r2",
			packageType: []string{"googet", "wua", "qfe"},
			shortName:   "windows",
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-cmd",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&inventoryTestSetup{
			image:       "projects/windows-cloud/global/images/family/windows-2012-r2-core",
			packageType: []string{"googet", "wua", "qfe"},
			shortName:   "windows",
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-cmd",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&inventoryTestSetup{
			image:       "projects/windows-cloud/global/images/family/windows-2016",
			packageType: []string{"googet", "wua", "qfe"},
			shortName:   "windows",
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-cmd",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&inventoryTestSetup{
			image:       "projects/windows-cloud/global/images/family/windows-2016-core",
			packageType: []string{"googet", "wua", "qfe"},
			shortName:   "windows",
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-cmd",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&inventoryTestSetup{
			image:       "projects/windows-cloud/global/images/family/windows-1709-core",
			packageType: []string{"googet", "wua", "qfe"},
			shortName:   "windows",
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-cmd",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&inventoryTestSetup{
			image:       "projects/windows-cloud/global/images/family/windows-1803-core",
			packageType: []string{"googet", "wua", "qfe"},
			shortName:   "windows",
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-cmd",
				Value: &utils.InstallOSConfigGooGet,
			},
		},

		// Debian images.
		&inventoryTestSetup{
			image:       "projects/debian-cloud/global/images/family/debian-9",
			packageType: []string{"deb"},
			shortName:   "debian",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigDeb,
			},
		},

		// Centos images.
		&inventoryTestSetup{
			image:       "projects/centos-cloud/global/images/family/centos-6",
			packageType: []string{"rpm"},
			shortName:   "centos",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigYumEL6,
			},
		},
		&inventoryTestSetup{
			image:       "projects/centos-cloud/global/images/family/centos-7",
			packageType: []string{"rpm"},
			shortName:   "centos",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigYumEL7,
			},
		},

		// RHEL images.
		&inventoryTestSetup{
			image:       "projects/rhel-cloud/global/images/family/rhel-6",
			packageType: []string{"rpm"},
			shortName:   "rhel",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigYumEL6,
			},
		},
		&inventoryTestSetup{
			image:       "projects/rhel-cloud/global/images/family/rhel-7",
			packageType: []string{"rpm"},
			shortName:   "rhel",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigYumEL7,
			},
		},

		// Ubuntu images
		&inventoryTestSetup{
			image:       "projects/ubuntu-os-cloud/global/images/family/ubuntu-1604-lts",
			packageType: []string{"deb"},
			shortName:   "ubuntu",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigDeb,
			},
		},
		&inventoryTestSetup{
			image:       "projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts",
			packageType: []string{"deb"},
			shortName:   "ubuntu",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigDeb,
			},
		},
	}

	var wg sync.WaitGroup
	tests := make(chan *junitxml.TestCase)
	for _, setup := range testSetup {
		wg.Add(1)
		go inventoryTestCase(ctx, setup, tests, &wg, logger, testCaseRegex)
	}

	go func() {
		wg.Wait()
		close(tests)
	}()

	for ret := range tests {
		testSuite.TestCase = append(testSuite.TestCase, ret)
	}

	logger.Printf("Finished TestSuite %q", testSuite.Name)
}

func runGatherInventoryTest(ctx context.Context, testSetup *inventoryTestSetup, testCase *junitxml.TestCase) (*apiBeta.GuestAttributes, bool) {
	testCase.Logf("Creating compute client")
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		testCase.WriteFailure("Error creating client: %v", err)
		return nil, false
	}

	testCase.Logf("Creating instance with image %q", testSetup.image)
	testSetup.name = fmt.Sprintf("inventory-test-%s-%s", path.Base(testSetup.image), utils.RandString(5))

	i := &api.Instance{
		Name:        testSetup.name,
		MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/n1-standard-2", testProject, testZone),
		NetworkInterfaces: []*api.NetworkInterface{
			&api.NetworkInterface{
				Network: "global/networks/default",
				AccessConfigs: []*api.AccessConfig{
					&api.AccessConfig{
						Type: "ONE_TO_ONE_NAT",
					},
				},
			},
		},
		Metadata: &api.Metadata{
			Items: []*api.MetadataItems{
				testSetup.startup,
				&api.MetadataItems{
					Key:   "enable-guest-attributes",
					Value: func() *string { v := "true"; return &v }(),
				},
			},
		},
		Disks: []*api.AttachedDisk{
			&api.AttachedDisk{
				AutoDelete: true,
				Boot:       true,
				InitializeParams: &api.AttachedDiskInitializeParams{
					SourceImage: testSetup.image,
				},
			},
		},
	}
	inst, err := compute.CreateInstance(client, testProject, testZone, i)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", err)
		return nil, false
	}
	defer inst.Cleanup()

	testCase.Logf("Waiting for agent install to complete")
	if err := inst.WaitForSerialOutput("osconfig install done", 1, 5*time.Second, 7*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return nil, false
	}

	return gatherInventory(client, testCase, inst.Project, inst.Zone, inst.Name)
}

func gatherInventory(client daisyCompute.Client, testCase *junitxml.TestCase, project, zone, name string) (*apiBeta.GuestAttributes, bool) {
	testCase.Logf("Checking inventory data")
	// It can take a long time to start collecting data, especially on Windows.
	var retryTime = 10 * time.Second
	for i := 0; ; i++ {
		time.Sleep(retryTime)

		ga, err := client.GetGuestAttributes(project, zone, name, "guestInventory/", "")
		totalRetryTime := time.Duration(i) * retryTime
		if err != nil && totalRetryTime > 25*time.Minute {
			testCase.WriteFailure("Error getting guest attributes: %v", err)
			return nil, false
		}
		if ga != nil {
			return ga, true
		}
		continue
	}
}

func runHostnameTest(ga *apiBeta.GuestAttributes, testSetup *inventoryTestSetup, testCase *junitxml.TestCase) {
	var hostname string
	for _, item := range ga.QueryValue.Items {
		if item.Key == "Hostname" {
			hostname = item.Value
			break
		}
	}

	if hostname == "" {
		testCase.WriteFailure("Hostname not found in GuestAttributes, QueryPath: %q", ga.QueryPath)
		return
	}

	if hostname != testSetup.name {
		testCase.WriteFailure("Hostname does not match expectation: got: %q, want: %q", hostname, testSetup.name)
	}
}

func runShortNameTest(ga *apiBeta.GuestAttributes, testSetup *inventoryTestSetup, testCase *junitxml.TestCase) {
	var shortName string
	for _, item := range ga.QueryValue.Items {
		if item.Key == "ShortName" {
			shortName = item.Value
			break
		}
	}

	if shortName == "" {
		testCase.WriteFailure("ShortName not found in GuestAttributes, QueryPath: %q", ga.QueryPath)
		return
	}

	if shortName != testSetup.shortName {
		testCase.WriteFailure("ShortName does not match expectation: got: %q, want: %q", shortName, testSetup.shortName)
	}
}

func runPackagesTest(ga *apiBeta.GuestAttributes, testSetup *inventoryTestSetup, testCase *junitxml.TestCase) {
	var packagesEncoded string
	for _, item := range ga.QueryValue.Items {
		if item.Key == "InstalledPackages" {
			packagesEncoded = item.Value
			break
		}
	}

	if packagesEncoded == "" {
		testCase.WriteFailure("InstalledPackages not found in GuestAttributes, QueryPath: %q", ga.QueryPath)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(packagesEncoded)
	if err != nil {
		testCase.WriteFailure(err.Error())
		return
	}

	zr, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		testCase.WriteFailure(err.Error())
		return
	}
	defer zr.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, zr); err != nil {
		testCase.WriteFailure(err.Error())
		return
	}

	var pkgs packages.Packages
	if err := json.Unmarshal(buf.Bytes(), &pkgs); err != nil {
		testCase.WriteFailure(err.Error())
		return
	}

	for _, pt := range testSetup.packageType {
		switch pt {
		case "googet":
			if len(pkgs.GooGet) < 1 {
				testCase.WriteFailure("No packages exported in InstalledPackages for %q", pt)
				return
			}
		case "deb":
			if len(pkgs.Deb) < 1 {
				testCase.WriteFailure("No packages exported in InstalledPackages for %q", pt)
				return
			}
		case "rpm":
			if len(pkgs.Rpm) < 1 {
				testCase.WriteFailure("No packages exported in InstalledPackages for %q", pt)
				return
			}
		case "pip":
			if len(pkgs.Pip) < 1 {
				testCase.WriteFailure("No packages exported in InstalledPackages for %q", pt)
				return
			}
		case "gem":
			if len(pkgs.Gem) < 1 {
				testCase.WriteFailure("No packages exported in InstalledPackages for %q", pt)
				return
			}
		case "wua":
			if len(pkgs.WUA) < 1 {
				testCase.WriteFailure("No packages exported in InstalledPackages for %q", pt)
				return
			}
		case "qfe":
			if len(pkgs.QFE) < 1 {
				testCase.WriteFailure("No packages exported in InstalledPackages for %q", pt)
				return
			}
		}
	}
}

func inventoryTestCase(ctx context.Context, testSetup *inventoryTestSetup, tests chan *junitxml.TestCase, wg *sync.WaitGroup, logger *log.Logger, regex *regexp.Regexp) {
	defer wg.Done()

	gatherInventoryTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[%s] Gather Inventory", testSetup.image))
	hostnameTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[%s] Check Hostname", testSetup.image))
	shortNameTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[%s] Check ShortName", testSetup.image))
	packageTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[%s] Check InstalledPackages", testSetup.image))

	if gatherInventoryTest.FilterTestCase(regex) {
		gatherInventoryTest.Finish(tests)

		hostnameTest.WriteSkipped("Setup skipped")
		hostnameTest.Finish(tests)
		shortNameTest.WriteSkipped("Setup skipped")
		hostnameTest.Finish(tests)
		packageTest.WriteSkipped("Setup skipped")
		packageTest.Finish(tests)
		return
	}

	logger.Printf("Running TestCase '%s.%q'", gatherInventoryTest.Classname, gatherInventoryTest.Name)
	ga, ok := runGatherInventoryTest(ctx, testSetup, gatherInventoryTest)
	gatherInventoryTest.Finish(tests)
	logger.Printf("TestCase '%s.%q' finished", gatherInventoryTest.Classname, gatherInventoryTest.Name)
	if !ok {
		hostnameTest.WriteFailure("Setup Failure")
		hostnameTest.Finish(tests)
		shortNameTest.WriteFailure("Setup Failure")
		shortNameTest.Finish(tests)
		packageTest.WriteFailure("Setup Failure")
		packageTest.Finish(tests)
		return
	}

	for tc, f := range map[*junitxml.TestCase]func(*apiBeta.GuestAttributes, *inventoryTestSetup, *junitxml.TestCase){
		hostnameTest:  runHostnameTest,
		shortNameTest: runShortNameTest,
		packageTest:   runPackagesTest,
	} {
		if tc.FilterTestCase(regex) {
			tc.Finish(tests)
		} else {
			logger.Printf("Running TestCase '%s.%q'", tc.Classname, tc.Name)
			f(ga, testSetup, tc)
			tc.Finish(tests)
			logger.Printf("TestCase '%s.%q' finished in %fs", tc.Classname, tc.Name, tc.Time)
		}
	}

}
