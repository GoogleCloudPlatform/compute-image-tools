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

// Package testsuite contains e2e tests for gce_windows_upgrade
package testsuite

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/common/gcp"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName              = "WindowsUpgradeTests"
	standardImage2008          = "projects/compute-image-tools-test/global/images/test-image-win2008r2-20200414"
	standardImage2012          = "projects/compute-image-tools-test/global/images/test-image-win2012r2-20220902"
	standardImage2016          = "projects/compute-image-tools-test/global/images/test-image-win2016-20220902"
	standardImage2019          = "projects/compute-image-tools-test/global/images/test-image-win2019-20220902"
	insufficientDiskSpaceImage = "projects/compute-image-tools-test/global/images/test-image-windows-2008r2-no-space"
	byolImage                  = "projects/compute-image-tools-test/global/images/test-image-windows-2008r2-byol"
)

var (
	cmds = map[e2e.CLITestType]string{
		e2e.Wrapper:                       "./gce_windows_upgrade",
		e2e.GcloudBetaProdWrapperLatest:   "gcloud",
		e2e.GcloudBetaLatestWrapperLatest: "gcloud",
	}
)

// TestSuite contains implementations of the e2e tests.
func TestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	testTypes := []e2e.CLITestType{
		e2e.Wrapper,
		e2e.GcloudBetaProdWrapperLatest,
		e2e.GcloudBetaLatestWrapperLatest,
	}

	testsMap := map[e2e.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}

	for _, testType := range testTypes {
		normalCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Normal case"))
		richParamsAndLatestInstallMedia := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Rich params and latest install media"))
		failedAndCleanup := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Failed and cleanup"))
		failedAndRollback := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Failed and rollback"))
		insufficientDiskSpace := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Insufficiant disk space"))
		testBYOL := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "BYOL"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}
		testsMap[testType][normalCase] = runWindowsUpgradeNormalCase
		testsMap[testType][richParamsAndLatestInstallMedia] = runWindowsUpgradeWithRichParamsAndLatestInstallMedia
		testsMap[testType][failedAndCleanup] = runWindowsUpgradeFailedAndCleanup
		testsMap[testType][failedAndRollback] = runWindowsUpgradeFailedAndRollback
		testsMap[testType][insufficientDiskSpace] = runWindowsUpgradeInsufficientDiskSpace
		testsMap[testType][testBYOL] = runWindowsUpgradeBYOL
	}

	// These tests only apply to wrapper since it's testing staging install media.
	stagingUpgrade2008To2012 := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] Staging test 2008 -> 2012", e2e.Wrapper))
	stagingUpgrade2012To2016 := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] Staging test 2012 -> 2016", e2e.Wrapper))
	stagingUpgrade2016To2019 := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] Staging test 2016 -> 2019", e2e.Wrapper))
	stagingUpgrade2019To2022 := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] Staging test 2019 -> 2022", e2e.Wrapper))
	testsMap[e2e.Wrapper][stagingUpgrade2008To2012] = runWindowsUpgradeStaging2008To2012
	testsMap[e2e.Wrapper][stagingUpgrade2012To2016] = runWindowsUpgradeStaging2012To2016
	testsMap[e2e.Wrapper][stagingUpgrade2016To2019] = runWindowsUpgradeStaging2016To2019
	testsMap[e2e.Wrapper][stagingUpgrade2019To2022] = runWindowsUpgradeStaging2019To2022

	if !e2e.GcloudAuth(logger, nil) {
		logger.Printf("Failed to run gcloud auth.")
		testSuite := junitxml.NewTestSuite(testSuiteName)
		testSuite.Failures = 1
		testSuite.Finish(testSuites)
		tswg.Done()
		return
	}

	e2e.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runWindowsUpgradeNormalCase(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-normal-case-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
		},
		e2e.GcloudBetaProdWrapperLatest: {
			"beta", "compute", "os-config", "os-upgrade", "--quiet", "--docker-image-tag=latest",
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-os=%v", "windows-2008r2"),
			fmt.Sprintf("--target-os=%v", "windows-2012r2"),
			instance,
		},
		e2e.GcloudBetaLatestWrapperLatest: {
			"beta", "compute", "os-config", "os-upgrade", "--quiet", "--docker-image-tag=latest",
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-os=%v", "windows-2008r2"),
			fmt.Sprintf("--target-os=%v", "windows-2012r2"),
			instance,
		},
	}
	runTest(ctx, standardImage2008, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		true, false, "", false, 0, false)
}

