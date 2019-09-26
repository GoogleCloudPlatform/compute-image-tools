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
func TestSuite(ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp, testProjectConfig *testconfig.Project) {

	testTypes := []testsuiteutils.TestType{
		testsuiteutils.Wrapper,
		testsuiteutils.GcloudProdWrapperLatest,
		testsuiteutils.GcloudLatestWrapperLatest,
	}

	testsMap := map[testsuiteutils.TestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, testsuiteutils.TestType){}

	for _, testType := range testTypes {
		imageExportRawTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][ImageExport] %v", testType, "Export Raw"))
		imageExportVMDKTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][ImageExport] %v", testType, "Export VMDK"))
		imageExportWithRichParamsTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][ImageExport] %v", testType, "Export with rich params"))
		imageExportWithSubnetWithoutNetworkTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][ImageExport] %v", testType, "Export with subnet but without network"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, testsuiteutils.TestType){}
		testsMap[testType][imageExportRawTestCase] = runImageExportRawTest
		testsMap[testType][imageExportVMDKTestCase] = runImageExportVMDKTest
		testsMap[testType][imageExportWithRichParamsTestCase] = runImageExportWithRichParamsTest
		testsMap[testType][imageExportWithSubnetWithoutNetworkTestCase] = runImageExportWithSubnetWithoutNetworkParamsTest
	}

	testsuiteutils.TestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex, testProjectConfig, testSuiteName, testsMap)
}

func runImageExportRawTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType testsuiteutils.TestType) {

	suffix := path.RandString(5)
	bucketName := fmt.Sprintf("%v-test-image", testProjectConfig.TestProjectID)
	objectName := fmt.Sprintf("e2e-export-raw-test-%v", suffix)
	fileURI := fmt.Sprintf("gs://%v/%v", bucketName, objectName)

	argsMap := map[testsuiteutils.TestType][]string{
		testsuiteutils.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			"-source_image=global/images/e2e-test-image-10g", fmt.Sprintf("-destination_uri=%v", fileURI),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		},
		testsuiteutils.GcloudProdWrapperLatest: {"beta", "compute", "images", "export", "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID), "--image=e2e-test-image-10g",
			fmt.Sprintf("--destination-uri=%v", fileURI), fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
		testsuiteutils.GcloudLatestWrapperLatest: {"beta", "compute", "images", "export", "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID), "--image=e2e-test-image-10g",
			fmt.Sprintf("--destination-uri=%v", fileURI), fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
	}

	runExportTest(ctx, argsMap[testType], testType, logger, testCase, bucketName, objectName)
}

func runImageExportVMDKTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType testsuiteutils.TestType) {

	suffix := path.RandString(5)
	bucketName := fmt.Sprintf("%v-test-image", testProjectConfig.TestProjectID)
	objectName := fmt.Sprintf("e2e-export-vmdk-test-%v", suffix)
	fileURI := fmt.Sprintf("gs://%v/%v", bucketName, objectName)

	argsMap := map[testsuiteutils.TestType][]string{
		testsuiteutils.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			"-source_image=global/images/e2e-test-image-10g", fmt.Sprintf("-destination_uri=%v", fileURI), "-format=vmdk",
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		},
		testsuiteutils.GcloudProdWrapperLatest: {"beta", "compute", "images", "export", "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID), "--image=e2e-test-image-10g",
			fmt.Sprintf("--destination-uri=%v", fileURI), "--export-format=vmdk", fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
		testsuiteutils.GcloudLatestWrapperLatest: {"beta", "compute", "images", "export", "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID), "--image=e2e-test-image-10g",
			fmt.Sprintf("--destination-uri=%v", fileURI), "--export-format=vmdk", fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
	}

	runExportTest(ctx, argsMap[testType], testType, logger, testCase, bucketName, objectName)
}

// Test most of params except -oauth, -compute_endpoint_override, and -scratch_bucket_gcs_path
func runImageExportWithRichParamsTest(ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project, testType testsuiteutils.TestType) {

	suffix := path.RandString(5)
	bucketName := fmt.Sprintf("%v-test-image", testProjectConfig.TestProjectID)
	objectName := fmt.Sprintf("e2e-export-rich-param-test-%v", suffix)
	fileURI := fmt.Sprintf("gs://%v/%v", bucketName, objectName)

	argsMap := map[testsuiteutils.TestType][]string{
		testsuiteutils.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			"-source_image=e2e-test-image-10g", fmt.Sprintf("-destination_uri=%v", fileURI),
			fmt.Sprintf("-network=%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("-subnet=%v-subnet-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
			"-timeout=2h", "-disable_gcs_logging", "-disable_cloud_logging", "-disable_stdout_logging",
			"-labels=key1=value1,key2=value",
		},
		testsuiteutils.GcloudProdWrapperLatest: {"beta", "compute", "images", "export", "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--network=%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%v-subnet-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
			"--timeout=2h", "--image=e2e-test-image-10g", fmt.Sprintf("--destination-uri=%v", fileURI),
		},
		testsuiteutils.GcloudLatestWrapperLatest: {"beta", "compute", "images", "export", "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--network=%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%v-subnet-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
			"--timeout=2h", "--image=e2e-test-image-10g", fmt.Sprintf("--destination-uri=%v", fileURI),
		},
	}

	runExportTest(ctx, argsMap[testType], testType, logger, testCase, bucketName, objectName)
}

func runImageExportWithSubnetWithoutNetworkParamsTest(ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project, testType testsuiteutils.TestType) {

	suffix := path.RandString(5)
	bucketName := fmt.Sprintf("%v-test-image", testProjectConfig.TestProjectID)
	objectName := fmt.Sprintf("e2e-export-subnet-test-%v", suffix)
	fileURI := fmt.Sprintf("gs://%v/%v", bucketName, objectName)

	argsMap := map[testsuiteutils.TestType][]string{
		testsuiteutils.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-subnet=%v-subnet-1", testProjectConfig.TestProjectID),
			"-source_image=global/images/e2e-test-image-10g", fmt.Sprintf("-destination_uri=%v", fileURI),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		},
		testsuiteutils.GcloudProdWrapperLatest: {"beta", "compute", "images", "export", "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%v-subnet-1", testProjectConfig.TestProjectID), "--image=e2e-test-image-10g",
			fmt.Sprintf("--destination-uri=%v", fileURI), fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
		testsuiteutils.GcloudLatestWrapperLatest: {"beta", "compute", "images", "export", "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%v-subnet-1", testProjectConfig.TestProjectID), "--image=e2e-test-image-10g",
			fmt.Sprintf("--destination-uri=%v", fileURI), fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
	}

	runExportTest(ctx, argsMap[testType], testType, logger, testCase, bucketName, objectName)
}

func runExportTest(ctx context.Context, args []string, testType testsuiteutils.TestType,
	logger *log.Logger, testCase *junitxml.TestCase, bucketName string, objectName string) {

	cmds := map[testsuiteutils.TestType]string{
		testsuiteutils.Wrapper:                   "./gce_vm_image_export",
		testsuiteutils.GcloudProdWrapperLatest:   "gcloud",
		testsuiteutils.GcloudLatestWrapperLatest: "gcloud",
	}

	if testsuiteutils.RunTestForTestType(cmds[testType], args, testType, logger, testCase) {
		verifyExportedImageFile(ctx, testCase, bucketName, objectName, logger)
	}
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
