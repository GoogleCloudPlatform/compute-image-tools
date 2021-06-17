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

// Package ovfmachineimageimporttestsuite contains e2e tests for machine image
// import cli tools
package ovfmachineimageimporttestsuite

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	ovfimporttestsuite "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e/gce_ovf_import/test_suites"
	"github.com/GoogleCloudPlatform/compute-image-tools/common/gcp"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "OVFMachineImageImportTests"
	ovaBucket     = "compute-image-tools-test-resources"
)

var (
	cmds = map[e2e.CLITestType]string{
		e2e.Wrapper:                       "./gce_ovf_import",
		e2e.GcloudBetaProdWrapperLatest:   "gcloud",
		e2e.GcloudBetaLatestWrapperLatest: "gcloud",
		e2e.GcloudGaLatestWrapperRelease:  "gcloud",
	}

	// Apply this as instance metadata if the OS config agent is not
	// supported for the platform or version being imported.
	skipOSConfigMetadata = map[string]string{"osconfig_not_supported": "true"}

	// argMap stores test args from e2e test CLI.
	argMap map[string]string
)

type ovfMachineImageImportTestProperties struct {
	machineImageName string
	storageLocation  string
	ovfimporttestsuite.OvfImportTestProperties
}

// TestSuite is image import test suite.
func TestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project, argMapInput map[string]string) {

	argMap = argMapInput

	testsMap := map[e2e.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}

	testTypes := []e2e.CLITestType{
		e2e.Wrapper,
		e2e.GcloudBetaProdWrapperLatest,
		e2e.GcloudBetaLatestWrapperLatest,
	}
	for _, testType := range testTypes {
		machineImageImportUbuntu3DisksNetworkSettingsNameTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Ubuntu 3 disks, one data disk larger than 10GB, Network setting (name only)"))
		machineImageImportWindows2012R2TwoDisksNetworkSettingsPathTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Windows 2012 R2 two disks, Network setting (path)"))
		machineImageImportStorageLocationTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Centos 7.4, Storage location"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}
		testsMap[testType][machineImageImportUbuntu3DisksNetworkSettingsNameTestCase] = runOVFMachineImageImportUbuntu3DisksNetworkSettingsName
		testsMap[testType][machineImageImportWindows2012R2TwoDisksNetworkSettingsPathTestCase] = runOVFMachineImageImportWindows2012R2TwoDisksNetworkSettingsPath
		testsMap[testType][machineImageImportStorageLocationTestCase] = runOVFMachineImageImportCentos74StorageLocation
	}

	// gcloud only tests
	machineImageImportDisabledDefaultServiceAccountSuccessTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "Machine image import without default service account, success by specifying a custom Compute service account"))
	machineImageImportDefaultServiceAccountWithMissingPermissionsSuccessTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "Machine image import without permission on default service account, success by specifying a custom Compute service account"))
	machineImageImportDisabledDefaultServiceAccountFailTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "Machine image import without default service account failed"))
	machineImageImportDefaultServiceAccountWithMissingPermissionsFailTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "Machine image import without permission on default service account failed"))
	machineImageImportDefaultServiceAccountCustomAccessScopeTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "Machine image import with default service account custom access scopes set"))
	testsMap[e2e.Wrapper][machineImageImportDisabledDefaultServiceAccountSuccessTestCase] = runMachineImageImportDisabledDefaultServiceAccountSuccessTest
	testsMap[e2e.Wrapper][machineImageImportDefaultServiceAccountWithMissingPermissionsSuccessTestCase] = runMachineImageImportOSDefaultServiceAccountWithMissingPermissionsSuccessTest
	testsMap[e2e.Wrapper][machineImageImportDisabledDefaultServiceAccountFailTestCase] = runMachineImageImportWithDisabledDefaultServiceAccountFailTest
	testsMap[e2e.Wrapper][machineImageImportDefaultServiceAccountWithMissingPermissionsFailTestCase] = runMachineImageImportDefaultServiceAccountWithMissingPermissionsFailTest
	testsMap[e2e.Wrapper][machineImageImportDefaultServiceAccountCustomAccessScopeTestCase] = runMachineImageImportDefaultServiceAccountCustomAccessScope

	e2e.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runOVFMachineImageImportUbuntu3DisksNetworkSettingsName(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-machine-image-ubuntu-3-disks-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{
			VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
				"scripts/ovf_import_test_ubuntu_3_disks.sh", logger),
			Zone:                  testProjectConfig.TestZone,
			ExpectedStartupOutput: "All tests passed!",
			FailureMatches:        []string{"TestFailed:"},
			SourceURI:             fmt.Sprintf("gs://%v/ova/ubuntu-1604-three-disks", ovaBucket),
			Os:                    "ubuntu-1604",
			InstanceMetadata:      skipOSConfigMetadata,
			MachineType:           "n1-standard-4",
			Network:               fmt.Sprintf("%v-vpc-1", testProjectConfig.TestProjectID),
			Subnet:                fmt.Sprintf("%v-subnet-1", testProjectConfig.TestProjectID),
			Tags:                  []string{"tag1", "tag2", "tag3"},
		},
	}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFMachineImageImportWindows2012R2TwoDisksNetworkSettingsPath(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	region, _ := paramhelper.GetRegion(testProjectConfig.TestZone)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-machine-image-w2k12-r2-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"scripts/ovf_import_test_windows_two_disks.ps1", logger),
			Zone:                  testProjectConfig.TestZone,
			ExpectedStartupOutput: "All Tests Passed",
			FailureMatches:        []string{"Test Failed:"},
			SourceURI:             fmt.Sprintf("gs://%v/ova/w2k12-r2", ovaBucket),
			Os:                    "windows-2012r2",
			MachineType:           "n1-standard-8",
			IsWindows:             true,
			Network:               fmt.Sprintf("global/networks/%v-vpc-1", testProjectConfig.TestProjectID),
			Subnet:                fmt.Sprintf("projects/%v/regions/%v/subnetworks/%v-subnet-1", testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			Tags:                  []string{"tag1", "tag2", "tag3"},
		}}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFMachineImageImportCentos74StorageLocation(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-gmi-storage-location-%v", suffix),
		storageLocation:  "asia-east2",
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{
			VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
				"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                  "asia-east2-a",
			ExpectedStartupOutput: "All tests passed!",
			FailureMatches:        []string{"FAILED:", "TestFailed:"},
			SourceURI:             fmt.Sprintf("gs://%v-asia/ova/centos-7.4/", ovaBucket),
			Os:                    "centos-7",
			MachineType:           "n2-standard-2",
		}}
	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runMachineImageImportDisabledDefaultServiceAccountSuccessTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, true)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}
	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-without-service-account-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                   "europe-west1-c",
			ExpectedStartupOutput:  "All tests passed!",
			FailureMatches:         []string{"FAILED:", "TestFailed:"},
			SourceURI:              fmt.Sprintf("gs://%v-eu/ova/centos-7.4/", ovaBucket),
			Os:                     "centos-7",
			MachineType:            "n1-standard-4",
			Project:                testVariables.ProjectID,
			ComputeServiceAccount:  testVariables.ComputeServiceAccount,
			InstanceServiceAccount: testVariables.InstanceServiceAccount,
			InstanceAccessScopes:   "https://www.googleapis.com/auth/compute,https://www.googleapis.com/auth/datastore",
		}}
	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

