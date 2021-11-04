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
	"strings"
	"sync"

	"google.golang.org/api/googleapi"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e"
	"github.com/GoogleCloudPlatform/compute-image-tools/common/gcp"
	computeApi "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "ImageImportCLI"
)

var (
	// argMap stores test args from e2e test CLI.
	argMap map[string]string
)

var cmds = map[e2e.CLITestType]string{
	e2e.Wrapper:                       "./gce_vm_image_import",
	e2e.GcloudBetaProdWrapperLatest:   "gcloud",
	e2e.GcloudBetaLatestWrapperLatest: "gcloud",
	e2e.GcloudGaLatestWrapperRelease:  "gcloud",
}

// CLITestSuite ensures that gcloud and the wrapper have consistent behavior for image imports.
func CLITestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project, argMapInput map[string]string) {

	argMap = argMapInput

	testTypes := []e2e.CLITestType{
		e2e.Wrapper,
		e2e.GcloudBetaProdWrapperLatest,
		e2e.GcloudBetaLatestWrapperLatest,
		e2e.GcloudGaLatestWrapperRelease,
	}

	testsMap := map[e2e.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}

	for _, testType := range testTypes {
		imageImportDataDiskTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Import data disk"))
		imageImportOSTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Import OS"))
		imageImportOSFromImageTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Import OS from image"))
		imageImportWithRichParamsTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Import with rich params"))
		imageImportWithDifferentNetworkParamStylesTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Import with different network param styles"))
		imageImportWithSubnetWithoutNetworkSpecifiedTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Import with subnet but without network"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}
		testsMap[testType][imageImportDataDiskTestCase] = runImageImportDataDiskTest
		testsMap[testType][imageImportOSTestCase] = runImageImportOSTest
		testsMap[testType][imageImportOSFromImageTestCase] = runImageImportOSFromImageTest
		testsMap[testType][imageImportWithRichParamsTestCase] = runImageImportWithRichParamsTest
		testsMap[testType][imageImportWithDifferentNetworkParamStylesTestCase] = runImageImportWithDifferentNetworkParamStyles
		testsMap[testType][imageImportWithSubnetWithoutNetworkSpecifiedTestCase] = runImageImportWithSubnetWithoutNetworkSpecified

		// TODO: recover this test only when shadow test is enabled.
		//imageImportShadowDiskCleanedUpWhenMainInflaterFails := junitxml.NewTestCase(
		//	testSuiteName, fmt.Sprintf("[%v] %v", testType, "Import shadow disk is cleaned up when main inflater fails"))
		//testsMap[testType][imageImportShadowDiskCleanedUpWhenMainInflaterFails] = runImageImportShadowDiskCleanedUpWhenMainInflaterFails
	}

	// Only test service account scenario for wrapper, till gcloud support it.
	imageImportOSWithDisabledDefaultServiceAccountSuccessTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.Wrapper, "Import OS without default service account, success by specifying a custom account"))
	imageImportOSDefaultServiceAccountWithMissingPermissionsSuccessTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.Wrapper, "Import OS without permission on default service account, success by specifying a custom account"))
	imageImportOSWithDisabledDefaultServiceAccountFailTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.Wrapper, "Import OS without default service account failed"))
	imageImportOSDefaultServiceAccountWithMissingPermissionsFailTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[%v] %v", e2e.Wrapper, "Import OS without permission on default service account failed"))
	testsMap[e2e.Wrapper][imageImportOSWithDisabledDefaultServiceAccountSuccessTestCase] = runImageImportOSWithDisabledDefaultServiceAccountServiceSuccessTest
	testsMap[e2e.Wrapper][imageImportOSDefaultServiceAccountWithMissingPermissionsSuccessTestCase] = runImageImportOSDefaultServiceAccountWithMissingPermissionsSuccessTest
	testsMap[e2e.Wrapper][imageImportOSWithDisabledDefaultServiceAccountFailTestCase] = runImageImportOSWithDisabledDefaultServiceAccountServiceFailTest
	testsMap[e2e.Wrapper][imageImportOSDefaultServiceAccountWithMissingPermissionsFailTestCase] = runImageImportOSDefaultServiceAccountWithMissingPermissionsFailTest

	e2e.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runImageImportDataDiskTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-data-disk-" + suffix
	zone := "europe-west2-c"
	sourceFile := fmt.Sprintf("gs://%v-test-image-eu/image-file-10g-vmdk", testProjectConfig.TestProjectID)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%s", imageName), "-data_disk",
			fmt.Sprintf("-source_file=%v", sourceFile),
			fmt.Sprintf("-zone=%v", zone),
		},
		e2e.GcloudBetaProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
		},
		e2e.GcloudBetaLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
		},
		e2e.GcloudGaLatestWrapperRelease: {"compute", "images", "import", imageName, "--quiet",
			"--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testProjectConfig.TestProjectID, imageName, logger, testCase)
}

func runImageImportOSTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-os-" + suffix
	zone := "asia-southeast1-c"
	sourceFile := fmt.Sprintf("gs://%v-test-image-asia/image-file-10g-vmdk", testProjectConfig.TestProjectID)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%v", imageName), "-os=debian-9",
			fmt.Sprintf("-source_file=%v", sourceFile),
			fmt.Sprintf("-zone=%v", zone),
		},
		e2e.GcloudBetaProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
		},
		e2e.GcloudBetaLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
		},
		e2e.GcloudGaLatestWrapperRelease: {"compute", "images", "import", imageName, "--quiet",
			"--os=debian-9", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testProjectConfig.TestProjectID, imageName, logger, testCase)
}

func runImageImportOSFromImageTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-os-from-image-" + suffix
	zone := "europe-west4-c"

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%v", imageName), "-os=debian-9", "-source_image=e2e-test-image-10g-eu",
			fmt.Sprintf("-zone=%v", zone),
		},
		e2e.GcloudBetaProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			"--source-image=e2e-test-image-10g-eu",
			fmt.Sprintf("--zone=%v", zone),
		},
		e2e.GcloudBetaLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			"--source-image=e2e-test-image-10g-eu",
			fmt.Sprintf("--zone=%v", zone),
		},
		e2e.GcloudGaLatestWrapperRelease: {"compute", "images", "import", imageName, "--quiet",
			"--os=debian-9", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			"--source-image=e2e-test-image-10g-eu",
			fmt.Sprintf("--zone=%v", zone),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testProjectConfig.TestProjectID, imageName, logger, testCase)
}

// Test most of params except -oauth, -compute_endpoint_override, and -scratch_bucket_gcs_path
func runImageImportWithRichParamsTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	family := "test-family"
	description := "test-description"
	labels := []string{"key1=value1", "key2=value2"}

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-rich-param-" + suffix

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%s", imageName), "-data_disk",
			fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			"-no_guest_environment", fmt.Sprintf("-family=%v", family), fmt.Sprintf("-description=%v", description),
			fmt.Sprintf("-network=%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("-subnet=%v-subnet-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
			"-timeout=2h", "-disable_gcs_logging", "-disable_cloud_logging", "-disable_stdout_logging",
			"-no_external_ip", fmt.Sprintf("-labels=%v", strings.Join(labels, ",")),
		},
		e2e.GcloudBetaProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			"--no-guest-environment",
			fmt.Sprintf("--network=%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%v-subnet-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone), "--timeout=2h",
		},
		e2e.GcloudBetaLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			"--no-guest-environment",
			fmt.Sprintf("--network=%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%v-subnet-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone), "--timeout=2h",
		},
		e2e.GcloudGaLatestWrapperRelease: {"compute", "images", "import", imageName, "--quiet",
			"--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			"--no-guest-environment",
			fmt.Sprintf("--network=%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%v-subnet-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone), "--timeout=2h",
		},
	}

	runImportTestWithExtraParams(ctx, argsMap[testType], testType, testProjectConfig.TestProjectID, imageName,
		logger, testCase, family, description, labels)
}

