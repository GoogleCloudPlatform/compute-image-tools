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
//  limitations under the License.

package packagemanagement

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	"github.com/kylelemons/godebug/pretty"
	api "google.golang.org/api/compute/v1"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	osconfigserver "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/osconfig_server"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_config"
)

const (
	testSuiteName = "PackageManagementTests"
	debianImage   = "projects/debian-cloud/global/images/family/debian-9"
	centosImage   = "projects/centos-cloud/global/images/family/centos-7"
	rhelImage     = "projects/rhel-cloud/global/images/family/rhel-7"
)

var (
	dump = &pretty.Config{IncludeUnexported: true}
)

type packageManagementTestSetup struct {
	image      string
	name       string
	fname      string
	osconfig   *osconfigpb.OsConfig
	assignment *osconfigpb.Assignment
	startup    *api.MetadataItems
	vstring    string
	vf         func(*compute.Instance, string, int64, time.Duration, time.Duration) error
}

// TestSuite is a PackageManagementTests test suite.
func TestSuite(ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite, logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp, testProjectConfig *testconfig.Project) {
	defer tswg.Done()

	if testSuiteRegex != nil && !testSuiteRegex.MatchString(testSuiteName) {
		return
	}

	testSuite := junitxml.NewTestSuite(testSuiteName)
	defer testSuite.Finish(testSuites)

	logger.Printf("Running TestSuite %q", testSuite.Name)
	testSetup := generateAllTestSetup(testProjectConfig)
	var wg sync.WaitGroup
	tests := make(chan *junitxml.TestCase)
	for _, setup := range testSetup {
		wg.Add(1)
		go packageManagementTestCase(ctx, setup, tests, &wg, logger, testCaseRegex, testProjectConfig)
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

func runCreateOsConfigTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *packageManagementTestSetup, logger *log.Logger, testProjectConfig *testconfig.Project) {

	parent := fmt.Sprintf("projects/%s", testProjectConfig.TestProjectID)
	oc, err := osconfigserver.CreateOsConfig(ctx, testSetup.osconfig, parent)
	if err != nil {
		testCase.WriteFailure("error while creating osconfig: \n%s\n", utils.GetStatusFromError(err))
		return
	}

	defer cleanupOsConfig(ctx, testCase, oc, testProjectConfig)
}

func runPackageRemovalTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *packageManagementTestSetup, logger *log.Logger, testProjectConfig *testconfig.Project) {

	parent := fmt.Sprintf("projects/%s", testProjectConfig.TestProjectID)
	oc, err := osconfigserver.CreateOsConfig(ctx, testSetup.osconfig, parent)

	if err != nil {
		testCase.WriteFailure("error while creating osconfig: \n%s\n", utils.GetStatusFromError(err))
		return
	}

	defer cleanupOsConfig(ctx, testCase, oc, testProjectConfig)

	assign, err := osconfigserver.CreateAssignment(ctx, testSetup.assignment, parent)
	if err != nil {
		testCase.WriteFailure("error while creating assignment: \n%s\n", utils.GetStatusFromError(err))
		return
	}

	defer cleanupAssignment(ctx, testCase, assign, testProjectConfig)

	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		testCase.WriteFailure("error creating client: %v", err)
		return
	}

	testCase.Logf("Creating instance with image %q", testSetup.image)
	//TODO: move instance definition to a common method
	i := &api.Instance{
		Name:        testSetup.name,
		MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/n1-standard-1", testProjectConfig.TestProjectID, testProjectConfig.TestZone),
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
		ServiceAccounts: []*api.ServiceAccount{
			&api.ServiceAccount{
				Email:  testProjectConfig.ServiceAccountEmail,
				Scopes: testProjectConfig.ServiceAccountScopes,
			},
		},
	}

	inst, err := compute.CreateInstance(client, testProjectConfig.TestProjectID, testProjectConfig.TestZone, i)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %s", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	testCase.Logf("Waiting for agent install to complete")
	if err := inst.WaitForSerialOutput("osconfig install done", 1, 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	// read the serial console once
	if err = testSetup.vf(inst, testSetup.vstring, 1, 10*time.Second, 600*time.Second); err != nil {
		testCase.WriteFailure("error while asserting: %v", err)
		return
	}
}

func runPackageInstallRemovalTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *packageManagementTestSetup, logger *log.Logger, testProjectConfig *testconfig.Project) {
	parent := fmt.Sprintf("projects/%s", testProjectConfig.TestProjectID)
	oc, err := osconfigserver.CreateOsConfig(ctx, testSetup.osconfig, parent)
	if err != nil {
		testCase.WriteFailure("error while creating osconfig: \n%s\n", utils.GetStatusFromError(err))
		return
	}

	defer cleanupOsConfig(ctx, testCase, oc, testProjectConfig)

	assign, err := osconfigserver.CreateAssignment(ctx, testSetup.assignment, parent)
	if err != nil {
		testCase.WriteFailure("error while creating assignment: \n%s\n", utils.GetStatusFromError(err))
		return
	}

	defer cleanupAssignment(ctx, testCase, assign, testProjectConfig)

	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		testCase.WriteFailure("error creating client: %v", err)
		return
	}

	testCase.Logf("Creating instance with image %q", testSetup.image)
	i := &api.Instance{
		Name:        testSetup.name,
		MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/n1-standard-1", testProjectConfig.TestProjectID, testProjectConfig.TestZone),
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
		ServiceAccounts: []*api.ServiceAccount{
			&api.ServiceAccount{
				Email:  testProjectConfig.ServiceAccountEmail,
				Scopes: testProjectConfig.ServiceAccountScopes,
			},
		},
	}

	inst, err := compute.CreateInstance(client, testProjectConfig.TestProjectID, testProjectConfig.TestZone, i)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	testCase.Logf("Waiting for agent install to complete")
	if err := inst.WaitForSerialOutput("osconfig install done", 1, 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	testCase.Logf("Agent installed successfully")

	// read the serial console once
	if err = testSetup.vf(inst, testSetup.vstring, 1, 10*time.Second, 60*time.Second); err != nil {
		testCase.WriteFailure("error while asserting: %v", err)
	}
}

func runPackageInstallTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *packageManagementTestSetup, logger *log.Logger, testProjectConfig *testconfig.Project) {
	parent := fmt.Sprintf("projects/%s", testProjectConfig.TestProjectID)
	oc, err := osconfigserver.CreateOsConfig(ctx, testSetup.osconfig, parent)
	if err != nil {
		testCase.WriteFailure("error while creating osconfig: \n%s\n", utils.GetStatusFromError(err))
		return
	}
	defer cleanupOsConfig(ctx, testCase, oc, testProjectConfig)

	assign, err := osconfigserver.CreateAssignment(ctx, testSetup.assignment, parent)
	if err != nil {
		testCase.WriteFailure("error while creating assignment: \n%s\n", utils.GetStatusFromError(err))
		return
	}
	defer cleanupAssignment(ctx, testCase, assign, testProjectConfig)

	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		testCase.WriteFailure("error creating client: %v", err)
		return
	}

	testCase.Logf("Creating instance with image %q", testSetup.image)
	i := &api.Instance{
		Name:        testSetup.name,
		MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/n1-standard-1", testProjectConfig.TestProjectID, testProjectConfig.TestZone),
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
		ServiceAccounts: []*api.ServiceAccount{
			&api.ServiceAccount{
				Email:  testProjectConfig.ServiceAccountEmail,
				Scopes: testProjectConfig.ServiceAccountScopes,
			},
		},
	}

	inst, err := compute.CreateInstance(client, testProjectConfig.TestProjectID, testProjectConfig.TestZone, i)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	testCase.Logf("Waiting for agent install to complete")
	if err := inst.WaitForSerialOutput("osconfig install done", 1, 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	testCase.Logf("Agent installed successfully")

	// read the serial console once
	if err = testSetup.vf(inst, testSetup.vstring, 1, 10*time.Second, 60*time.Second); err != nil {
		testCase.WriteFailure("error while asserting: %v", err)
	}
}

func packageManagementTestCase(ctx context.Context, testSetup *packageManagementTestSetup, tests chan *junitxml.TestCase, wg *sync.WaitGroup, logger *log.Logger, regex *regexp.Regexp, testProjectConfig *testconfig.Project) {
	defer wg.Done()

	createOsConfigTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[%s/CreateOsConfig] Create OsConfig", testSetup.image))
	packageInstallTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[%s/PackageInstall] Package installation", testSetup.image))
	packageRemovalTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[%s/PackageRemoval] Package removal", testSetup.image))
	packageInstallRemovalTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[%s/PackageInstallRemoval] Package no change", testSetup.image))

	for tc, f := range map[*junitxml.TestCase]func(context.Context, *junitxml.TestCase, *packageManagementTestSetup, *log.Logger, *testconfig.Project){
		createOsConfigTest:        runCreateOsConfigTest,
		packageInstallTest:        runPackageInstallTest,
		packageRemovalTest:        runPackageRemovalTest,
		packageInstallRemovalTest: runPackageInstallRemovalTest,
	} {
		tfname := strings.ToLower(strings.Replace(testSetup.fname, "test", "", 1))
		ttc := strings.ToLower(getTestNameFromTestCase(tc.Name))
		if strings.Compare(tfname, ttc) != 0 {
			continue
		}
		if tc.FilterTestCase(regex) {
			tc.Finish(tests)
		} else {
			logger.Printf("Running TestCase %s.%q", tc.Classname, tc.Name)
			f(ctx, tc, testSetup, logger, testProjectConfig)
			tc.Finish(tests)
			logger.Printf("TestCase %s.%q finished in %fs", tc.Classname, tc.Name, tc.Time)
		}
	}
}

func cleanupOsConfig(ctx context.Context, testCase *junitxml.TestCase, oc *osconfigserver.OsConfig, testProjectConfig *testconfig.Project) {
	err := oc.Cleanup(ctx, testProjectConfig.TestProjectID)
	if err != nil {
		testCase.WriteFailure(fmt.Sprintf("error while deleting osconfig: %s", utils.GetStatusFromError(err)))
	}
}

func cleanupAssignment(ctx context.Context, testCase *junitxml.TestCase, assignment *osconfigserver.Assignment, testProjectConfig *testconfig.Project) {
	err := assignment.Cleanup(ctx, testProjectConfig.TestProjectID)
	if err != nil {
		testCase.WriteFailure(fmt.Sprintf("error while deleting assignment: %s", utils.GetStatusFromError(err)))
	}
}

func getTestNameFromTestCase(tc string) string {
	re := regexp.MustCompile(`\[[^]]*\]`)
	ss := re.FindAllString(tc, -1)
	var ret []string
	for _, s := range ss {
		ret = append(ret, s[1:len(s)-1])
	}
	return filepath.Base(ret[1])
}
