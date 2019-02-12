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

// Package osconfigserver contains wrapper around osconfig service APIs and helper methods
package osconfigserver

import (
	"context"
	"fmt"

	osconfig "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

// OsConfig is a wrapper struct around osconfig
type OsConfig struct {
	*osconfigpb.OsConfig
}

// OsConfigIterator is a wrapper struct around OsConfigIterator
type OsConfigIterator struct {
	*osconfig.OsConfigIterator
}

// CreateOsConfig is a wrapper around createOsConfig API
func CreateOsConfig(ctx context.Context, oc *osconfigpb.OsConfig, parent string) (*OsConfig, error) {
	client, err := GetOsConfigClient(ctx)
	if err != nil {
		return nil, err
	}

	req := &osconfigpb.CreateOsConfigRequest{
		Parent:   parent,
		OsConfig: oc,
	}

	res, err := client.CreateOsConfig(ctx, req)
	if err != nil {
		return nil, err
	}
	return &OsConfig{OsConfig: res}, nil
}

// ListOsConfigs is a wrapper around listOsConfigs API
func ListOsConfigs(ctx context.Context, req *osconfigpb.ListOsConfigsRequest) *OsConfigIterator {
	client, err := GetOsConfigClient(ctx)
	if err != nil {
		return nil
	}

	resp := client.ListOsConfigs(ctx, req)

	return &OsConfigIterator{OsConfigIterator: resp}
}

// Cleanup function will cleanup the osconfig created under project
func (o *OsConfig) Cleanup(ctx context.Context, projectID string) error {
	client, err := GetOsConfigClient(ctx)
	if err != nil {
		return err
	}

	deleteReq := &osconfigpb.DeleteOsConfigRequest{
		Name: fmt.Sprintf("projects/%s/osConfigs/%s", projectID, o.Name),
	}
	return client.DeleteOsConfig(ctx, deleteReq)
}
