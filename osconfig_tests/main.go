//  Copyright 2018 Google Inc. All Rights Reserved.
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

package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/gcp_clients"

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_suites/inventory"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_suites/package_management"
)

var (
	testSuiteFilter = flag.String("test_suite_filter", "", "test suite filter")
	testCaseFilter  = flag.String("test_case_filter", "", "test case filter")
	outDir          = flag.String("out_dir", "/tmp", "junit xml directory")
	testProjectID   = flag.String("test_project_id", "", "test project id")
	testZone        = flag.String("test_zone", "", "test zone")
)

var testFunctions = []func(context.Context, *sync.WaitGroup, chan *junitxml.TestSuite, *log.Logger, *regexp.Regexp, *regexp.Regexp, *testconfig.Project){
	packagemanagement.TestSuite,
	inventory.TestSuite,
}

func main() {
	flag.Parse()
	ctx := context.Background()

	config.SetConfig()
	gcpclients.PopulateClients(ctx)

	if len(strings.TrimSpace(*testProjectID)) == 0 {
		fmt.Println("-test_project_id is invalid")
		os.Exit(1)
	}

	// TODO: check if the zone is valid, fail fast
	if len(strings.TrimSpace(*testZone)) == 0 {
		fmt.Println("-test_zone is invalid")
		os.Exit(1)
	}

	pr := testconfig.GetProject(*testProjectID, *testZone)

	var testSuiteRegex *regexp.Regexp
	if *testSuiteFilter != "" {
		var err error
		testSuiteRegex, err = regexp.Compile(*testSuiteFilter)
		if err != nil {
			fmt.Println("-testSuiteFilter flag not valid:", err)
			os.Exit(1)
		}
	}

	var testCaseRegex *regexp.Regexp
	if *testCaseFilter != "" {
		var err error
		testCaseRegex, err = regexp.Compile(*testCaseFilter)
		if err != nil {
			fmt.Println("-testCaseFilter flag not valid:", err)
			os.Exit(1)
		}
	}

	logger := log.New(os.Stdout, "[OsConfigTests] ", 0)
	logger.Println("Starting...")

	tests := make(chan *junitxml.TestSuite)
	var wg sync.WaitGroup
	for _, tf := range testFunctions {
		wg.Add(1)
		go tf(ctx, &wg, tests, logger, testSuiteRegex, testCaseRegex, pr)
	}
	go func() {
		wg.Wait()
		close(tests)
	}()

	var testSuites []*junitxml.TestSuite
	for ret := range tests {
		testSuites = append(testSuites, ret)
		testSuiteOutPath := filepath.Join(*outDir, fmt.Sprintf("junit_%s.xml", ret.Name))
		if err := os.MkdirAll(filepath.Dir(testSuiteOutPath), 0770); err != nil {
			log.Fatal(err)
		}

		logger.Printf("Creating junit xml file: %s", testSuiteOutPath)
		d, err := xml.MarshalIndent(ret, "  ", "   ")
		if err != nil {
			log.Fatal(err)
		}

		if err := ioutil.WriteFile(testSuiteOutPath, d, 0644); err != nil {
			log.Fatal(err)
		}
	}

	var buf bytes.Buffer
	for _, ts := range testSuites {
		if ts.Failures > 0 {
			buf.WriteString(fmt.Sprintf("TestSuite %q has errors:\n", ts.Name))
			for _, tc := range ts.TestCase {
				if tc.Failure != nil {
					buf.WriteString(fmt.Sprintf(" - %q: %s\n", tc.Name, tc.Failure.FailMessage))
				}
			}

		}
	}

	if buf.Len() > 0 {
		logger.Fatalf("%sExiting with exit code 1", buf.String())
	}
	logger.Print("All test cases completed successfully.")
}
