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

// Package testsuiteutils contains e2e tests utils for image import/export cli tools
package testsuiteutils

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2etestutils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2etestutils/test_config"
)

// TestSuite is image import test suite.
func TestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project, testSuiteName string, testsMap map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project)) {

	defer tswg.Done()

	if testSuiteRegex != nil && !testSuiteRegex.MatchString(testSuiteName) {
		return
	}

	testSuite := junitxml.NewTestSuite(testSuiteName)
	defer testSuite.Finish(testSuites)
	logger.Printf("Running TestSuite %q", testSuite.Name)
	tests := runTestCases(ctx, logger, testCaseRegex, testProjectConfig, testsMap)

	for ret := range tests {
		testSuite.TestCase = append(testSuite.TestCase, ret)
	}

	logger.Printf("Finished TestSuite %q", testSuite.Name)
}

func runTestCases(
	ctx context.Context, logger *log.Logger, regex *regexp.Regexp,
	testProjectConfig *testconfig.Project, testsMap map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project)) chan *junitxml.TestCase {

	var wg sync.WaitGroup
	tests := make(chan *junitxml.TestCase)
	for tc, f := range testsMap {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, tc *junitxml.TestCase, f func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project)) {
			defer wg.Done()
			if tc.FilterTestCase(regex) {
				tc.Finish(tests)
			} else {
				defer tc.Finish(tests)
				logger.Printf("Running TestCase %s.%q", tc.Classname, tc.Name)
				f(ctx, tc, logger, testProjectConfig)
				logger.Printf("TestCase %s.%q finished in %fs", tc.Classname, tc.Name, tc.Time)
			}
		}(ctx, &wg, tc, f)
	}

	go func() {
		wg.Wait()
		close(tests)
	}()

	return tests
}

// RunCliTool executes cli tool with params
func RunCliTool(logger *log.Logger, testCase *junitxml.TestCase, cmdString string, args []string) {
	logger.Printf("[%v] Running command: '%s %s'", testCase.Name, cmdString, strings.Join(args, " "))
	cmd := exec.Command(fmt.Sprintf("./%s", cmdString), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logger.Fatalf("Error running cmd: %v\n", err.Error())
	}
}
