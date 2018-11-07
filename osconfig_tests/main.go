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
	"os/signal"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/junit"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_suites/inventory"
)

const (
	defaultParallelCount = 5
)

var (
	oauth           = flag.String("oauth", "", "path to oauth json file")
	testSuiteFilter = flag.String("test_suite_filter", "", "test suite filter")
	testCaseFilter  = flag.String("test_case_filter", "", "test case filter")
	outPath         = flag.String("out_path", "junit.xml", "junit xml path")
)

var testFunctions = []func(context.Context, *sync.WaitGroup, chan *junit.TestSuite, *log.Logger, *regexp.Regexp, *regexp.Regexp){
	inventory.TestSuite,
}

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())

	if err := os.MkdirAll(filepath.Dir(*outPath), 0770); err != nil {
		log.Fatal(err)
	}

	var testSuiteRegex *regexp.Regexp
	if *testSuiteFilter != "" {
		var err error
		testSuiteRegex, err = regexp.Compile(*testSuiteFilter)
		if err != nil {
			fmt.Println("-testCaseFilter flag not valid:", err)
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

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		select {
		case <-c:
			fmt.Printf("\nCtrl-C caught, cancel currently unimplemented\n")
			cancel()
		}
	}()

	logger := log.New(os.Stdout, "[OSConfigTests] ", 0)
	logger.Println("Starting...")

	tests := make(chan *junit.TestSuite)
	var wg sync.WaitGroup
	for _, tf := range testFunctions {
		wg.Add(1)
		go tf(ctx, &wg, tests, logger, testSuiteRegex, testCaseRegex)
	}
	go func() {
		wg.Wait()
		close(tests)
	}()

	var testSuites []*junit.TestSuite
	for ret := range tests {
		testSuites = append(testSuites, ret)
	}

	logger.Printf("Creating junit xml file: %q", *outPath)
	d, err := xml.MarshalIndent(testSuites, "  ", "   ")
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile(*outPath, d, 0644); err != nil {
		log.Fatal(err)
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
