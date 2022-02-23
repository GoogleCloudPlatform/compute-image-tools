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

// Package onestepimporttestsuites contains e2e tests for gce_onestep_image_import
package onestepimporttestsuites

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e"
	"github.com/GoogleCloudPlatform/compute-image-tools/common/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "OnestepImageImportTests"
)

var (
	// argMap stores test args from e2e test CLI.
	argMap map[string]string
)

// OnestepImageImportSuite contains implementations of the e2e tests.
func OnestepImageImportSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project, argMapInput map[string]string) {

	argMap = argMapInput

	testTypes := []e2e.CLITestType{
		e2e.Wrapper,
		e2e.GcloudBetaProdWrapperLatest,
		e2e.GcloudBetaLatestWrapperLatest,
	}

	testsMap := map[e2e.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}

	for _, testType := range testTypes {
		onestepImageImportFromAWSUbuntuAMI := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "From AWS Ubuntu-1804 AMI"))
		onestepImageImportFromAWSUbuntuVMDK := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "From AWS Ubuntu-1804 VMDK"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}
		testsMap[testType][onestepImageImportFromAWSUbuntuAMI] = runOnestepImageImportFromAWSLinuxAMI
		testsMap[testType][onestepImageImportFromAWSUbuntuVMDK] = runOnestepImageImportFromAWSLinuxVMDK
	}

	// Only test windows for wrapper to reduce the test time. The aws-related code
	// logic for windows are exactly the same as for linux, so no need to
	// duplicate them too much.
	onestepImageImportFromAWSWindowsAMI := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.Wrapper, "From AWS Windows-2019 AMI"))
	onestepImageImportFromAWSWindowsVMDK := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.Wrapper, "From AWS Windows-2019 VMDK"))
	testsMap[e2e.Wrapper][onestepImageImportFromAWSWindowsAMI] = runOnestepImageImportFromAWSWindowsAMI
	testsMap[e2e.Wrapper][onestepImageImportFromAWSWindowsVMDK] = runOnestepImageImportFromAWSWindowsVMDK

	// Only test service account scenario for wrapper, till gcloud support it.
	onestepImageImportWithDisabledDefaultServiceAccountSuccess := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.Wrapper, "Without default service account"))
	onestepImageImportDefaultServiceAccountWithMissingPermissionsSuccess := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.Wrapper, "Without default service account"))
	testsMap[e2e.Wrapper][onestepImageImportWithDisabledDefaultServiceAccountSuccess] = runOnestepImageImportWithDisabledDefaultServiceAccountSuccess
	testsMap[e2e.Wrapper][onestepImageImportDefaultServiceAccountWithMissingPermissionsSuccess] = runOnestepImageImportDefaultServiceAccountWithMissingPermissionsSuccess

	if !e2e.GcloudAuth(logger, nil) {
		logger.Printf("Failed to run gcloud auth.")
		testSuite := junitxml.NewTestSuite(testSuiteName)
		testSuite.Failures = 1
		testSuite.Finish(testSuites)
		tswg.Done()
		return
	}

	if !getAWSTestArgs() {
		e2e.Failure(nil, logger, fmt.Sprintln("Failed to get aws test args"))
		testSuite := junitxml.NewTestSuite(testSuiteName)
		testSuite.Failures = 1
		testSuite.Finish(testSuites)
		tswg.Done()
		return
	}

	if err := setAWSAuth(logger, nil); err != nil {
		e2e.Failure(nil, logger, fmt.Sprintf("Failed to get aws credentials: %v\n", err))
		testSuite := junitxml.NewTestSuite(testSuiteName)
		testSuite.Failures = 1
		testSuite.Finish(testSuites)
		tswg.Done()
		return
	}

	e2e.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runOnestepImageImportFromAWSLinuxAMI(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	imageName := "e2e-test-onestep-image-import" + path.RandString(5)

	props := &onestepImportAWSTestProperties{
		imageName:     imageName,
		amiID:         ubuntuAMIID,
		os:            "ubuntu-1804",
		startupScript: "post_translate_test.sh",
		skipOSConfig:  "true",
	}

	runOnestepImportTest(ctx, props, testProjectConfig.TestProjectID, testProjectConfig.TestZone, testType, logger, testCase)
}

