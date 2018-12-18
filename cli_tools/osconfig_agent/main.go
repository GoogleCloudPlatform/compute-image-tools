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
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/inventory"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/ospackage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
	"google.golang.org/api/option"
)

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

	action := flag.Arg(0)
	if action == "inventory" {
		// Just run inventory and exit.
		inventory.RunInventory()
		logger.Close()
		os.Exit(0)
	}

	if action == "ospackage" {
		// Just run SetConfig and exit.
		client, err := osconfig.NewClient(ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OAuthPath()))
		if err != nil {
			logger.Fatalf("NewClient Error: %v", err)
		}

		res, err := config.Instance()
		if err != nil {
			logger.Fatalf("get instance error: %v", err)
		}

		resp, err := service.LookupConfigs(ctx, client, res)
		if err != nil {
			logger.Fatalf("LookupConfigs error: %v", err)
		}
		ospackage.SetConfig(resp)
		os.Exit(0)
	}

	if err := service.Run(ctx, action); err != nil {
		logger.Fatalf(err.Error())
	}
}
