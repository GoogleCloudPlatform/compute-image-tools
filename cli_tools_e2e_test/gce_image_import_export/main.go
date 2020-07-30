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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/gce_image_import_export/test_suites/export"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/gce_image_import_export/test_suites/import"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/gce_image_import_export/test_suites/onestep_import"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

func main() {
	exportTestSuccess := e2etestutils.RunTestsAndOutput([]func(context.Context, *sync.WaitGroup, chan *junitxml.TestSuite, *log.Logger,
		*regexp.Regexp, *regexp.Regexp, *testconfig.Project){exporttestsuites.TestSuite},
		"[ImageExportTests]")
	importCLIRegressionSuccess := e2etestutils.RunTestsAndOutput([]func(context.Context, *sync.WaitGroup, chan *junitxml.TestSuite, *log.Logger,
		*regexp.Regexp, *regexp.Regexp, *testconfig.Project){importtestsuites.CLITestSuite},
		"[CLIRegressionImageImportTests]")
	imageImportSuccess := e2etestutils.RunTestsAndOutput([]func(context.Context, *sync.WaitGroup, chan *junitxml.TestSuite, *log.Logger,
		*regexp.Regexp, *regexp.Regexp, *testconfig.Project){importtestsuites.ImageImportSuite},
		"[ImageImportTests]")
	onestepImageImportSuccess := e2etestutils.RunTestsAndOutput([]func(context.Context, *sync.WaitGroup, chan *junitxml.TestSuite, *log.Logger,
		*regexp.Regexp, *regexp.Regexp, *testconfig.Project){onestepimporttestsuites.OnestepImageImportSuite},
		"[OnestepImageImportTests]")
	if !exportTestSuccess || !importCLIRegressionSuccess || !imageImportSuccess || !onestepImageImportSuccess {
		os.Exit(1)
	}
}
