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
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
)

// Make file paths mutable
var (
	WorkflowDir                = "daisy_workflows/image_import/"
	ImportWorkflow             = "import_image.wf.json"
	ImportFromImageWorkflow    = "import_from_image.wf.json"
	ImportAndTranslateWorkflow = "import_and_translate.wf.json"
)

const (
	logPrefix = "[import-image]"
)

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

func (i importer) runImport(varMap map[string]string, importWorkflowPath string) (*daisy.Workflow, error) {

	workflow, err := daisycommon.ParseWorkflow(importWorkflowPath, varMap,
		i.env.Project, i.env.Zone, i.env.ScratchBucketGcsPath, i.env.Oauth,
		i.translation.Timeout.String(), i.translation.CustomWorkflow, i.env.GcsLogsDisabled,
		i.env.CloudLogsDisabled, i.env.StdoutLogsDisabled)

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
			UserLabels:      i.img.Labels,
			BuildIDLabelKey: "gce-image-import-build-id",
			ImageLocation:   i.img.StorageLocation,
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
		daisyutils.UpdateAllInstanceNoExternalIP(w, i.env.NoExternalIP)
		if i.translation.UefiCompatible {
			daisyutils.UpdateToUEFICompatible(w)
		}
	}

	return workflow, workflow.RunWithModifiers(context.Background(), preValidateWorkflowModifier, postValidateWorkflowModifier)
}

type importer struct {
	storageClient *storage.Client
	env           Environment
	img           ImageSpec
	translation   TranslationSpec
}

// NewImporter constructs a new Importer instance.
func NewImporter(
	storageClient *storage.Client,
	env Environment,
	img ImageSpec,
	translation TranslationSpec) Importer {

	return importer{storageClient: storageClient, env: env, img: img, translation: translation}
}

// Importer runs the import workflow.
type Importer interface {
	Run(ctx context.Context) (*daisy.Workflow, error)
}

// Run runs import workflow.
func (i importer) Run(ctx context.Context) (*daisy.Workflow, error) {
	log.SetPrefix(logPrefix + " ")

	source, err := initAndValidateSource(i.translation.SourceFile, i.translation.SourceImage, i.storageClient)
	if err != nil {
		return nil, err
	}

	importWorkflowPath, translateWorkflowPath := getWorkflowPaths(
		source, i.translation.DataDisk, i.translation.OS,
		i.translation.CustomWorkflow, i.env.CurrentExecutablePath)

	varMap := buildDaisyVars(source, translateWorkflowPath, i.img.Name, i.img.Family,
		i.img.Description, i.env.Region, i.env.Subnet, i.env.Network,
		i.translation.NoGuestEnvironment, i.translation.SysprepWindows)

	var w *daisy.Workflow
	if w, err = i.runImport(varMap, importWorkflowPath); err != nil {

		daisyutils.PostProcessDErrorForNetworkFlag("image import", err, i.env.Network, w)

		return customizeErrorToDetectionResults(i.translation.OS, w, err)
	}
	return w, nil
}

// Environment describes where the import is running.
type Environment struct {
	ClientID              string
	Project               string
	Network               string
	Subnet                string
	Zone                  string
	Region                string
	ScratchBucketGcsPath  string
	Oauth                 string
	ComputeEndpoint       string
	CurrentExecutablePath string
	GcsLogsDisabled       bool
	CloudLogsDisabled     bool
	StdoutLogsDisabled    bool
	NoExternalIP          bool
}

// ImageSpec describes the metadata of the final image.
type ImageSpec struct {
	Name            string
	Family          string
	Description     string
	Labels          map[string]string
	StorageLocation string
}

// TranslationSpec describes the source and operations
// applied to an imported image.
type TranslationSpec struct {
	SourceFile         string
	SourceImage        string
	DataDisk           bool
	OS                 string
	NoGuestEnvironment bool
	Timeout            time.Duration
	CustomWorkflow     string
	UefiCompatible     bool
	SysprepWindows     bool
}