func runOnestepImageImportFromAWSLinuxVMDK(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	imageName := "e2e-test-onestep-image-import" + path.RandString(5)

	props := &onestepImportAWSTestProperties{
		imageName:         imageName,
		sourceAMIFilePath: ubuntuVMDKFilePath,
		os:                "ubuntu-1804",
		startupScript:     "post_translate_test.sh",
		skipOSConfig:      "true",
	}

	runOnestepImportTest(ctx, props, testProjectConfig.TestProjectID, testProjectConfig.TestZone, testType, logger, testCase)
}

func runOnestepImageImportFromAWSWindowsAMI(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	imageName := "e2e-test-onestep-image-import" + path.RandString(5)

	props := &onestepImportAWSTestProperties{
		imageName:     imageName,
		amiID:         windowsAMIID,
		os:            "windows-2019",
		timeout:       "4h",
		startupScript: "post_translate_test.ps1",
	}

	runOnestepImportTest(ctx, props, testProjectConfig.TestProjectID, testProjectConfig.TestZone, testType, logger, testCase)
}

func runOnestepImageImportFromAWSWindowsVMDK(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {
	imageName := "e2e-test-onestep-image-import" + path.RandString(5)

	props := &onestepImportAWSTestProperties{
		imageName:         imageName,
		sourceAMIFilePath: windowsVMDKFilePath,
		os:                "windows-2019",
		startupScript:     "post_translate_test.ps1",
	}

	runOnestepImportTest(ctx, props, testProjectConfig.TestProjectID, testProjectConfig.TestZone, testType, logger, testCase)
}

// With a disabled default service account, import success by specifying a custom account.
func runOnestepImageImportWithDisabledDefaultServiceAccountSuccess(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	serviceAccountTestVariables, ok := e2e.GetServiceAccountTestVariables(argMap, true)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}

	imageName := "e2e-onestep-no-default-service-account-" + path.RandString(5)

	props := &onestepImportAWSTestProperties{
		imageName:             imageName,
		sourceAMIFilePath:     ubuntuVMDKFilePath,
		os:                    "ubuntu-1804",
		startupScript:         "post_translate_test.sh",
		skipOSConfig:          "true",
		computeServiceAccount: serviceAccountTestVariables.ComputeServiceAccount,
	}

	runOnestepImportTest(ctx, props, serviceAccountTestVariables.ProjectID, testProjectConfig.TestZone, testType, logger, testCase)
}

// With insufficient permissions on default service account, import success by specifying a custom account.
func runOnestepImageImportDefaultServiceAccountWithMissingPermissionsSuccess(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	serviceAccountTestVariables, ok := e2e.GetServiceAccountTestVariables(argMap, false)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}

	imageName := "e2e-onestep-no-account-permission-" + path.RandString(5)

	props := &onestepImportAWSTestProperties{
		imageName:             imageName,
		sourceAMIFilePath:     ubuntuVMDKFilePath,
		os:                    "ubuntu-1804",
		startupScript:         "post_translate_test.sh",
		skipOSConfig:          "true",
		computeServiceAccount: serviceAccountTestVariables.ComputeServiceAccount,
	}

	runOnestepImportTest(ctx, props, serviceAccountTestVariables.ProjectID, testProjectConfig.TestZone, testType, logger, testCase)
}