func runWindowsUpgradeStaging2008To2012(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	runWindowsUpgradeStaging(ctx, testCase, logger,
		testProjectConfig, testType, standardImage2008, "windows-2008r2", "windows-2012r2", "6.3")
}

func runWindowsUpgradeStaging2012To2016(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	runWindowsUpgradeStaging(ctx, testCase, logger,
		testProjectConfig, testType, standardImage2012, "windows-2012r2", "windows-2016", "10.0")
}

func runWindowsUpgradeStaging2016To2019(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	runWindowsUpgradeStaging(ctx, testCase, logger,
		testProjectConfig, testType, standardImage2016, "windows-2016", "windows-2019", "10.0")
}

func runWindowsUpgradeStaging2019To2022(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	runWindowsUpgradeStaging(ctx, testCase, logger,
		testProjectConfig, testType, standardImage2019, "windows-2019", "windows-2022", "10.0")
}

func runWindowsUpgradeStaging(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType, testImage string, sourceOS string, targetOS string,
	expectedVersion string) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-staging-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", sourceOS),
			fmt.Sprintf("-target-os=%v", targetOS),
			fmt.Sprintf("-instance=%v", instance),
			fmt.Sprintf("-use-staging-install-media=true"),
		},
	}
	runStagingTest(ctx, testImage, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase, expectedVersion)
}

func runWindowsUpgradeWithRichParamsAndLatestInstallMedia(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-rich-params-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
			fmt.Sprintf("-create-machine-backup=false"),
			fmt.Sprintf("-auto-rollback"),
			fmt.Sprintf("-timeout=2h"),
			fmt.Sprintf("-project=%v", "compute-image-test-pool-002"),
			fmt.Sprintf("-zone=%v", "fake-zone"),
			"-use-staging-install-media",
		},
		e2e.GcloudBetaProdWrapperLatest: {
			"beta", "compute", "os-config", "os-upgrade", "--quiet", "--docker-image-tag=latest",
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-os=%v", "windows-2008r2"),
			fmt.Sprintf("--target-os=%v", "windows-2012r2"),
			fmt.Sprintf("--no-create-machine-backup"),
			fmt.Sprintf("--auto-rollback"),
			fmt.Sprintf("--timeout=2h"),
			fmt.Sprintf("--zone=%v", "us-east1-b"),
			"--use-staging-install-media",
			instance,
		},
		e2e.GcloudBetaLatestWrapperLatest: {
			"beta", "compute", "os-config", "os-upgrade", "--quiet", "--docker-image-tag=latest",
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-os=%v", "windows-2008r2"),
			fmt.Sprintf("--target-os=%v", "windows-2012r2"),
			fmt.Sprintf("--no-create-machine-backup"),
			fmt.Sprintf("--auto-rollback"),
			fmt.Sprintf("--timeout=2h"),
			fmt.Sprintf("--zone=%v", "us-east1-b"),
			"--use-staging-install-media",
			instance,
		},
	}
	runTest(ctx, standardImage2008, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		true, false, "original", true, 2, false)
}

// this test is cli only, since gcloud can't accept ctrl+c and cleanup
func runWindowsUpgradeFailedAndCleanup(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-failed-and-cleanup-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
		},
	}
	runTest(ctx, standardImage2008, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		false, true, "", false, 0, false)
}

// this test is cli only, since gcloud can't accept ctrl+c and cleanup
func runWindowsUpgradeFailedAndRollback(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-failed-and-rollback-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
			fmt.Sprintf("-create-machine-backup=false"),
			fmt.Sprintf("-auto-rollback"),
		},
	}
	runTest(ctx, standardImage2008, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		false, true, "original-backup", true, 2, false)
}

