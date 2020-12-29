//  Copyright 2020 Google Inc. All Rights Reserved.
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

package cli

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/option"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/files"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/importer"
)

// Main starts an image import.
func Main(args []string, toolLogger logging.ToolLogger, workflowDir string) error {
	logging.RedirectGlobalLogsToUser(toolLogger)

	ctx := context.Background()

	// 1. Parse the args without validating or populating. Splitting parsing and
	// validation allows us to log the intermediate, non-validated values, if
	// there's an error setting up dependencies.
	importArgs, err := importer.NewImportArguments(args, files.MakeAbsolute(workflowDir))
	if err != nil {
		logFailure(importArgs, err)
		return err
	}

	// 2. Setup dependencies.
	storageClient, err := storage.NewStorageClient(
		ctx, toolLogger, option.WithCredentialsFile(importArgs.Oauth))
	if err != nil {
		logFailure(importArgs, err)
		return err
	}
	computeClient, err := param.CreateComputeClient(
		&ctx, importArgs.Oauth, importArgs.ComputeEndpoint)
	if err != nil {
		logFailure(importArgs, err)
		return err
	}
	metadataGCE := &compute.MetadataGCE{}
	paramPopulator := param.NewPopulator(
		metadataGCE,
		storageClient,
		storage.NewResourceLocationRetriever(metadataGCE, computeClient),
		storage.NewScratchBucketCreator(ctx, storageClient),
	)

	// 3. Parse, validate, and populate arguments.
	if err = importArgs.ValidateAndPopulate(
		paramPopulator, importer.NewSourceFactory(storageClient)); err != nil {
		logFailure(importArgs, err)
		return err
	}

	importRunner, err := importer.NewImporter(importArgs, computeClient, *storageClient, toolLogger)
	if err != nil {
		logFailure(importArgs, err)
		return err
	}

	importClosure := func() (service.Loggable, error) {
		err := importRunner.Run(ctx)
		return service.NewOutputInfoLoggable(toolLogger.ReadOutputInfo()), userFriendlyError(err, importArgs)
	}

	project := importArgs.Project
	if err := service.RunWithServerLogging(
		service.ImageImportAction, initLoggingParams(importArgs), &project, importClosure); err != nil {
		return err
	}
	return nil
}

func userFriendlyError(err error, importArgs importer.ImportArguments) error {
	if strings.Contains(err.Error(), "constraints/compute.vmExternalIpAccess") {
		return fmt.Errorf("constraint constraints/compute.vmExternalIpAccess "+
			"violated for project %v. For more information about importing disks using "+
			"networks that don't allow external IP addresses, see "+
			"https://cloud.google.com/compute/docs/import/importing-virtual-disks#no-external-ip",
			importArgs.Project)
	}
	return err
}

// logFailure sends a message to the logging framework, and is expected to be
// used when a validation failure causes the import to not run.
func logFailure(allArgs importer.ImportArguments, cause error) {
	noOpCallback := func() (service.Loggable, error) {
		return nil, cause
	}
	// Ignoring the returned error since its a copy of
	// the return value from the callback.
	_ = service.RunWithServerLogging(
		service.ImageImportAction, initLoggingParams(allArgs), nil, noOpCallback)
}

func initLoggingParams(args importer.ImportArguments) service.InputParams {
	return service.InputParams{
		ImageImportParams: &service.ImageImportParams{
			CommonParams: &service.CommonParams{
				ClientID:                args.ClientID,
				ClientVersion:           args.ClientVersion,
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
			ImageName:             args.ImageName,
			DataDisk:              args.DataDisk,
			OS:                    args.OS,
			SourceFile:            args.SourceFile,
			SourceImage:           args.SourceImage,
			NoGuestEnvironment:    args.NoGuestEnvironment,
			Family:                args.Family,
			Description:           args.Description,
			NoExternalIP:          args.NoExternalIP,
			StorageLocation:       args.StorageLocation,
			ComputeServiceAccount: args.ComputeServiceAccount,
		},
	}
}