// With insufficient permissions on default service account, import success by specifying a custom account.
func runMachineImageImportOSDefaultServiceAccountWithMissingPermissionsSuccessTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, false)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}
	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-missing-cse-permissions-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                   "us-west1-c",
			ExpectedStartupOutput:  "All tests passed!",
			FailureMatches:         []string{"FAILED:", "TestFailed:"},
			SourceURI:              fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
			Os:                     "centos-7",
			MachineType:            "n1-standard-4",
			Project:                testVariables.ProjectID,
			ComputeServiceAccount:  testVariables.ComputeServiceAccount,
			InstanceServiceAccount: testVariables.InstanceServiceAccount,
		}}
	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

// With insufficient permissions on default service account, import failed.
func runMachineImageImportWithDisabledDefaultServiceAccountFailTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, true)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}
	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-missing-permn-on-default-csa-fail-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                  testProjectConfig.TestZone,
			ExpectedStartupOutput: "All tests passed!",
			FailureMatches:        []string{"FAILED:", "TestFailed:"},
			SourceURI:             fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
			Os:                    "centos-7",
			MachineType:           "n1-standard-4",
			Project:               testVariables.ProjectID,
		}}
	e2e.RunTestCommandAssertErrorMessage(cmds[testType], buildTestArgs(props, testProjectConfig)[testType], "Failed to download GCS path", logger, testCase)
}

// With insufficient permissions on default service account, import failed.
func runMachineImageImportDefaultServiceAccountWithMissingPermissionsFailTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, false)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}
	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-insufficient-perm-default-csa-fail-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                  testProjectConfig.TestZone,
			ExpectedStartupOutput: "All tests passed!",
			FailureMatches:        []string{"FAILED:", "TestFailed:"},
			SourceURI:             fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
			Os:                    "centos-7",
			MachineType:           "n1-standard-4",
			Project:               testVariables.ProjectID,
		}}
	e2e.RunTestCommandAssertErrorMessage(cmds[testType], buildTestArgs(props, testProjectConfig)[testType], "Failed to download GCS path", logger, testCase)
}

