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

package importtestsuites

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/assert"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/utils"
)

const (
	suite = "ImageImport"
)

type testCase struct {
	caseName string

	// Imported image name.
	imageName string

	// Specify either image for file.
	source string

	// This is passed directly to the `--os` flag of gce_vm_image_import.
	os string

	// When empty, the test is expected to pass. When non-empty, the
	// actual error message must contain this for the test to pass.
	expectedError string

	// Additional args passed to gce_vm_image_import.
	extraArgs []string

	// Whether the image-under-test is expected to have the OS config agent installed.
	osConfigNotSupported bool

	// Expect to see all given strings in guestOsFeatures
	expectedGuestOsFeatures   []string

	// Expect to see none of given strings in guestOsFeatures
	unexpectedGuestOsFeatures []string

	// Whether to add "inspect" arg
	inspect bool
}

var basicCases = []*testCase{
	{
		caseName: "debian-9",
		source:   "projects/compute-image-tools-test/global/images/debian-9-translate",
		os:       "debian-9",
	}, {
		caseName:             "ubuntu-1404",
		source:               "projects/compute-image-tools-test/global/images/ubuntu-1404-img-import",
		os:                   "ubuntu-1404",
		osConfigNotSupported: true,
		inspect:              true,
	}, {
		caseName:             "ubuntu-1604",
		source:               "projects/compute-image-tools-test/global/images/ubuntu-1604-vmware-import",
		os:                   "ubuntu-1604",
		osConfigNotSupported: true,
		inspect:              true,
	}, {
		caseName:             "ubuntu-1804",
		source:               "gs://compute-image-tools-test-resources/ubuntu-1804-vmware.vmdk",
		os:                   "ubuntu-1804",
		osConfigNotSupported: true,
		inspect:              true,
	}, {
		caseName:             "ubuntu-2004",
		source:               "projects/compute-image-tools-test/global/images/ubuntu-2004",
		os:                   "ubuntu-2004",
		osConfigNotSupported: true,
		inspect:              true,
	}, {
		caseName:             "ubuntu-2004-aws",
		source:               "projects/compute-image-tools-test/global/images/ubuntu-2004-aws",
		os:                   "ubuntu-2004",
		osConfigNotSupported: true,
		inspect:              true,
	}, {
		caseName:      "incorrect OS specified",
		source:        "projects/compute-image-tools-test/global/images/debian-9-translate",
		os:            "opensuse-15",
		expectedError: "\"debian-9\" was detected on your disk, but \"opensuse-15\" was specified",
		inspect:       true,
	},
	// EL
	{
		caseName: "el-centos-7-8",
		source:   "projects/compute-image-tools-test/global/images/centos-7-8",
		os:       "centos-7",
		inspect:  true,
	}, {
		caseName: "el-centos-8-0",
		source:   "projects/compute-image-tools-test/global/images/centos-8-import",
		os:       "centos-8",
		inspect:  true,
	}, {
		caseName: "el-centos-8-2",
		source:   "projects/compute-image-tools-test/global/images/centos-8-2",
		os:       "centos-8",
		inspect:  true,
	}, {
		caseName:  "el-rhel-7-uefi",
		source:    "projects/compute-image-tools-test/global/images/linux-uefi-no-guestosfeature-rhel7",
		os:        "rhel-7",
		extraArgs: []string{"-uefi_compatible=true"},
		inspect:   true,
	}, {
		caseName: "el-rhel-7-8",
		source:   "projects/compute-image-tools-test/global/images/rhel-7-8",
		os:       "rhel-7",
		inspect:  true,
	}, {
		caseName: "el-rhel-8-0",
		source:   "projects/compute-image-tools-test/global/images/rhel-8-0",
		os:       "rhel-8",
		inspect:  true,
	}, {
		caseName: "el-rhel-8-2",
		source:   "projects/compute-image-tools-test/global/images/rhel-8-2",
		os:       "rhel-8",
		inspect:  true,
	}, {
		caseName:  "windows-2019-uefi",
		source:    "projects/compute-image-tools-test/global/images/windows-2019-uefi-nodrivers",
		os:        "windows-2019",
		extraArgs: []string{"-uefi_compatible=true"},
		inspect:   true,
	}, {
		caseName: "windows-10-x86-byol",
		source:   "projects/compute-image-tools-test/global/images/windows-10-1909-ent-x86-nodrivers",
		os:       "windows-10-x86-byol",
		inspect:  true,
	},
}

