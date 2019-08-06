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

// Package guestpolicies GuestPolicy osconfig agent tests.
package guestpolicies

import (
	"context"
	"fmt"
	"log"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/config"
	gcpclients "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/gcp_clients"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	osconfigV1alpha2 "github.com/GoogleCloudPlatform/osconfig/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha2"
	"github.com/kylelemons/godebug/pretty"
	computeApi "google.golang.org/api/compute/v1"

	osconfigpb "github.com/GoogleCloudPlatform/osconfig/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha2"
)

var (
	testSuiteName = "OSPackage"
)

var (
	dump = &pretty.Config{IncludeUnexported: true}
)

const (
	packageInstallFunction            = "pkginstall"
	packageRemovalFunction            = "pkgremoval"
	packageInstallFromNewRepoFunction = "pkgfromnewrepo"
	packageUpdateFunction             = "pkgupdate"
	packageNoUpdateFunction           = "pkgnoupdate"
)

type guestPolicyTestSetup struct {
	image         string
	guestPolicyID string
	instanceName  string
	testName      string
	guestPolicy   *osconfigpb.GuestPolicy
	startup       *computeApi.MetadataItems
	machineType   string
	queryPath     string
	assertTimeout time.Duration
}

func newGuestPolicyTestSetup(image, instanceName, testName, queryPath, machineType string, gp *osconfigpb.GuestPolicy, startup *computeApi.MetadataItems, assertTimeout time.Duration) *guestPolicyTestSetup {
	return &guestPolicyTestSetup{
		image:         image,
		guestPolicyID: instanceName,
		instanceName:  instanceName,
		guestPolicy:   gp,
		testName:      testName,
		machineType:   machineType,
		queryPath:     queryPath,
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

// We only want to create one GuestPolicy at a time to limit QPS.
var gpMx sync.Mutex

func createGuestPolicy(ctx context.Context, client *osconfigV1alpha2.Client, req *osconfigpb.CreateGuestPolicyRequest) (*osconfigpb.GuestPolicy, error) {
	gpMx.Lock()
	defer gpMx.Unlock()
	return client.CreateGuestPolicy(ctx, req)
}

func runTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *guestPolicyTestSetup, logger *log.Logger, logwg *sync.WaitGroup, testProjectConfig *testconfig.Project) {
	parent := fmt.Sprintf("projects/%s", testProjectConfig.TestProjectID)

	client, err := gcpclients.GetOsConfigClientV1alpha2()
	if err != nil {
		testCase.WriteFailure("Error getting osconfig client: %v", err)
		return
	}

	req := &osconfigpb.CreateGuestPolicyRequest{
		Parent:        parent,
		GuestPolicyId: testSetup.guestPolicyID,
		GuestPolicy:   testSetup.guestPolicy,
	}

	res, err := createGuestPolicy(ctx, client, req)
	if err != nil {
		testCase.WriteFailure("Error running CreateGuestPolicy: %s", utils.GetStatusFromError(err))
		return
	}
	defer cleanupGuestPolicy(ctx, testCase, res)

	computeClient, err := gcpclients.GetComputeClient()
	if err != nil {
		testCase.WriteFailure("Error getting compute client: %v", err)
		return
	}

	testCase.Logf("Creating instance with image %q", testSetup.image)
	var metadataItems []*computeApi.MetadataItems
	metadataItems = append(metadataItems, testSetup.startup)
	metadataItems = append(metadataItems, compute.BuildInstanceMetadataItem("os-config-enabled-prerelease-features", "ospackage"))
	inst, err := utils.CreateComputeInstance(metadataItems, computeClient, testSetup.machineType, testSetup.image, testSetup.instanceName, testProjectConfig.TestProjectID, testProjectConfig.GetZone(), testProjectConfig.ServiceAccountEmail, testProjectConfig.ServiceAccountScopes)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %s", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	storageClient, err := gcpclients.GetStorageClient()
	if err != nil {
		testCase.WriteFailure("Error getting storage client: %v", err)
	}
	logwg.Add(1)
	go inst.StreamSerialOutput(ctx, storageClient, path.Join(testSuiteName, config.LogsPath()), config.LogBucket(), logwg, 1, config.LogPushInterval())

	testCase.Logf("Waiting for agent install to complete")
	if _, err := inst.WaitForGuestAttributes("osconfig_tests/install_done", 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	if _, err := inst.WaitForGuestAttributes(testSetup.queryPath, 10*time.Second, testSetup.assertTimeout); err != nil {
		testCase.WriteFailure("error while asserting: %v", err)
		return
	}
}

func packageManagementTestCase(ctx context.Context, testSetup *guestPolicyTestSetup, tests chan *junitxml.TestCase, wg *sync.WaitGroup, logger *log.Logger, regex *regexp.Regexp, testProjectConfig *testconfig.Project) {
	defer wg.Done()

	var logwg sync.WaitGroup

	tc, err := getTestCaseFromTestSetUp(testSetup)
	if err != nil {
		logger.Fatalf("invalid testcase: %+v", err)
		return
	}
	if tc.FilterTestCase(regex) {
		tc.Finish(tests)
	} else {
		logger.Printf("Running TestCase %q", tc.Name)
		runTest(ctx, tc, testSetup, logger, &logwg, testProjectConfig)
		tc.Finish(tests)
		logger.Printf("TestCase %q finished in %fs", tc.Name, tc.Time)
	}

	logwg.Wait()
}

// factory method to get testcase from the testsetup
func getTestCaseFromTestSetUp(testSetup *guestPolicyTestSetup) (*junitxml.TestCase, error) {
	var tc *junitxml.TestCase

	switch testSetup.testName {
	case packageInstallFunction:
		tc = junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Package installation] [%s]", path.Base(testSetup.image)))
	case packageRemovalFunction:
		tc = junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Package removal] [%s]", path.Base(testSetup.image)))
	case packageInstallFromNewRepoFunction:
		tc = junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Add a new package from new repository] [%s]", path.Base(testSetup.image)))
	case packageUpdateFunction:
		tc = junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Package update] [%s]", path.Base(testSetup.image)))
	case packageNoUpdateFunction:
		tc = junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Package install doesn't update] [%s]", path.Base(testSetup.image)))
	default:
		return nil, fmt.Errorf("unknown test function name: %s", testSetup.testName)
	}

	return tc, nil
}

func cleanupGuestPolicy(ctx context.Context, testCase *junitxml.TestCase, gp *osconfigpb.GuestPolicy) {
	client, err := gcpclients.GetOsConfigClientV1alpha2()
	if err != nil {
		testCase.WriteFailure(fmt.Sprintf("Error while deleting osconfig: %s", utils.GetStatusFromError(err)))
	}

	if err := client.DeleteGuestPolicy(ctx, &osconfigpb.DeleteGuestPolicyRequest{Name: gp.GetName()}); err != nil {
		testCase.WriteFailure(fmt.Sprintf("Error while deleting osconfig: %s", utils.GetStatusFromError(err)))
	}
}
