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

// Package importtestsuites contains e2e tests for image import cli tools
package importtestsuites

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/gce_image_import_export_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/gce_image_import_export_tests/test_suites"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "ImageImportTests"
)

// TestSuite is image import test suite.
func TestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	imageImportDataDiskTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageImport] %v", "Import data disk"))
	imageImportOSTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageImport] %v", "Import OS"))
	imageImportOSFromImageTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[ImageImport] %v", "Import OS from image"))

	testsMap := map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project){
		imageImportDataDiskTestCase:    runImageImportDataDiskTest,
		imageImportOSTestCase:          runImageImportOSTest,
		imageImportOSFromImageTestCase: runImageImportOSFromImageTest,
	}

	testsuiteutils.TestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runImageImportDataDiskTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := pathutils.RandString(5)
	imageName := "e2e-test-image-import-data-disk-" + suffix
	cmd := "gce_vm_image_import"
	args := []string{"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("-image_name=%s", imageName), "-data_disk", fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID)}
	testsuiteutils.RunCliTool(logger, testCase, cmd, args)

	verifyImportedImage(ctx, testCase, testProjectConfig, imageName, logger)
}

func runImageImportOSTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := pathutils.RandString(5)
	imageName := "e2e-test-image-import-os-" + suffix
	cmd := "gce_vm_image_import"
	args := []string{"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("-image_name=%v", imageName), "-os=debian-9", fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID)}
	testsuiteutils.RunCliTool(logger, testCase, cmd, args)

	verifyImportedImage(ctx, testCase, testProjectConfig, imageName, logger)
}

func runImageImportOSFromImageTest(
	ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project) {

	suffix := pathutils.RandString(5)
	imageName := "e2e-test-image-import-os-from-image-" + suffix
	cmd := "gce_vm_image_import"
	args := []string{"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("-image_name=%v", imageName), "-os=debian-9", "-source_image=e2e-test-image-10g"}
	testsuiteutils.RunCliTool(logger, testCase, cmd, args)

	verifyImportedImage(ctx, testCase, testProjectConfig, imageName, logger)
}

func verifyImportedImage(ctx context.Context, testCase *junitxml.TestCase,
	testProjectConfig *testconfig.Project, imageName string, logger *log.Logger) {
	logger.Printf("Verifying imported image...")
	image, err := compute.CreateImageObject(ctx, testProjectConfig.TestProjectID, imageName)
	if err != nil {
		testCase.WriteFailure("Error creating compute api client: %v", err)
		logger.Printf("Error creating compute api client: %v", err)
		return
	}

	if err := image.Exists(); err != nil {
		testCase.WriteFailure("Image '%v' doesn't exist after import: %v", imageName, err)
		logger.Printf("Image '%v' doesn't exist after import: %v", imageName, err)
		return
	}
	logger.Printf("Image '%v' exists! Import success.", imageName)

	if err := image.Cleanup(); err != nil {
		logger.Printf("Image '%v' failed to clean up.", imageName)
	} else {
		logger.Printf("Image '%v' cleaned up.", imageName)
	}
}
