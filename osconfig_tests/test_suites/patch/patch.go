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

	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	gcpclients "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/gcp_clients"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/kylelemons/godebug/pretty"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/_internal/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha2"
)

const (
	testSuiteName = "OSPatch"
)

var (
	dump       = &pretty.Config{IncludeUnexported: true}
	testSuffix = utils.RandString(3)
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
		shouldReboot := true
		f := func() {
			runRebootPatchTest(ctx, tc, s, testProjectConfig, &osconfigpb.PatchConfig{Apt: &osconfigpb.AptSettings{Type: osconfigpb.AptSettings_DIST}}, shouldReboot)
		}
		go runTestCase(tc, f, tests, &wg, logger, testCaseRegex)
	}
	// Test that PatchConfig_NEVER prevents reboot.
	for _, setup := range oldImageTestSetup() {
		wg.Add(1)
		s := setup
		tc := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[PatchJob does not reboot] [%s]", s.testName))
		pc := &osconfigpb.PatchConfig{RebootConfig: osconfigpb.PatchConfig_NEVER, Apt: &osconfigpb.AptSettings{Type: osconfigpb.AptSettings_DIST}}
		shouldReboot := false
		f := func() { runRebootPatchTest(ctx, tc, s, testProjectConfig, pc, shouldReboot) }
		go runTestCase(tc, f, tests, &wg, logger, testCaseRegex)
	}
	// Test that pre- and post-patch steps run as expected.
	for _, setup := range headImageTestSetup() {
		wg.Add(1)
		s := setup
		tc := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[PatchJob runs pre-step and post-step] [%s]", s.testName))
		pc := patchConfigWithPrePostSteps()
		f := func() { runExecutePatchJobTest(ctx, tc, s, testProjectConfig, pc) }
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
	// Test Zypper specific functionality, this just tests that using these settings doesn't break anything.
	for _, setup := range suseHeadImageTestSetup() {
		wg.Add(1)
		s := setup
		tc := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[Zypper WithOptional, WithUpdate, Categories and Severities] [%s]", s.testName))
		f := func() {
			runExecutePatchJobTest(ctx, tc, s, testProjectConfig, &osconfigpb.PatchConfig{
				Zypper: &osconfigpb.ZypperSettings{WithOptional: true, WithUpdate: true, Categories: []string{"security", "recommended", "feature"}, Severities: []string{"critical", "important", "moderate", "low"}}})
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

func awaitPatchJob(ctx context.Context, job *osconfigpb.PatchJob, timeout time.Duration) (*osconfigpb.PatchJob, error) {
	client, err := gcpclients.GetOsConfigClientV1alpha2()
	if err != nil {
		return nil, err
	}
	tick := time.Tick(10 * time.Second)
	timedout := time.Tick(timeout)
	for {
		select {
		case <-timedout:
			return nil, errors.New("timed out while waiting for patch job to complete")
		case <-tick:
			req := &osconfigpb.GetPatchJobRequest{
				Name: job.GetName(),
			}
			res, err := client.GetPatchJob(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("error while fetching patch job: %s", utils.GetStatusFromError(err))
			}

			if isPatchJobFailureState(res.State) {
				return nil, fmt.Errorf("failure status %v with message '%s'", res.State, job.GetErrorMessage())
			}

			if res.State == osconfigpb.PatchJob_SUCCEEDED {
				if res.InstanceDetailsSummary.GetInstancesSucceeded() < 1 && res.InstanceDetailsSummary.GetInstancesSucceededRebootRequired() < 1 {
					return nil, errors.New("completed with no instances patched")
				}
				return res, nil
			}
		}
	}
}

func runExecutePatchJobTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *patchTestSetup, testProjectConfig *testconfig.Project, pc *osconfigpb.PatchConfig) {
	computeClient, err := gcpclients.GetComputeClient()
	if err != nil {
		testCase.WriteFailure("Error getting compute client: %v", err)
		return
	}

	testCase.Logf("Creating instance with image %q", testSetup.image)
	name := fmt.Sprintf("patch-test-%s-%s-%s", path.Base(testSetup.testName), testSuffix, utils.RandString(5))
	zone := testProjectConfig.AquireZone()
	defer testProjectConfig.ReleaseZone(zone)
	inst, err := utils.CreateComputeInstance(testSetup.metadata, computeClient, testSetup.machineType, testSetup.image, name, testProjectConfig.TestProjectID, zone, testProjectConfig.ServiceAccountEmail, testProjectConfig.ServiceAccountScopes)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	testCase.Logf("Waiting for agent install to complete")
	if _, err := inst.WaitForGuestAttributes("osconfig_tests/install_done", 5*time.Second, 15*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	testCase.Logf("Agent installed successfully")
	if err := inst.AddMetadata(compute.BuildInstanceMetadataItem("windows-startup-script-ps1", windowsRecordBoot), compute.BuildInstanceMetadataItem("startup-script", linuxRecordBoot)); err != nil {
		testCase.WriteFailure("Error setting metadata: %v", err)
		return
	}

	parent := fmt.Sprintf("projects/%s", testProjectConfig.TestProjectID)
	osconfigClient, err := gcpclients.GetOsConfigClientV1alpha2()
	if err != nil {
		testCase.WriteFailure("Error getting osconfig client: %v", err)
		return
	}

	req := &osconfigpb.ExecutePatchJobRequest{
		Parent:      parent,
		Description: "testing patch job run",
		Filter:      fmt.Sprintf("name=\"%s\"", name),
		Duration:    &duration.Duration{Seconds: int64(testSetup.assertTimeout / time.Second)},
		PatchConfig: pc,
	}
	testCase.Logf("Running ExecutePatchJob")
	job, err := osconfigClient.ExecutePatchJob(ctx, req)
	if err != nil {
		testCase.WriteFailure("Error running ExecutePatchJob: %s", utils.GetStatusFromError(err))
		return
	}

	testCase.Logf("Started patch job %q", job.GetName())
	if _, err := awaitPatchJob(ctx, job, testSetup.assertTimeout); err != nil {
		testCase.WriteFailure("Patch job %q error: %v", job.GetName(), err)
		return
	}

	if pc.GetPreStep() != nil && pc.GetPostStep() != nil {
		validatePrePostStepSuccess(inst, testCase)
	}
}

func runRebootPatchTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *patchTestSetup, testProjectConfig *testconfig.Project, pc *osconfigpb.PatchConfig, shouldReboot bool) {
	computeClient, err := gcpclients.GetComputeClient()
	if err != nil {
		testCase.WriteFailure("Error getting compute client: %v", err)
		return
	}

	testCase.Logf("Creating instance with image %q", testSetup.image)
	name := fmt.Sprintf("patch-reboot-%s-%s-%s", path.Base(testSetup.testName), testSuffix, utils.RandString(5))
	zone := testProjectConfig.AquireZone()
	defer testProjectConfig.ReleaseZone(zone)
	inst, err := utils.CreateComputeInstance(testSetup.metadata, computeClient, testSetup.machineType, testSetup.image, name, testProjectConfig.TestProjectID, zone, testProjectConfig.ServiceAccountEmail, testProjectConfig.ServiceAccountScopes)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	testCase.Logf("Waiting for agent install to complete")
	if _, err := inst.WaitForGuestAttributes("osconfig_tests/install_done", 5*time.Second, 15*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	testCase.Logf("Agent installed successfully")
	if err := inst.AddMetadata(compute.BuildInstanceMetadataItem("windows-startup-script-ps1", windowsRecordBoot), compute.BuildInstanceMetadataItem("startup-script", linuxRecordBoot)); err != nil {
		testCase.WriteFailure("Error setting metadata: %v", err)
		return
	}

	parent := fmt.Sprintf("projects/%s", testProjectConfig.TestProjectID)
	osconfigClient, err := gcpclients.GetOsConfigClientV1alpha2()
	if err != nil {
		testCase.WriteFailure("Error getting osconfig client: %v", err)
		return
	}

	req := &osconfigpb.ExecutePatchJobRequest{
		Parent:      parent,
		Description: "testing patch job reboot",
		Filter:      fmt.Sprintf("name=\"%s\"", name),
		Duration:    &duration.Duration{Seconds: int64(testSetup.assertTimeout / time.Second)},
		PatchConfig: pc,
	}
	testCase.Logf("Running ExecutePatchJob")
	job, err := osconfigClient.ExecutePatchJob(ctx, req)
	if err != nil {
		testCase.WriteFailure("Error running ExecutePatchJob: %s", utils.GetStatusFromError(err))
		return
	}

	testCase.Logf("Started patch job '%s'", job.GetName())
	pj, err := awaitPatchJob(ctx, job, testSetup.assertTimeout)
	if err != nil {
		testCase.WriteFailure("Patch job '%s' error: %v", job.GetName(), err)
	}

	// If shouldReboot is true that instance should not report a pending reboot.
	if shouldReboot && pj.GetInstanceDetailsSummary().GetInstancesSucceededRebootRequired() > 0 {
		testCase.WriteFailure("PatchJob finished with status InstancesSucceededRebootRequired.")
		return
	}
	// If shouldReboot is false that instance should report a pending reboot.
	if !shouldReboot && pj.GetInstanceDetailsSummary().GetInstancesSucceededRebootRequired() == 0 {
		testCase.WriteFailure("PatchJob should have finished with status InstancesSucceededRebootRequired.")
		return
	}

	testCase.Logf("Checking reboot count")
	attr, err := inst.GetGuestAttributes("osconfig_tests/boot_count")
	if err != nil {
		testCase.WriteFailure("Error retrieving boot count: %v", err)
		return
	}
	if len(attr) == 0 {
		testCase.WriteFailure("Error retrieving boot count: osconfig_tests/boot_count attribute empty")
		return
	}
	num, err := strconv.Atoi(attr[0].Value)
	if err != nil {
		testCase.WriteFailure("Error parsing boot count: %v", err)
		return
	}
	if shouldReboot && num < 2 {
		testCase.WriteFailure("Instance should have booted at least 2 times, boot num: %d.", num)
		return
	}
	if !shouldReboot && num > 1 {
		testCase.WriteFailure("Instance should not have booted more that 1 time, boot num: %d.", num)
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

func patchConfigWithPrePostSteps() *osconfigpb.PatchConfig {
	linuxPreStepConfig := &osconfigpb.ExecStepConfig{Executable: &osconfigpb.ExecStepConfig_LocalPath{LocalPath: "./linux_local_pre_patch_script.sh"}, Interpreter: osconfigpb.ExecStepConfig_SHELL}
	windowsPreStepConfig := &osconfigpb.ExecStepConfig{Executable: &osconfigpb.ExecStepConfig_GcsObject{GcsObject: &osconfigpb.GcsObject{Bucket: "osconfig-agent-end2end-test-resources", Object: "OSPatch/windows_gcs_pre_patch_script.ps1", GenerationNumber: 1571249543230832}}, Interpreter: osconfigpb.ExecStepConfig_POWERSHELL}
	linuxPostStepConfig := &osconfigpb.ExecStepConfig{Executable: &osconfigpb.ExecStepConfig_GcsObject{GcsObject: &osconfigpb.GcsObject{Bucket: "osconfig-agent-end2end-test-resources", Object: "OSPatch/linux_gcs_post_patch_script", GenerationNumber: 1570567792146617}}}
	windowsPostStepConfig := &osconfigpb.ExecStepConfig{Executable: &osconfigpb.ExecStepConfig_LocalPath{LocalPath: "C:\\Windows\\System32\\windows_local_post_patch_script.ps1"}, Interpreter: osconfigpb.ExecStepConfig_POWERSHELL}

	preStep := &osconfigpb.ExecStep{LinuxExecStepConfig: linuxPreStepConfig, WindowsExecStepConfig: windowsPreStepConfig}
	postStep := &osconfigpb.ExecStep{LinuxExecStepConfig: linuxPostStepConfig, WindowsExecStepConfig: windowsPostStepConfig}

	return &osconfigpb.PatchConfig{PreStep: preStep, PostStep: postStep}
}

func validatePrePostStepSuccess(inst *compute.Instance, testCase *junitxml.TestCase) {
	if _, err := inst.WaitForGuestAttributes("osconfig_tests/pre_step_ran", 5*time.Second, 1*time.Minute); err != nil {
		testCase.WriteFailure("error while asserting: %v", err)
		return
	}
	if _, err := inst.WaitForGuestAttributes("osconfig_tests/post_step_ran", 5*time.Second, 1*time.Minute); err != nil {
		testCase.WriteFailure("error while asserting: %v", err)
		return
	}
}
