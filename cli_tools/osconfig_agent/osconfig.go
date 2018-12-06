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

	osconfig "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/osinfo"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
)

func getConfigTypes(info *osinfo.DistributionInfo) []osconfigpb.LookupConfigsRequest_ConfigType {
	var configTypes []osconfigpb.LookupConfigsRequest_ConfigType

	if packages.AptExists {
		configTypes = append(configTypes, osconfigpb.LookupConfigsRequest_APT)
	}
	if packages.YumExists {
		configTypes = append(configTypes, osconfigpb.LookupConfigsRequest_YUM)
	}
	if packages.GooGetExists {
		configTypes = append(configTypes, osconfigpb.LookupConfigsRequest_GOO)
	}
	if info.ShortName == osinfo.Windows {
		configTypes = append(configTypes, osconfigpb.LookupConfigsRequest_WINDOWS_UPDATE)
	}

	// TODO: Remove this override once testing is complete.
	// --------------------------------------
	configTypes = []osconfigpb.LookupConfigsRequest_ConfigType{
		osconfigpb.LookupConfigsRequest_APT,
		osconfigpb.LookupConfigsRequest_YUM,
		osconfigpb.LookupConfigsRequest_GOO,
		osconfigpb.LookupConfigsRequest_WINDOWS_UPDATE,
	}
	// --------------------------------------

	return configTypes
}

func lookupConfigs(ctx context.Context, client *osconfig.Client, resource string) (*osconfigpb.LookupConfigsResponse, error) {
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
	logDebugf("LookupConfigs request:\n%s\n\n", dump.Sprint(req))

	res, err := client.LookupConfigs(ctx, req)
	if err != nil {
		return nil, err
	}
	logDebugf("LookupConfigs response:\n%s\n\n", dump.Sprint(res))

	return res, nil
}
