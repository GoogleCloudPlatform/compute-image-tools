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

// Package exporttestsuites contains e2e tests for image export cli tools
package exporttestsuites

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/gce_image_import_export_tests/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/gce_image_import_export_tests/test_suites"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "ImageExportTests"
)

// TestSuite is image export test suite.
func TestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	imageExportRawTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageExport] %v", "Export Raw"))
	imageExportVMDKTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageExport] %v", "Export VMDK"))
	imageExportWithRichParamsTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageExport] %v", "Export with rich params"))
	imageExportWithSubnetWithoutNetworkTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageExport] %v", "Export with subnet but without network"))

	testsMap := map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project){
		imageExportRawTestCase:                      runImageExportRawTest,
		imageExportVMDKTestCase:                     runImageExportVMDKTest,
		imageExportWithRichParamsTestCase:           runImageExportWithRichParamsTest,
		imageExportWithSubnetWithoutNetworkTestCase: runImageExportWithSubnetWithoutNetworkParamsTest,
	}

	testsuiteutils.TestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runImageExportRawTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := path.RandString(5)
	bucketName := fmt.Sprintf("%v-test-image", testProjectConfig.TestProjectID)
	objectName := fmt.Sprintf("e2e-export-raw-test-%v", suffix)
	fileURI := fmt.Sprintf("gs://%v/%v", bucketName, objectName)
	cmd := "gce_vm_image_export"
	args := []string{"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
		"-source_image=global/images/e2e-test-image-10g", fmt.Sprintf("-destination_uri=%v", fileURI)}
	if err := testsuiteutils.RunCliTool(logger, testCase, cmd, args); err != nil {
		logger.Printf("Error running cmd: %v\n", err)
		testCase.WriteFailure("Error running cmd: %v", err)
		return
	}

	verifyExportedImageFile(ctx, testCase, bucketName, objectName, logger)
}

func runImageExportVMDKTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := path.RandString(5)
	bucketName := fmt.Sprintf("%v-test-image", testProjectConfig.TestProjectID)
	objectName := fmt.Sprintf("e2e-export-vmdk-test-%v", suffix)
	fileURI := fmt.Sprintf("gs://%v/%v", bucketName, objectName)
	cmd := "gce_vm_image_export"
	args := []string{"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
		"-source_image=global/images/e2e-test-image-10g", fmt.Sprintf("-destination_uri=%v", fileURI), "-format=vmdk"}
	if err := testsuiteutils.RunCliTool(logger, testCase, cmd, args); err != nil {
		logger.Printf("Error running cmd: %v\n", err)
		testCase.WriteFailure("Error running cmd: %v", err)
		return
	}

	verifyExportedImageFile(ctx, testCase, bucketName, objectName, logger)
}

// Test most of params except -oauth, -compute_endpoint_override, and -scratch_bucket_gcs_path
func runImageExportWithRichParamsTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := path.RandString(5)
	bucketName := fmt.Sprintf("%v-test-image", testProjectConfig.TestProjectID)
	objectName := fmt.Sprintf("e2e-export-rich-param-test-%v", suffix)
	fileURI := fmt.Sprintf("gs://%v/%v", bucketName, objectName)
	cmd := "gce_vm_image_export"
	args := []string{"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
		"-source_image=global/images/e2e-test-image-10g", fmt.Sprintf("-destination_uri=%v", fileURI),
		fmt.Sprintf("-network=%v-vpc-1", testProjectConfig.TestProjectID),
		fmt.Sprintf("-subnet=%v-subnet-1", testProjectConfig.TestProjectID),
		fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		"-timeout=2h", "-disable_gcs_logging", "-disable_cloud_logging", "-disable_stdout_logging",
		"-labels=key1=value1,key2=value"}
	if err := testsuiteutils.RunCliTool(logger, testCase, cmd, args); err != nil {
		logger.Printf("Error running cmd: %v\n", err)
		testCase.WriteFailure("Error running cmd: %v", err)
		return
	}

	verifyExportedImageFile(ctx, testCase, bucketName, objectName, logger)
}

func runImageExportWithSubnetWithoutNetworkParamsTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := path.RandString(5)
	bucketName := fmt.Sprintf("%v-test-image", testProjectConfig.TestProjectID)
	objectName := fmt.Sprintf("e2e-export-subnet-test-%v", suffix)
	fileURI := fmt.Sprintf("gs://%v/%v", bucketName, objectName)
	cmd := "gce_vm_image_export"
	args := []string{"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("-subnet=%v-subnet-1", testProjectConfig.TestProjectID),
		"-source_image=global/images/e2e-test-image-10g", fmt.Sprintf("-destination_uri=%v", fileURI)}
	if err := testsuiteutils.RunCliTool(logger, testCase, cmd, args); err != nil {
		logger.Printf("Error running cmd: %v\n", err)
		testCase.WriteFailure("Error running cmd: %v", err)
		return
	}

	verifyExportedImageFile(ctx, testCase, bucketName, objectName, logger)
}

func verifyExportedImageFile(ctx context.Context, testCase *junitxml.TestCase, bucketName string,
	objectName string, logger *log.Logger) {
	logger.Printf("Verifying exported file...")
	file, err := storage.CreateFileObject(ctx, bucketName, objectName)
	if err != nil {
		testCase.WriteFailure("File '%v' doesn't exist after export: %v", objectName, err)
		logger.Printf("File '%v' doesn't exist after export: %v", objectName, err)
		return
	}
	logger.Printf("File '%v' exists! Export success.", objectName)

	if err := file.Cleanup(); err != nil {
		logger.Printf("File '%v' failed to clean up.", objectName)
	} else {
		logger.Printf("File '%v' cleaned up.", objectName)
	}
}
