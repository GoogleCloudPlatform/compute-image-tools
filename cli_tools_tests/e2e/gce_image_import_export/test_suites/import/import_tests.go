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
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e"
	"github.com/GoogleCloudPlatform/compute-image-tools/common/gcp"
	"github.com/GoogleCloudPlatform/compute-image-tools/common/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	suite           = "ImageImport"
	slesOnDemandTip = "For on-demand conversion failures, see daisy_workflows/image_import/suse/suse_import/README.md"
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
	requiredGuestOsFeatures []string

	expectLicense string

	// Expect to see none of given strings in guestOsFeatures
	notAllowedGuestOsFeatures []string

	// Additional information to show on failure to assist with debugging.
	tip string
}

var basicCases = []*testCase{
	// Disk image formats
	{
		caseName: "vhd-ubuntu-1804",
		source:   "gs://compute-image-tools-test-resources/ubuntu-1804-azure.vhd",
		os:       "ubuntu-1804",
	},
	{
		caseName: "vpc-file-format-ubuntu-1804",
		source:   "gs://compute-image-tools-test-resources/ubuntu-1804.vpc",
		os:       "ubuntu-1804",
	},

	// Error messages
	{
		caseName:      "incorrect OS specified",
		source:        "projects/compute-image-tools-test/global/images/debian-9-translate",
		os:            "opensuse-15",
		expectedError: "\"debian-9\" was detected on your disk, but \"opensuse-15\" was specified",
	},
	{
		caseName:      "SLES import with no OS on disk",
		source:        "gs://compute-image-tools-test-resources/empty-10gb.qcow2",
		os:            "opensuse-15",
		expectedError: "no operating systems found",
	},

	// Debian
	{
		caseName: "debian-9",
		source:   "projects/compute-image-tools-test/global/images/debian-9-translate",
		os:       "debian-9",
	},
	{
		caseName: "debian-10",
		source:   "projects/compute-image-tools-test/global/images/debian-10",
		os:       "debian-10",
	},
	{
		caseName: "debian-11",
		source:   "projects/compute-image-tools-test/global/images/debian-11",
	},

	// Ubuntu
	{
		caseName:             "ubuntu-1404",
		source:               "projects/compute-image-tools-test/global/images/ubuntu-1404-img-import",
		os:                   "ubuntu-1404",
		osConfigNotSupported: true,
	}, {
		caseName: "ubuntu-1604",
		source:   "projects/compute-image-tools-test/global/images/ubuntu-1604-vmware-import",
		os:       "ubuntu-1604",
	}, {
		caseName: "ubuntu-1804",
		source:   "gs://compute-image-tools-test-resources/ubuntu-1804-vmware.vmdk",
		os:       "ubuntu-1804",
	}, {
		caseName: "ubuntu-2004",
		source:   "projects/compute-image-tools-test/global/images/ubuntu-2004",
		os:       "ubuntu-2004",
	}, {
		caseName: "ubuntu-2004-aws",
		source:   "projects/compute-image-tools-test/global/images/ubuntu-2004-aws",
		os:       "ubuntu-2004",
	},

	// OpenSUSE
	{
		caseName:             "opensuse-15-1",
		source:               "projects/compute-image-tools-test/global/images/opensuse-15-1",
		os:                   "opensuse-15",
		osConfigNotSupported: true,
	},
	{
		caseName:             "opensuse-15-2",
		source:               "projects/compute-image-tools-test/global/images/opensuse-15-2",
		os:                   "opensuse-15",
		osConfigNotSupported: true,
	},

	// SLES: BYOL
	// Uses a mixture of tactics for specifying BYOL, all of which should be successful.
	{
		caseName:             "sles-12-5-byol",
		source:               "projects/compute-image-tools-test/global/images/sles-12-5-registered",
		expectLicense:        "https://www.googleapis.com/compute/v1/projects/suse-byos-cloud/global/licenses/sles-12-byos",
		extraArgs:            []string{"-byol"}, // -byol with OS detection
		osConfigNotSupported: true,
	}, {
		caseName:             "sles-sap-12-5-byol",
		source:               "projects/compute-image-tools-test/global/images/sles-sap-12-5-registered",
		os:                   "sles-sap-12-byol",
		extraArgs:            []string{"-byol"}, // -byol specified when not required
		osConfigNotSupported: true,
	}, {
		caseName:             "sles-15-2-byol",
		source:               "projects/compute-image-tools-test/global/images/sles-15-2-registered",
		os:                   "sles-15",
		extraArgs:            []string{"-byol"}, // -byol transforms sles-15 to sles-15-byol
		osConfigNotSupported: true,
	}, {
		caseName:             "sles-sap-15-2-byol",
		source:               "projects/compute-image-tools-test/global/images/sles-sap-15-2-registered",
		os:                   "sles-sap-15-byol", // No -byol flag
		osConfigNotSupported: true,
	},

	// SLES: On-demand
	{
		caseName:             "sles-12-4-on-demand",
		source:               "projects/compute-image-tools-test/global/images/sles-12-4-unregistered",
		os:                   "sles-12",
		osConfigNotSupported: true,
		tip:                  slesOnDemandTip,
	}, {
		caseName:             "sles-12-5-on-demand",
		source:               "projects/compute-image-tools-test/global/images/sles-12-5-unregistered",
		os:                   "sles-12",
		osConfigNotSupported: true,
		tip:                  slesOnDemandTip,
	}, {
		caseName:             "sles-sap-12-4-on-demand",
		source:               "projects/compute-image-tools-test/global/images/sles-sap-12-4-unregistered",
		os:                   "sles-sap-12",
		osConfigNotSupported: true,
		tip:                  slesOnDemandTip,
	}, {
		caseName:             "sles-sap-12-5-on-demand",
		source:               "projects/compute-image-tools-test/global/images/sles-sap-12-5-unregistered",
		os:                   "sles-sap-12",
		osConfigNotSupported: true,
		tip:                  slesOnDemandTip,
	}, {
		caseName:             "sles-15-0-on-demand",
		source:               "projects/compute-image-tools-test/global/images/sles-15-0-unregistered",
		os:                   "sles-15",
		osConfigNotSupported: true,
		tip:                  slesOnDemandTip,
	}, {
		caseName:             "sles-15-1-on-demand",
		source:               "projects/compute-image-tools-test/global/images/sles-15-1-unregistered",
		os:                   "sles-15",
		osConfigNotSupported: true,
		tip:                  slesOnDemandTip,
	}, {
		caseName:             "sles-15-2-on-demand",
		source:               "projects/compute-image-tools-test/global/images/sles-15-2-unregistered",
		os:                   "sles-15",
		osConfigNotSupported: true,
		tip:                  slesOnDemandTip,
	}, {
		caseName:             "sles-15-3-on-demand",
		source:               "projects/compute-image-tools-test/global/images/sles-15-3-unregistered",
		os:                   "sles-15",
		osConfigNotSupported: true,
		tip:                  slesOnDemandTip,
	}, {
		caseName:             "sles-sap-15-0-on-demand",
		source:               "projects/compute-image-tools-test/global/images/sles-sap-15-0-unregistered",
		os:                   "sles-sap-15",
		osConfigNotSupported: true,
		tip:                  slesOnDemandTip,
	}, {
		caseName:             "sles-sap-15-1-on-demand",
		source:               "projects/compute-image-tools-test/global/images/sles-sap-15-1-unregistered",
		os:                   "sles-sap-15",
		osConfigNotSupported: true,
		tip:                  slesOnDemandTip,
	}, {
		caseName:             "sles-sap-15-2-on-demand",
		source:               "projects/compute-image-tools-test/global/images/sles-sap-15-2-unregistered",
		os:                   "sles-sap-15",
		osConfigNotSupported: true,
		tip:                  slesOnDemandTip,
	},

	// EL
	{
		caseName: "el-centos-7-8",
		source:   "projects/compute-image-tools-test/global/images/centos-7-8",
		os:       "centos-7",
	}, {
		caseName: "el-centos-8-0",
		source:   "projects/compute-image-tools-test/global/images/centos-8-import",
		os:       "centos-8",
	}, {
		caseName: "el-centos-8-3",
		source:   "projects/compute-image-tools-test/global/images/centos-8-3",
		os:       "centos-8",
	}, {
		caseName: "el-rhel-6-10",
		source:   "projects/compute-image-tools-test/global/images/rhel-6-10",
		os:       "rhel-6",
	}, {
		caseName:  "el-rhel-7-uefi",
		source:    "projects/compute-image-tools-test/global/images/linux-uefi-no-guestosfeature-rhel7",
		os:        "rhel-7",
		extraArgs: []string{"-uefi_compatible=true"},
	}, {
		caseName: "el-rhel-7-8",
		source:   "projects/compute-image-tools-test/global/images/rhel-7-8",
		os:       "rhel-7",
	}, {
		caseName: "el-rhel-8-0",
		source:   "projects/compute-image-tools-test/global/images/rhel-8-0",
		os:       "rhel-8",
	}, {
		caseName: "el-rhel-8-2",
		source:   "projects/compute-image-tools-test/global/images/rhel-8-2",
		os:       "rhel-8",
	}, {
		caseName: "el-rocky-8-4",
		source:   "projects/compute-image-tools-test/global/images/rocky-8-4",
	},

	// EL - Error cases
	{
		// Don't fail when /lib/modules contains directories other than kernel modules.
		// Spec: http://linux-training.be/sysadmin/ch28.html#idp68123376
		// Bug: b/168774581
		caseName: "el-allow-extra-dirs-in-lib-modules",
		source:   "projects/compute-image-tools-test/global/images/el-depmod-extra-lib-modules",
		os:       "centos-8",
	}, {
		// Fail when a package isn't found, and alert user with useful message.
		caseName:      "el-package-not-found",
		source:        "projects/compute-image-tools-test/global/images/centos-7-missing-repo",
		os:            "centos-7",
		expectedError: "There are no enabled repos",
	}, {
		// Fail when yum has an unreachable repo.
		caseName:      "el-unreachable-repos",
		source:        "projects/compute-image-tools-test/global/images/centos-8-cdrom-repo",
		os:            "centos-8",
		expectedError: "Ensure all configured repos are reachable",
	}, {
		// Fail when imported as RHEL BYOL, but image does not have valid subscription.
		caseName: "rhel-byol-without-subscription",
		source:   "projects/compute-image-tools-test/global/images/rhel-8-0",
		os:       "rhel-8-byol",
		expectedError: "subscription-manager did not find an active subscription.*" +
			"Omit `-byol` to register with on-demand licensing.",
	}, {
		// Fail when `yum` not found.
		caseName:      "el-yum-not-found",
		source:        "projects/compute-image-tools-test/global/images/manjaro",
		os:            "centos-8",
		expectedError: "Verify the disk's OS: `yum` not found.",
	},

	// Windows
	{
		caseName:                "windows-2019-uefi",
		source:                  "projects/compute-image-tools-test/global/images/windows-2019-uefi-nodrivers",
		expectLicense:           "https://www.googleapis.com/compute/v1/projects/windows-cloud/global/licenses/windows-server-2019-dc",
		requiredGuestOsFeatures: []string{"WINDOWS"},
	}, {
		caseName: "windows-10-x86-byol",
		source:   "projects/compute-image-tools-test/global/images/windows-10-1909-ent-x86-nodrivers",
		os:       "windows-10-x86-byol",
	},
}