func runImageImportWithDifferentNetworkParamStyles(ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-subnet-" + suffix
	region, _ := paramhelper.GetRegion(testProjectConfig.TestZone)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%s", imageName), "-data_disk",
			fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("-network=global/networks/%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("-subnet=projects/%v/regions/%v/subnetworks/%v-subnet-1",
				testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		},
		e2e.GcloudBetaProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--network=global/networks/%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=projects/%v/regions/%v/subnetworks/%v-subnet-1",
				testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
		e2e.GcloudBetaLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--network=global/networks/%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=projects/%v/regions/%v/subnetworks/%v-subnet-1",
				testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
		e2e.GcloudGaLatestWrapperRelease: {"compute", "images", "import", imageName, "--quiet",
			"--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--network=global/networks/%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=projects/%v/regions/%v/subnetworks/%v-subnet-1",
				testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testProjectConfig.TestProjectID, imageName, logger, testCase)
}

func runImageImportWithSubnetWithoutNetworkSpecified(ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	destinationImage := "e2e-test-image-import-subnet-" + suffix

	// This project doesn't have a 'default' network. Import will fail if a worker
	// isn't configured to use the custom network and subnet.
	project := "compute-image-test-custom-vpc"
	subnet := "regions/us-central1/subnetworks/unrestricted-egress"
	zone := "us-central1-a"
	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", project),
			fmt.Sprintf("-image_name=%s", destinationImage),
			fmt.Sprintf("-source_file=gs://%v-test-image/ubuntu-1804.vpc", testProjectConfig.TestProjectID),
			fmt.Sprintf("-subnet=%s", subnet),
			fmt.Sprintf("-zone=%v", zone),
		},
		e2e.GcloudBetaProdWrapperLatest: {"beta", "compute", "images", "import", destinationImage, "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--project=%v", project),
			fmt.Sprintf("--source-file=gs://%v-test-image/ubuntu-1804.vpc", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%s", subnet),
			fmt.Sprintf("--zone=%v", zone),
		},
		e2e.GcloudBetaLatestWrapperLatest: {"beta", "compute", "images", "import", destinationImage, "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--project=%v", project),
			fmt.Sprintf("--source-file=gs://%v-test-image/ubuntu-1804.vpc", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%s", subnet),
			fmt.Sprintf("--zone=%v", zone),
		},
		e2e.GcloudGaLatestWrapperRelease: {"compute", "images", "import", destinationImage, "--quiet",
			fmt.Sprintf("--project=%v", project),
			fmt.Sprintf("--source-file=gs://%v-test-image/ubuntu-1804.vpc", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%s", subnet),
			fmt.Sprintf("--zone=%v", zone),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testProjectConfig.TestProjectID, destinationImage, logger, testCase)
}

func runImageImportShadowDiskCleanedUpWhenMainInflaterFails(ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-bad-network-" + suffix

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%s", imageName), "-data_disk",
			fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			// main inflater will fail because vpc-1 network is a custom network and requires a subnet flag that is not provided
			fmt.Sprintf("-network=global/networks/%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
			fmt.Sprintf("-execution_id=%v", suffix),
		},
	}

	args := argsMap[testType]
	if args == nil {
		return
	}

	e2e.RunTestCommandAssertErrorMessage(cmds[testType], args, "googleapi: Error 400: Invalid value for field 'resource.networkInterfaces[0]'", logger, testCase)

	// Try get shadow disk.
	shadowDiskName := "shadow-disk-" + suffix
	logger.Printf("Verifying shadow disk cleanup...")
	client, err := computeApi.NewClient(ctx)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Failed to create compute API client: %v", err))
		return
	}
	_, err = client.GetDisk(testProjectConfig.TestProjectID, testProjectConfig.TestZone, shadowDiskName)

	// Expect 404 error to ensure shadow disk has been cleaned up.
	if apiErr, ok := err.(*googleapi.Error); !ok || apiErr.Code != 404 {
		e2e.Failure(testCase, logger, fmt.Sprintf("Shadow disk '%v' not cleaned up.", shadowDiskName))
		return
	}
}

