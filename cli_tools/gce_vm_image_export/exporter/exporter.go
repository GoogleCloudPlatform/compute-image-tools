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
	"strconv"
	"strings"

	"google.golang.org/api/option"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	stringutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/string"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

// Make file paths mutable
var (
	WorkflowDir              = "daisy_workflows/export/"
	ExportWorkflow           = "image_export.wf.json"
	ExportAndConvertWorkflow = "image_export_ext.wf.json"
)

// Parameter key shared with external packages
const (
	ClientIDFlagKey           = "client_id"
	DestinationURIFlagKey     = "destination_uri"
	SourceImageFlagKey        = "source_image"
	SourceDiskSnapshotFlagKey = "source_disk_snapshot"

	targetSizeGBKey = "target-size-gb"
	sourceSizeGBKey = "source-size-gb"
)

func validateAndParseFlags(destinationURI string, sourceImage string, sourceDiskSnapshot string, labels string) (map[string]string, error) {
	if err := validation.ValidateStringFlagNotEmpty(destinationURI, DestinationURIFlagKey); err != nil {
		return nil, err
	}
	if err := validation.ValidateExactlyOneOfStringFlagNotEmpty(map[string]string{
		SourceImageFlagKey:        sourceImage,
		SourceDiskSnapshotFlagKey: sourceDiskSnapshot,
	}); err != nil {
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

func buildDaisyVars(destinationURI string, sourceImage string, sourceDiskSnapshot string, bufferDiskSizeGb int64, format string, network string,
	subnet string, region string, computeServiceAccount string) map[string]string {

	destinationURI = strings.TrimSpace(destinationURI)
	sourceImage = strings.TrimSpace(sourceImage)
	sourceDiskSnapshot = strings.TrimSpace(sourceDiskSnapshot)
	format = strings.TrimSpace(format)
	network = strings.TrimSpace(network)
	subnet = strings.TrimSpace(subnet)
	region = strings.TrimSpace(region)
	computeServiceAccount = strings.TrimSpace(computeServiceAccount)

	varMap := map[string]string{}

	varMap["destination"] = destinationURI

	if sourceImage != "" {
		varMap["source_image"] = param.GetGlobalResourcePath(
			"images", sourceImage)
	}

	if sourceDiskSnapshot != "" {
		varMap["source_disk_snapshot"] = param.GetGlobalResourcePath("snapshots", sourceDiskSnapshot)
	}

	if bufferDiskSizeGb > 0 {
		varMap["export_instance_disk_size"] = strconv.FormatInt(bufferDiskSizeGb, 10)
	}

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

// Run runs export workflow.
func Run(clientID string, destinationURI string, sourceImage string, sourceDiskSnapshot string, format string,
	project *string, network string, subnet string, zone string, timeout string,
	scratchBucketGcsPath string, oauth string, ce string, computeServiceAccount string, gcsLogsDisabled bool,
	cloudLogsDisabled bool, stdoutLogsDisabled bool, labels string, currentExecutablePath string, logger logging.Logger) error {

	userLabels, err := validateAndParseFlags(destinationURI, sourceImage, sourceDiskSnapshot, labels)
	if err != nil {
		return err
	}

	ctx := context.Background()
	metadataGCE := &compute.MetadataGCE{}
	storageClient, err := storage.NewStorageClient(
		ctx, logger, option.WithCredentialsFile(oauth))
	if err != nil {
		return err
	}
	defer storageClient.Close()

	scratchBucketCreator := storage.NewScratchBucketCreator(ctx, storageClient)
	computeClient, err := param.CreateComputeClient(&ctx, oauth, ce)
	if err != nil {
		return err
	}
	resourceLocationRetriever := storage.NewResourceLocationRetriever(metadataGCE, computeClient)

	region := new(string)
	paramPopulator := param.NewPopulator(param.NewNetworkResolver(computeClient), metadataGCE, storageClient, resourceLocationRetriever, scratchBucketCreator)
	err = paramPopulator.PopulateMissingParameters(project, clientID, &zone, region, &scratchBucketGcsPath, destinationURI, nil, &network, &subnet)
	if err != nil {
		return err
	}

	var bufferDiskSizeGb int64
	if sourceImage != "" {
		if bufferDiskSizeGb, err = validateImageExists(computeClient, *project, sourceImage); err != nil {
			return err
		}
	} else {
		if bufferDiskSizeGb, err = validateSnapshotExists(computeClient, *project, sourceDiskSnapshot); err != nil {
			return err
		}
	}

	varMap := buildDaisyVars(destinationURI, sourceImage, sourceDiskSnapshot, bufferDiskSizeGb, format, network, subnet, *region, computeServiceAccount)

	workflowProvider := func() (*daisy.Workflow, error) {
		return daisy.NewFromFile(getWorkflowPath(format, currentExecutablePath))
	}

	env := daisyutils.EnvironmentSettings{
		Project:               *project,
		Zone:                  zone,
		GCSPath:               scratchBucketGcsPath,
		OAuth:                 oauth,
		Timeout:               timeout,
		ComputeEndpoint:       ce,
		DisableGCSLogs:        gcsLogsDisabled,
		DisableCloudLogs:      cloudLogsDisabled,
		DisableStdoutLogs:     stdoutLogsDisabled,
		Network:               network,
		Subnet:                subnet,
		ComputeServiceAccount: computeServiceAccount,
		Labels:                userLabels,
		ExecutionID:           os.Getenv(os.Getenv(daisyutils.BuildIDOSEnvVarName)),
		Tool: daisyutils.Tool{
			HumanReadableName: "gce image export",
			ResourceLabelName: "gce-image-export",
		},
	}

	if env.ExecutionID == "" {
		env.ExecutionID = path.RandString(5)
	}
	values, err := daisyutils.NewDaisyWorker(workflowProvider, env, logger).RunAndReadSerialValues(
		varMap, targetSizeGBKey, sourceSizeGBKey)
	logger.Metric(&pb.OutputInfo{
		SourcesSizeGb: []int64{stringutils.SafeStringToInt(values[sourceSizeGBKey])},
		TargetsSizeGb: []int64{stringutils.SafeStringToInt(values[targetSizeGBKey])},
	})
	return err
}

// validateImageExists checks whether imageName exists in the specified project.
//
// This validates when imageName is a valid image name, and skips validation if
// the imageName is a URI, or something that's not recognized as an image.
// The simplistic validation avoids false negatives; Daisy has robust logic for
// interpreting the various permutations of specifying an image and project,
// and we don't want to copy that here, since this is a convenience method to create
// user-friendly messages.
func validateImageExists(computeClient daisyCompute.Client, project string, imageUri string) (diskSizeGb int64, err error) {
	// try to get image even before validation in case it's a valid URL,
	// in order to obtain its size
	var imageName string
	if err := validation.ValidateImageName(imageUri); err != nil {
		if project, imageName, err = validation.ValidateImageUri(imageUri); err != nil {
			return diskSizeGb, nil
		}
	} else {
		imageName = imageUri
	}
	log.Printf("Fetghing image %q from project %q.", imageName, project)
	image, err := computeClient.GetImage(project, imageName)
	if err != nil {
		log.Printf("Error when fetching image %q: %q.", imageUri, err)
		return diskSizeGb, daisy.Errf("Image %q not found", imageUri)
	}
	return image.DiskSizeGb, nil
}

// validateSnapshotExists checks whether snapshotName exists in the specified project.
//
// This validates when snapshotName is a valid snapshot name, and skips validation if
// the snapshotName is a URI, or something that's not recognized as a snapshot.
// The simplistic validation avoids false negatives; Daisy has robust logic for
// interpreting the various permutations of specifying a snapshot and project,
// and we don't want to copy that here, since this is a convenience method to create
// user-friendly messages.
func validateSnapshotExists(computeClient daisyCompute.Client, project string, snapshotName string) (diskSizeGb int64, err error) {
	// try to get snapshot even before validation in case it's a valid URL,
	// in order to obtain its size

	if err := validation.ValidateSnapshotName(snapshotName); err != nil {
		return diskSizeGb, nil
	}
	snapshot, err := computeClient.GetSnapshot(project, snapshotName)
	if err != nil {
		log.Printf("Error when fetching snapshot %q: %q.", snapshotName, err)
		return diskSizeGb, daisy.Errf("Snapshot %q not found", snapshotName)
	}

	return snapshot.DiskSizeGb, nil
}
