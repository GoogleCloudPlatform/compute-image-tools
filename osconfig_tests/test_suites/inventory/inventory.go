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
	apiBeta "google.golang.org/api/compute/v0.beta"
	api "google.golang.org/api/compute/v1"
)

const testSuiteName = "InventoryTests"

// TODO: Should these be configurable via flags?
const testProject = "compute-image-test-pool-001"
const testZone = "us-central1-c"

// TODO: Move to the new combined osconfig package, also make this easily available to other tests.
var installInventoryDeb = `echo 'deb http://packages.cloud.google.com/apt google-compute-engine-inventory-unstable main' >> /etc/apt/sources.list
apt-get update
apt-get install -y google-compute-engine-inventory
echo 'inventory install done'`
var installInventoryGooGet = `c:\programdata\googet\googet.exe -noconfirm install -sources https://packages.cloud.google.com/yuck/repos/google-compute-engine-staging google-compute-engine-inventory
echo 'inventory install done'`

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

	if testSuiteRegex != nil && testSuiteRegex.MatchString(testSuiteName) {
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
				Value: &installInventoryGooGet,
			},
		},
		&inventoryTestSetup{
			image:       "projects/windows-cloud/global/images/family/windows-2012-r2",
			packageType: []string{"googet", "wua", "qfe"},
			shortName:   "windows",
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-cmd",
				Value: &installInventoryGooGet,
			},
		},
		&inventoryTestSetup{
			image:       "projects/windows-cloud/global/images/family/windows-2016",
			packageType: []string{"googet", "wua", "qfe"},
			shortName:   "windows",
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-cmd",
				Value: &installInventoryGooGet,
			},
		},

		// Debian images.
		&inventoryTestSetup{
			image:       "projects/debian-cloud/global/images/family/debian-9",
			packageType: []string{"deb"},
			shortName:   "debian",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &installInventoryDeb,
			},
		},

		// Ubuntu images
		&inventoryTestSetup{
			image:       "projects/ubuntu-os-cloud/global/images/family/ubuntu-1404-lts",
			packageType: []string{"deb", "pip", "gem"},
			shortName:   "ubuntu",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &installInventoryDeb,
			},
		},
		&inventoryTestSetup{
			image:       "projects/ubuntu-os-cloud/global/images/family/ubuntu-1604-lts",
			packageType: []string{"deb", "pip", "gem"},
			shortName:   "ubuntu",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &installInventoryDeb,
			},
		},
		&inventoryTestSetup{
			image:       "projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts",
			packageType: []string{"deb", "pip", "gem"},
			shortName:   "ubuntu",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &installInventoryDeb,
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
	testSetup.name = fmt.Sprintf("inventory-test-%s-%s", path.Base(testSetup.image), compute.RandString(5))

	i := &api.Instance{
		Name:        testSetup.name,
		MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/n1-standard-1", testProject, testZone),
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
	if err := inst.WaitForSerialOutput("inventory install done", 1, 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for inventory agent install: %v", err)
		return nil, false
	}

	testCase.Logf("Checking inventory data")
	ga, err := client.GetGuestAttributes(inst.Project, inst.Zone, inst.Name, "guestInventory/", "")
	if err != nil {
		testCase.WriteFailure("Error getting guest attributes: %v", err)
		return nil, false
	}

	return ga, true
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

	gatherInventoryTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("Gather Inventory [%s]", testSetup.image))
	defer gatherInventoryTest.Finish(tests)

	hostnameTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("Check Hostname [%s]", testSetup.image))
	defer hostnameTest.Finish(tests)

	shortNameTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("Check ShortName [%s]", testSetup.image))
	defer hostnameTest.Finish(tests)

	packageTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("Check InstalledPackages [%s]", testSetup.image))
	defer hostnameTest.Finish(tests)

	if gatherInventoryTest.FilterTestCase(regex) {
		hostnameTest.WriteSkipped("Setup skipped")
		shortNameTest.WriteSkipped("Setup skipped")
		packageTest.WriteSkipped("Setup skipped")
		return
	}

	logger.Printf("Running TestCase '%s.%q'", gatherInventoryTest.Classname, gatherInventoryTest.Name)
	ga, ok := runGatherInventoryTest(ctx, testSetup, gatherInventoryTest)
	logger.Printf("TestCase '%s.%q' finished", gatherInventoryTest.Classname, gatherInventoryTest.Name)
	if !ok {
		hostnameTest.WriteSkipped("Setup Failure")
		shortNameTest.WriteSkipped("Setup Failure")
		packageTest.WriteSkipped("Setup Failure")
		return
	}

	for tc, f := range map[*junitxml.TestCase]func(*apiBeta.GuestAttributes, *inventoryTestSetup, *junitxml.TestCase){
		hostnameTest:  runHostnameTest,
		shortNameTest: runShortNameTest,
		packageTest:   runPackagesTest,
	} {
		if !tc.FilterTestCase(regex) {
			logger.Printf("Running TestCase '%s.%q'", tc.Classname, tc.Name)
			f(ga, testSetup, tc)
			logger.Printf("TestCase '%s.%q' finished", tc.Classname, tc.Name)
		}
	}

}