// With a disabled default service account, import success by specifying a custom account.
func runImageImportOSWithDisabledDefaultServiceAccountServiceSuccessTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, true)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}

	suffix := path.RandString(5)
	imageName := "e2e-test-import-without-service-account-" + suffix
	zone := "asia-northeast1-c"
	sourceFile := fmt.Sprintf("gs://%v-test-image-asia/image-file-10g-vmdk", testProjectConfig.TestProjectID)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testVariables.ProjectID),
			fmt.Sprintf("-image_name=%v", imageName), "-os=debian-9",
			fmt.Sprintf("-source_file=%v", sourceFile),
			fmt.Sprintf("-zone=%v", zone),
			fmt.Sprintf("-compute_service_account=%v", testVariables.ComputeServiceAccount),
		},
		e2e.GcloudBetaProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", testVariables.ComputeServiceAccount),
		},
		e2e.GcloudBetaLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", testVariables.ComputeServiceAccount),
		},
		e2e.GcloudGaLatestWrapperRelease: {"compute", "images", "import", imageName, "--quiet",
			"--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", testVariables.ComputeServiceAccount),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testVariables.ProjectID, imageName, logger, testCase)
}

// With insufficient permissions on default service account, import success by specifying a custom account.
func runImageImportOSDefaultServiceAccountWithMissingPermissionsSuccessTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, false)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}

	suffix := path.RandString(5)
	imageName := "e2e-test-import-missing-permission-" + suffix
	zone := "europe-west3-c"
	sourceFile := fmt.Sprintf("gs://%v-test-image-eu/image-file-10g-vmdk", testProjectConfig.TestProjectID)

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testVariables.ProjectID),
			fmt.Sprintf("-image_name=%v", imageName), "-os=debian-9",
			fmt.Sprintf("-source_file=%v", sourceFile),
			fmt.Sprintf("-zone=%v", zone),
			fmt.Sprintf("-compute_service_account=%v", testVariables.ComputeServiceAccount),
		},
		e2e.GcloudBetaProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", testVariables.ComputeServiceAccount),
		},
		e2e.GcloudBetaLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", testVariables.ComputeServiceAccount),
		},
		e2e.GcloudGaLatestWrapperRelease: {"compute", "images", "import", imageName, "--quiet",
			"--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=%v", sourceFile),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", testVariables.ComputeServiceAccount),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testVariables.ProjectID, imageName, logger, testCase)
}

// With a disabled default service account, import failed.
func runImageImportOSWithDisabledDefaultServiceAccountServiceFailTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, true)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}

	suffix := path.RandString(5)
	imageName := "e2e-test-import-without-service-account-fail-" + suffix
	defaultAccount := "default"
	zone := "us-east1-c"

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testVariables.ProjectID),
			fmt.Sprintf("-image_name=%v", imageName), "-os=debian-9",
			fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("-zone=%v", zone),
			fmt.Sprintf("-compute_service_account=%v", defaultAccount),
		},
		e2e.GcloudBetaProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", defaultAccount),
		},
		e2e.GcloudBetaLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", defaultAccount),
		},
		e2e.GcloudGaLatestWrapperRelease: {"compute", "images", "import", imageName, "--quiet",
			"--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", defaultAccount),
		},
	}

	e2e.RunTestCommandAssertErrorMessage(cmds[testType], argsMap[testType], "Failed to download GCS path", logger, testCase)
}

