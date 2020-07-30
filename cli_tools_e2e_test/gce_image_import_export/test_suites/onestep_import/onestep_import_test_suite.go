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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "OnestepImageImportTests"
)

// OnestepImageImportSuite contains implementations of the e2e tests.
func OnestepImageImportSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	testTypes := []utils.CLITestType{
		utils.Wrapper,
		utils.GcloudProdWrapperLatest,
		utils.GcloudLatestWrapperLatest,
	}

	testsMap := map[utils.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, utils.CLITestType){}

	for _, testType := range testTypes {
		onestepImageImportFromAWSUbuntuAMI := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OnestepImageImport] %v", testType, "Onestep image import from AWS Ubuntu-1804 AMI"))
		onestepImageImportFromAWSUbuntuVMDK := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OnestepImageImport] %v", testType, "Onestep image import from AWS Ubuntu-1804 VMDK"))
		onestepImageImportFromAWSWindowsAMI := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OnestepImageImport] %v", testType, "Onestep image import from AWS Windows-2019 AMI"))
		onestepImageImportFromAWSWindowsVMDK := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OnestepImageImport] %v", testType, "Onestep image import from AWS Windows-2019 VMDK"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, utils.CLITestType){}
		testsMap[testType][onestepImageImportFromAWSUbuntuAMI] = runOnestepImageImportFromAWSUbuntuAMI
		testsMap[testType][onestepImageImportFromAWSUbuntuVMDK] = runOnestepImageImportFromAWSUbuntuVMDK
		testsMap[testType][onestepImageImportFromAWSWindowsAMI] = runOnestepImageImportFromAWSWindowsAMI
		testsMap[testType][onestepImageImportFromAWSWindowsVMDK] = runOnestepImageImportFromAWSWindowsVMDK
	}

	if !utils.GcloudAuth(logger, nil) {
		logger.Printf("Failed to run gcloud auth.")
		testSuite := junitxml.NewTestSuite(testSuiteName)
		testSuite.Failures = 1
		testSuite.Finish(testSuites)
		tswg.Done()
		return
	}

	if err := setAWSAuth(logger, nil); err != nil {
		utils.Failure(nil, logger, fmt.Sprintf("Failed to get aws credentials: %v\n", err))
		testSuite := junitxml.NewTestSuite(testSuiteName)
		testSuite.Failures = 1
		testSuite.Finish(testSuites)
		tswg.Done()
		return
	}

	utils.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runOnestepImageImportFromAWSUbuntuAMI(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {
	imageName := "e2e-test-onestep-image-import" + path.RandString(5)

	props := &onestepImportAWSTestProperties{
		imageName:         imageName,
		os:                "ubuntu-1804",
		amiID:             ubuntuAMIID,
		amiExportLocation: awsBucket,
		startupScript:     "post_translate_test.sh",
	}

	runOnestepImportTest(ctx, props, testProjectConfig, testType, logger, testCase)
}

func runOnestepImageImportFromAWSUbuntuVMDK(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {
	imageName := "e2e-test-onestep-image-import" + path.RandString(5)

	props := &onestepImportAWSTestProperties{
		imageName:         imageName,
		sourceAMIFilePath: ubuntuVMDKFilePath,
		os:                "ubuntu-1804",
		startupScript:     "post_translate_test.sh",
	}

	runOnestepImportTest(ctx, props, testProjectConfig, testType, logger, testCase)
}

func runOnestepImageImportFromAWSWindowsAMI(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {
	imageName := "e2e-test-onestep-image-import" + path.RandString(5)

	props := &onestepImportAWSTestProperties{
		imageName:         imageName,
		amiID:             windowsAMIID,
		amiExportLocation: awsBucket,
		os:                "windows-2019",
		timeout:           "4h",
		startupScript:     "post_translate_test.ps1",
	}

	runOnestepImportTest(ctx, props, testProjectConfig, testType, logger, testCase)
}

func runOnestepImageImportFromAWSWindowsVMDK(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {
	imageName := "e2e-test-onestep-image-import" + path.RandString(5)

	props := &onestepImportAWSTestProperties{
		imageName:         imageName,
		sourceAMIFilePath: windowsVMDKFilePath,
		os:                "windows-2019",
		startupScript:     "post_translate_test.ps1",
	}

	runOnestepImportTest(ctx, props, testProjectConfig, testType, logger, testCase)
}

func runOnestepImportTest(ctx context.Context, props *onestepImportAWSTestProperties, testConfig *testconfig.Project, testType utils.CLITestType,
	logger *log.Logger, testCase *junitxml.TestCase) {
	args := buildTestArgs(props, testConfig)[testType]

	cmds := map[utils.CLITestType]string{
		utils.Wrapper:                   "./gce_onestep_image_import",
		utils.GcloudProdWrapperLatest:   "gcloud",
		utils.GcloudLatestWrapperLatest: "gcloud",
	}

	if utils.RunTestForTestType(cmds[testType], args, testType, logger, testCase) {
		verifyImportedImageFile(ctx, testCase, props, testConfig, logger)
	}
}

// buildTestArgs build args for tests.
func buildTestArgs(props *onestepImportAWSTestProperties, testProjectConfig *testconfig.Project) map[utils.CLITestType][]string {
	gcloudArgs := []string{
		"beta", "compute", "images", "import", "--quiet",
		"--docker-image-tag=latest", props.imageName,
		fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("--aws-access-key-id=%v", awsAccessKeyID),
		fmt.Sprintf("--aws-secret-access-key=%v", awsSecretAccessKey),
		fmt.Sprintf("--aws-session-token=%v", awsSessionToken),
		fmt.Sprintf("--aws-region=%v", awsRegion),
		fmt.Sprintf("--os=%v", props.os),
	}
	wrapperArgs := []string{
		"-client_id=e2e",
		fmt.Sprintf("-image_name=%v", props.imageName),
		fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("-aws_access_key_id=%v", awsAccessKeyID),
		fmt.Sprintf("-aws_secret_access_key=%v", awsSecretAccessKey),
		fmt.Sprintf("-aws_session_token=%v", awsSessionToken),
		fmt.Sprintf("-aws_region=%v", awsRegion),
		fmt.Sprintf("-os=%v", props.os),
	}

	if props.sourceAMIFilePath != "" {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--aws-source-ami-file-path=%v", props.sourceAMIFilePath))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-aws_source_ami_file_path=%v", props.sourceAMIFilePath))
	}
	if props.amiID != "" {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--aws-ami-id=%v", props.amiID))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-aws_ami_id=%v", props.amiID))
	}
	if props.amiExportLocation != "" {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--aws-ami-export-location=%v", props.amiExportLocation))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-aws_ami_export_location=%v", props.amiExportLocation))
	}

	if props.timeout != "" {
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--timeout=%v", props.timeout))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-timeout=%v", props.timeout))
	}

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper:                   wrapperArgs,
		utils.GcloudProdWrapperLatest:   gcloudArgs,
		utils.GcloudLatestWrapperLatest: gcloudArgs,
	}
	return argsMap
}

// verifyImportedImageFile boots the instance and executes a startup script containing tests.
func verifyImportedImageFile(ctx context.Context, testCase *junitxml.TestCase, props *onestepImportAWSTestProperties, testProjectConfig *testconfig.Project, logger *log.Logger) {
	wf, err := daisy.NewFromFile("post_translate_test.wf.json")
	if err != nil {
		utils.Failure(testCase, logger, fmt.Sprintf("Failed post translate test: %v\n", err))
		return
	}

	imagePath := fmt.Sprintf("projects/%s/global/images/%s", testProjectConfig.TestProjectID, props.imageName)

	wf.Vars = map[string]daisy.Var{
		"image_under_test": {
			Value: imagePath,
		},
		"startup_script": {
			Value: props.startupScript,
		},
	}

	wf.Logger = logging.AsDaisyLogger(logger)
	wf.Project = testProjectConfig.TestProjectID
	wf.Zone = testProjectConfig.TestZone
	err = wf.Run(ctx)
	if err != nil {
		utils.Failure(testCase, logger, fmt.Sprintf("Failed post translate test: %v\n", err))
	}
}
