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
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/guest-logging-go/logger"
	osconfig "github.com/GoogleCloudPlatform/osconfig/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	"google.golang.org/api/option"
)

var (
	storageClient  *storage.Client
	computeClient  compute.Client
	osconfigClient *osconfig.Client

	populateClientOnce sync.Once
)

func populateClients(ctx context.Context) error {
	var err error
	populateClientOnce.Do(func() {
		err = createComputeClient(ctx)
		err = createOsConfigClient(ctx)
		err = createStorageClient(ctx)
	})
	return err
}

func createComputeClient(ctx context.Context) error {
	var err error
	computeClient, err = compute.NewClient(ctx,  option.WithCredentialsFile(config.OauthPath()))
	if err != nil {
		return err
	}
	return nil
}

func createStorageClient(ctx context.Context) error{
	logger.Debugf("creating storage client\n")
	var err error
	storageClient, err = storage.NewClient(ctx, option.WithCredentialsFile(config.OauthPath()))
	if err != nil {
		return err
	}
	return nil
}

func createOsConfigClient(ctx context.Context) error{
	logger.Debugf("creating osconfig client\n")
	var err error
	osconfigClient, err = osconfig.NewClient(ctx, option.WithCredentialsFile(config.OauthPath()), option.WithEndpoint(config.SvcEndpoint()))
	if err != nil {
		return err
	}
	return nil
}

// GetComputeClient returns a singleton GCP client for osconfig tests
func GetComputeClient(ctx context.Context) (compute.Client, error) {
	if err := populateClients(ctx); err != nil {
		return nil, err
	}
	return computeClient, nil
}

// GetStorageClient returns a singleton GCP client for osconfig tests
func GetStorageClient(ctx context.Context) (*storage.Client, error) {
	if storageClient == nil {
		return nil, fmt.Errorf("storage client was not initialized")
	}
	return storageClient, nil
}

// GetOsConfigClient returns a singleton GCP client for osconfig tests
func GetOsConfigClient(ctx context.Context) (*osconfig.Client, error) {
	if err := populateClients(ctx); err != nil {
		return nil, err
	}
	return osconfigClient, nil
}
