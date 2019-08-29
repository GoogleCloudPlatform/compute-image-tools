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

package importer

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
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
	WorkflowDir                = "daisy_workflows/image_import/"
	ImportWorkflow             = "import_image.wf.json"
	ImportFromImageWorkflow    = "import_from_image.wf.json"
	ImportAndTranslateWorkflow = "import_and_translate.wf.json"
)

// Parameter key shared with other packages
const (
	ImageNameFlagKey = "image_name"
	ClientIDFlagKey  = "client_id"
	buildIDOSEnv     = "BUILD_ID"
)

func validateAndParseFlags(clientID string, imageName string, sourceFile string, sourceImage string, dataDisk bool, osID string, customTranWorkflow string, labels string) (
	string, string, map[string]string, error) {

	if err := validation.ValidateStringFlagNotEmpty(imageName, ImageNameFlagKey); err != nil {
		return "", "", nil, err
	}
	if err := validation.ValidateStringFlagNotEmpty(clientID, ClientIDFlagKey); err != nil {
		return "", "", nil, err
	}

	if !dataDisk && osID == "" && customTranWorkflow == "" {
		return "", "", nil, daisy.Errf("-data_disk, or -os, or -custom_translate_workflow has to be specified")
	}

	if dataDisk && (osID != "" || customTranWorkflow != "") {
		return "", "", nil, daisy.Errf("when -data_disk is specified, -os and -custom_translate_workflow should be empty")
	}

	if osID != "" && customTranWorkflow != "" {
		return "", "", nil, daisy.Errf("-os and -custom_translate_workflow can't be both specified")
	}

	if sourceFile == "" && sourceImage == "" {
		return "", "", nil, daisy.Errf("-source_file or -source_image has to be specified")
	}

	if sourceFile != "" && sourceImage != "" {
		return "", "", nil, daisy.Errf("either -source_file or -source_image has to be specified, but not both %v %v", sourceFile, sourceImage)
	}

	if osID != "" {
		if err := daisyutils.ValidateOS(osID); err != nil {
			return "", "", nil, err
		}
	}

	var sourceBucketName, sourceObjectName string

	if sourceFile != "" {
		var err error
		sourceBucketName, sourceObjectName, err = storage.SplitGCSPath(sourceFile)
		if err != nil {
			return "", "", nil, daisy.Errf("failed to split source file GCS path: %v", err)
		}
	}

	var userLabels map[string]string
	if labels != "" {
		var err error
		userLabels, err = param.ParseKeyValues(labels)
		derr := daisy.ToDError(err)
		if derr != nil {
			return "", "", nil, derr
		}
	}

	return sourceBucketName, sourceObjectName, userLabels, nil
}

// validate source file is not a compression file by checking file header.
func validateSourceFile(storageClient domain.StorageClientInterface, sourceBucketName, sourceObjectName string) error {
	rc, err := storageClient.GetObjectReader(sourceBucketName, sourceObjectName)
	if err != nil {
		return daisy.Errf("failed to read GCS file when validating source file: unable to open file from bucket %q, file %q: %v", sourceBucketName, sourceObjectName, err)
	}
	defer rc.Close()

	// Detect whether it's a compressed file by extracting compressed file header
	if _, err = gzip.NewReader(rc); err == nil {
		return daisy.Errf("cannot import an image from a compressed file. Please provide a path to an uncompressed image file. If the compressed file is an image exported from Google Compute Engine, please use 'images create' instead")
	}

	return nil
}

// Returns main workflow and translate workflow paths (if any)
func getWorkflowPaths(dataDisk bool, osID, sourceImage, customTranWorkflow, currentExecutablePath string) (string, string) {
	if sourceImage != "" {
		return path.ToWorkingDir(WorkflowDir+ImportFromImageWorkflow, currentExecutablePath), getTranslateWorkflowPath(customTranWorkflow, osID)
	}
	if dataDisk {
		return path.ToWorkingDir(WorkflowDir+ImportWorkflow, currentExecutablePath), ""
	}
	return path.ToWorkingDir(WorkflowDir+ImportAndTranslateWorkflow, currentExecutablePath), getTranslateWorkflowPath(customTranWorkflow, osID)
}

func getTranslateWorkflowPath(customTranslateWorkflow, osID string) string {
	if customTranslateWorkflow != "" {
		return customTranslateWorkflow
	}
	return daisyutils.GetTranslateWorkflowPath(osID)
}