var inspectUEFICases = []*testCase{
	{
		caseName: "inspect-uefi-linux-uefi-rhel-7",
		// source created from projects/gce-uefi-images/global/images/rhel-7-v20200403
		source:                  "gs://compute-image-tools-test-resources/uefi/linux-uefi-rhel-7.vmdk",
		os:                      "rhel-7",
		expectedGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-linux-uefi-rhel-7-from-image",
		// image created from projects/gce-uefi-images/global/images/rhel-7-v20200403 and removed UEFI_COMPATIBLE
		source:                  "projects/compute-image-tools-test/global/images/linux-uefi-no-guestosfeature-rhel7",
		os:                      "rhel-7",
		expectedGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-linux-nonuefi-debian-9",
		// source created from projects/debian-cloud/global/images/debian-9-stretch-v20200714
		source:                    "gs://compute-image-tools-test-resources/uefi/linux-nonuefi-debian-9.vmdk",
		os:                        "debian-9",
		unexpectedGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-linux-hybrid-ubuntu-1804",
		// source created from projects/gce-uefi-images/global/images/ubuntu-1804-bionic-v20200317
		source:                  "gs://compute-image-tools-test-resources/uefi/linux-hybrid-ubuntu-1804.vmdk",
		os:                      "ubuntu-1804",
		osConfigNotSupported:    true,
		expectedGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-linux-mbr-uefi-rhel-7",
		// source created from projects/gce-uefi-images/global/images/ubuntu-1804-bionic-v20200317 and converted from GPT to MBR
		source:                  "gs://compute-image-tools-test-resources/uefi/linux-ubuntu-mbr-uefi.vmdk",
		os:                      "ubuntu-1804",
		osConfigNotSupported:    true,
		expectedGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-windows-uefi-windows",
		// source created from projects/gce-uefi-images/global/images/windows-server-2019-dc-core-v20200609
		source:                  "gs://compute-image-tools-test-resources/uefi/windows-uefi-2019.vmdk",
		os:                      "windows-2019",
		expectedGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-windows-nonuefi-windows",
		// source created from projects/windows-cloud/global/images/windows-server-2019-dc-v20200114
		source:                    "gs://compute-image-tools-test-resources/uefi/windows-nonuefi-2019.vmdk",
		os:                        "windows-2019",
		unexpectedGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	},
}

func (t *testCase) run(ctx context.Context, junit *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	start := time.Now()
	logger = t.createTestScopedLogger(junit, logger)
	t.imageName = "e2e-test-image-import" + path.RandString(5)
	imagePath := fmt.Sprintf("projects/%s/global/images/%s", testProjectConfig.TestProjectID, t.imageName)

	importLogs, err := t.runImport(junit, logger, testProjectConfig, t.imageName)

	if t.expectedError != "" {
		t.verifyExpectedError(junit, err, importLogs)
	} else if err != nil {
		t.writeImportFailed(junit, importLogs)
	} else {
		t.verifyImage(ctx, junit, logger, testProjectConfig)
		err = t.runPostTranslateTest(ctx, imagePath, testProjectConfig, logger)
		if err != nil {
			junit.WriteFailure("Failed post translate test: %v", err)
		}
	}
	junit.Time = time.Now().Sub(start).Seconds()
}

func (t *testCase) verifyImage(ctx context.Context, junit *junitxml.TestCase, logger *log.Logger, testProjectConfig *testconfig.Project) {
	logger.Printf("Verifying imported image...")
	image, err := compute.CreateImageObject(ctx, testProjectConfig.TestProjectID, t.imageName)
	if err != nil {
		junit.WriteFailure("Image '%v' doesn't exist after import: %v", t.imageName, err)
		logger.Printf("Image '%v' doesn't exist after import: %v", t.imageName, err)
		return
	}
	logger.Printf("Image '%v' exists! Import success.", t.imageName)

	assert.GuestOSFeatures(t.expectedGuestOsFeatures, t.unexpectedGuestOsFeatures, image.GuestOsFeatures, junit, logger)
}