func runWindowsUpgradeInsufficientDiskSpace(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-insufficient-disk-space-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
			fmt.Sprintf("-auto-rollback"),
		},
		e2e.GcloudBetaProdWrapperLatest: {
			"beta", "compute", "os-config", "os-upgrade", "--quiet", "--docker-image-tag=latest",
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-os=%v", "windows-2008r2"),
			fmt.Sprintf("--target-os=%v", "windows-2012r2"),
			fmt.Sprintf("--auto-rollback"),
			instance,
		},
		e2e.GcloudBetaLatestWrapperLatest: {
			"beta", "compute", "os-config", "os-upgrade", "--quiet", "--docker-image-tag=latest",
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-os=%v", "windows-2008r2"),
			fmt.Sprintf("--target-os=%v", "windows-2012r2"),
			fmt.Sprintf("--auto-rollback"),
			instance,
		},
	}
	runTest(ctx, insufficientDiskSpaceImage, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		false, false, "original", true, 0, false)
}

func runWindowsUpgradeBYOL(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-byol-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
			fmt.Sprintf("-create-machine-backup=false"),
		},
		e2e.GcloudBetaProdWrapperLatest: {
			"beta", "compute", "os-config", "os-upgrade", "--quiet", "--docker-image-tag=latest",
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-os=%v", "windows-2008r2"),
			fmt.Sprintf("--target-os=%v", "windows-2012r2"),
			fmt.Sprintf("--no-create-machine-backup"),
			instance,
		},
		e2e.GcloudBetaLatestWrapperLatest: {
			"beta", "compute", "os-config", "os-upgrade", "--quiet", "--docker-image-tag=latest",
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-os=%v", "windows-2008r2"),
			fmt.Sprintf("--target-os=%v", "windows-2012r2"),
			fmt.Sprintf("--no-create-machine-backup"),
			instance,
		},
	}
	runTest(ctx, byolImage, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		false, false, "", false, 0, true)
}

func runTest(ctx context.Context, image string, args []string, testType e2e.CLITestType,
	testProjectConfig *testconfig.Project, instanceName string, logger *log.Logger, testCase *junitxml.TestCase,
	expectSuccess bool, triggerFailure bool, expectedScriptURL string, autoRollback bool, dataDiskCount int, expectValidationFailure bool) {

	if args == nil {
		return
	}
	cmd, ok := cmds[testType]
	if !ok {
		return
	}

	// create the test instance
	if !e2e.RunTestCommand("gcloud", []string{
		"compute", "instances", "create", fmt.Sprintf("--image=%v", image),
		"--boot-disk-type=pd-ssd", "--machine-type=n1-standard-4", fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID), instanceName,
	}, logger, testCase) {
		return
	}

	defer cleanupTestInstance(testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName, logger, testCase)

	// create and attach data disks
	for dataDiskIndex := 1; dataDiskIndex <= dataDiskCount; dataDiskIndex++ {
		diskName := fmt.Sprintf("%v-%v", instanceName, dataDiskIndex)
		if !e2e.RunTestCommand("gcloud", []string{
			"compute", "disks", "create", fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID), "--size=10gb",
			"--image=projects/compute-image-tools-test/global/images/empty-ntfs-10g",
			diskName,
		}, logger, testCase) {
			return
		}

		if !e2e.RunTestCommand("gcloud", []string{
			"compute", "instances", "attach-disk", instanceName, fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--disk=%v", diskName),
		}, logger, testCase) {
			return
		}
	}

	// set original startup script to metadata
	if expectedScriptURL != "" {
		key := "windows-startup-script-url"
		if expectedScriptURL == "original-backup" {
			key = "windows-startup-script-url-backup"
		}
		_, err := gcp.SetMetadata(ctx, testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName,
			key, expectedScriptURL, true)
		if err != nil {
			e2e.Failure(testCase, logger, fmt.Sprintf("Failed to set metadata for %v: %v", instanceName, err))
			return
		}
	}

	var success bool
	if testType == e2e.Wrapper {
		cmd := e2e.RunTestCommandAsync(cmd, args, logger, testCase)

		go func() {
			// send an INT signal to fail the upgrade
			if triggerFailure {
				// wait for "preparation" to finish
				instance, err := gcp.CreateInstanceObject(ctx, testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName, true)
				if err != nil {
					e2e.Failure(testCase, logger, fmt.Sprintf("Failed to fetch instance object for %v: %v", instanceName, err))
					return
				}
				expectedOutput := "Beginning upgrade startup script."
				logger.Printf("[%v] Waiting for `%v` in instance serial console.", instanceName,
					expectedOutput)
				if err := instance.WaitForSerialOutput(
					expectedOutput, nil, 1, 15*time.Second, 20*time.Minute); err != nil {
					testCase.WriteFailure("Error during waiting for preparation finished: %v", err)
					return
				}

				err = cmd.Process.Signal(os.Interrupt)
				if err != nil {
					e2e.Failure(testCase, logger, fmt.Sprintf("Failed to send ctrl+c to upgrade %v: %v", instanceName, err))
					return
				}
			}
		}()

		err := cmd.Wait()
		if err != nil {
			success = false
		} else {
			success = true
		}
	} else {
		isLatestGcloud := testType == e2e.GcloudBetaLatestWrapperLatest
		if !e2e.GcloudUpdate(logger, testCase, isLatestGcloud) {
			success = false
		} else {
			success = e2e.RunTestCommandIgnoringError(cmd, args, logger, testCase)
		}
	}

	verifyUpgradedInstance(ctx, logger, testCase, testProjectConfig, instanceName, success,
		expectSuccess, expectedScriptURL, autoRollback, dataDiskCount, expectValidationFailure)
}

