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
)

func lookupConfigs(ctx context.Context, client *osconfig.Client, resource string) (*osconfigpb.LookupConfigsResponse, error) {
	req := &osconfigpb.LookupConfigsRequest{
		Resource: resource,
		ConfigTypes: []osconfigpb.LookupConfigsRequest_ConfigType{
			osconfigpb.LookupConfigsRequest_APT,
			osconfigpb.LookupConfigsRequest_YUM,
			osconfigpb.LookupConfigsRequest_GOO,
			osconfigpb.LookupConfigsRequest_WINDOWS_UPDATE},
	}

	return client.LookupConfigs(ctx, req)
}