// createTestScopedLogger returns a new logger that is prefixed with the name of the test.
func (t testCase) createTestScopedLogger(junit *junitxml.TestCase, logger *log.Logger) *log.Logger {
	return log.New(logger.Writer(), junit.Name+" ", logger.Flags())
}

// runImport runs an image import workflow, and returns its console output and error.
func (t testCase) runImport(junit *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, imageName string) (*bytes.Buffer, error) {
	args := []string{
		"-client_id", "e2e",
		"-os", t.os,
		"-project", testProjectConfig.TestProjectID,
		"-zone", testProjectConfig.TestZone,
		"-image_name", imageName,
	}
	if t.inspect {
		args = append(args, "-inspect")
	}
	if strings.Contains(t.source, "gs://") {
		args = append(args, "-source_file", t.source)
	} else {
		args = append(args, "-source_image", t.source)
	}
	if t.extraArgs != nil {
		args = append(args, t.extraArgs...)
	}
	cmd := exec.Command("./gce_vm_image_import", args...)
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = io.MultiWriter(cmdOutput, logging.AsWriter(logger))
	cmd.Stderr = io.MultiWriter(cmdOutput, logging.AsWriter(logger))
	err := cmd.Start()
	if err != nil {
		return cmdOutput, err
	}
	return cmdOutput, cmd.Wait()
}

func (t testCase) writeImportFailed(junit *junitxml.TestCase, importLog *bytes.Buffer) {
	// The Logf messages are shown in the test framework UI.
	junit.Logf(importLog.String())
	junit.WriteFailure("Import failed: %q", getLastLine(importLog))
}

func (t testCase) verifyExpectedError(junit *junitxml.TestCase, actualErr error, importLog *bytes.Buffer) {
	if actualErr == nil {
		junit.WriteFailure("Expected failed import: %q.", t.expectedError)
		return
	}

	lastLine := getLastLine(importLog)
	if !strings.Contains(lastLine, t.expectedError) {
		junit.WriteFailure("Message shown to user %q did not contain %q.",
			lastLine, t.expectedError)
	}
}

// runPostTranslateTest boots the instance and executes a startup script containing tests.
func (t testCase) runPostTranslateTest(ctx context.Context, imagePath string,
	testProjectConfig *testconfig.Project, logger *log.Logger) error {
	wf, err := daisy.NewFromFile("post_translate_test.wf.json")
	if err != nil {
		return err
	}

	varMap := map[string]string{
		"image_under_test":            imagePath,
		"path_to_post_translate_test": t.testScript(),
		"osconfig_not_supported":      strconv.FormatBool(t.osConfigNotSupported),
	}

	for k, v := range varMap {
		wf.AddVar(k, v)
	}

	wf.Logger = logging.AsDaisyLogger(logger)
	wf.Project = testProjectConfig.TestProjectID
	wf.Zone = testProjectConfig.TestZone
	err = wf.Run(ctx)
	return err
}

func (t testCase) testScript() string {
	if strings.Contains(t.os, "windows") {
		return "post_translate_test.ps1"
	}
	return "post_translate_test.sh"
}

func getAllTestCases() []*testCase {
	var cases []*testCase
	cases = append(cases, basicCases...)
	cases = append(cases, inspectUEFICases...)
	return cases
}

// ImageImportSuite performs image imports, and verifies that the results are bootable and are
// are able to perform basic GCP operations. The suite includes support for negative test cases,
// where error messages are validated against expected error messages.
func ImageImportSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	junits := map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, utils.CLITestType){}

	cases := getAllTestCases()
	for _, tc := range cases {
		testCase := *tc
		junit := junitxml.NewTestCase(
			suite, fmt.Sprintf("[%v]", tc.caseName))
		junits[junit] = testCase.run
	}

	testsMap := map[utils.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, utils.CLITestType){
		utils.Wrapper: junits,
	}

	utils.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, suite, testsMap)
}

// getLastLine consumes a buffer, and returns its last line.
func getLastLine(buffer *bytes.Buffer) string {
	var lastLine string
	scanner := bufio.NewScanner(buffer)
	for scanner.Scan() {
		lastLine = scanner.Text()
	}
	return lastLine
}
