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

package main

import (
	"context"
	"log"
	"os"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e/gce_windows_upgrade/test_suites/windows_upgrade"
)

func main() {
	windowsUpgradeTestSuccess := e2etestutils.RunTestsAndOutput([]func(context.Context, *sync.WaitGroup, chan *junitxml.TestSuite, *log.Logger,
		*regexp.Regexp, *regexp.Regexp, *testconfig.Project){testsuite.TestSuite},
		"[WindowsUpgradeTests]")
	if !windowsUpgradeTestSuccess {
		os.Exit(1)
	}
}
