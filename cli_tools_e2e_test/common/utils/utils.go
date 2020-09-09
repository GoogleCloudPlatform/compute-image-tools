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

// Package utils contains e2e tests utils for cli tools e2e tests
package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

// CLITestType defines which type of test is going to be executed
type CLITestType string

// List all test types here
const (
	Wrapper                   CLITestType = "1 wrapper"
	GcloudProdWrapperLatest   CLITestType = "2 gcloud-prod wrapper-latest"
	GcloudLatestWrapperLatest CLITestType = "3 gcloud-latest wrapper-latest"
)

var (
	gcloudUpdateLock = sync.Mutex{}
)

// CLITestSuite executes given test suite.
func CLITestSuite(ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project, testSuiteName string, testsMap map[CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, CLITestType)) {

	defer tswg.Done()

	if testSuiteRegex != nil && !testSuiteRegex.MatchString(testSuiteName) {
		return
	}

	testSuite := junitxml.NewTestSuite(testSuiteName)
	defer testSuite.Finish(testSuites)
	logger.Printf("Running CLITestSuite %q", testSuite.Name)
	tests := runTestCases(ctx, logger, testCaseRegex, testProjectConfig, testSuite.Name, testsMap)

	for ret := range tests {
		testSuite.TestCase = append(testSuite.TestCase, ret)
	}

	logger.Printf("Finished CLITestSuite %q", testSuite.Name)
}

func runTestCases(ctx context.Context, logger *log.Logger, regex *regexp.Regexp,
	testProjectConfig *testconfig.Project, testSuiteName string, testsMap map[CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, CLITestType)) chan *junitxml.TestCase {

	tests := make(chan *junitxml.TestCase)
	var ttwg sync.WaitGroup
	ttwg.Add(len(testsMap))
	tts := make([]string, 0, len(testsMap))
	for tt := range testsMap {
		tts = append(tts, string(tt))
	}
	sort.Strings(tts)
	go func() {
		for _, ttStr := range tts {
			tt := CLITestType(ttStr)
			m := testsMap[tt]
			logger.Printf("=== Running CLITestSuite %v for test type %v ===", testSuiteName, tt)

			var wg sync.WaitGroup
			for tc, f := range m {
				wg.Add(1)
				go func(ctx context.Context, wg *sync.WaitGroup, tc *junitxml.TestCase, tt CLITestType, f func(
					context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, CLITestType)) {

					defer wg.Done()
					if tc.FilterTestCase(regex) {
						tc.Finish(tests)
					} else {
						defer logger.Printf("TestCase %s.%q finished in %fs", tc.Classname, tc.Name, tc.Time)
						defer tc.Finish(tests)
						logger.Printf("Running TestCase %s.%q", tc.Classname, tc.Name)
						f(ctx, tc, logger, testProjectConfig, tt)
					}
				}(ctx, &wg, tc, tt, f)
			}
			wg.Wait()

			ttwg.Done()
			logger.Printf("=== Finished running CLITestSuite %v for test type %v ===", testSuiteName, tt)
		}
	}()

	go func() {
		ttwg.Wait()
		close(tests)
	}()

	return tests
}

