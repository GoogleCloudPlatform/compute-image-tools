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
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
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

	// Specify either image for file.
	source string

	// This is passed directly to the `--os` flag of gce_vm_image_import.
	os string

	// When empty, the test is expected to pass. When non-empty, the
	// actual error message must contain this for the test to pass.
	expectedError string
}

var cases = []testCase{
	{
		caseName: "debian-9",
		source:   "projects/compute-image-tools-test/global/images/debian-9-translate",
		os:       "debian-9",
	}, {
		caseName:      "incorrect OS specified",
		source:        "projects/compute-image-tools-test/global/images/debian-9-translate",
		os:            "opensuse-15",
		expectedError: "\"debian-9\" was detected on your disk, but \"opensuse-15\" was specified",
	},
}

func (t testCase) run(ctx context.Context, junit *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {
	logger = t.createTestScopedLogger(junit, logger)
	imageName := "e2e-test-image-import" + path.RandString(5)
	imagePath := fmt.Sprintf("projects/%s/global/images/%s", testProjectConfig.TestProjectID, imageName)

	importLogs, err := t.runImport(junit, logger, testProjectConfig, imageName)

	if t.expectedError != "" {
		t.verifyExpectedError(junit, err, importLogs)
	} else if err != nil {
		t.writeImportFailed(junit, importLogs)
	} else {
		err = t.runPostTranslateTest(ctx, imagePath, testProjectConfig, logger)
		if err != nil {
			junit.WriteFailure("Failed post translate test: %v", err)
		}
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
		"-os", t.os,
		"-project", testProjectConfig.TestProjectID,
		"-zone", testProjectConfig.TestZone,
		"-image_name", imageName,
	}
	if strings.Contains(t.source, "gs://") {
		args = append(args, "-source_file", t.source)
	} else {
		args = append(args, "-source_image", t.source)
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
	wf.Vars = map[string]daisy.Var{
		"image_under_test": {
			Value: imagePath,
		},
	}
	wf.Logger = logging.AsDaisyLogger(logger)
	wf.Project = testProjectConfig.TestProjectID
	wf.Zone = testProjectConfig.TestZone
	err = wf.Run(ctx)
	return err
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

	for _, testCase := range cases {
		junit := junitxml.NewTestCase(
			suite, fmt.Sprintf("[%v]", testCase.caseName))
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
