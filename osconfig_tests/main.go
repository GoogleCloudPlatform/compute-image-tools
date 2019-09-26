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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/config"
	gcpclients "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/gcp_clients"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_suites/guestpolicies"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_suites/inventory"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_suites/patch"

	_ "google.golang.org/genproto/googleapis/rpc/errdetails"
)

var testFunctions = []func(context.Context, *sync.WaitGroup, chan *junitxml.TestSuite, *log.Logger, *regexp.Regexp, *regexp.Regexp, *testconfig.Project){
	guestpolicies.TestSuite,
	inventory.TestSuite,
	patch.TestSuite,
}

func main() {
	ctx := context.Background()

	pr := testconfig.GetProject(*config.TestProjectID, config.Zones())

	if err := gcpclients.PopulateClients(ctx); err != nil {
		log.Fatal(err)
	}

	logger := log.New(os.Stdout, "[OsConfigTests] ", 0)
	logger.Println("Starting...")

	tests := make(chan *junitxml.TestSuite)
	var wg sync.WaitGroup
	for _, tf := range testFunctions {
		wg.Add(1)
		go tf(ctx, &wg, tests, logger, config.TestSuiteFilter(), config.TestCaseFilter(), pr)
	}
	go func() {
		wg.Wait()
		close(tests)
	}()

	var testSuites []*junitxml.TestSuite
	for ret := range tests {
		testSuites = append(testSuites, ret)
		testSuiteOutPath := filepath.Join(*config.OutDir, fmt.Sprintf("junit_%s.xml", ret.Name))
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