var inspectUEFICases = []*testCase{
	{
		caseName: "inspect-uefi-linux-uefi-rhel-7",
		// source created from projects/gce-uefi-images/global/images/rhel-7-v20200403
		source:                  "gs://compute-image-tools-test-resources/uefi/linux-uefi-rhel-7.vmdk",
		os:                      "rhel-7",
		requiredGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-linux-uefi-rhel-7-from-image",
		// image created from projects/gce-uefi-images/global/images/rhel-7-v20200403 and removed UEFI_COMPATIBLE
		source:                  "projects/compute-image-tools-test/global/images/linux-uefi-no-guestosfeature-rhel7",
		os:                      "rhel-7",
		requiredGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-linux-nonuefi-debian-9",
		// source created from projects/debian-cloud/global/images/debian-9-stretch-v20200714
		source:                    "gs://compute-image-tools-test-resources/uefi/linux-nonuefi-debian-9.vmdk",
		os:                        "debian-9",
		notAllowedGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-linux-dual-protective-mbr-ubuntu-1804",
		// source created from projects/gce-uefi-images/global/images/ubuntu-1804-bionic-v20200317
		source:                    "gs://compute-image-tools-test-resources/uefi/linux-protective-mbr-ubuntu-1804.vmdk",
		os:                        "ubuntu-1804",
		notAllowedGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-linux-dual-hybrid-mbr-ubuntu-2004",
		// source created from scratch
		source:                    "gs://compute-image-tools-test-resources/uefi/linux-hybrid-mbr-ubuntu-2004.vmdk",
		os:                        "ubuntu-2004",
		notAllowedGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-linux-uefi-mbr-ubuntu-1804",
		// source created from projects/gce-uefi-images/global/images/ubuntu-1804-bionic-v20200317 and converted from GPT to MBR
		source:                  "gs://compute-image-tools-test-resources/uefi/linux-ubuntu-mbr-uefi.vmdk",
		os:                      "ubuntu-1804",
		requiredGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-windows-uefi",
		// source created from projects/gce-uefi-images/global/images/windows-server-2019-dc-core-v20200609
		source:                  "gs://compute-image-tools-test-resources/uefi/windows-uefi-2019.vmdk",
		os:                      "windows-2019",
		requiredGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	}, {
		caseName: "inspect-uefi-windows-nonuefi",
		// source created from projects/windows-cloud/global/images/windows-server-2019-dc-v20200114
		source:                    "gs://compute-image-tools-test-resources/uefi/windows-nonuefi-2019.vmdk",
		os:                        "windows-2019",
		notAllowedGuestOsFeatures: []string{"UEFI_COMPATIBLE"},
	},
}

