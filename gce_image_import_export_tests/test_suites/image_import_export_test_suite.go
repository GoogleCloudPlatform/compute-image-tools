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

// Package importexporttestsuites contains e2e tests for image import/export cli tools
package importexporttestsuites

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/gce_image_import_export_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/test_common/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/test_common/test_config"
)

const (
	testSuiteName = "ImageImportExportTests"
)

// TestSuite is image import test suite.
func TestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	defer tswg.Done()

	if testSuiteRegex != nil && !testSuiteRegex.MatchString(testSuiteName) {
		return
	}

	testSuite := junitxml.NewTestSuite(testSuiteName)
	defer testSuite.Finish(testSuites)
	logger.Printf("Running TestSuite %q", testSuite.Name)
	tests := runTestCases(ctx, logger, testCaseRegex, testProjectConfig)

	for ret := range tests {
		testSuite.TestCase = append(testSuite.TestCase, ret)
	}

	logger.Printf("Finished TestSuite %q", testSuite.Name)
}

func runTestCases(
	ctx context.Context, logger *log.Logger, regex *regexp.Regexp,
	testProjectConfig *testconfig.Project) chan *junitxml.TestCase{

	imageImportDataDiskTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageImport] %v", "Import data disk"))
	/*
	imageImportOSTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageImport] %v", "Import OS"))
	imageImportOSFromImageTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageImport] %v", "Import OS from image"))
	imageExportRawTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageExport] %v", "Export Raw"))
	imageExportVMDKTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageExport] %v", "Export VMDK"))
*/
	testsMap := map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project){
		imageImportDataDiskTestCase: runImageImportDataDiskTest,
//		imageImportOSTestCase: runImageImportOSTest,
//		imageImportOSFromImageTestCase: runImageImportOSFromImageTest,
//		imageExportRawTestCase: runImageExportRawTest,
//		imageExportVMDKTestCase: runImageExportVMDKTest,
	}

	var wg sync.WaitGroup
	tests := make(chan *junitxml.TestCase)
	for tc, f := range testsMap {
		wg.Add(1)
		go func(wg *sync.WaitGroup, tc *junitxml.TestCase, f func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project)) {
			defer wg.Done()
			if tc.FilterTestCase(regex) {
				tc.Finish(tests)
			} else {
				logger.Printf("Running TestCase %s.%q", tc.Classname, tc.Name)
				f(ctx, tc, logger, testProjectConfig)
				tc.Finish(tests)
				logger.Printf("TestCase %s.%q finished in %fs", tc.Classname, tc.Name, tc.Time)
			}
		}(&wg, tc, f)
	}

	go func() {
		wg.Wait()
		close(tests)
	}()

	return tests
}

func runImageImportDataDiskTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := pathutils.RandString(5)
	imageName := "e2e-test-image-import-data-disk-" + suffix
	cmd := "gce_vm_image_import"
	args := []string{"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID), fmt.Sprintf("-image_name=%s", imageName), "-data_disk", "-source_file=gs://compute-image-test-pool-001-test-image/image-file-10g-vmdk"}
	runCliTool(logger, testCase, cmd, args)

	// Verify the result
	// TODO: get image
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		testCase.WriteFailure("error creating client: %v", err)
		return
	}
	image := compute.CreateImageObject(client, testProjectConfig.TestProjectID, imageName)
	err = image.Exists()
	if err != nil {
		logger.Fatalf("Image '%v' doesn't exist after import: %v", imageName, err)
	}
}

func runImageImportOSTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := pathutils.RandString(5)
	cmd := "gce_vm_image_import"
	args := []string{"-client_id=e2e", "-image_name=e2e_test_image_import_os_" + suffix, "-data_disk", "-source_image=image1"}
	runCliTool(logger, testCase, cmd, args)

	// Verify the result
	// TODO: get image

}

func runImageImportOSFromImageTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := pathutils.RandString(5)
	cmd := "gce_vm_image_import"
	args := []string{"-client_id=e2e", "-image_name=e2e_test_image_import_os_from_image_" + suffix, "-data_disk", "-source_image=image1"}
	runCliTool(logger, testCase, cmd, args)

	// Verify the result
	// TODO: get image

}

func runImageExportRawTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := pathutils.RandString(5)
	cmd := "gce_vm_image_export"
	args := []string{"-client_id=e2e", "-image_name=e2e_test_image_export_raw_" + suffix, "-data_disk", "-source_image=image1"}
	runCliTool(logger, testCase, cmd, args)

	// Verify the result
	// TODO: get file

}

func runImageExportVMDKTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := pathutils.RandString(5)
	cmd := "gce_vm_image_export"
	args := []string{"-client_id=e2e", "-image_name=e2e_test_image_export_vmdk_" + suffix, "-data_disk", "-source_image=image1"}
	runCliTool(logger, testCase, cmd, args)

	// Verify the result
	// TODO: get file

}

func runCliTool(logger *log.Logger, testCase *junitxml.TestCase, cmdString string, args []string) {
	// Execute cli tool
	logger.Printf("[%v] Running command: '%s %s'", testCase.Name, cmdString, strings.Join(args, " "))
	cmd := exec.Command(fmt.Sprintf("./%s", cmdString), args...)
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		logger.Fatalf("Error running cmd: %v\n", err.Error())
	}
}