// RunCliTool runs a cli tool with given args
func RunCliTool(logger *log.Logger, testCase *junitxml.TestCase, cmdString string, args []string) error {
	prefix := "Test Env"
	if testCase != nil {
		prefix = testCase.Name
	}
	logger.Printf("[%v] Running command: '%s %s'", prefix, cmdString, strings.Join(args, " "))
	cmd := exec.Command(cmdString, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunTestCommand runs given test command
func RunTestCommand(cmd string, args []string, logger *log.Logger, testCase *junitxml.TestCase) bool {
	if err := RunCliTool(logger, testCase, cmd, args); err != nil {
		Failure(testCase, logger, fmt.Sprintf("Error running cmd: %v\n", err))
		return false
	}
	return true
}

// RunTestCommandIgnoringError runs given test command. The test case won't be marked as fail even error happens.
func RunTestCommandIgnoringError(cmd string, args []string, logger *log.Logger, testCase *junitxml.TestCase) bool {
	if err := RunCliTool(logger, testCase, cmd, args); err != nil {
		logger.Printf("[%v] %v", testCase.Name, fmt.Sprintf("Error running cmd: %v\n", err))
		return false
	}
	return true
}

func runCliToolAsync(logger *log.Logger, testCase *junitxml.TestCase, cmdString string, args []string) (*exec.Cmd, error) {
	logger.Printf("[%v] Running command: '%s %s'", testCase.Name, cmdString, strings.Join(args, " "))
	cmd := exec.Command(cmdString, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	return cmd, err
}

// RunTestCommandAsync runs given test command asynchronously
func RunTestCommandAsync(cmd string, args []string, logger *log.Logger, testCase *junitxml.TestCase) *exec.Cmd {
	cmdPtr, err := runCliToolAsync(logger, testCase, cmd, args)
	if err != nil {
		Failure(testCase, logger, fmt.Sprintf("Error starting cmd: %v\n", err))
		return nil
	}
	return cmdPtr
}

// GcloudAuth runs "gcloud auth"
func GcloudAuth(logger *log.Logger, testCase *junitxml.TestCase) bool {
	// This file exists in test env. For local testing, download a creds file from project
	// compute-image-tools-test.
	credsPath := "/etc/compute-image-tools-test-service-account/creds.json"
	cmd := "gcloud"
	args := []string{"auth", "activate-service-account", "--key-file=" + credsPath}
	if err := RunCliTool(logger, testCase, cmd, args); err != nil {
		Failure(testCase, logger, fmt.Sprintf("Error running cmd: %v\n", err))
		return false
	}
	return true
}

// GcloudUpdate runs "gcloud update" to pull either latest or prod version
func GcloudUpdate(logger *log.Logger, testCase *junitxml.TestCase, latest bool) bool {
	gcloudUpdateLock.Lock()
	defer gcloudUpdateLock.Unlock()

	// auth is required for gcloud updates
	if !GcloudAuth(logger, testCase) {
		return false
	}

	cmd := "gcloud"

	if latest {
		args := []string{"components", "repositories", "add",
			"https://storage.googleapis.com/cloud-sdk-testing/ci/staging/components-2.json", "--quiet"}
		if err := RunCliTool(logger, testCase, cmd, args); err != nil {
			logger.Printf("Error running cmd: %v\n", err)
			testCase.WriteFailure("Error running cmd: %v", err)
			return false
		}
	} else {
		args := []string{"components", "repositories", "remove", "--all"}
		if err := RunCliTool(logger, testCase, cmd, args); err != nil {
			logger.Printf("Error running cmd: %v\n", err)
			testCase.WriteFailure("Error running cmd: %v", err)
			return false
		}
	}

	args := []string{"components", "update", "--quiet"}
	if err := RunCliTool(logger, testCase, cmd, args); err != nil {
		logger.Printf("Error running cmd: %v\n", err)
		testCase.WriteFailure("Error running cmd: %v", err)
		return false
	}

	// an additional auth is required if updated through a different repository
	if !GcloudAuth(logger, testCase) {
		return false
	}

	return true
}

// RunTestForTestType runs test for given test type
func RunTestForTestType(cmd string, args []string, testType CLITestType, logger *log.Logger, testCase *junitxml.TestCase) bool {
	switch testType {
	case Wrapper:
		if !RunTestCommand(cmd, args, logger, testCase) {
			return false
		}
	case GcloudProdWrapperLatest:
		if !GcloudUpdate(logger, testCase, false) {
			return false
		}
		if !RunTestCommand(cmd, args, logger, testCase) {
			return false
		}
	case GcloudLatestWrapperLatest:
		if !GcloudUpdate(logger, testCase, true) {
			return false
		}
		if !RunTestCommand(cmd, args, logger, testCase) {
			return false
		}
	}
	return true
}

// Failure logs failure message to both test case output and logger.
func Failure(testCase *junitxml.TestCase, logger *log.Logger, msg string) {
	prefix := "Test Env"
	if testCase != nil {
		prefix = testCase.Name
		testCase.WriteFailure(msg)
	}
	logger.Printf("[%v] %v", prefix, msg)
}

// ContainsSubString checks whether the string slice contains a substring anywhere.
func ContainsSubString(strs []string, s string) bool {
	for _, str := range strs {
		if strings.Contains(str, s) {
			return true
		}
	}
	return false
}