func (t *testCase) run(ctx context.Context, junit *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	start := time.Now()
	logger = t.createTestScopedLogger(junit, logger)
	t.imageName = "e2e-test-image-import" + path.RandString(5)
	imagePath := fmt.Sprintf("projects/%s/global/images/%s", testProjectConfig.TestProjectID, t.imageName)

	importLogs, err := t.runImport(junit, logger, testProjectConfig, t.imageName)

	if importLogs == nil {
		panic("Expected importLogs to be non-nil")
	}

	if t.expectedError != "" {
		t.verifyExpectedError(junit, err, importLogs)
	} else if err != nil {
		junit.Logf(importLogs.String())
		t.writeFailure(junit, "Import failed: %q", getLastLine(importLogs))
	} else {
		t.verifyImage(ctx, junit, logger, testProjectConfig)
		err = t.runPostTranslateTest(ctx, imagePath, testProjectConfig, logger)
		if err != nil {
			t.writeFailure(junit, "Failed post translate test: %v", err)
		}
	}
	junit.Time = time.Now().Sub(start).Seconds()
}

func (t *testCase) verifyImage(ctx context.Context, junit *junitxml.TestCase, logger *log.Logger, testProjectConfig *testconfig.Project) {
	logger.Printf("Verifying imported image...")
	image, err := gcp.CreateImageObject(ctx, testProjectConfig.TestProjectID, t.imageName)
	if err != nil {
		junit.Logf("Image '%v' doesn't exist after import: %v", t.imageName, err)
		t.writeFailure(junit, "Image '%v' doesn't exist after import: %v", t.imageName, err)
		return
	}
	logger.Printf("Image '%v' exists! Import finished.", t.imageName)

	e2e.GuestOSFeatures(t.requiredGuestOsFeatures, t.notAllowedGuestOsFeatures, image.GuestOsFeatures, junit, logger)
	if t.expectLicense != "" {
		e2e.ContainsAll(image.Licenses, []string{t.expectLicense}, junit, logger,
			fmt.Sprintf("Expected license %s. Actual licenses %v", t.expectLicense, image.Licenses))
	}
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
		"-project", testProjectConfig.TestProjectID,
		"-zone", testProjectConfig.TestZone,
		"-image_name", imageName,
	}
	if t.os != "" {
		args = append(args, "-os", t.os)
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

func (t testCase) writeFailure(junit *junitxml.TestCase, msg string, args ...interface{}) {
	if t.tip != "" {
		junit.Logf(t.tip)
	}
	junit.WriteFailure(msg, args)
}

func (t testCase) verifyExpectedError(junit *junitxml.TestCase, actualErr error, importLog *bytes.Buffer) {
	if actualErr == nil {
		t.writeFailure(junit, "Expected failed import: %q.", t.expectedError)
		return
	}

	lastLine := getLastLine(importLog)

	if !regexp.MustCompile(t.expectedError).MatchString(lastLine) {
		t.writeFailure(junit, "Message shown to user %q did not contain %q.",
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
	if strings.Contains(t.os, "windows") ||
		strings.Contains(t.imageName, "windows") ||
		strings.Contains(t.caseName, "windows") {
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
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}

	cases := getAllTestCases()
	for _, tc := range cases {
		testCase := *tc
		junit := junitxml.NewTestCase(
			suite, fmt.Sprintf("[%v]", tc.caseName))
		junits[junit] = testCase.run
	}

	testsMap := map[e2e.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){
		e2e.Wrapper: junits,
	}

	e2e.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
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