// Ensure custom access scopes are set on the machine image even when default service account is used
func runMachineImageImportDefaultServiceAccountCustomAccessScope(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, false)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}
	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-custom-sc-default-sa-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                  "asia-southeast1-c",
			ExpectedStartupOutput: "All tests passed!",
			FailureMatches:        []string{"FAILED:", "TestFailed:"},
			SourceURI:             fmt.Sprintf("gs://%v-asia/ova/centos-7.4/", ovaBucket),
			Os:                    "centos-7",
			MachineType:           "n1-standard-4",
			Project:               testVariables.ProjectID,
			InstanceAccessScopes:  "https://www.googleapis.com/auth/compute,https://www.googleapis.com/auth/datastore",
		}}
	e2e.RunTestCommandAssertErrorMessage(cmds[testType], buildTestArgs(props, testProjectConfig)[testType], "Failed to download GCS path", logger, testCase)
}

func buildTestArgs(props *ovfMachineImageImportTestProperties, testProjectConfig *testconfig.Project) map[e2e.CLITestType][]string {
	gcloudBetaArgs := []string{
		"beta", "compute", "machine-images", "import", props.machineImageName, "--quiet",
		"--docker-image-tag=latest"}
	gcloudArgs := []string{"compute", "machine-images", "import", props.machineImageName, "--quiet"}
	wrapperArgs := []string{
		"-client-id=e2e",
		fmt.Sprintf("-machine-image-name=%s", props.machineImageName),
	}
	if props.storageLocation != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--storage-location=%v", props.storageLocation))
		gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--storage-location=%v", props.storageLocation))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-machine-image-storage-location=%v", props.storageLocation))
	}
	return ovfimporttestsuite.BuildArgsMap(&props.OvfImportTestProperties, testProjectConfig, gcloudBetaArgs, gcloudArgs, wrapperArgs)
}

func runOVFMachineImageImportTest(ctx context.Context, args []string, testType e2e.CLITestType,
	testProjectConfig *testconfig.Project, logger *log.Logger, testCase *junitxml.TestCase,
	props *ovfMachineImageImportTestProperties) {

	if e2e.RunTestForTestType(cmds[testType], args, testType, logger, testCase) {
		verifyImportedMachineImage(ctx, testCase, testProjectConfig, logger, props)
	}
}

func verifyImportedMachineImage(
	ctx context.Context, testCase *junitxml.TestCase, testProjectConfig *testconfig.Project,
	logger *log.Logger, props *ovfMachineImageImportTestProperties) {

	project := ovfimporttestsuite.GetProject(&props.OvfImportTestProperties, testProjectConfig)
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Error creating client: %v", err))
		return
	}

	logger.Printf("Verifying imported machine image...")
	testInstanceName := props.machineImageName + "-test-instance"
	logger.Printf("Creating `%v` test instance.", testInstanceName)

	// Verify storage location
	if props.storageLocation != "" {
		gmi, gmiErr := gcp.CreateMachineImageObject(ctx, project, props.machineImageName)
		if gmiErr != nil {
			e2e.Failure(testCase, logger, fmt.Sprintf("Error when loading machine image '%v': %v", props.machineImageName, err))
			return
		}
		storageLocationMatch := false
		for _, storageLocation := range gmi.StorageLocations {
			storageLocationMatch = storageLocationMatch || strings.Contains(storageLocation, props.storageLocation)
		}
		if !storageLocationMatch {
			e2e.Failure(testCase, logger,
				fmt.Sprintf("Machine image storage locations (%v) do not contain storage location from the flag:: %v",
					strings.Join(gmi.StorageLocations, ","), props.storageLocation))
			return
		}
	}

	instance, err := gcp.CreateInstanceBeta(
		ctx, project, props.Zone, testInstanceName, props.IsWindows, props.machineImageName)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Error when creating test instance `%v` from machine image '%v': %v", testInstanceName, props.machineImageName, err))
		return
	}

	// Clean-up
	defer func() {
		logger.Printf("Deleting instance `%v`", testInstanceName)
		if err := instance.Cleanup(); err != nil {
			logger.Printf("Instance '%v' failed to clean up: %v", testInstanceName, err)
		} else {
			logger.Printf("Instance '%v' cleaned up.", testInstanceName)
		}
		logger.Printf("Deleting machine image `%v`", props.machineImageName)
		if err := client.DeleteMachineImage(project, props.machineImageName); err != nil {
			logger.Printf("Machine image '%v' failed to clean up: %v", props.machineImageName, err)
		} else {
			logger.Printf("Machine image '%v' cleaned up.", props.machineImageName)
		}
	}()

	ovfimporttestsuite.VerifyInstance(instance, client, testCase, project, logger, &props.OvfImportTestProperties)
}
