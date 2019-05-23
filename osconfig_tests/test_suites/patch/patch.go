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

// Package patch contains end to end tests for patch management
package patch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path"
	"regexp"
	"strconv"
	"sync"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	gcpclients "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/gcp_clients"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/junitxml"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/kylelemons/godebug/pretty"
	api "google.golang.org/api/compute/v1"

	osconfigpb "github.com/GoogleCloudPlatform/osconfig/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

const (
	testSuiteName = "OSPatch"
)

var (
	dump   = &pretty.Config{IncludeUnexported: true}
	suffix = utils.RandString(5)
)

// TestSuite is a OSPatch test suite.
func TestSuite(ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite, logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp, testProjectConfig *testconfig.Project) {
	defer tswg.Done()

	if testSuiteRegex != nil && !testSuiteRegex.MatchString(testSuiteName) {
		return
	}

	testSuite := junitxml.NewTestSuite(testSuiteName)
	defer testSuite.Finish(testSuites)

	logger.Printf("Running TestSuite %q", testSuite.Name)

	var wg sync.WaitGroup
	tests := make(chan *junitxml.TestCase)
	// Basic functionality smoke test against all latest images.
	for _, setup := range headImageTestSetup() {
		wg.Add(1)
		s := setup
		tc := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Execute PatchJob] [%s]", s.testName))
		f := func() { runExecutePatchJobTest(ctx, tc, s, testProjectConfig, nil) }
		go runTestCase(tc, f, tests, &wg, logger, testCaseRegex)
	}
	// Test that updates trigger reboot as expected.
	for _, setup := range oldImageTestSetup() {
		wg.Add(1)
		s := setup
		tc := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[PatchJob triggers reboot] [%s]", s.testName))
		f := func() {
			minBootCount := 2
			maxBootCount := 5
			runRebootPatchTest(ctx, tc, s, testProjectConfig, &osconfigpb.PatchConfig{Apt: &osconfigpb.AptSettings{Type: osconfigpb.AptSettings_DIST}}, minBootCount, maxBootCount)
		}
		go runTestCase(tc, f, tests, &wg, logger, testCaseRegex)
	}
	// Test that PatchConfig_NEVER prevents reboot.
	for _, setup := range oldImageTestSetup() {
		wg.Add(1)
		s := setup
		tc := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[PatchJob does not reboot] [%s]", s.testName))
		pc := &osconfigpb.PatchConfig{RebootConfig: osconfigpb.PatchConfig_NEVER, Apt: &osconfigpb.AptSettings{Type: osconfigpb.AptSettings_DIST}}
		f := func() { runRebootPatchTest(ctx, tc, s, testProjectConfig, pc, 1, 1) }
		go runTestCase(tc, f, tests, &wg, logger, testCaseRegex)
	}
	// Test APT specific functionality, this just tests that using these settings doesn't break anything.
	for _, setup := range aptHeadImageTestSetup() {
		wg.Add(1)
		s := setup
		tc := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[APT dist-upgrade] [%s]", s.testName))
		f := func() {
			runExecutePatchJobTest(ctx, tc, s, testProjectConfig, &osconfigpb.PatchConfig{Apt: &osconfigpb.AptSettings{Type: osconfigpb.AptSettings_DIST}})
		}
		go runTestCase(tc, f, tests, &wg, logger, testCaseRegex)
	}
	// Test YUM specific functionality, this just tests that using these settings doesn't break anything.
	for _, setup := range yumHeadImageTestSetup() {
		wg.Add(1)
		s := setup
		tc := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[YUM security, minimal and excludes] [%s]", s.testName))
		f := func() {
			runExecutePatchJobTest(ctx, tc, s, testProjectConfig, &osconfigpb.PatchConfig{Yum: &osconfigpb.YumSettings{Security: true, Minimal: true, Excludes: []string{"pkg1", "pkg2"}}})
		}
		go runTestCase(tc, f, tests, &wg, logger, testCaseRegex)
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

func awaitPatchJob(ctx context.Context, job *osconfigpb.PatchJob, timeout time.Duration) error {
	client, err := gcpclients.GetOsConfigClient(ctx)
	if err != nil {
		return err
	}
	tick := time.Tick(10 * time.Second)
	timedout := time.Tick(timeout)
	for {
		select {
		case <-timedout:
			return errors.New("timed out")
		case <-tick:
			req := &osconfigpb.GetPatchJobRequest{
				Name: job.GetName(),
			}
			res, err := client.GetPatchJob(ctx, req)
			if err != nil {
				return fmt.Errorf("error while fetching patch job: %s", utils.GetStatusFromError(err))
			}

			if isPatchJobFailureState(res.State) {
				return fmt.Errorf("failure status %v with message '%s'", res.State, job.GetErrorMessage())
			}

			if res.State == osconfigpb.PatchJob_SUCCEEDED {
				if res.InstanceDetailsSummary.GetInstancesSucceeded() < 1 {
					return errors.New("completed with no instances patched")
				}
				return nil
			}
		}
	}
}

func runExecutePatchJobTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *patchTestSetup, testProjectConfig *testconfig.Project, pc *osconfigpb.PatchConfig) {
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		testCase.WriteFailure("error creating client: %v", err)
		return
	}

	testCase.Logf("Creating instance with image %q", testSetup.image)
	var metadataItems []*api.MetadataItems
	metadataItems = append(metadataItems, testSetup.startup)
	metadataItems = append(metadataItems, compute.BuildInstanceMetadataItem("os-config-enabled-prerelease-features", "ospatch"))
	name := fmt.Sprintf("patch-test-%s-%s", path.Base(testSetup.testName), suffix)
	inst, err := utils.CreateComputeInstance(metadataItems, client, testSetup.machineType, testSetup.image, name, testProjectConfig.TestProjectID, testProjectConfig.GetZone(), testProjectConfig.ServiceAccountEmail, testProjectConfig.ServiceAccountScopes)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	testCase.Logf("Waiting for agent install to complete")
	if _, err := inst.WaitForGuestAttribute("osconfig_tests/", "install_done", 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	testCase.Logf("Agent installed successfully")

	parent := fmt.Sprintf("projects/%s", testProjectConfig.TestProjectID)
	osconfigClient, err := gcpclients.GetOsConfigClient(ctx)

	req := &osconfigpb.ExecutePatchJobRequest{
		Parent:      parent,
		Description: "testing patch job run",
		Filter:      fmt.Sprintf("name=\"%s\"", name),
		Duration:    &duration.Duration{Seconds: int64(testSetup.assertTimeout / time.Second)},
		PatchConfig: pc,
	}
	job, err := osconfigClient.ExecutePatchJob(ctx, req)

	if err != nil {
		testCase.WriteFailure("error while executing patch job: \n%s\n", utils.GetStatusFromError(err))
		return
	}

	testCase.Logf("Started patch job '%s'", job.GetName())
	if err := awaitPatchJob(ctx, job, testSetup.assertTimeout); err != nil {
		testCase.WriteFailure("Patch job '%s' error: %v", job.GetName(), err)
	}
}

func runRebootPatchTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *patchTestSetup, testProjectConfig *testconfig.Project, pc *osconfigpb.PatchConfig, minBootCount, maxBootCount int) {
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		testCase.WriteFailure("error creating client: %v", err)
		return
	}

	testCase.Logf("Creating instance with image %q", testSetup.image)
	var metadataItems []*api.MetadataItems
	metadataItems = append(metadataItems, testSetup.startup)
	metadataItems = append(metadataItems, compute.BuildInstanceMetadataItem("os-config-enabled-prerelease-features", "ospatch"))
	name := fmt.Sprintf("patch-reboot-%s-%s", path.Base(testSetup.testName), suffix)
	inst, err := utils.CreateComputeInstance(metadataItems, client, testSetup.machineType, testSetup.image, name, testProjectConfig.TestProjectID, testProjectConfig.GetZone(), testProjectConfig.ServiceAccountEmail, testProjectConfig.ServiceAccountScopes)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	testCase.Logf("Waiting for agent install to complete")
	if _, err := inst.WaitForGuestAttribute("osconfig_tests/", "install_done", 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	testCase.Logf("Agent installed successfully")

	parent := fmt.Sprintf("projects/%s", testProjectConfig.TestProjectID)
	osconfigClient, err := gcpclients.GetOsConfigClient(ctx)

	req := &osconfigpb.ExecutePatchJobRequest{
		Parent:      parent,
		Description: "testing patch job reboot",
		Filter:      fmt.Sprintf("name=\"%s\"", name),
		Duration:    &duration.Duration{Seconds: int64(testSetup.assertTimeout / time.Second)},
		PatchConfig: &osconfigpb.PatchConfig{Apt: &osconfigpb.AptSettings{Type: osconfigpb.AptSettings_DIST}},
	}
	job, err := osconfigClient.ExecutePatchJob(ctx, req)

	if err != nil {
		testCase.WriteFailure("error while executing patch job: \n%s\n", utils.GetStatusFromError(err))
		return
	}

	testCase.Logf("Started patch job '%s'", job.GetName())
	if err := awaitPatchJob(ctx, job, testSetup.assertTimeout); err != nil {
		testCase.WriteFailure("Patch job '%s' error: %v", job.GetName(), err)
	}

	testCase.Logf("Checking reboot count")
	attr, err := inst.GetGuestAttribute("osconfig_tests/", "boot_count")
	if err != nil {
		testCase.WriteFailure("Error retrieving boot count: %v", err)
		return
	}
	num, err := strconv.Atoi(attr)
	if err != nil {
		testCase.WriteFailure("Error parsing boot count: %v", err)
		return
	}
	if num < minBootCount || num > maxBootCount {
		testCase.WriteFailure("Instance rebooted %d times, minBootCount: %d, maxBootCount: %d", num, minBootCount, maxBootCount)
		return
	}
}

func isPatchJobFailureState(state osconfigpb.PatchJob_State) bool {
	return state == osconfigpb.PatchJob_COMPLETED_WITH_ERRORS ||
		state == osconfigpb.PatchJob_TIMED_OUT ||
		state == osconfigpb.PatchJob_CANCELED
}

func runTestCase(tc *junitxml.TestCase, f func(), tests chan *junitxml.TestCase, wg *sync.WaitGroup, logger *log.Logger, regex *regexp.Regexp) {
	defer wg.Done()

	if tc.FilterTestCase(regex) {
		tc.Finish(tests)
	} else {
		logger.Printf("Running TestCase %q", tc.Name)
		f()
		tc.Finish(tests)
		logger.Printf("TestCase %q finished in %fs", tc.Name, tc.Time)
	}
}
