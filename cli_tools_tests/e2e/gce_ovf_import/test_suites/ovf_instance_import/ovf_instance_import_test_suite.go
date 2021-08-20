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
	"log"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e"
	ovfimporttestsuite "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e/gce_ovf_import/test_suites"
	"github.com/GoogleCloudPlatform/compute-image-tools/common/gcp"
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

	// argMap stores test args from e2e test CLI.
	argMap map[string]string
)

type ovfInstanceImportTestProperties struct {
	ovfimporttestsuite.OvfImportTestProperties
	instanceName string
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
		e2e.GcloudGaLatestWrapperRelease,
	}
	for _, testType := range testTypes {
		instanceImportUbuntu3DisksTestCaseNetworkSettingsName := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Ubuntu 3 disks, one data disk larger than 10GB, network setting (name only)"))
		instanceImportWindows2012R2TwoDisksNetworkSettingsPath := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Windows 2012 R2 two disks, network setting (path)"))
		instanceImportWindows2016 := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Windows 2016"))
		instanceImportWindows2008R2FourNICs := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Windows 2008r2 - Four NICs"))
		instanceImportDebian9 := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Debian 9"))
		instanceImportUbuntu16FromVirtualBox := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Ubuntu 1604 from Virtualbox"))
		instanceImportUbuntu16FromAWS := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Ubuntu 1604 from AWS"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}
		testsMap[testType][instanceImportUbuntu3DisksTestCaseNetworkSettingsName] = runOVFInstanceImportUbuntu3DisksNetworkSettingsName
		testsMap[testType][instanceImportWindows2012R2TwoDisksNetworkSettingsPath] = runOVFInstanceImportWindows2012R2TwoDisksNetworkSettingsPath
		testsMap[testType][instanceImportWindows2016] = runOVFInstanceImportWindows2016
		testsMap[testType][instanceImportWindows2008R2FourNICs] = runOVFInstanceImportWindows2008R2FourNICs
		testsMap[testType][instanceImportDebian9] = runOVFInstanceImportDebian9
		testsMap[testType][instanceImportUbuntu16FromVirtualBox] = runOVFInstanceImportUbuntu16FromVirtualBox
		testsMap[testType][instanceImportUbuntu16FromAWS] = runOVFInstanceImportUbuntu16FromAWS
	}

	// gcloud only tests
	instanceImportDisabledDefaultServiceAccountSuccessTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "No default service account, success by specifying custom Compute service account (Oracle Linux as CentOS)"))
	instanceImportDefaultServiceAccountWithMissingPermissionsSuccessTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "No permission on default service account, success by specifying a custom Compute service account"))
	instanceImportDisabledDefaultServiceAccountFailTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "No default service account, failed"))
	instanceImportDefaultServiceAccountWithMissingPermissionsFailTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "No permission on default service account, failed"))
	instanceImportDefaultServiceAccountCustomAccessScopeTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "No default service account, custom access scopes"))
	instanceImportDefaultServiceAccountNoAccessScopeTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "No default service account, no access scopes"))
	instanceImportNoServiceAccountTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.GcloudBetaLatestWrapperLatest, "No service account"))

	testsMap[e2e.Wrapper][instanceImportDisabledDefaultServiceAccountSuccessTestCase] = runInstanceImportDisabledDefaultServiceAccountSuccessTest
	testsMap[e2e.Wrapper][instanceImportDefaultServiceAccountWithMissingPermissionsSuccessTestCase] = runInstanceImportDefaultServiceAccountWithMissingPermissionsSuccessTest
	testsMap[e2e.Wrapper][instanceImportDisabledDefaultServiceAccountFailTestCase] = runInstanceImportWithDisabledDefaultServiceAccountFailTest
	testsMap[e2e.Wrapper][instanceImportDefaultServiceAccountWithMissingPermissionsFailTestCase] = runInstanceImportDefaultServiceAccountWithMissingPermissionsFailTest
	testsMap[e2e.Wrapper][instanceImportDefaultServiceAccountCustomAccessScopeTestCase] = runInstanceImportDefaultServiceAccountCustomAccessScope
	testsMap[e2e.Wrapper][instanceImportDefaultServiceAccountNoAccessScopeTestCase] = runInstanceImportDefaultServiceAccountNoAccessScope
	testsMap[e2e.Wrapper][instanceImportNoServiceAccountTestCase] = runInstanceImportNoServiceAccount

	e2e.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runOVFInstanceImportUbuntu3DisksNetworkSettingsName(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-ubuntu-3-disks-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{
			VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
				"scripts/ovf_import_test_ubuntu_3_disks.sh", logger),
			Zone:                  testProjectConfig.TestZone,
			ExpectedStartupOutput: "All tests passed!",
			FailureMatches:        []string{"FAILED:", "TestFailed:"},
			SourceURI:             fmt.Sprintf("gs://%v/ova/ubuntu-1604-three-disks", ovaBucket),
			Os:                    "ubuntu-1604",
			MachineType:           "n1-standard-4",
			Network:               fmt.Sprintf("%v-vpc-1", testProjectConfig.TestProjectID),
			Subnet:                fmt.Sprintf("%v-subnet-1", testProjectConfig.TestProjectID),
			Tags:                  []string{"tag1", "tag2", "tag3"},
		}}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportWindows2012R2TwoDisksNetworkSettingsPath(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	region, _ := paramhelper.GetRegion(testProjectConfig.TestZone)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-w2k12-r2-%v", suffix),
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
			Subnet: fmt.Sprintf("projects/%v/regions/%v/subnetworks/%v-subnet-1",
				testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			Tags: []string{"tag1", "tag2", "tag3"},
		}}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportWindows2016(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-w2k16-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.ps1", logger),
			Zone:                  "asia-northeast1-a",
			ExpectedStartupOutput: "All Tests Passed",
			FailureMatches:        []string{"Test Failed:"},
			SourceURI:             fmt.Sprintf("gs://%v-asia/ova/w2k16/w2k16.ovf", ovaBucket),
			Os:                    "windows-2016",
			MachineType:           "n2-standard-2",
			IsWindows:             true,
		}}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportWindows2008R2FourNICs(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-w2k8r2-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.ps1", logger),
			Zone:                  "europe-west2-c",
			ExpectedStartupOutput: "All Tests Passed",
			FailureMatches:        []string{"Test Failed:"},
			SourceURI:             fmt.Sprintf("gs://%v-eu/ova/win2008r2-all-updates-four-nic.ova", ovaBucket),
			Os:                    "windows-2008r2",
			InstanceMetadata:      skipOSConfigMetadata,
			IsWindows:             true,
		}}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportDebian9(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	// no startup script as this OVA has issues running it (possibly due to no SSH allowed)
	// b/141321520
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-debian-9-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{
			Zone:        "us-west1-c",
			SourceURI:   fmt.Sprintf("gs://%v/ova/bitnami-tomcat-8.5.43-0-linux-debian-9-x86_64.ova", ovaBucket),
			Os:          "debian-9",
			MachineType: "n1-standard-4",
		}}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportUbuntu16FromVirtualBox(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-virtualbox-6-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                  "asia-southeast1-c",
			ExpectedStartupOutput: "All tests passed!",
			FailureMatches:        []string{"FAILED:", "TestFailed:"},
			SourceURI:             fmt.Sprintf("gs://%v-asia/ova/ubuntu-16.04-virtualbox.ova", ovaBucket),
			Os:                    "ubuntu-1604",
			MachineType:           "n1-standard-4",
		},
	}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runOVFInstanceImportUbuntu16FromAWS(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-instance-aws-ova-ubuntu-1604-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{
			Zone:        "europe-west4-c",
			SourceURI:   fmt.Sprintf("gs://%v-eu/ova/aws-ova-ubuntu-1604.ova", ovaBucket),
			Os:          "ubuntu-1604",
			MachineType: "n1-standard-4",
		},
	}

	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func runInstanceImportDisabledDefaultServiceAccountSuccessTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, true)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}
	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-without-service-account-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{
			VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
				"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                   "asia-northeast1-c",
			ExpectedStartupOutput:  "All tests passed!",
			FailureMatches:         []string{"FAILED:", "TestFailed:"},
			SourceURI:              fmt.Sprintf("gs://%v-asia/ova/OL7U9_x86_64-olvm-b77.ova", ovaBucket),
			Os:                     "centos-7",
			MachineType:            "n1-standard-4",
			Project:                testVariables.ProjectID,
			ComputeServiceAccount:  testVariables.ComputeServiceAccount,
			InstanceServiceAccount: testVariables.InstanceServiceAccount,
			InstanceAccessScopes:   "https://www.googleapis.com/auth/compute,https://www.googleapis.com/auth/datastore",
		},
	}
	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

