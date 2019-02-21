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

// osconfig_agent interacts with the osconfig api.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/inventory"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/ospackage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/patch"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
)

var version string

func init() {
	// We do this here so the -X value doesn't need the full path.
	config.SetVersion(version)
}

type logWritter struct{}

func (l *logWritter) Write(b []byte) (int, error) {
	logger.Debug(logger.LogEntry{CallDepth: 3, Message: string(b)})
	return len(b), nil
}

func main() {
	flag.Parse()
	ctx := context.Background()

	if config.Debug() {
		packages.DebugLogger = log.New(&logWritter{}, "", 0)
	}

	proj, err := config.Project()
	if err != nil {
		logger.Fatalf(err.Error())
	}

	logger.Init(ctx, proj)
	defer logger.Close()

	action := flag.Arg(0)
	if action == "inventory" {
		// Just run inventory and exit.
		inventory.RunInventory()
		return
	}

	if action == "ospackage" {
		// Just run SetConfig and exit.
		res, err := config.Instance()
		if err != nil {
			logger.Close()
			logger.Fatalf("get instance error: %v", err)
		}
		if err := ospackage.RunOsConfig(ctx, res, false); err != nil {
			logger.Close()
			logger.Fatalf(err.Error())
		}
		return
	}

	if action == "ospatch" {
		patch.RunPatchAgent(ctx)
		return
	}

	if action == "" {
		action = "run"
	}
	if err := service.Run(ctx, action); err != nil {
		logger.Fatalf(err.Error())
	}
}