func runStagingTest(ctx context.Context, image string, args []string, testType e2e.CLITestType,
	testProjectConfig *testconfig.Project, instanceName string, logger *log.Logger, testCase *junitxml.TestCase, expectedVersion string) {

	if args == nil {
		return
	}
	cmd, ok := cmds[testType]
	if !ok {
		return
	}

	// create the test instance
	if !e2e.RunTestCommand("gcloud", []string{
		"compute", "instances", "create", fmt.Sprintf("--image=%v", image),
		"--boot-disk-type=pd-ssd", "--machine-type=n1-standard-4", fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID), instanceName,
	}, logger, testCase) {
		return
	}

	defer cleanupTestInstance(testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName, logger, testCase)

	var success bool
	if testType == e2e.Wrapper {
		cmd := e2e.RunTestCommandAsync(cmd, args, logger, testCase)

		err := cmd.Wait()
		if err != nil {
			success = false
		} else {
			success = true
		}
	}

	if success != true {
		e2e.Failure(testCase, logger, fmt.Sprintf("Actual success: %v, expect success: true", success))
		return
	}

	instance, err := gcp.CreateInstanceObject(ctx, testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName, true)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Failed to fetch instance object for %v: %v", instanceName, err))
		return
	}
	verifyOSVersion(instance, testCase, instanceName, logger, expectedVersion)
}

func verifyUpgradedInstance(ctx context.Context, logger *log.Logger, testCase *junitxml.TestCase,
	testProjectConfig *testconfig.Project, instanceName string, success bool, expectSuccess bool,
	expectedScriptURL string, autoRollback bool, dataDiskCount int, expectValidationFailure bool) {

	if success != expectSuccess {
		e2e.Failure(testCase, logger, fmt.Sprintf("Actual success: %v, expect success: %v", success, expectSuccess))
		return
	}

	instance, err := gcp.CreateInstanceObject(ctx, testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName, true)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Failed to fetch instance object for %v: %v", instanceName, err))
		return
	}

	logger.Printf("Verifying upgraded instance `%v`...", instanceName)

	if !verifyLicensesAndDisks(instance, expectValidationFailure, logger, testCase, expectSuccess, autoRollback, dataDiskCount) {
		return
	}

	if expectSuccess {
		if !verifyOSVersion(instance, testCase, instanceName, logger, "6.3") {
			return
		}
	} else {
		verifyRollback(autoRollback, instance, testProjectConfig, instanceName, testCase, logger)
	}

	verifyCleanup(instance, testCase, logger, expectedScriptURL)
}

