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
	"errors"
	"fmt"
	"log"

	osconfig "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

type OsConfig struct {
	*osconfigpb.OsConfig
}

type OsConfigIterator struct {
	*osconfig.OsConfigIterator
}

// CreateOsConfig is a wrapper around createOsConfig API
func CreateOsConfig(ctx context.Context, logger *log.Logger, oc *OsConfig, parent string) (*OsConfig, error) {
	client, err := GetOsConfigClient(ctx, logger)

	if err != nil {
		return nil, err
	}

	req := &osconfigpb.CreateOsConfigRequest{
		Parent:   parent,
		OsConfig: oc.OsConfig,
	}

	logger.Printf("create osconfig request:\n%s\n\n", dump.Sprint(req))

	res, err := client.CreateOsConfig(ctx, req)
	if err != nil {
		logger.Printf("error while creating osconfig:\n%s\n\n", err)
		return nil, err
	}
	logger.Printf("create osconfig response:\n%s\n\n", dump.Sprint(res))

	return &OsConfig{OsConfig: res}, nil
}

// ListOsConfigs is a wrapper around listOsConfigs API
func ListOsConfigs(ctx context.Context, logger *log.Logger, req *osconfigpb.ListOsConfigsRequest) *OsConfigIterator {
	client, err := GetOsConfigClient(ctx, logger)

	if err != nil {
		return nil
	}

	logger.Printf("List osconfig request:\n%s\n\n", dump.Sprint(req))

	resp := client.ListOsConfigs(ctx, req)
	if resp == nil {
		logger.Printf("error while listing osconfig:\n%v\n\n", resp)
		return nil
	}

	return &OsConfigIterator{OsConfigIterator: resp}
}

// Cleanup function will cleanup the osconfig created under project
func (o *OsConfig) Cleanup(ctx context.Context, logger *log.Logger) error {
	client, err := GetOsConfigClient(ctx, logger)

	if err != nil {
		return err
	}

	logger.Printf("Deleting osconfig...")

	deleteReq := &osconfigpb.DeleteOsConfigRequest{
		Name: fmt.Sprintf("projects/compute-image-test-pool-001/osConfigs/%s", o.Name),
	}
	ok := client.DeleteOsConfig(ctx, deleteReq)
	if ok != nil {
		logger.Printf("error while cleaning up")
		return errors.New(fmt.Sprintf("error while cleaning up the osconfig: %s\n", ok))
	}

	logger.Printf("OsConfig cleanup done.")
	return nil
}