// With insufficient permissions on default service account, import failed.
func runImageImportOSDefaultServiceAccountWithMissingPermissionsFailTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	testVariables, ok := e2e.GetServiceAccountTestVariables(argMap, false)
	if !ok {
		e2e.Failure(testCase, logger, fmt.Sprintln("Failed to get service account test args"))
		return
	}

	suffix := path.RandString(5)
	imageName := "e2e-test-missing-permission-fail-" + suffix
	defaultAccount := "default"
	zone := "us-east4-c"

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testVariables.ProjectID),
			fmt.Sprintf("-image_name=%v", imageName), "-os=debian-9",
			fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("-zone=%v", zone),
			fmt.Sprintf("-compute_service_account=%v", defaultAccount),
		},
		e2e.GcloudBetaProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", defaultAccount),
		},
		e2e.GcloudBetaLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", defaultAccount),
		},
		e2e.GcloudGaLatestWrapperRelease: {"compute", "images", "import", imageName, "--quiet",
			"--os=debian-9", fmt.Sprintf("--project=%v", testVariables.ProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", zone),
			fmt.Sprintf("--compute_service_account=%v", defaultAccount),
		},
	}

	e2e.RunTestCommandAssertErrorMessage(cmds[testType], argsMap[testType], "Failed to download GCS path", logger, testCase)
}

func runImportTest(ctx context.Context, args []string, testType e2e.CLITestType,
	projectID string, imageName string, logger *log.Logger, testCase *junitxml.TestCase) {

	runImportTestWithExtraParams(ctx, args, testType, projectID, imageName, logger, testCase, "", "", nil)
}

func runImportTestWithExtraParams(ctx context.Context, args []string, testType e2e.CLITestType,
	projectID string, imageName string, logger *log.Logger, testCase *junitxml.TestCase,
	expectedFamily string, expectedDescription string, expectedLabels []string) {

	// "family", "description" and "labels" hasn't been supported by gcloud
	if testType != e2e.Wrapper {
		expectedFamily = ""
		expectedDescription = ""
		expectedLabels = nil
	}

	if e2e.RunTestForTestType(cmds[testType], args, testType, logger, testCase) {
		verifyImportedImage(ctx, testCase, projectID, imageName, logger, expectedFamily,
			expectedDescription, expectedLabels)
	}
}

func verifyImportedImage(ctx context.Context, testCase *junitxml.TestCase,
	projectID string, imageName string, logger *log.Logger,
	expectedFamily string, expectedDescription string, expectedLabels []string) {

	logger.Printf("Verifying imported image...")
	image, err := gcp.CreateImageObject(ctx, projectID, imageName)
	if err != nil {
		testCase.WriteFailure("Image '%v' doesn't exist after import: %v", imageName, err)
		logger.Printf("Image '%v' doesn't exist after import: %v", imageName, err)
		return
	}
	logger.Printf("Image '%v' exists! Import success.", imageName)

	if expectedFamily != "" && image.Family != expectedFamily {
		e2e.Failure(testCase, logger, fmt.Sprintf("Image '%v' family expect: %v, actual: %v", imageName, expectedFamily, image.Family))
	}

	if expectedDescription != "" && image.Description != expectedDescription {
		e2e.Failure(testCase, logger, fmt.Sprintf("Image '%v' description expect: %v, actual: %v", imageName, expectedDescription, image.Description))
	}

	if expectedLabels != nil {
		imageLabels := make([]string, 0, len(image.Labels))
		for k, v := range image.Labels {
			imageLabels = append(imageLabels, k+"="+v)
		}
		e2e.ContainsAll(imageLabels, expectedLabels, testCase, logger,
			fmt.Sprintf("Image '%v' labels expect: %v, actual: %v", imageName, strings.Join(expectedLabels, ","), strings.Join(imageLabels, ",")))
	}

	if err := image.Cleanup(); err != nil {
		logger.Printf("Image '%v' failed to clean up.", imageName)
	} else {
		logger.Printf("Image '%v' cleaned up.", imageName)
	}
}
