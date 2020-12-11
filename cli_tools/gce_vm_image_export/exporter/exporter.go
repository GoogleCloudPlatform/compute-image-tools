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
	"log"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/option"
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

const (
	logPrefix = "[image-export]"
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
	subnet string, region string, computeServiceAccount string) map[string]string {

	destinationURI = strings.TrimSpace(destinationURI)
	sourceImage = strings.TrimSpace(sourceImage)
	format = strings.TrimSpace(format)
	network = strings.TrimSpace(network)
	subnet = strings.TrimSpace(subnet)
	region = strings.TrimSpace(region)
	computeServiceAccount = strings.TrimSpace(computeServiceAccount)

	varMap := map[string]string{}

	varMap["destination"] = destinationURI

	varMap["source_image"] = param.GetGlobalResourcePath(
		"images", sourceImage)

	if format != "" {
		varMap["format"] = format
	}
	if subnet != "" {
		varMap["export_subnet"] = param.GetRegionalResourcePath(
			region, "subnetworks", subnet)

		// When subnet is set, we need to grant a value to network to avoid fallback to default
		if network == "" {
			varMap["export_network"] = ""
		}
	}
	if network != "" {
		varMap["export_network"] = param.GetGlobalResourcePath(
			"networks", network)
	}
	if computeServiceAccount != "" {
		varMap["compute_service_account"] = computeServiceAccount
	}
	return varMap
}

func runExportWorkflow(ctx context.Context, exportWorkflowPath string, varMap map[string]string,
	project string, zone string, timeout string, scratchBucketGcsPath string, oauth string, ce string,
	gcsLogsDisabled bool, cloudLogsDisabled bool, stdoutLogsDisabled bool,
	userLabels map[string]string) (*daisy.Workflow, error) {

	workflow, err := daisycommon.ParseWorkflow(exportWorkflowPath, varMap,
		project, zone, scratchBucketGcsPath, oauth, timeout, ce, gcsLogsDisabled,
		cloudLogsDisabled, stdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	preValidateWorkflowModifier := func(w *daisy.Workflow) {
		w.SetLogProcessHook(daisyutils.RemovePrivacyLogTag)
	}

	postValidateWorkflowModifier := func(w *daisy.Workflow) {
		w.LogWorkflowInfo("Cloud Build ID: %s", os.Getenv(daisyutils.BuildIDOSEnvVarName))
		rl := &daisyutils.ResourceLabeler{
			BuildID: os.Getenv("BUILD_ID"), UserLabels: userLabels, BuildIDLabelKey: "gce-image-export-build-id",
			InstanceLabelKeyRetriever: func(instanceName string) string {
				return "gce-image-export-tmp"
			},
			DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
				return "gce-image-export-tmp"
			},
			ImageLabelKeyRetriever: func(imageName string) string {
				return "gce-image-export"
			}}
		rl.LabelResources(w)
	}

	err = workflow.RunWithModifiers(ctx, preValidateWorkflowModifier, postValidateWorkflowModifier)
	return workflow, err
}

// Run runs export workflow.
func Run(clientID string, destinationURI string, sourceImage string, format string,
	project *string, network string, subnet string, zone string, timeout string,
	scratchBucketGcsPath string, oauth string, ce string, computeServiceAccount string, gcsLogsDisabled bool,
	cloudLogsDisabled bool, stdoutLogsDisabled bool, labels string, currentExecutablePath string) (*daisy.Workflow, error) {

	log.SetPrefix(logPrefix + " ")

	userLabels, err := validateAndParseFlags(clientID, destinationURI, sourceImage, labels)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	metadataGCE := &compute.MetadataGCE{}
	storageClient, err := storage.NewStorageClient(
		ctx, logging.NewToolLogger(logPrefix), option.WithCredentialsFile(oauth))
	if err != nil {
		return nil, err
	}
	defer storageClient.Close()

	scratchBucketCreator := storage.NewScratchBucketCreator(ctx, storageClient)
	computeClient, err := param.CreateComputeClient(&ctx, oauth, ce)
	if err != nil {
		return nil, err
	}
	resourceLocationRetriever := storage.NewResourceLocationRetriever(metadataGCE, computeClient)

	region := new(string)
	paramPopulator := param.NewPopulator(metadataGCE, storageClient, resourceLocationRetriever, scratchBucketCreator)
	err = paramPopulator.PopulateMissingParameters(project, clientID, &zone, region, &scratchBucketGcsPath, destinationURI, nil)
	if err != nil {
		return nil, err
	}

	varMap := buildDaisyVars(destinationURI, sourceImage, format, network, subnet, *region, computeServiceAccount)

	var w *daisy.Workflow
	if w, err = runExportWorkflow(ctx, getWorkflowPath(format, currentExecutablePath), varMap, *project,
		zone, timeout, scratchBucketGcsPath, oauth, ce, gcsLogsDisabled, cloudLogsDisabled,
		stdoutLogsDisabled, userLabels); err != nil {

		daisyutils.PostProcessDErrorForNetworkFlag("image export", err, network, w)

		return w, err
	}
	return w, nil
}
