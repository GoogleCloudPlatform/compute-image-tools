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
	"os"
	"strconv"
	"strings"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
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
)

const (
	logPrefix = "[import-image]"
)

func validateAndParseFlags(clientID string, imageName string, dataDisk bool, osID string, customTranWorkflow string, labels string) (
	map[string]string, error) {

	if err := validation.ValidateStringFlagNotEmpty(imageName, ImageNameFlagKey); err != nil {
		return nil, err
	}
	if err := validation.ValidateStringFlagNotEmpty(clientID, ClientIDFlagKey); err != nil {
		return nil, err
	}

	if !dataDisk && osID == "" && customTranWorkflow == "" {
		return nil, daisy.Errf("-data_disk, or -os, or -custom_translate_workflow has to be specified")
	}

	if dataDisk && (osID != "" || customTranWorkflow != "") {
		return nil, daisy.Errf("when -data_disk is specified, -os and -custom_translate_workflow should be empty")
	}

	if osID != "" && customTranWorkflow != "" {
		return nil, daisy.Errf("-os and -custom_translate_workflow can't be both specified")
	}

	if osID != "" {
		if err := daisyutils.ValidateOS(osID); err != nil {
			return nil, err
		}
	}

	var userLabels map[string]string
	if labels != "" {
		var err error
		userLabels, err = param.ParseKeyValues(labels)
		derr := daisy.ToDError(err)
		if derr != nil {
			return nil, derr
		}
	}

	return userLabels, nil
}

// Returns main workflow and translate workflow paths (if any)
func getWorkflowPaths(source resource, dataDisk bool, osID, customTranWorkflow, currentExecutablePath string) (string, string) {
	if isImage(source) {
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

func buildDaisyVars(source resource, translateWorkflowPath, imageName, family, description,
	region, subnet, network string, noGuestEnvironment bool, sysprepWindows bool) map[string]string {

	varMap := map[string]string{}

	varMap["image_name"] = strings.ToLower(strings.TrimSpace(imageName))
	if translateWorkflowPath != "" {
		varMap["translate_workflow"] = translateWorkflowPath
		varMap["install_gce_packages"] = strconv.FormatBool(!noGuestEnvironment)
		varMap["is_windows"] = strconv.FormatBool(strings.Contains(translateWorkflowPath, "windows"))
		varMap["sysprep_windows"] = strconv.FormatBool(sysprepWindows)
	}
	if isFile(source) {
		varMap["source_disk_file"] = source.path()
	} else {
		varMap["source_image"] = source.path()
	}
	varMap["family"] = strings.TrimSpace(family)
	varMap["description"] = strings.TrimSpace(description)
	if subnet != "" {
		varMap["import_subnet"] = param.GetRegionalResourcePath(strings.TrimSpace(region),
			"subnetworks", strings.TrimSpace(subnet))
		// When subnet is set, we need to grant a value to network to avoid fallback to default
		if network == "" {
			varMap["import_network"] = ""
		}
	}
	if network != "" {
		varMap["import_network"] = param.GetGlobalResourcePath("networks", strings.TrimSpace(network))
	}
	return varMap
}

func runImport(varMap map[string]string, importWorkflowPath string, zone string,
	timeout string, project string, scratchBucketGcsPath string, oauth string, ce string,
	gcsLogsDisabled bool, cloudLogsDisabled bool, stdoutLogsDisabled bool, noExternalIP bool,
	userLabels map[string]string, storageLocation string, uefiCompatible bool) (*daisy.Workflow, error) {

	workflow, err := daisycommon.ParseWorkflow(importWorkflowPath, varMap,
		project, zone, scratchBucketGcsPath, oauth, timeout, ce, gcsLogsDisabled,
		cloudLogsDisabled, stdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	preValidateWorkflowModifier := func(w *daisy.Workflow) {
		w.SetLogProcessHook(daisyutils.RemovePrivacyLogTag)
	}

	postValidateWorkflowModifier := func(w *daisy.Workflow) {
		buildID := os.Getenv(daisyutils.BuildIDOSEnvVarName)
		w.LogWorkflowInfo("Cloud Build ID: %s", buildID)
		rl := &daisyutils.ResourceLabeler{
			BuildID:         buildID,
			UserLabels:      userLabels,
			BuildIDLabelKey: "gce-image-import-build-id",
			ImageLocation:   storageLocation,
			InstanceLabelKeyRetriever: func(instanceName string) string {
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
		if uefiCompatible {
			daisyutils.UpdateToUEFICompatible(w)
		}
	}

	return workflow, workflow.RunWithModifiers(context.Background(), preValidateWorkflowModifier, postValidateWorkflowModifier)
}

// Run runs import workflow.
func Run(clientID string, imageName string, dataDisk bool, osID string, customTranWorkflow string,
	sourceFile string, sourceImage string, noGuestEnvironment bool, family string, description string,
	network string, subnet string, zone string, timeout string, project *string,
	scratchBucketGcsPath string, oauth string, ce string, gcsLogsDisabled bool, cloudLogsDisabled bool,
	stdoutLogsDisabled bool, noExternalIP bool, labels string, currentExecutablePath string, storageLocation string,
	uefiCompatible bool, sysprepWindows bool, storageClient *storage.Client,
	paramPopulator param.Populator) (*daisy.Workflow, error) {

	log.SetPrefix(logPrefix + " ")

	userLabels, err := validateAndParseFlags(clientID, imageName,
		dataDisk, osID, customTranWorkflow, labels)
	if err != nil {
		return nil, err
	}

	region := new(string)
	err = paramPopulator.PopulateMissingParameters(
		project, &zone, region, &scratchBucketGcsPath, sourceFile, &storageLocation)
	if err != nil {
		return nil, err
	}

	source, err := initAndValidateSource(sourceFile, sourceImage, storageClient)
	if err != nil {
		return nil, err
	}

	importWorkflowPath, translateWorkflowPath := getWorkflowPaths(source, dataDisk, osID,
		customTranWorkflow, currentExecutablePath)

	varMap := buildDaisyVars(source, translateWorkflowPath, imageName, family,
		description, *region, subnet, network, noGuestEnvironment, sysprepWindows)

	var w *daisy.Workflow
	if w, err = runImport(varMap, importWorkflowPath, zone, timeout, *project, scratchBucketGcsPath,
		oauth, ce, gcsLogsDisabled, cloudLogsDisabled, stdoutLogsDisabled,
		noExternalIP, userLabels, storageLocation, uefiCompatible); err != nil {

		daisyutils.PostProcessDErrorForNetworkFlag("image import", err, network, w)

		return customizeErrorToDetectionResults(osID, w, err)
	}
	return w, nil
}
