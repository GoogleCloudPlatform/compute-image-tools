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
	"path"
	"regexp"
	"sync"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/gcp_clients"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/osconfig_server"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	"github.com/kylelemons/godebug/pretty"
	api "google.golang.org/api/compute/v1"

	osconfigpb "github.com/GoogleCloudPlatform/osconfig/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

var (
	testSuiteName = "OSPackage"
	debianImages  = []string{"projects/ubuntu-os-cloud/global/images/family/ubuntu-1604-lts", "projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts", "projects/debian-cloud/global/images/family/debian-9"}
	centosImages  = []string{"projects/centos-cloud/global/images/family/centos-6", "projects/centos-cloud/global/images/family/centos-7"}
	rhelImages    = []string{"projects/rhel-cloud/global/images/family/rhel-6", "projects/rhel-cloud/global/images/family/rhel-7"}
	windowsImages = []string{"projects/windows-cloud/global/images/family/windows-2008-r2",
		"projects/windows-cloud/global/images/family/windows-2012-r2",
		"projects/windows-cloud/global/images/family/windows-2012-r2-core",
		"projects/windows-cloud/global/images/family/windows-2016",
		"projects/windows-cloud/global/images/family/windows-2016-core",
		"projects/windows-cloud/global/images/family/windows-1709-core",
		"projects/windows-cloud/global/images/family/windows-1803-core",
		"projects/windows-cloud/global/images/family/windows-1809-core",
		"projects/windows-cloud/global/images/family/windows-2019-core",
		"projects/windows-cloud/global/images/family/windows-2019"}
)

var (
	dump = &pretty.Config{IncludeUnexported: true}
)

type packageMangementTestFunctionName string

const (
	createOsConfigFunction            = "createosconfig"
	packageInstallFunction            = "packageinstall"
	packageRemovalFunction            = "packageremoval"
	packageInstallRemovalFunction     = "packageinstallremoval"
	packageInstallFromNewRepoFunction = "packageinstallfromnewrepo"
)

type packageManagementTestSetup struct {
	image         string
	name          string
	fname         packageMangementTestFunctionName // this is used to identify the test case for this test setup
	osconfig      *osconfigpb.OsConfig
	assignment    *osconfigpb.Assignment
	startup       *api.MetadataItems
	vstring       string
	assertTimeout time.Duration
	vf            func(*compute.Instance, string, int64, time.Duration, time.Duration) error
}

func newPackageManagementTestSetup(setup **packageManagementTestSetup, image, name string, fname packageMangementTestFunctionName, vs string, oc *osconfigpb.OsConfig, assignment *osconfigpb.Assignment, startup *api.MetadataItems, assertTimeout time.Duration, vf func(*compute.Instance, string, int64, time.Duration, time.Duration) error) {
	*setup = &packageManagementTestSetup{
		image:         image,
		name:          name,
		osconfig:      oc,
		assignment:    assignment,
		fname:         fname,
		vf:            vf,
		vstring:       vs,
		assertTimeout: assertTimeout,
		startup:       startup,
	}
}

// TestSuite is a OSPackage test suite.
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

func runCreateOsConfigTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *packageManagementTestSetup, logger *log.Logger, logwg *sync.WaitGroup, testProjectConfig *testconfig.Project) {

	parent := fmt.Sprintf("projects/%s", testProjectConfig.TestProjectID)
	oc, err := osconfigserver.CreateOsConfig(ctx, testSetup.osconfig, parent)
	if err != nil {
		testCase.WriteFailure("error while creating osconfig: \n%s\n", utils.GetStatusFromError(err))
		return
	}

	defer cleanupOsConfig(ctx, testCase, oc, testProjectConfig)
}

func runPackageRemovalTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *packageManagementTestSetup, logger *log.Logger, logwg *sync.WaitGroup, testProjectConfig *testconfig.Project) {

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
	var metadataItems []*api.MetadataItems
	metadataItems = append(metadataItems, testSetup.startup)
	metadataItems = append(metadataItems, compute.BuildInstanceMetadataItem("os-config-enabled-prerelease-features", "ospackage"))
	inst, err := utils.CreateComputeInstance(metadataItems, client, "n1-standard-4", testSetup.image, testSetup.name, testProjectConfig.TestProjectID, testProjectConfig.GetZone(), testProjectConfig.ServiceAccountEmail, testProjectConfig.ServiceAccountScopes)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %s", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	storageClient, err := gcpclients.GetStorageClient(ctx)
	if err != nil {
		testCase.WriteFailure("Error getting storage client: %v", err)
	}
	logwg.Add(1)
	go inst.StreamSerialOutput(ctx, storageClient, path.Join(testSuiteName, config.LogsPath()), config.LogBucket(), logwg, 1, config.LogPushInterval())

	testCase.Logf("Waiting for agent install to complete")
	if err := inst.WaitForSerialOutput("osconfig install done", 1, 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	// read the serial console once
	if err = testSetup.vf(inst, testSetup.vstring, 1, 10*time.Second, testSetup.assertTimeout); err != nil {
		testCase.WriteFailure("error while asserting: %v", err)
		return
	}
}

func runPackageInstallRemovalTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *packageManagementTestSetup, logger *log.Logger, logwg *sync.WaitGroup, testProjectConfig *testconfig.Project) {
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
	var metadataItems []*api.MetadataItems
	metadataItems = append(metadataItems, testSetup.startup)
	metadataItems = append(metadataItems, compute.BuildInstanceMetadataItem("os-config-enabled-prerelease-features", "ospackage"))
	inst, err := utils.CreateComputeInstance(metadataItems, client, "n1-standard-4", testSetup.image, testSetup.name, testProjectConfig.TestProjectID, testProjectConfig.GetZone(), testProjectConfig.ServiceAccountEmail, testProjectConfig.ServiceAccountScopes)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	storageClient, err := gcpclients.GetStorageClient(ctx)
	if err != nil {
		testCase.WriteFailure("Error getting storage client: %v", err)
	}
	logwg.Add(1)
	go inst.StreamSerialOutput(ctx, storageClient, path.Join(testSuiteName, config.LogsPath()), config.LogBucket(), logwg, 1, config.LogPushInterval())

	testCase.Logf("Waiting for agent install to complete")
	if err := inst.WaitForSerialOutput("osconfig install done", 1, 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	testCase.Logf("Agent installed successfully")

	// read the serial console once
	if err = testSetup.vf(inst, testSetup.vstring, 1, 10*time.Second, testSetup.assertTimeout); err != nil {
		testCase.WriteFailure("error while asserting: %v", err)
	}
}

func runPackageInstallTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *packageManagementTestSetup, logger *log.Logger, logwg *sync.WaitGroup, testProjectConfig *testconfig.Project) {
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
	var metadataItems []*api.MetadataItems
	metadataItems = append(metadataItems, testSetup.startup)
	metadataItems = append(metadataItems, compute.BuildInstanceMetadataItem("os-config-enabled-prerelease-features", "ospackage"))
	inst, err := utils.CreateComputeInstance(metadataItems, client, "n1-standard-4", testSetup.image, testSetup.name, testProjectConfig.TestProjectID, testProjectConfig.GetZone(), testProjectConfig.ServiceAccountEmail, testProjectConfig.ServiceAccountScopes)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	storageClient, err := gcpclients.GetStorageClient(ctx)
	if err != nil {
		testCase.WriteFailure("Error getting storage client: %v", err)
	}
	logwg.Add(1)
	go inst.StreamSerialOutput(ctx, storageClient, path.Join(testSuiteName, config.LogsPath()), config.LogBucket(), logwg, 1, config.LogPushInterval())

	testCase.Logf("Waiting for agent install to complete")
	if err := inst.WaitForSerialOutput("osconfig install done", 1, 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	testCase.Logf("Agent installed successfully")

	// read the serial console once
	if err = testSetup.vf(inst, testSetup.vstring, 1, 10*time.Second, testSetup.assertTimeout); err != nil {
		testCase.WriteFailure("error while asserting: %v", err)
	}
}

func runPackageInstallFromNewRepoTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *packageManagementTestSetup, logger *log.Logger, logwg *sync.WaitGroup, testProjectConfig *testconfig.Project) {
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
	var metadataItems []*api.MetadataItems
	metadataItems = append(metadataItems, testSetup.startup)
	metadataItems = append(metadataItems, compute.BuildInstanceMetadataItem("os-config-enabled-prerelease-features", "ospackage"))
	inst, err := utils.CreateComputeInstance(metadataItems, client, "n1-standard-4", testSetup.image, testSetup.name, testProjectConfig.TestProjectID, testProjectConfig.GetZone(), testProjectConfig.ServiceAccountEmail, testProjectConfig.ServiceAccountScopes)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	storageClient, err := gcpclients.GetStorageClient(ctx)
	if err != nil {
		testCase.WriteFailure("Error getting storage client: %v", err)
	}
	logwg.Add(1)
	go inst.StreamSerialOutput(ctx, storageClient, path.Join(testSuiteName, config.LogsPath()), config.LogBucket(), logwg, 1, config.LogPushInterval())

	testCase.Logf("Waiting for agent install to complete")
	if err := inst.WaitForSerialOutput("osconfig install done", 1, 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	testCase.Logf("Agent installed successfully")

	// read the serial console once
	if err = testSetup.vf(inst, testSetup.vstring, 1, 10*time.Second, testSetup.assertTimeout); err != nil {
		testCase.WriteFailure("error while asserting: %v", err)
	}
}

func packageManagementTestCase(ctx context.Context, testSetup *packageManagementTestSetup, tests chan *junitxml.TestCase, wg *sync.WaitGroup, logger *log.Logger, regex *regexp.Regexp, testProjectConfig *testconfig.Project) {
	defer wg.Done()

	var logwg sync.WaitGroup

	tc, f, err := getTestCaseFromTestSetUp(testSetup)
	if err != nil {
		logger.Fatalf("invalid testcase: %+v", err)
		return
	}
	if tc.FilterTestCase(regex) {
		tc.Finish(tests)
	} else {
		logger.Printf("Running TestCase %q", tc.Name)
		f(ctx, tc, testSetup, logger, &logwg, testProjectConfig)
		tc.Finish(tests)
		logger.Printf("TestCase %q finished in %fs", tc.Name, tc.Time)
	}

	logwg.Wait()
}

// factory method to get testcase from the testsetup
func getTestCaseFromTestSetUp(testSetup *packageManagementTestSetup) (*junitxml.TestCase, func(context.Context, *junitxml.TestCase, *packageManagementTestSetup, *log.Logger, *sync.WaitGroup, *testconfig.Project), error) {
	var tc *junitxml.TestCase
	var f func(context.Context, *junitxml.TestCase, *packageManagementTestSetup, *log.Logger, *sync.WaitGroup, *testconfig.Project)

	switch testSetup.fname {
	case createOsConfigFunction:
		tc = junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Create OsConfig] [%s]", path.Base(testSetup.image)))
		f = runCreateOsConfigTest
	case packageInstallFunction:
		tc = junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Package installation] [%s]", path.Base(testSetup.image)))
		f = runPackageInstallTest
	case packageRemovalFunction:
		tc = junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Package removal] [%s]", path.Base(testSetup.image)))
		f = runPackageRemovalTest
	case packageInstallRemovalFunction:
		tc = junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Package no change] [%s]", path.Base(testSetup.image)))
		f = runPackageInstallRemovalTest
	case packageInstallFromNewRepoFunction:
		tc = junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Add a new package from new repository] [%s]", path.Base(testSetup.image)))
		f = runPackageInstallFromNewRepoTest
	default:
		return nil, nil, fmt.Errorf("unknown test function name: %s", testSetup.fname)
	}

	return tc, f, nil
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