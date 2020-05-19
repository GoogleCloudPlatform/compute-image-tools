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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	computeUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/utils"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "OVFMachineImageImportTests"
	ovaBucket     = "compute-image-tools-test-resources"
)

var (
	cmds = map[utils.CLITestType]string{
		utils.Wrapper:                   "./gce_ovf_import",
		utils.GcloudProdWrapperLatest:   "gcloud",
		utils.GcloudLatestWrapperLatest: "gcloud",
	}
)

type ovfMachineImageImportTestProperties struct {
	machineImageName          string
	isWindows                 bool
	expectedStartupOutput     string
	verificationStartupScript string
	zone                      string
	sourceURI                 string
	os                        string
	machineType               string
	network                   string
	subnet                    string
	storageLocation           string
}

// TestSuite is image import test suite.
func TestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	testsMap := map[utils.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, utils.CLITestType){}

	testTypes := []utils.CLITestType{
		utils.Wrapper,
		utils.GcloudProdWrapperLatest,
		utils.GcloudLatestWrapperLatest,
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
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, utils.CLITestType){}
		testsMap[testType][machineImageImportUbuntu3DisksTestCase] = runOVFMachineImageImportUbuntu3Disks
		testsMap[testType][machineImageImportWindows2012R2TwoDisks] = runOVFMachineImageImportWindows2012R2TwoDisks
		testsMap[testType][machineImageImportNetworkSettingsName] = runOVFMachineImageImportNetworkSettingsName
		testsMap[testType][machineImageImportNetworkSettingsPath] = runOVFMachineImageImportNetworkSettingsPath
		testsMap[testType][machineImageImportStorageLocation] = runOVFMachineImageImportStorageLocation
	}

	utils.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runOVFMachineImageImportUbuntu3Disks(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-machine-image-ubuntu-3-disks-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"scripts/ovf_import_test_ubuntu_3_disks.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		sourceURI:             fmt.Sprintf("gs://%v/ova/ubuntu-1604-three-disks", ovaBucket),
		os:                    "ubuntu-1604",
		machineType:           "n1-standard-4"}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFMachineImageImportCentos68(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-machine-image-centos-6-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		sourceURI:             fmt.Sprintf("gs://%v/", ovaBucket),
		os:                    "centos-6",
		machineType:           "n1-standard-4",
	}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFMachineImageImportWindows2012R2TwoDisks(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-machine-image-w2k12-r2-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"scripts/ovf_import_test_windows_two_disks.ps1", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All Tests Passed",
		sourceURI:             fmt.Sprintf("gs://%v/ova/w2k12-r2", ovaBucket),
		os:                    "windows-2012r2",
		machineType:           "n1-standard-8",
		isWindows:             true,
	}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFMachineImageImportStorageLocation(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-machine-image-w2k16-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.ps1", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All Tests Passed",
		sourceURI:             fmt.Sprintf("gs://%v/ova/w2k16/w2k16.ovf", ovaBucket),
		os:                    "windows-2016",
		machineType:           "n2-standard-2",
		isWindows:             true,
		storageLocation:       "us-west2",
	}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFMachineImageImportNetworkSettingsName(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-network-name-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		sourceURI:             fmt.Sprintf("gs://%v/", ovaBucket),
		os:                    "centos-6",
		machineType:           "n1-standard-4",
		network:               fmt.Sprintf("%v-vpc-1", testProjectConfig.TestProjectID),
		subnet:                fmt.Sprintf("%v-subnet-1", testProjectConfig.TestProjectID),
	}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFMachineImageImportNetworkSettingsPath(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	region, _ := paramhelper.GetRegion(testProjectConfig.TestZone)
	props := &ovfMachineImageImportTestProperties{
		machineImageName: fmt.Sprintf("test-network-path-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		sourceURI:             fmt.Sprintf("gs://%v/", ovaBucket),
		os:                    "centos-6",
		machineType:           "n1-standard-4",
		network:               fmt.Sprintf("global/networks/%v-vpc-1", testProjectConfig.TestProjectID),
		subnet:                fmt.Sprintf("projects/%v/regions/%v/subnetworks/%v-subnet-1", testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
	}

	runOVFMachineImageImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func buildTestArgs(props *ovfMachineImageImportTestProperties, testProjectConfig *testconfig.Project) map[utils.CLITestType][]string {
	gcloudArgs := []string{
		"beta", "compute", "machine-images", "import", props.machineImageName, "--quiet",
		"--docker-image-tag=latest",
		fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("--source-uri=%v", props.sourceURI),
		fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
	}
	wrapperArgs := []string{"-client-id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("-machine-image-name=%s", props.machineImageName),
		fmt.Sprintf("-ovf-gcs-path=%v", props.sourceURI),
		fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
	}

	if props.os != "" {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--os=%v", props.os))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-os=%v", props.os))
	}
	if props.machineType != "" {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--machine-type=%v", props.machineType))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-machine-type=%v", props.machineType))
	}
	if props.network != "" {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--network=%v", props.network))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-network=%v", props.network))
	}
	if props.subnet != "" {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--subnet=%v", props.subnet))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-subnet=%v", props.subnet))
	}
	if props.storageLocation != "" {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--storage-location=%v", props.storageLocation))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-machine-image-storage-location=%v", props.storageLocation))
	}

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper:                   wrapperArgs,
		utils.GcloudProdWrapperLatest:   gcloudArgs,
		utils.GcloudLatestWrapperLatest: gcloudArgs,
	}
	return argsMap
}

