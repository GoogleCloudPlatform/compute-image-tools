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

// GCE OVF export tool
package main

import (
	"context"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	ovfexporter "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/exporter"
)

func createInstanceExportInputParams(args ovfexportdomain.OVFExportArgs) service.InputParams {
	return service.InputParams{
		InstanceExportParams: &service.InstanceExportParams{
			CommonParams:     createCommonInputParams(args),
			DestinationURI:   args.DestinationURI,
			InstanceName:     args.InstanceName,
			DiskExportFormat: args.DiskExportFormat,
			OS:               args.OsID,
			NoExternalIP:     args.NoExternalIP,
		},
	}
}

func createMachineImageExportInputParams(args ovfexportdomain.OVFExportArgs) service.InputParams {
	return service.InputParams{
		MachineImageExportParams: &service.MachineImageExportParams{
			CommonParams:     createCommonInputParams(args),
			DestinationURI:   args.DestinationURI,
			MachineImageName: args.MachineImageName,
			DiskExportFormat: args.DiskExportFormat,
			OS:               args.OsID,
			NoExternalIP:     args.NoExternalIP,
		},
	}
}

func createCommonInputParams(args ovfexportdomain.OVFExportArgs) *service.CommonParams {
	return &service.CommonParams{
		ClientID:                args.ClientID,
		ClientVersion:           args.ClientVersion,
		Network:                 args.Network,
		Subnet:                  args.Subnet,
		Zone:                    args.Zone,
		Timeout:                 args.Timeout.String(),
		Project:                 args.Project,
		ObfuscatedProject:       service.Hash(args.Project),
		ScratchBucketGcsPath:    args.ScratchBucketGcsPath,
		Oauth:                   args.Oauth,
		ComputeEndpointOverride: args.Ce,
		DisableGcsLogging:       args.GcsLogsDisabled,
		DisableCloudLogging:     args.CloudLogsDisabled,
		DisableStdoutLogging:    args.StdoutLogsDisabled,
	}
}

func createInputParams(args ovfexportdomain.OVFExportArgs) (string, service.InputParams) {
	if args.IsInstanceExport() {
		return service.InstanceExportAction, createInstanceExportInputParams(args)
	}
	return service.MachineImageExportAction, createMachineImageExportInputParams(args)
}

// logFailure sends a message to the logging framework, and is expected to be
// used when a validation failure causes the export to not run.
func logFailure(allArgs ovfexportdomain.OVFExportArgs, cause error) {
	noOpCallback := func() (service.Loggable, error) {
		return nil, cause
	}
	// Ignoring the returned error since its a copy of
	// the return value from the callback.
	action, inputParams := createInputParams(allArgs)
	_ = service.RunWithServerLogging(action, inputParams, nil, noOpCallback)
}

func runExport(args []string) error {
	logger := logging.NewToolLogger(ovfexporter.LogPrefix)
	logging.RedirectGlobalLogsToUser(logger)

	exportArgs, err := ovfexportdomain.NewOVFExportArgs(args)
	if err != nil {
		logFailure(*exportArgs, err)
		return err
	}

	var oe *ovfexporter.OVFExporter
	if oe, err = ovfexporter.NewOVFExporter(exportArgs, logger); err != nil {
		return err
	}
	ctx := context.Background()

	exporterClosure := func() (service.Loggable, error) {
		err := oe.Run(ctx)
		return service.NewOutputInfoLoggable(logger.ReadOutputInfo()), err
	}
	action, inputParams := createInputParams(*exportArgs)
	if err := service.RunWithServerLogging(action, inputParams, &exportArgs.Project, exporterClosure); err != nil {
		return err
	}
	return nil
}

func main() {
	if err := runExport(os.Args[1:]); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
