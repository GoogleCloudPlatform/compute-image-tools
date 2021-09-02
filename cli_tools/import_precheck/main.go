//  Copyright 2017 Google Inc. All Rights Reserved.
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
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/import_precheck/precheck"

	"github.com/GoogleCloudPlatform/osconfig/osinfo"
	"github.com/google/logger"
)

const logPath = "out.log"

var (
	log *logger.Logger
)

func getChecks(osInfo *osinfo.OSInfo) []precheck.Check {
	return []precheck.Check{
		&precheck.OSVersionCheck{OSInfo: osInfo},
		&precheck.DisksCheck{},
		&precheck.SSHCheck{},
		&precheck.PowershellCheck{},
		&precheck.SHA2DriverSigningCheck{OSInfo: osInfo},
	}
}

func main() {
	var err error
	lf, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		logger.Fatalf("failed to open log file: %v", err)
	}
	defer lf.Close()
	mw := io.MultiWriter(lf, os.Stdout)
	log = logger.Init("Precheck", false, false, mw)
	defer log.Close()

	if err = precheck.CheckRoot(); err != nil {
		logger.Fatal(err)
	}

	osInfo, err := osinfo.Get()
	if err != nil {
		logger.Fatal(err)
	}

	checks := getChecks(osInfo)
	wg := sync.WaitGroup{}
	for _, c := range checks {
		wg.Add(1)
		go func(c precheck.Check) {
			defer wg.Done()
			report, err := c.Run()
			if err != nil {
				log.Errorf("%s error: %v", c.GetName(), err)
			} else {
				fmt.Println(report.String())
			}
		}(c)
	}
	wg.Wait()
}
