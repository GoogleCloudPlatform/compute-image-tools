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

// Package apipoller contains utilities to periodically poll the service for configurations.
package apipoller

import (
	"context"
	"time"

	osconfigAgent "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/inventory"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/osconfig"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/patch"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/osinfo"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/option"
)

var dump = &pretty.Config{IncludeUnexported: true}

// Poll periodically calls the service to pull the latest applicaple configurations.
func Poll(ctx context.Context) {
	client, err := osconfigAgent.NewClient(ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OAuthPath()))
	if err != nil {
		logger.Fatalf("NewClient Error: %v", err)
	}

	res, err := config.Instance()
	if err != nil {
		logger.Fatalf("get instance error: %v", err)
	}

	patch.PatchInit()
	ticker := time.NewTicker(config.SvcPollInterval())
	for {
		resp, err := lookupConfigs(ctx, client, res)
		if err != nil {
			logger.Errorf("lookupConfigs error: %v", err)
		} else {
			osconfig.SetOsConfig(resp)
			patch.SetPatchPolicies(resp.PatchPolicies)
		}
		inventory.RunInventory()

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func lookupConfigs(ctx context.Context, client *osconfigAgent.Client, resource string) (*osconfigpb.LookupConfigsResponse, error) {
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
	}
	logger.Debugf("LookupConfigs request:\n%s\n\n", dump.Sprint(req))

	res, err := client.LookupConfigs(ctx, req)
	if err != nil {
		return nil, err
	}
	logger.Debugf("LookupConfigs response:\n%s\n\n", dump.Sprint(res))

	return res, nil
}
