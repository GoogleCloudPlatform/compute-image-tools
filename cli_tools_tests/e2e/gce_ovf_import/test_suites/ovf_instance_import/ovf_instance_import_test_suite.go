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

// Package ovfinstanceimporttestsuite contains e2e tests for instance import cli tools
package ovfinstanceimporttestsuite

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
	testSuiteName = "OVFInstanceImportTests"
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
)

type ovfInstanceImportTestProperties struct {
	instanceName              string
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
	instanceMetadata          map[string]string
}

// TestSuite is image import test suite.
func TestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	testsMap := map[e2e.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}

	testTypes := []e2e.CLITestType{
		e2e.Wrapper,
		e2e.GcloudBetaProdWrapperLatest,
		e2e.GcloudBetaLatestWrapperLatest,
		e2e.GcloudGaLatestWrapperRelease,
	}
	for _, testType := range testTypes {
		instanceImportUbuntu3DisksTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFInstanceImport] %v", testType, "Ubuntu 3 disks, one data disk larger than 10GB"))
		instanceImportCentos74 := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFInstanceImport] %v", testType, "Centos 7.4"))
		instanceImportWindows2012R2TwoDisks := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFInstanceImport] %v", testType, "Windows 2012 R2 two disks"))
		instanceImportWindows2016 := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFInstanceImport] %v", testType, "Windows 2016"))
		instanceImportWindows2008R2FourNICs := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFInstanceImport] %v", testType, "Windows 2008r2 - Four NICs"))
		instanceImportDebian9 := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFInstanceImport] %v", testType, "Debian 9"))
		instanceImportUbuntu16FromVirtualBox := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFInstanceImport] %v", testType, "Ubuntu 1604 from Virtualbox"))
		instanceImportUbuntu16FromAWS := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFInstanceImport] %v", testType, "Ubuntu 1604 from AWS"))
		instanceImportNetworkSettingsName := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFInstanceImport] %v", testType, "Test network setting (name only)"))
		instanceImportNetworkSettingsPath := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFInstanceImport] %v", testType, "Test network setting (path)"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}
		testsMap[testType][instanceImportUbuntu3DisksTestCase] = runOVFInstanceImportUbuntu3Disks
		testsMap[testType][instanceImportCentos74] = runOVFInstanceImportCentos74
		testsMap[testType][instanceImportWindows2012R2TwoDisks] = runOVFInstanceImportWindows2012R2TwoDisks
		testsMap[testType][instanceImportWindows2016] = runOVFInstanceImportWindows2016
		testsMap[testType][instanceImportWindows2008R2FourNICs] = runOVFInstanceImportWindows2008R2FourNICs
		testsMap[testType][instanceImportDebian9] = runOVFInstanceImportDebian9
		testsMap[testType][instanceImportUbuntu16FromVirtualBox] = runOVFInstanceImportUbuntu16FromVirtualBox
		testsMap[testType][instanceImportUbuntu16FromAWS] = runOVFInstanceImportUbuntu16FromAWS
		testsMap[testType][instanceImportNetworkSettingsName] = runOVFInstanceImportNetworkSettingsName
		testsMap[testType][instanceImportNetworkSettingsPath] = runOVFInstanceImportNetworkSettingsPath
	}

	e2e.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runOVFInstanceImportUbuntu3Disks(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-ubuntu-3-disks-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"scripts/ovf_import_test_ubuntu_3_disks.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		failureMatches:        []string{"FAILED:", "TestFailed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/ubuntu-1604-three-disks", ovaBucket),
		instanceMetadata:      skipOSConfigMetadata,
		os:                    "ubuntu-1604",
		machineType:           "n1-standard-4"}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportCentos74(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-centos-7-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		failureMatches:        []string{"FAILED:", "TestFailed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
		os:                    "centos-7",
		machineType:           "n1-standard-4",
	}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportWindows2012R2TwoDisks(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-w2k12-r2-%v", suffix),
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

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportWindows2016(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-w2k16-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.ps1", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All Tests Passed",
		failureMatches:        []string{"Test Failed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/w2k16/w2k16.ovf", ovaBucket),
		os:                    "windows-2016",
		machineType:           "n2-standard-2",
		isWindows:             true,
	}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportWindows2008R2FourNICs(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-w2k8r2-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.ps1", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All Tests Passed",
		failureMatches:        []string{"Test Failed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/win2008r2-all-updates-four-nic.ova", ovaBucket),
		os:                    "windows-2008r2",
		instanceMetadata:      skipOSConfigMetadata,
		isWindows:             true,
	}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportDebian9(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	// no startup script as this OVA has issues running it (possibly due to no SSH allowed)
	// b/141321520
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-debian-9-%v", suffix),
		zone:         testProjectConfig.TestZone,
		sourceURI:    fmt.Sprintf("gs://%v/ova/bitnami-tomcat-8.5.43-0-linux-debian-9-x86_64.ova", ovaBucket),
		os:           "debian-9",
		machineType:  "n1-standard-4",
	}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportUbuntu16FromVirtualBox(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-virtualbox-6-%v", suffix),
		verificationStartupScript: loadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		failureMatches:        []string{"FAILED:", "TestFailed:"},
		sourceURI:             fmt.Sprintf("gs://%v/ova/ubuntu-16.04-virtualbox.ova", ovaBucket),
		os:                    "ubuntu-1604",
		instanceMetadata:      skipOSConfigMetadata,
		machineType:           "n1-standard-4",
	}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportUbuntu16FromAWS(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-aws-ova-ubuntu-1604-%v", suffix),
		zone:         testProjectConfig.TestZone,
		sourceURI:    fmt.Sprintf("gs://%v/ova/aws-ova-ubuntu-1604.ova", ovaBucket),
		os:           "ubuntu-1604",
		machineType:  "n1-standard-4",
	}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportNetworkSettingsName(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-network-name-%v", suffix),
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

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportNetworkSettingsPath(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	region, _ := paramhelper.GetRegion(testProjectConfig.TestZone)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-network-path-%v", suffix),
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

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func buildTestArgs(props *ovfInstanceImportTestProperties, testProjectConfig *testconfig.Project) map[e2e.CLITestType][]string {
	gcloudBetaArgs := []string{
		"beta", "compute", "instances", "import", props.instanceName, "--quiet",
		"--docker-image-tag=latest",
		fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("--source-uri=%v", props.sourceURI),
		fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
	}
	gcloudArgs := []string{
		"compute", "instances", "import", props.instanceName, "--quiet",
		fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("--source-uri=%v", props.sourceURI),
		fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
	}
	wrapperArgs := []string{"-client-id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("-instance-names=%s", props.instanceName),
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

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper:                       wrapperArgs,
		e2e.GcloudBetaProdWrapperLatest:   gcloudBetaArgs,
		e2e.GcloudBetaLatestWrapperLatest: gcloudBetaArgs,
		e2e.GcloudGaLatestWrapperRelease:  gcloudArgs,
	}
	return argsMap
}

func runOVFInstanceImportTest(ctx context.Context, args []string, testType e2e.CLITestType,
	testProjectConfig *testconfig.Project, logger *log.Logger, testCase *junitxml.TestCase,
	props *ovfInstanceImportTestProperties) {

	if e2e.RunTestForTestType(cmds[testType], args, testType, logger, testCase) {
		verifyImportedInstance(ctx, testCase, testProjectConfig, logger, props)
	}
}

func verifyImportedInstance(
	ctx context.Context, testCase *junitxml.TestCase, testProjectConfig *testconfig.Project,
	logger *log.Logger, props *ovfInstanceImportTestProperties) {

	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Error creating client: %v", err))
		return
	}

	logger.Printf("Verifying imported instance...")
	instance, err := gcp.CreateInstanceObject(ctx, testProjectConfig.TestProjectID, props.zone, props.instanceName, props.isWindows)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Image '%v' doesn't exist after import: %v", props.instanceName, err))
		return
	}

	defer func() {
		logger.Printf("Deleting instance `%v`", props.instanceName)
		if err := instance.Cleanup(); err != nil {
			logger.Printf("Instance '%v' failed to clean up: %v", props.instanceName, err)
		} else {
			logger.Printf("Instance '%v' cleaned up.", props.instanceName)
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

	logger.Printf("[%v] Stopping instance before restarting with test startup script", props.instanceName)
	err = client.StopInstance(
		testProjectConfig.TestProjectID, props.zone, props.instanceName)

	if err != nil {
		testCase.WriteFailure("Error stopping imported instance: %v", err)
		return
	}

	if props.verificationStartupScript == "" {
		logger.Printf("[%v] Will not set test startup script to instance metadata as it's not defined", props.instanceName)
		return
	}

	err = instance.StartWithScriptCode(props.verificationStartupScript, props.instanceMetadata)
	if err != nil {
		testCase.WriteFailure("Error starting instance `%v` with script: %v", props.instanceName, err)
		return
	}
	logger.Printf("[%v] Waiting for `%v` in instance serial console.", props.instanceName,
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
