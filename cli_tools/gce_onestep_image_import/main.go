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
	onestepImporter "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_onestep_image_import/onestep_importer"
)


func main() {
	log.SetPrefix("[onestep-import-image] ")
	// 1. Parse
	importer, err := onestepImporter.Parse(os.Args[1:])
	if err != nil {
		terminate(importer, err)
	}

	importEntry := func() (service.Loggable, error) {
		return importer.Run()
	}

	project := importer.Project
	if err := service.RunWithServerLogging(
		service.OneStepImageImportAction, initLoggingParams(importer), &project, importEntry); err != nil {
		os.Exit(1)
	}
}

func initLoggingParams(args *onestepImporter.ImportArguments) service.InputParams {
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
			NoGuestEnvironment: args.NoGuestEnvironment,
			Family:             args.Family,
			Description:        args.Description,
			NoExternalIP:       args.NoExternalIP,
			StorageLocation:    args.StorageLocation,
			CloudProvider: 			args.CloudProvider,
		},
	}
}


// terminate is used when there is a failure prior to running import. It sends
// a message to the logging framework, and then executes os.Exit(1).
func terminate(allArgs *onestepImporter.ImportArguments, cause error) {
	noOpCallback := func() (service.Loggable, error) {
		return nil, cause
	}
	// Ignoring the returned error since its a copy of
	// the return value from the callback.
	_ = service.RunWithServerLogging(
		service.OneStepImageImportAction, service.InputParams{}, nil, noOpCallback)
	os.Exit(1)
}




