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
	"context"
	"log"

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
)

const (
	logPrefix = "[import-image]"
)

func validateAndParseFlags(clientID string, imageName string, sourceFile string, sourceImage string, dataDisk bool, osID string, customTranWorkflow string, labels string) (
		string, string, map[string]string, error) {

	// call original validate, then do own validation
	return "", "", nil, nil
}

func runImport(ctx context.Context, varMap map[string]string, importWorkflowPath string, zone string,
		timeout string, project string, scratchBucketGcsPath string, oauth string, ce string,
		gcsLogsDisabled bool, cloudLogsDisabled bool, stdoutLogsDisabled bool, kmsKey string,
		kmsKeyring string, kmsLocation string, kmsProject string, noExternalIP bool,
		userLabels map[string]string, storageLocation string, uefiCompatible bool) (*daisy.Workflow, error) {

	// call original importer
	return nil, nil
}

// Run runs import workflow.
func Run(clientID string, imageName string, dataDisk bool, osID string, customTranWorkflow string,
		sourceFile string, noGuestEnvironment bool, family string, description string,
		network string, subnet string, zone string, timeout string, project *string,
		scratchBucketGcsPath string, oauth string, ce string, gcsLogsDisabled bool, cloudLogsDisabled bool,
		stdoutLogsDisabled bool, kmsKey string, kmsKeyring string, kmsLocation string, kmsProject string,
		noExternalIP bool, labels string, currentExecutablePath string, storageLocation string,
		uefiCompatible bool) (*daisy.Workflow, error) {

	log.SetPrefix(logPrefix + " ")

	/*
	sourceBucketName, sourceObjectName, userLabels, err := validateAndParseFlags(clientID, imageName,
		sourceFile, "", dataDisk, osID, customTranWorkflow, labels)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	metadataGCE := &compute.MetadataGCE{}
	storageClient, err := storage.NewStorageClient(
		ctx, logging.NewLogger(logPrefix), oauth)
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
	err = param.PopulateMissingParameters(project, &zone, region, &scratchBucketGcsPath,
		sourceFile, &storageLocation, metadataGCE, scratchBucketCreator, resourceLocationRetriever, storageClient)
	if err != nil {
		return nil, err
	}
	*/

	// do s3 thing
	// 1. export
	// 2. copy

	/*
	if sourceFile != "" {
		err = validateSourceFile(storageClient, sourceBucketName, sourceObjectName)
	}
	if err != nil {
		return nil, err
	}*/


	/*
	var w *daisy.Workflow
	if w, err = runImport(ctx, varMap, importWorkflowPath, zone, timeout, *project, scratchBucketGcsPath,
		oauth, ce, gcsLogsDisabled, cloudLogsDisabled, stdoutLogsDisabled, kmsKey, kmsKeyring,
		kmsLocation, kmsProject, noExternalIP, userLabels, storageLocation, uefiCompatible); err != nil {

		daisyutils.PostProcessDErrorForNetworkFlag("image import", err, network, w)

		return w, err
	}
	return w, nil
	 */
	return nil, nil
}
