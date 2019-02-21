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

// Package service runs the osconfig service.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/inventory"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/ospackage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/patch"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/tasker"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/service"
)

func run(ctx context.Context) {
	patch.Init(ctx)

	res, err := config.Instance()
	if err != nil {
		logger.Fatalf("get instance error: %v", err)
	}

	ticker := time.NewTicker(config.SvcPollInterval())
	for {
		if err := ospackage.RunOsConfig(ctx, res, true); err != nil {
			logger.Errorf(err.Error())
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

// Run registers a service to periodically call the osconfig enpoint to pull
// the latest applicaple configurations and apply them.
func Run(ctx context.Context, action string) error {
	if err := service.Register(ctx, "google_osconfig_agent", "Google OSConfig Agent", "", run, action); err != nil {
		return fmt.Errorf("service.Register error: %v", err)
	}
	return nil
}
