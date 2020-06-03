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

// GCE VM image import tool
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/importer"
)

func main() {
	log.SetPrefix("[import-image] ")
	ctx := context.Background()

	importArgs, err := parseAndPopulateArgs(ctx)
	if err != nil {
		terminate(importArgs, err)
	}

	importClosure := func() (service.Loggable, error) {
		wf, e := importer.NewImporter(importArgs).Run(ctx)
		return service.NewLoggableFromWorkflow(wf), e
	}

	project := importArgs.Project
	if err := service.RunWithServerLogging(
		service.ImageImportAction, initLoggingParams(importArgs), &project, importClosure); err != nil {
		os.Exit(1)
	}
}

func parseAndPopulateArgs(ctx context.Context) (importer.ImportArguments, error) {
	// 1. Parse the args without validating or populating. Splitting parsing and
	// validation allows us to log the intermediate, non-validated values, if
	// there's an error setting up dependencies.
	parsed, err := importer.NewImportArguments(os.Args[1:])
	if err != nil {
		terminate(parsed, err)
	}

	// 2. Setup dependencies.
	storageClient, err := storage.NewStorageClient(
		ctx, logging.NewDefaultLogger(), parsed.Oauth)
	if err != nil {
		terminate(parsed, err)
	}
	computeClient, err := param.CreateComputeClient(
		&ctx, parsed.Oauth, parsed.ComputeEndpoint)
	if err != nil {
		terminate(parsed, err)
	}
	metadataGCE := &compute.MetadataGCE{}
	paramPopulator := param.NewPopulator(
		metadataGCE,
		storageClient,
		storage.NewResourceLocationRetriever(metadataGCE, computeClient),
		storage.NewScratchBucketCreator(ctx, storageClient),
	)

	// 3. Parse, validate, and populate arguments.
	return parsed, parsed.ValidateAndPopulate(
		paramPopulator, importer.NewSourceFactory(storageClient))
}

// terminate is used when there is a failure prior to running import. It sends
// a message to the logging framework, and then executes os.Exit(1).
func terminate(allArgs importer.ImportArguments, cause error) {
	noOpCallback := func() (service.Loggable, error) {
		return nil, cause
	}
	// Ignoring the returned error since its a copy of
	// the return value from the callback.
	_ = service.RunWithServerLogging(
		service.ImageImportAction, initLoggingParams(allArgs), nil, noOpCallback)
	os.Exit(1)
}

func initLoggingParams(args importer.ImportArguments) service.InputParams {
	return service.InputParams{
		ImageImportParams: &service.ImageImportParams{
			CommonParams: &service.CommonParams{
				ClientID:                args.ClientID,
				Network:                 args.Network,
				Subnet:                  args.Subnet,
				Zone:                    args.Zone,
				Timeout:                 args.Timeout.String(),
				Project:                 args.Project,
				ObfuscatedProject:       service.Hash(args.Project),
				Labels:                  fmt.Sprintf("%v", args.Labels),
				ScratchBucketGcsPath:    args.ScratchBucketGcsPath,
				Oauth:                   args.Oauth,
				ComputeEndpointOverride: args.ComputeEndpoint,
				DisableGcsLogging:       args.GcsLogsDisabled,
				DisableCloudLogging:     args.CloudLogsDisabled,
				DisableStdoutLogging:    args.StdoutLogsDisabled,
			},
			ImageName:          args.ImageName,
			DataDisk:           args.DataDisk,
			OS:                 args.OS,
			SourceFile:         args.SourceFile,
			SourceImage:        args.SourceImage,
			NoGuestEnvironment: args.NoGuestEnvironment,
			Family:             args.Family,
			Description:        args.Description,
			NoExternalIP:       args.NoExternalIP,
			StorageLocation:    args.StorageLocation,
		},
	}
}
