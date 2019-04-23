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

package gcpclients

import (
	"cloud.google.com/go/storage"
	"context"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/config"
	"google.golang.org/api/option"
)

var (
	storageClient  *storage.Client
	osconfigClient *osconfig.Client
)

// GetStorageClient returns a singleton GCP client for osconfig tests
func GetStorageClient(ctx context.Context) (*storage.Client, error) {
	if storageClient != nil {
		return storageClient, nil
	}

	return storage.NewClient(ctx, option.WithCredentialsFile(config.OauthPath()))
}

// GetOsConfigClient returns a singleton GCP client for osconfig tests
func GetOsConfigClient(ctx context.Context) (*osconfig.Client, error) {
	if osconfigClient != nil {
		return osconfigClient, nil
	}

	return osconfig.NewClient(ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OauthPath()))
}