// With insufficient permissions on default service account, import success by specifying a custom account.
func runInstanceImportDefaultServiceAccountWithMissingPermissionsSuccessTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, false)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}
	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-missing-ce-permissions-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                   "europe-west3-c",
			ExpectedStartupOutput:  "All tests passed!",
			FailureMatches:         []string{"FAILED:", "TestFailed:"},
			SourceURI:              fmt.Sprintf("gs://%v-eu/ova/centos-7.4/", ovaBucket),
			Os:                     "centos-7",
			MachineType:            "n1-standard-4",
			Project:                testVariables.ProjectID,
			ComputeServiceAccount:  testVariables.ComputeServiceAccount,
			InstanceServiceAccount: testVariables.InstanceServiceAccount,
		},
	}
	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

// With insufficient permissions on default service account, import failed.
func runInstanceImportWithDisabledDefaultServiceAccountFailTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, true)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}
	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-missing-permission-on-default-csa-fail-%v", suffix),
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
func runInstanceImportDefaultServiceAccountWithMissingPermissionsFailTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, false)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}
	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-insufficient-permission-default-csa-fail-%v", suffix),
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

// Ensure custom access scopes are set on the instance even when default service account is used
func runInstanceImportDefaultServiceAccountCustomAccessScope(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-scopes-on-default-cse-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                  "us-east1-c",
			ExpectedStartupOutput: "All tests passed!",
			FailureMatches:        []string{"FAILED:", "TestFailed:"},
			SourceURI:             fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
			Os:                    "centos-7",
			MachineType:           "n1-standard-4",
			Project:               testProjectConfig.TestProjectID,
			InstanceAccessScopes:  "https://www.googleapis.com/auth/compute,https://www.googleapis.com/auth/datastore",
		}}
	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