func verifyLicensesAndDisks(instance *gcp.Instance, expectValidationFailure bool, logger *log.Logger,
	testCase *junitxml.TestCase, expectSuccess bool, autoRollback bool, dataDiskCount int) bool {

	hasBootDisk := false
	for _, disk := range instance.Disks {
		if !disk.Boot {
			continue
		}

		if !expectValidationFailure {
			logger.Printf("Existing licenses: %v", disk.Licenses)
			if !e2e.ContainsSubString(disk.Licenses, "projects/windows-cloud/global/licenses/windows-server-2008-r2-dc") {
				e2e.Failure(testCase, logger, "Original 2008r2 license not found.")
			}
			containsAdditionalLicense := e2e.ContainsSubString(disk.Licenses, "projects/windows-cloud/global/licenses/windows-server-2012-r2-dc-in-place-upgrade")
			// success case
			if expectSuccess {
				if !containsAdditionalLicense {
					e2e.Failure(testCase, logger, "Additional license not found.")
				}
			} else {
				if autoRollback {
					// rollback case
					if containsAdditionalLicense {
						e2e.Failure(testCase, logger, "Additional license found after rollback.")
					}
				} else {
					// cleanup case
					if !containsAdditionalLicense {
						e2e.Failure(testCase, logger, "Additional license not found.")
					}
				}
			}
		}

		hasBootDisk = true
	}
	if !hasBootDisk {
		e2e.Failure(testCase, logger, "Boot disk not found.")
		return false
	}
	if len(instance.Disks) != dataDiskCount+1 {
		e2e.Failure(testCase, logger, fmt.Sprintf("Count of disks not match: expect %v, actual %v.", dataDiskCount+1, len(instance.Disks)))
	}
	return true
}

func verifyOSVersion(instance *gcp.Instance, testCase *junitxml.TestCase, instanceName string, logger *log.Logger, expectedVersion string) bool {
	err := instance.RestartWithScriptCode("$ver=[System.Environment]::OSVersion.Version\n" +
		"Write-Host \"windows_upgrade_verify_version=$($ver.Major).$($ver.Minor)\"")
	if err != nil {
		testCase.WriteFailure("Error starting instance `%v` with script: `%v`", instanceName, err)
		return false
	}
	expectedOutput := fmt.Sprintf("windows_upgrade_verify_version=%v", expectedVersion)
	logger.Printf("[%v] Waiting for `%v` in instance serial console.", instanceName,
		expectedOutput)
	if err := instance.WaitForSerialOutput(
		expectedOutput, nil, 1, 15*time.Second, 15*time.Minute); err != nil {
		testCase.WriteFailure("Error during validation: %v", err)
	}
	return true
}

func verifyRollback(autoRollback bool, instance *gcp.Instance, testProjectConfig *testconfig.Project, instanceName string, testCase *junitxml.TestCase, logger *log.Logger) {
	if autoRollback {
		// original boot disk name == instance name by default
		originalOSDisk, err := instance.Client.GetDisk(testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)
		if err != nil {
			e2e.Failure(testCase, logger, "Failed to get original OS disk.")
		}
		if len(originalOSDisk.Users) == 0 {
			e2e.Failure(testCase, logger, "Original OS disk didn't get rollback.")
		}
	}
}

// verify cleanup: install media, startup script & backup
func verifyCleanup(instance *gcp.Instance, testCase *junitxml.TestCase, logger *log.Logger, expectedScriptURL string) {
	for _, d := range instance.Disks {
		if strings.HasSuffix(d.Source, "global/images/family/windows-install-media") {
			e2e.Failure(testCase, logger, "Install media is not cleaned up.")
		}
	}
	var windowsStartupScriptURL string
	var windowsStartupScriptURLBackup string
	for _, i := range instance.Metadata.Items {
		if i.Key == "windows-startup-script-url" && i.Value != nil {
			windowsStartupScriptURL = *i.Value
		} else if i.Key == "windows-startup-script-url-backup" && i.Value != nil {
			windowsStartupScriptURLBackup = *i.Value
		}
	}
	if expectedScriptURL != windowsStartupScriptURL {
		e2e.Failure(testCase, logger, fmt.Sprintf("Unexpected startup script URL: %v", windowsStartupScriptURL))
	}
	if windowsStartupScriptURLBackup != "" {
		e2e.Failure(testCase, logger, fmt.Sprintf("Unexpected startup script URL backup: %v", windowsStartupScriptURLBackup))
	}
}

func cleanupTestInstance(project, zone, instanceName string, logger *log.Logger, testCase *junitxml.TestCase) {
	// Run gcloud to delete the instance, ignoring error.
	e2e.RunCliTool(logger, testCase, "gcloud", []string{
		"compute", "instances", "delete", "--quiet",
		fmt.Sprintf("--zone=%v", zone),
		fmt.Sprintf("--project=%v", project), instanceName,
	})
}