func runOnestepImportTest(ctx context.Context, props *onestepImportAWSTestProperties, testProjectID string, testZone string, testType e2e.CLITestType,
	logger *log.Logger, testCase *junitxml.TestCase) {
	args := buildTestArgs(props, testProjectID, testZone)[testType]

	cmds := map[e2e.CLITestType]string{
		e2e.Wrapper:                       "./gce_onestep_image_import",
		e2e.GcloudBetaProdWrapperLatest:   "gcloud",
		e2e.GcloudBetaLatestWrapperLatest: "gcloud",
	}

	if e2e.RunTestForTestType(cmds[testType], args, testType, logger, testCase) {
		verifyImportedImageFile(ctx, testCase, props, testProjectID, testZone, logger)
	}
}

// buildTestArgs build args for tests.
func buildTestArgs(props *onestepImportAWSTestProperties, testProjectID string, testZone string) map[e2e.CLITestType][]string {
	gcloudArgs := []string{
		"beta", "compute", "images", "import", "--quiet",
		"--docker-image-tag=latest", props.imageName,
		fmt.Sprintf("--project=%v", testProjectID),
		fmt.Sprintf("--zone=%v", testZone),
		fmt.Sprintf("--aws-access-key-id=%v", awsAccessKeyID),
		fmt.Sprintf("--aws-secret-access-key=%v", awsSecretAccessKey),
		fmt.Sprintf("--aws-session-token=%v", awsSessionToken),
		fmt.Sprintf("--aws-region=%v", awsRegion),
		fmt.Sprintf("--os=%v", props.os),
		fmt.Sprintf("--compute-service-account=%v", props.computeServiceAccount),
	}

	wrapperArgs := []string{
		"-client_id=e2e",
		fmt.Sprintf("-image_name=%v", props.imageName),
		fmt.Sprintf("-project=%v", testProjectID),
		fmt.Sprintf("-zone=%v", testZone),
		fmt.Sprintf("-aws_access_key_id=%v", awsAccessKeyID),
		fmt.Sprintf("-aws_secret_access_key=%v", awsSecretAccessKey),
		fmt.Sprintf("-aws_session_token=%v", awsSessionToken),
		fmt.Sprintf("-aws_region=%v", awsRegion),
		fmt.Sprintf("-os=%v", props.os),
		fmt.Sprintf("-compute_service_account=%v", props.computeServiceAccount),
	}

	if props.amiID != "" {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--aws-ami-export-location=%v", awsBucket))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-aws_ami_export_location=%v", awsBucket))
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--aws-ami-id=%v", props.amiID))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-aws_ami_id=%v", props.amiID))
	} else {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--aws-source-ami-file-path=%v", props.sourceAMIFilePath))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-aws_source_ami_file_path=%v", props.sourceAMIFilePath))
	}

	if props.timeout != "" {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--timeout=%v", props.timeout))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-timeout=%v", props.timeout))
	}

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper:                       wrapperArgs,
		e2e.GcloudBetaProdWrapperLatest:   gcloudArgs,
		e2e.GcloudBetaLatestWrapperLatest: gcloudArgs,
	}
	return argsMap
}

// verifyImportedImageFile boots the instance and executes a startup script containing tests.
func verifyImportedImageFile(ctx context.Context, testCase *junitxml.TestCase, props *onestepImportAWSTestProperties, testProjectID string, testZone string, logger *log.Logger) {
	wf, err := daisy.NewFromFile("post_translate_test.wf.json")
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Failed post translate test: %v\n", err))
		return
	}

	imagePath := fmt.Sprintf("projects/%s/global/images/%s", testProjectID, props.imageName)

	wf.Vars = map[string]daisy.Var{
		"image_under_test": {
			Value: imagePath,
		},
		"path_to_post_translate_test": {
			Value: props.startupScript,
		},
		"osconfig_not_supported": {
			Value: props.skipOSConfig,
		},
		"compute_service_account": {
			Value: props.computeServiceAccount,
		},
	}

	wf.Logger = logging.AsDaisyLogger(logger)
	wf.Project = testProjectID
	wf.Zone = testZone
	err = wf.Run(ctx)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Failed post translate test: %v\n", err))
	}
}
