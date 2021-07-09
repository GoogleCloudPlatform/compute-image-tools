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

// GCE one-step image import tool
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_onestep_image_import/onestep_importer"
)

func main() {
	log.SetPrefix("[import-image-from-external-cloud] ")
	// 1. Parse flags
	importerArgs, err := importer.NewOneStepImportArguments(os.Args[1:])
	if err != nil {
		log.Println(err)
		printHelpLink()
		os.Exit(1)
	}

	importEntry := func() (service.Loggable, error) {
		loggable, importErr := importer.Run(importerArgs)
		if importErr != nil {
			printHelpLink()
		}
		return loggable, importErr
	}

	// 2. Run Onestep Importer
	if err := service.RunWithServerLogging(
		service.OneStepImageImportAction, initLoggingParams(importerArgs), importerArgs.ProjectPtr, importEntry); err != nil {
		os.Exit(1)
	}
}

func printHelpLink() {
	log.Println("Refer to https://cloud.google.com/compute/docs/import/import-aws-image for more detailed help.")
}

func initLoggingParams(args *importer.OneStepImportArguments) service.InputParams {
	return service.InputParams{
		OnestepImageImportParams: &service.OnestepImageImportParams{
			CommonParams: &service.CommonParams{
				ClientID:                args.ClientID,
				ClientVersion:           args.ClientVersion,
				Network:                 args.Network,
				Subnet:                  args.Subnet,
				Zone:                    args.Zone,
				Timeout:                 args.Timeout.String(),
				Project:                 *args.ProjectPtr,
				ObfuscatedProject:       service.Hash(*args.ProjectPtr),
				Labels:                  fmt.Sprintf("%v", args.Labels),
				ScratchBucketGcsPath:    args.ScratchBucketGcsPath,
				Oauth:                   args.Oauth,
				ComputeEndpointOverride: args.ComputeEndpoint,
				DisableGcsLogging:       args.GcsLogsDisabled,
				DisableCloudLogging:     args.CloudLogsDisabled,
				DisableStdoutLogging:    args.StdoutLogsDisabled,
			},
			ImageName:             args.ImageName,
			OS:                    args.OS,
			NoGuestEnvironment:    args.NoGuestEnvironment,
			Family:                args.Family,
			Description:           args.Description,
			NoExternalIP:          args.NoExternalIP,
			StorageLocation:       args.StorageLocation,
			ComputeServiceAccount: args.ComputeServiceAccount,
			AWSAMIID:              args.AWSAMIID,
			AWSAMIExportLocation:  args.AWSAMIExportLocation,
			AWSSourceAMIFilePath:  args.AWSSourceAMIFilePath,
		},
	}
}