func runOVFMachineImageImportTest(ctx context.Context, args []string, testType utils.CLITestType,
	testProjectConfig *testconfig.Project, logger *log.Logger, testCase *junitxml.TestCase,
	props *ovfMachineImageImportTestProperties) {

	if utils.RunTestForTestType(cmds[testType], args, testType, logger, testCase) {
		verifyImportedMachineImage(ctx, testCase, testProjectConfig, logger, props)
	}
}

func verifyImportedMachineImage(
	ctx context.Context, testCase *junitxml.TestCase, testProjectConfig *testconfig.Project,
	logger *log.Logger, props *ovfMachineImageImportTestProperties) {

	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		utils.Failure(testCase, logger, fmt.Sprintf("Error creating client: %v", err))
		return
	}

	logger.Printf("Verifying imported machine image...")
	testInstanceName := props.machineImageName + "-test-instance"
	logger.Printf("Creating `%v` test instance.", testInstanceName)
	instance, err := computeUtils.CreateInstanceBeta(
		ctx, testProjectConfig.TestProjectID, props.zone, testInstanceName, props.isWindows, props.machineImageName)
	if err != nil {
		utils.Failure(testCase, logger, fmt.Sprintf("Error when creating test instance `%v` from machine image '%v': %v", testInstanceName, props.machineImageName, err))
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
		if err := client.DeleteMachineImage(testProjectConfig.TestProjectID, props.machineImageName); err != nil {
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
		utils.Failure(testCase, logger, fmt.Sprintf("Instance zone `%v` doesn't match requested zone `%v`",
			instance.Zone, props.zone))
		return
	}

	if props.verificationStartupScript == "" {
		logger.Printf("[%v] Will not set test startup script to test instance metadata as it's not defined", props.machineImageName)
		return
	}

	err = instance.StartWithScript(props.verificationStartupScript)
	if err != nil {
		testCase.WriteFailure("Error starting instance `%v` with script: `%v`: %v", testInstanceName, err)
		return
	}
	logger.Printf("[%v] Waiting for `%v` in instance serial console.", testInstanceName,
		props.expectedStartupOutput)
	if err := instance.WaitForSerialOutput(
		props.expectedStartupOutput, 1, 5*time.Second, 15*time.Minute); err != nil {
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
