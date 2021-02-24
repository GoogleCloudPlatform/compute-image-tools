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
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

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
	machineImageName          string
	isWindows                 bool
	expectedStartupOutput     string
	failureMatches            []string
	verificationStartupScript string
	zone                      string
	sourceURI                 string
	os                        string
	machineType               string
	network                   string
	subnet                    string
	storageLocation           string
	instanceMetadata          map[string]string
	project                   string
	computeServiceAccount     string
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
		machineImageImportUbuntu3DisksTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFMachineImageImport] %v", testType, "Ubuntu 3 disks, one data disk larger than 10GB"))
		machineImageImportWindows2012R2TwoDisks := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFMachineImageImport] %v", testType, "Windows 2012 R2 two disks"))
		machineImageImportNetworkSettingsName := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFMachineImageImport] %v", testType, "Network setting (name only)"))
		machineImageImportNetworkSettingsPath := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFMachineImageImport] %v", testType, "Network setting (path)"))
		machineImageImportStorageLocation := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFMachineImageImport] %v", testType, "Storage location"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}
		testsMap[testType][machineImageImportUbuntu3DisksTestCase] = runOVFMachineImageImportUbuntu3Disks
		testsMap[testType][machineImageImportWindows2012R2TwoDisks] = runOVFMachineImageImportWindows2012R2TwoDisks
		testsMap[testType][machineImageImportNetworkSettingsName] = runOVFMachineImageImportNetworkSettingsName
		testsMap[testType][machineImageImportNetworkSettingsPath] = runOVFMachineImageImportNetworkSettingsPath
		testsMap[testType][machineImageImportStorageLocation] = runOVFMachineImageImportStorageLocation
	}

	// Only test service account scenario for wrapper, till gcloud supports it.
	machineImageImportDisabledDefaultServiceAccountSuccessTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v][CLI] %v", e2e.Wrapper, "Machine image import without default service account, success by specifying a custom Compute service account"))
	machineImageImportDefaultServiceAccountWithMissingPermissionsSuccessTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v][CLI] %v", e2e.Wrapper, "Machine image import without permission on default service account, success by specifying a custom Compute service account"))
	machineImageImportDisabledDefaultServiceAccountFailTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v][CLI] %v", e2e.Wrapper, "Machine image import without default service account failed"))
	machineImageImportDefaultServiceAccountWithMissingPermissionsFailTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v][CLI] %v", e2e.Wrapper, "Machine image import without permission on default service account failed"))
	testsMap[e2e.Wrapper][machineImageImportDisabledDefaultServiceAccountSuccessTestCase] = runMachineImageImportDisabledDefaultServiceAccountSuccessTest
	testsMap[e2e.Wrapper][machineImageImportDefaultServiceAccountWithMissingPermissionsSuccessTestCase] = runMachineImageImportOSDefaultServiceAccountWithMissingPermissionsSuccessTest
	testsMap[e2e.Wrapper][machineImageImportDisabledDefaultServiceAccountFailTestCase] = runMachineImageImportWithDisabledDefaultServiceAccountFailTest
	testsMap[e2e.Wrapper][machineImageImportDefaultServiceAccountWithMissingPermissionsFailTestCase] = runMachineImageImportDefaultServiceAccountWithMissingPermissionsFailTest

	e2e.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runOVFMachineImageImportUbuntu3Disks(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-machine-image-ubuntu-3-disks-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"scripts/ovf_import_test_ubuntu_3_disks.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		failureMatches:        []string{"TestFailed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/ubuntu-1604-three-disks", ovaBucket),
		os:                    "ubuntu-1604",
		instanceMetadata:      skipOSConfigMetadata,
		machineType:           "n1-standard-4"}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFMachineImageImportWindows2012R2TwoDisks(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-machine-image-w2k12-r2-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"scripts/ovf_import_test_windows_two_disks.ps1", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All Tests Passed",
		failureMatches:        []string{"Test Failed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/w2k12-r2", ovaBucket),
		os:                    "windows-2012r2",
		machineType:           "n1-standard-8",
		isWindows:             true,
	}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFMachineImageImportStorageLocation(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-machine-image-w2k16-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.ps1", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All Tests Passed",
		failureMatches:        []string{"Test Failed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/w2k16/w2k16.ovf", ovaBucket),
		os:                    "windows-2016",
		machineType:           "n2-standard-2",
		isWindows:             true,
		storageLocation:       "us-west2",
	}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFMachineImageImportNetworkSettingsName(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-network-name-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		failureMatches:        []string{"FAILED:", "TestFailed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
		os:                    "centos-7",
		machineType:           "n1-standard-4",
		network:               fmt.Sprintf("%v-vpc-1", testProjectConfig.TestProjectID),
		subnet:                fmt.Sprintf("%v-subnet-1", testProjectConfig.TestProjectID),
	}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFMachineImageImportNetworkSettingsPath(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	region, _ := paramhelper.GetRegion(testProjectConfig.TestZone)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-network-path-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		failureMatches:        []string{"FAILED:", "TestFailed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
		os:                    "centos-7",
		machineType:           "n1-standard-4",
		network:               fmt.Sprintf("global/networks/%v-vpc-1", testProjectConfig.TestProjectID),
		subnet:                fmt.Sprintf("projects/%v/regions/%v/subnetworks/%v-subnet-1", testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
	}

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
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		failureMatches:        []string{"FAILED:", "TestFailed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
		os:                    "centos-7",
		machineType:           "n1-standard-4",
		project:               testVariables.ProjectID,
		computeServiceAccount: testVariables.ComputeServiceAccount,
	}
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
		machineImageName: fmt.Sprintf("test-missing-ce-permissions-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		failureMatches:        []string{"FAILED:", "TestFailed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
		os:                    "centos-7",
		machineType:           "n1-standard-4",
		project:               testVariables.ProjectID,
		computeServiceAccount: testVariables.ComputeServiceAccount,
	}
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
		machineImageName: fmt.Sprintf("test-missing-permission-on-default-csa-fail-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		failureMatches:        []string{"FAILED:", "TestFailed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
		os:                    "centos-7",
		machineType:           "n1-standard-4",
		project:               testVariables.ProjectID,
	}
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
		machineImageName: fmt.Sprintf("test-insufficient-permission-default-csa-fail-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		failureMatches:        []string{"FAILED:", "TestFailed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
		os:                    "centos-7",
		machineType:           "n1-standard-4",
		project:               testVariables.ProjectID,
	}
	e2e.RunTestCommandAssertErrorMessage(cmds[testType], buildTestArgs(props, testProjectConfig)[testType], "Failed to download GCS path", logger, testCase)
}

func getProject(props *ovfMachineImageImportTestProperties, testProjectConfig *testconfig.Project) string {
	if props.project != "" {
		return props.project
	}
	return testProjectConfig.TestProjectID
}

func buildTestArgs(props *ovfMachineImageImportTestProperties, testProjectConfig *testconfig.Project) map[e2e.CLITestType][]string {
	project := getProject(props, testProjectConfig)
	gcloudBetaArgs := []string{
		"beta", "compute", "machine-images", "import", props.machineImageName, "--quiet",
		"--docker-image-tag=latest",
		fmt.Sprintf("--project=%v", project),
		fmt.Sprintf("--source-uri=%v", props.sourceURI),
		fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
	}
	gcloudArgs := []string{
		"compute", "machine-images", "import", props.machineImageName, "--quiet",
		fmt.Sprintf("--project=%v", project),
		fmt.Sprintf("--source-uri=%v", props.sourceURI),
		fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
	}
	wrapperArgs := []string{"-client-id=e2e", fmt.Sprintf("-project=%v", project),
		fmt.Sprintf("-machine-image-name=%s", props.machineImageName),
		fmt.Sprintf("-ovf-gcs-path=%v", props.sourceURI),
		fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		fmt.Sprintf("-build-id=%v", path.RandString(10)),
	}

	if props.os != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--os=%v", props.os))
		gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--os=%v", props.os))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-os=%v", props.os))
	}
	if props.machineType != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--machine-type=%v", props.machineType))
		gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--machine-type=%v", props.machineType))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-machine-type=%v", props.machineType))
	}
	if props.network != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--network=%v", props.network))
		gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--network=%v", props.network))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-network=%v", props.network))
	}
	if props.subnet != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--subnet=%v", props.subnet))
		gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--subnet=%v", props.subnet))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-subnet=%v", props.subnet))
	}
	if props.storageLocation != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--storage-location=%v", props.storageLocation))
		gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--storage-location=%v", props.storageLocation))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-machine-image-storage-location=%v", props.storageLocation))
	}
	if props.computeServiceAccount != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--compute-service-account=%v", props.computeServiceAccount))
		gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--compute-service-account=%v", props.computeServiceAccount))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-compute-service-account=%v", props.computeServiceAccount))
	}

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper:                       wrapperArgs,
		e2e.GcloudBetaProdWrapperLatest:   gcloudBetaArgs,
		e2e.GcloudBetaLatestWrapperLatest: gcloudBetaArgs,
		e2e.GcloudGaLatestWrapperRelease:  gcloudArgs,
	}
	return argsMap
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

	project := getProject(props, testProjectConfig)
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Error creating client: %v", err))
		return
	}

	logger.Printf("Verifying imported machine image...")
	testInstanceName := props.machineImageName + "-test-instance"
	logger.Printf("Creating `%v` test instance.", testInstanceName)
	computeServiceAccount := "default"
	if props.computeServiceAccount != "" {
		computeServiceAccount = props.computeServiceAccount
	}
	instance, err := gcp.CreateInstanceBeta(
		ctx, project, props.zone, testInstanceName, props.isWindows, props.machineImageName, computeServiceAccount)
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

	// The boot disk for a Windows instance must have the WINDOWS GuestOSFeature,
	// while the boot disk for other operating systems shouldn't have it.
	for _, disk := range instance.Disks {
		if !disk.Boot {
			continue
		}

		hasWindowsFeature := false
		for _, feature := range disk.GuestOsFeatures {
			if "WINDOWS" == feature.Type {
				hasWindowsFeature = true
				break
			}
		}

		if props.isWindows && !hasWindowsFeature {
			testCase.WriteFailure(
				"Windows boot disk missing WINDOWS GuestOsFeature. Features found=%v",
				disk.GuestOsFeatures)
		} else if !props.isWindows && hasWindowsFeature {
			testCase.WriteFailure(
				"Non-Windows boot disk includes WINDOWS GuestOsFeature. Features found=%v",
				disk.GuestOsFeatures)
		}
	}

	if props.machineType != "" && !strings.HasSuffix(instance.MachineType, props.machineType) {
		testCase.WriteFailure(
			"Instance machine type `%v` doesn't match the expected machine type `%v`",
			instance.MachineType, props.machineType)
		return
	}

	if !strings.HasSuffix(instance.Zone, props.zone) {
		e2e.Failure(testCase, logger, fmt.Sprintf("Instance zone `%v` doesn't match requested zone `%v`",
			instance.Zone, props.zone))
		return
	}

	if props.verificationStartupScript == "" {
		logger.Printf("[%v] Will not set test startup script to test instance metadata as it's not defined", props.machineImageName)
		return
	}

	err = instance.StartWithScriptCode(props.verificationStartupScript, props.instanceMetadata)
	if err != nil {
		testCase.WriteFailure("Error starting instance `%v` with script: `%v`: %v", testInstanceName, err)
		return
	}
	logger.Printf("[%v] Waiting for `%v` in instance serial console.", testInstanceName,
		props.expectedStartupOutput)
	if err := instance.WaitForSerialOutput(
		props.expectedStartupOutput, props.failureMatches, 1, 5*time.Second, 15*time.Minute); err != nil {
		testCase.WriteFailure("Error during VM validation: %v", err)
	}
}

func loadScriptContent(scriptPath string, logger *log.Logger) string {
	scriptContent, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		logger.Printf("Error loading script `%v`: %v", scriptPath, err)
		os.Exit(1)
	}
	return string(scriptContent)
}
