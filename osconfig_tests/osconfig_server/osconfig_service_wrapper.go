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

// Package contains wrapper around osconfig service APIs and helper methods
package osconfig_server

import (
	"context"
	"fmt"
	"log"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/junitxml"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/option"

	osconfig "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

var dump = &pretty.Config{IncludeUnexported: true}
var osconfigClient *osconfig.Client

func getOsConfigClient(ctx context.Context, logger *log.Logger) (*osconfig.Client, error) {

	if osconfigClient != nil {
		return osconfigClient, nil
	}

	client, err := osconfig.NewClient(ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OAuthPath()))

	if err != nil {
		logger.Printf("error while creating osconfig client: %s\n", err)
	}
	return client, err
}

func CreateOsConfig(ctx context.Context, logger *log.Logger, req *osconfigpb.CreateOsConfigRequest) (*osconfigpb.OsConfig, error) {
	client, err := getOsConfigClient(ctx, logger)

	if err != nil {
		return nil, err
	}

	logger.Printf("create osconfig request:\n%s\n\n", dump.Sprint(req))

	res, err := client.CreateOsConfig(ctx, req)
	if err != nil {
		logger.Printf("error while creating osconfig:\n%s\n\n", err)
		return nil, err
	}
	logger.Printf("create osconfig response:\n%s\n\n", dump.Sprint(res))

	return res, nil
}

func ListOsConfigs(ctx context.Context, logger *log.Logger, req *osconfigpb.ListOsConfigsRequest) *osconfig.OsConfigIterator {
	client, err := getOsConfigClient(ctx, logger)

	if err != nil {
		return nil
	}

	logger.Printf("List osconfig request:\n%s\n\n", dump.Sprint(req))

	resp := client.ListOsConfigs(ctx, req)
	if resp == nil {
		logger.Printf("error while listing osconfig:\n%s\n\n", *resp)
		return nil
	}

	return resp
}

func DeleteOsConfig(ctx context.Context, logger *log.Logger, req *osconfigpb.DeleteOsConfigRequest) error {
	client, err := getOsConfigClient(ctx, logger)

	if err != nil {
		return err
	}

	logger.Printf("Delete osconfig request:\n%s\n\n", dump.Sprint(req))

	ok := client.DeleteOsConfig(ctx, req)
	if ok != nil {
		logger.Printf("error while deleting osconfig:\n%s\n\n", ok)
		return nil
	}
	return ok
}

// This function will cleanup all the osconfig created under project
// Assumption is that this project is only used by this test application
func CleanupOsConfig(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger, osConfig *osconfigpb.OsConfig) {

	logger.Printf("Starting OsConfig cleanup...")

	deleteReq := &osconfigpb.DeleteOsConfigRequest{
		Name: fmt.Sprintf("projects/compute-image-test-pool-001/osConfigs/%s", osConfig.Name),
	}
	ok := DeleteOsConfig(ctx, logger, deleteReq)
	if ok != nil {
		testCase.WriteFailure("error while cleaning up")
		return
	}

	logger.Printf("OsConfig cleanup done.")

}