// Ensure no access scopes are set on the instance when default service account is used
func runInstanceImportDefaultServiceAccountNoAccessScope(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-default-sa-no-scopes-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                   "us-east4-c",
			ExpectedStartupOutput:  "All tests passed!",
			FailureMatches:         []string{"FAILED:", "TestFailed:"},
			SourceURI:              fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
			Os:                     "centos-7",
			MachineType:            "n1-standard-4",
			Project:                testProjectConfig.TestProjectID,
			NoInstanceAccessScopes: true,
		}}
	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

// Ensure no service account is set on imported instances
func runInstanceImportNoServiceAccount(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	suffix := path.RandString(5)
	props := &ovfInstanceImportTestProperties{
		instanceName: fmt.Sprintf("test-no-service-account-%v", suffix),
		OvfImportTestProperties: ovfimporttestsuite.OvfImportTestProperties{VerificationStartupScript: ovfimporttestsuite.LoadScriptContent(
			"daisy_integration_tests/scripts/post_translate_test.sh", logger),
			Zone:                     testProjectConfig.TestZone,
			ExpectedStartupOutput:    "All tests passed!",
			FailureMatches:           []string{"FAILED:", "TestFailed:"},
			SourceURI:                fmt.Sprintf("gs://%v/ova/centos-7.4/", ovaBucket),
			Os:                       "centos-7",
			MachineType:              "n1-standard-4",
			Project:                  testProjectConfig.TestProjectID,
			NoInstanceServiceAccount: true,
		}}
	runOVFInstanceImportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func buildTestArgs(props *ovfInstanceImportTestProperties, testProjectConfig *testconfig.Project) map[e2e.CLITestType][]string {
	gcloudBetaArgs := []string{
		"beta", "compute", "instances", "import", props.instanceName, "--quiet",
		"--docker-image-tag=latest"}
	gcloudArgs := []string{"compute", "instances", "import", props.instanceName, "--quiet"}
	wrapperArgs := []string{"-client-id=e2e", fmt.Sprintf("-instance-names=%s", props.instanceName)}
	return ovfimporttestsuite.BuildArgsMap(&props.OvfImportTestProperties, testProjectConfig, gcloudBetaArgs, gcloudArgs, wrapperArgs)
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

	project := ovfimporttestsuite.GetProject(&props.OvfImportTestProperties, testProjectConfig)
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Error creating client: %v", err))
		return
	}
	logger.Printf("Verifying imported instance...")
	instance, err := gcp.CreateInstanceBetaObject(ctx, project, props.Zone, props.instanceName, props.IsWindows)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Instance '%v' doesn't exist after import: %v", props.instanceName, err))
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

	ovfimporttestsuite.VerifyInstance(instance, client, testCase, project, logger, &props.OvfImportTestProperties)
}
