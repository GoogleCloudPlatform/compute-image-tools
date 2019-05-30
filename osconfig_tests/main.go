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
	"context"
	"github.com/GoogleCloudPlatform/compute-image-tools/test_common/junitxml"
	"log"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_suites/inventory"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_suites/package_management"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_suites/patch"
	"github.com/GoogleCloudPlatform/compute-image-tools/test_common"
	"github.com/GoogleCloudPlatform/compute-image-tools/test_common/test_config"
)

var testFunctions = []func(context.Context, *sync.WaitGroup, chan *junitxml.TestSuite, *log.Logger, *regexp.Regexp, *regexp.Regexp, *testconfig.Project){
	packagemanagement.TestSuite,
	inventory.TestSuite,
	patch.TestSuite,
}

func main() {
	testcommon.LaunchTests(testFunctions, "[OsConfigTests]")
}
