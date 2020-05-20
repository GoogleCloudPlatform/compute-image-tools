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
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/args"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/importer"
)

func main() {
	log.SetPrefix("[import-image] ")
	ctx := context.Background()

	importArgs, err := parseAllArgs(ctx)
	if err != nil {
		terminate(err)
	}

	importerClosure := func() (service.Loggable, error) {
		wf, e := importer.NewImporter(importArgs.Env, importArgs.Img, importArgs.Translation).Run(ctx)
		return service.NewLoggableFromWorkflow(wf), e
	}

	project := importArgs.Env.Project
	if err := service.RunWithServerLogging(
		service.ImageImportAction, initLogging(importArgs), &project, importerClosure); err != nil {
		os.Exit(1)
	}
}

func parseAllArgs(ctx context.Context) (args.ParsedArguments, error) {
	// 1. Parse auth flags, which are required to initialize the API clients
	// that are used for subsequent validation and population.
	fs := flag.NewFlagSet("auth-flags", flag.ContinueOnError)
	oauth := flag.String("oauth", "", "Path to oauth json file.")
	ce := flag.String("compute_endpoint_override", "", "API endpoint to override default.")
	// Don't write parse errors to stdout, instead propagate them via an
	// exception since we use flag.ContinueOnError.
	fs.SetOutput(ioutil.Discard)
	// Ignoring parse errors here since FlagSet.Parse reports that there are extra
	// flags (such as client_id) that are not defined. That's okay: we just want the
	// authentication flags now, and we'll re-parse everything next.
	_ = fs.Parse(os.Args[1:])

	// 2. Setup dependencies.
	storageClient, err := storage.NewStorageClient(ctx, logging.NewDefaultLogger(), *oauth)
	if err != nil {
		terminate(err)
	}
	sourceFactory := importer.NewSourceFactory(storageClient)
	computeClient, err := param.CreateComputeClient(&ctx, *oauth, *ce)
	if err != nil {
		return args.ParsedArguments{}, err
	}
	metadataGCE := &compute.MetadataGCE{}
	paramPopulator := param.NewPopulator(
		metadataGCE,
		storageClient,
		storage.NewResourceLocationRetriever(metadataGCE, computeClient),
		storage.NewScratchBucketCreator(ctx, storageClient),
	)

	// 3. Parse, validate, and populate arguments.
	return args.ParseArgs(os.Args[1:], paramPopulator, sourceFactory)
}

// terminate is used when there is a failure prior to running import. It sends
// a message to the logging framework, and then executes os.Exit(1).
func terminate(cause error) {
	noopCallback := func() (service.Loggable, error) {
		return nil, cause
	}
	// Ignoring the returned error since its a copy of
	// the return value from the callback.
	_ = service.RunWithServerLogging(service.ImageImportAction, service.InputParams{}, nil, noopCallback)
	os.Exit(1)
}

func initLogging(args args.ParsedArguments) service.InputParams {
	env := args.Env
	translation := args.Translation
	img := args.Img
	return service.InputParams{
		ImageImportParams: &service.ImageImportParams{
			CommonParams: &service.CommonParams{
				ClientID:             env.ClientID,
				Network:              env.Network,
				Subnet:               env.Subnet,
				Zone:                 env.Zone,
				Timeout:              translation.Timeout.String(),
				Project:              env.Project,
				ObfuscatedProject:    service.Hash(env.Project),
				DisableGcsLogging:    env.GcsLogsDisabled,
				DisableCloudLogging:  env.CloudLogsDisabled,
				DisableStdoutLogging: env.StdoutLogsDisabled,
			},
			ImageName:          img.Name,
			DataDisk:           translation.DataDisk,
			OS:                 translation.OS,
			SourceFile:         translation.SourceFile,
			SourceImage:        translation.SourceImage,
			NoGuestEnvironment: translation.NoGuestEnvironment,
			Family:             img.Family,
			NoExternalIP:       env.NoExternalIP,
			StorageLocation:    img.StorageLocation,
		},
	}
}