func buildDaisyVars(translateWorkflowPath, imageName, sourceFile, sourceImage, family, description,
	region, subnet, network string, noGuestEnvironment bool) map[string]string {

	varMap := map[string]string{}

	varMap["image_name"] = strings.ToLower(imageName)
	if translateWorkflowPath != "" {
		varMap["translate_workflow"] = translateWorkflowPath
		varMap["install_gce_packages"] = strconv.FormatBool(!noGuestEnvironment)
	}
	if sourceFile != "" {
		varMap["source_disk_file"] = sourceFile
	}
	if sourceImage != "" {
		varMap["source_image"] = fmt.Sprintf("global/images/%v", sourceImage)
	}
	varMap["family"] = family
	varMap["description"] = description
	if subnet != "" {
		varMap["import_subnet"] = fmt.Sprintf("regions/%v/subnetworks/%v", region, subnet)
		// When subnet is set, we need to grant a value to network to avoid fallback to default
		if network == "" {
			varMap["import_network"] = ""
		}
	}
	if network != "" {
		varMap["import_network"] = fmt.Sprintf("global/networks/%v", network)
	}
	return varMap
}

func runImport(ctx context.Context, varMap map[string]string, importWorkflowPath string, zone string,
	timeout string, project string, scratchBucketGcsPath string, oauth string, ce string,
	gcsLogsDisabled bool, cloudLogsDisabled bool, stdoutLogsDisabled bool, kmsKey string,
	kmsKeyring string, kmsLocation string, kmsProject string, noExternalIP bool,
	userLabels map[string]string, storageLocation string) error {

	workflow, err := daisycommon.ParseWorkflow(importWorkflowPath, varMap,
		project, zone, scratchBucketGcsPath, oauth, timeout, ce, gcsLogsDisabled,
		cloudLogsDisabled, stdoutLogsDisabled)
	if err != nil {
		return err
	}

	workflowModifier := func(w *daisy.Workflow) {
		workflow.LogWorkflowInfo("Cloud Build ID: %s", os.Getenv(buildIDOSEnv))
		rl := &daisyutils.ResourceLabeler{
			BuildID:         os.Getenv(buildIDOSEnv),
			UserLabels:      userLabels,
			BuildIDLabelKey: "gce-image-import-build-id",
			ImageLocation:   storageLocation,
			InstanceLabelKeyRetriever: func(instance *daisy.Instance) string {
				return "gce-image-import-tmp"
			},
			DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
				return "gce-image-import-tmp"
			},
			ImageLabelKeyRetriever: func(imageName string) string {
				imageTypeLabel := "gce-image-import"
				if strings.Contains(imageName, "untranslated") {
					imageTypeLabel = "gce-image-import-tmp"
				}
				return imageTypeLabel
			}}
		rl.LabelResources(w)
		daisyutils.UpdateAllInstanceNoExternalIP(w, noExternalIP)
	}

	return workflow.RunWithModifiers(ctx, nil, workflowModifier)
}

// Run runs import workflow.
func Run(clientID string, imageName string, dataDisk bool, osID string, customTranWorkflow string,
	sourceFile string, sourceImage string, noGuestEnvironment bool, family string, description string,
	network string, subnet string, zone string, timeout string, project string,
	scratchBucketGcsPath string, oauth string, ce string, gcsLogsDisabled bool, cloudLogsDisabled bool,
	stdoutLogsDisabled bool, kmsKey string, kmsKeyring string, kmsLocation string, kmsProject string,
	noExternalIP bool, labels string, currentExecutablePath string, storageLocation string) error {

	sourceBucketName, sourceObjectName, userLabels, err := validateAndParseFlags(clientID, imageName,
		sourceFile, sourceImage, dataDisk, osID, customTranWorkflow, labels)
	if err != nil {
		return err
	}

	ctx := context.Background()
	metadataGCE := &compute.MetadataGCE{}
	storageClient, err := storage.NewStorageClient(
		ctx, logging.NewLogger("[image-import]"), oauth)
	if err != nil {
		return daisy.Errf("error creating storage client: %v", err)
	}
	defer storageClient.Close()

	scratchBucketCreator := storage.NewScratchBucketCreator(ctx, storageClient)
	computeClient, err := param.CreateComputeClient(&ctx, oauth, ce)
	if err != nil {
		return err
	}
	zoneRetriever := storage.NewZoneRetriever(metadataGCE, computeClient)

	region := new(string)
	err = param.PopulateMissingParameters(&project, &zone, region, &scratchBucketGcsPath,
		sourceFile, metadataGCE, scratchBucketCreator, zoneRetriever, storageClient)
	if err != nil {
		return err
	}

	if sourceFile != "" {
		err = validateSourceFile(storageClient, sourceBucketName, sourceObjectName)
	}
	if err != nil {
		return err
	}

	importWorkflowPath, translateWorkflowPath := getWorkflowPaths(dataDisk, osID, sourceImage,
		customTranWorkflow, currentExecutablePath)

	varMap := buildDaisyVars(translateWorkflowPath, imageName, sourceFile, sourceImage, family,
		description, *region, subnet, network, noGuestEnvironment)

	if err = runImport(ctx, varMap, importWorkflowPath, zone, timeout, project, scratchBucketGcsPath,
		oauth, ce, gcsLogsDisabled, cloudLogsDisabled, stdoutLogsDisabled, kmsKey, kmsKeyring,
		kmsLocation, kmsProject, noExternalIP, userLabels, storageLocation); err != nil {

		return err
	}
	return nil
}
