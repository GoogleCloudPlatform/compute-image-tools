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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/inventory"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/ospackage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/tasker"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/osinfo"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/service"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/option"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

var dump = &pretty.Config{IncludeUnexported: true}

func runOsConfig(ctx context.Context, res string) error {
	client, err := osconfig.NewClient(ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OAuthPath()))
	if err != nil {
		return fmt.Errorf("osconfig.NewClient Error: %v", err)
	}

	resp, err := LookupConfigs(ctx, client, res)
	if err != nil {
		return fmt.Errorf("LookupConfigs error: %v", err)
	}
	tasker.Enqueue("Set package config", func() { ospackage.SetConfig(resp) })
	return nil
}

func run(ctx context.Context) {
	//patch.Init(ctx)

	res, err := config.Instance()
	if err != nil {
		logger.Fatalf("get instance error: %v", err)
	}

	ticker := time.NewTicker(config.SvcPollInterval())
	configError := false
	for {
		err := runOsConfig(ctx, res)
		// Only log osconfig error on the first error so as not to spam the logs.
		// TODO: Remove once the API is in beta.
		if err != nil && !configError {
			logger.Errorf(err.Error())
			configError = true
		} else {
			configError = false
		}

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

// LookupConfigs looks up osconfigs.
func LookupConfigs(ctx context.Context, client *osconfig.Client, resource string) (*osconfigpb.LookupConfigsResponse, error) {
	info, err := osinfo.GetDistributionInfo()
	if err != nil {
		return nil, err
	}

	req := &osconfigpb.LookupConfigsRequest{
		Resource: resource,
		OsInfo: &osconfigpb.LookupConfigsRequest_OsInfo{
			OsLongName:     info.LongName,
			OsShortName:    info.ShortName,
			OsVersion:      info.Version,
			OsKernel:       info.Kernel,
			OsArchitecture: info.Architecture,
		},
		ConfigTypes: []osconfigpb.LookupConfigsRequest_ConfigType{
			osconfigpb.LookupConfigsRequest_GOO,
			osconfigpb.LookupConfigsRequest_WINDOWS_UPDATE,
			osconfigpb.LookupConfigsRequest_APT,
			osconfigpb.LookupConfigsRequest_YUM,
			osconfigpb.LookupConfigsRequest_ZYPPER,
		},
	}
	logger.Debugf("LookupConfigs request:\n%s\n\n", dump.Sprint(req))

	res, err := client.LookupConfigs(ctx, req)
	if err != nil {
		return nil, err
	}
	logger.Debugf("LookupConfigs response:\n%s\n\n", dump.Sprint(res))

	return res, nil
}

// Run registers a service to periodically call the osconfig enpoint to pull
// the latest applicaple configurations and apply them.
func Run(ctx context.Context, action string) error {
	if err := service.Register(ctx, "google_osconfig_agent", "Google OSConfig Agent", "", run, action); err != nil {
		return fmt.Errorf("service.Register error: %v", err)
	}
	return nil
}
