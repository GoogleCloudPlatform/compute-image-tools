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

// Package exporter defines GCE VM image exporter
package exporter

import (
	"context"
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// Make file paths mutable
var (
	WorkflowDir              = "daisy_workflows/export/"
	ExportWorkflow           = "image_export.wf.json"
	ExportAndConvertWorkflow = "image_export_ext.wf.json"
)

// Parameter key shared with external packages
const (
	ClientIDFlagKey       = "client_id"
	DestinationURIFlagKey = "destination_uri"
	SourceImageFlagKey    = "source_image"
)

func validateAndParseFlags(clientID string, destinationURI string, sourceImage string, labels string) (
	map[string]string, error) {

	if err := validation.ValidateStringFlagNotEmpty(clientID, ClientIDFlagKey); err != nil {
		return nil, err
	}
	if err := validation.ValidateStringFlagNotEmpty(destinationURI, DestinationURIFlagKey); err != nil {
		return nil, err
	}
	if err := validation.ValidateStringFlagNotEmpty(sourceImage, SourceImageFlagKey); err != nil {
		return nil, err
	}

	if labels != "" {
		var err error
		userLabels, err := param.ParseKeyValues(labels)
		if err != nil {
			return nil, err
		}
		return userLabels, nil
	}
	return nil, nil
}

func getWorkflowPath(format string, currentExecutablePath string) string {
	if format == "" {
		return path.ToWorkingDir(WorkflowDir+ExportWorkflow, currentExecutablePath)
	}

	return path.ToWorkingDir(WorkflowDir+ExportAndConvertWorkflow, currentExecutablePath)
}

func buildDaisyVars(destinationURI string, sourceImage string, format string, network string,
	subnet string, region string) map[string]string {

	varMap := map[string]string{}

	varMap["destination"] = destinationURI

	varMap["source_image"] = sourceImage

	if format != "" {
		varMap["format"] = format
	}
	if subnet != "" {
		varMap["export_subnet"] = fmt.Sprintf("regions/%v/subnetworks/%v", region, subnet)
		// When subnet is set, we need to grant a value to network to avoid fallback to default
		if network == "" {
			varMap["export_network"] = ""
		}
	}
	if network != "" {
		varMap["export_network"] = fmt.Sprintf("global/networks/%v", network)
	}
	return varMap
}

func runExportWorkflow(ctx context.Context, exportWorkflowPath string, varMap map[string]string,
	project string, zone string, timeout string, scratchBucketGcsPath string, oauth string, ce string,
	gcsLogsDisabled bool, cloudLogsDisabled bool, stdoutLogsDisabled bool,
	userLabels map[string]string) error {

	workflow, err := daisycommon.ParseWorkflow(exportWorkflowPath, varMap,
		project, zone, scratchBucketGcsPath, oauth, timeout, ce, gcsLogsDisabled,
		cloudLogsDisabled, stdoutLogsDisabled)
	if err != nil {
		return err
	}

	workflowModifier := func(w *daisy.Workflow) {
		rl := &daisyutils.ResourceLabeler{
			BuildID: os.Getenv("BUILD_ID"), UserLabels: userLabels, BuildIDLabelKey: "gce-image-export-build-id",
			InstanceLabelKeyRetriever: func(instance *daisy.Instance) string {
				return "gce-image-export-tmp"
			},
			DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
				return "gce-image-export-tmp"
			},
			ImageLabelKeyRetriever: func(image *daisy.Image) string {
				return "gce-image-export"
			}}
		rl.LabelResources(w)
	}

	return workflow.RunWithModifiers(ctx, nil, workflowModifier)
}

// Run runs export workflow.
func Run(clientID, destinationURI, sourceImage, format, project, network, subnet, zone, timeout,
	scratchBucketGcsPath, oauth, ce string, gcsLogsDisabled, cloudLogsDisabled,
	stdoutLogsDisabled bool, labels, currentExecutablePath string) error {

	var userLabels map[string]string
	var err error
	if userLabels, err = validateAndParseFlags(clientID, destinationURI, sourceImage, labels); err != nil {
		return err
	}

	ctx := context.Background()
	metadataGCE := &compute.MetadataGCE{}
	storageClient, err := storage.NewStorageClient(
		ctx, logging.NewLogger("[image-export]"), oauth)
	if err != nil {
		return fmt.Errorf("error creating storage client %v", err)
	}
	defer storageClient.Close()

	scratchBucketCreator := storage.NewScratchBucketCreator(ctx, storageClient)
	zoneRetriever, err := storage.NewZoneRetriever(metadataGCE, param.CreateComputeClient(&ctx, oauth, ce))
	if err != nil {
		return err
	}

	region := new(string)
	err = param.PopulateMissingParameters(&project, &zone, region, &scratchBucketGcsPath,
		destinationURI, metadataGCE, scratchBucketCreator, zoneRetriever, storageClient)
	if err != nil {
		return err
	}

	varMap := buildDaisyVars(destinationURI, sourceImage, format, network, subnet, *region)

	if err := runExportWorkflow(ctx, getWorkflowPath(format, currentExecutablePath), varMap, project,
		zone, timeout, scratchBucketGcsPath, oauth, ce, gcsLogsDisabled, cloudLogsDisabled,
		stdoutLogsDisabled, userLabels); err != nil {
		return err
	}
	return nil
}
