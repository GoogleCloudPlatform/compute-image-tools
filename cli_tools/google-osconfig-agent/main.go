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
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/inventory"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/ospackage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/ospatch"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/tasker"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/service"
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

func run(ctx context.Context) {
	if config.OSPatchEnabled() {
		ospatch.Init(ctx)
	}

	res, err := config.Instance()
	if err != nil {
		logger.Fatalf("get instance error: %v", err)
	}

	ticker := time.NewTicker(config.SvcPollInterval())
	for {
		if config.OSPackageEnabled() {
			if err := ospackage.Run(ctx, res, true); err != nil {
				logger.Errorf(err.Error())
			}
		}

		// This should always run after ospackage.SetConfig.
		tasker.Enqueue("Gather instance inventory", inventory.RunInventory)

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			logger.Close()
			return
		}
	}
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

	switch action := flag.Arg(0); action {
	case "":
		if err := service.Register(ctx, "google_osconfig_agent", "Google OSConfig Agent", "", run, "run"); err != nil {
			logger.Fatalf("service.Register error: %v", err)
		}
	case "noservice":
		run(ctx)
		return
	case "inventory":
		inventory.RunInventory()
		return
	case "ospackage":
		res, err := config.Instance()
		if err != nil {
			logger.Fatalf("get instance error: %v", err)
		}
		if err := ospackage.Run(ctx, res, false); err != nil {
			logger.Fatalf(err.Error())
		}
		return
	case "ospatch":
		ospatch.Run(ctx)
		return
	default:
		logger.Fatalf("Unknown arg %q", action)
	}
}
