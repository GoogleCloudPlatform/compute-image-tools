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
	"os"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
)

// Make file paths mutable
var (
	WorkflowDir                = "daisy_workflows/image_import/"
	ImportWorkflow             = "import_image.wf.json"
	ImportFromImageWorkflow    = "import_from_image.wf.json"
	ImportAndTranslateWorkflow = "import_and_translate.wf.json"
)

// Returns main workflow and translate workflow paths (if any)
func getWorkflowPaths(source Source, dataDisk bool, osID, customTranWorkflow, currentExecutablePath string) (string, string) {
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

func buildDaisyVars(source Source, translateWorkflowPath, imageName, family, description,
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
		varMap["source_disk_file"] = source.Path()
	} else {
		varMap["source_image"] = source.Path()
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

func (importer importer) runImport(varMap map[string]string, importWorkflowPath string) (*daisy.Workflow, error) {

	workflow, err := daisycommon.ParseWorkflow(importWorkflowPath, varMap, importer.Project, importer.Zone,
		importer.ScratchBucketGcsPath, importer.Oauth, importer.Timeout.String(), importer.CustomWorkflow,
		importer.GcsLogsDisabled, importer.CloudLogsDisabled, importer.StdoutLogsDisabled)

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
			UserLabels:      importer.Labels,
			BuildIDLabelKey: "gce-image-import-build-id",
			ImageLocation:   importer.StorageLocation,
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
		daisyutils.UpdateAllInstanceNoExternalIP(w, importer.NoExternalIP)
		if importer.UefiCompatible {
			daisyutils.UpdateToUEFICompatible(w)
		}
	}

	return workflow, workflow.RunWithModifiers(context.Background(), preValidateWorkflowModifier, postValidateWorkflowModifier)
}

type importer struct {
	ImportArguments
}

// NewImporter constructs an Importer instance.
func NewImporter(importArguments ImportArguments) Importer {
	return importer{ImportArguments: importArguments}
}

// Importer runs the import workflow.
type Importer interface {
	Run(ctx context.Context) (*daisy.Workflow, error)
}

// Run runs import workflow.
func (importer importer) Run(ctx context.Context) (w *daisy.Workflow, err error) {
	importWorkflowPath, translateWorkflowPath := getWorkflowPaths(
		importer.Source, importer.DataDisk, importer.OS,
		importer.CustomWorkflow, importer.CurrentExecutablePath)

	varMap := buildDaisyVars(importer.Source, translateWorkflowPath, importer.ImageName,
		importer.Family, importer.Description, importer.Region, importer.Subnet,
		importer.Network, importer.NoGuestEnvironment, importer.SysprepWindows)

	if w, err = importer.runImport(varMap, importWorkflowPath); err != nil {

		daisyutils.PostProcessDErrorForNetworkFlag("image import", err, importer.Network, w)

		return w, customizeErrorToDetectionResults(importer.OS,
			w.GetSerialConsoleOutputValue("detected_distro"),
			w.GetSerialConsoleOutputValue("detected_major_version"),
			w.GetSerialConsoleOutputValue("detected_minor_version"), err)
	}
	return w, nil
}
