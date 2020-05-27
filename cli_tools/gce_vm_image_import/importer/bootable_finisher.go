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

package importer

import (
	"context"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"

	daisy_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
)

type bootableFinisher struct {
	workflow        *daisy.Workflow
	userLabels      map[string]string
	storageLocation string
	uefiCompatible  bool
	noExternalIP    bool
	network         string
	OS              string
}

func (d bootableFinisher) run(ctx context.Context) (err error) {
	err = d.workflow.RunWithModifiers(ctx, d.preValidateFunc(), d.postValidateFunc())
	if err != nil {
		daisy_utils.PostProcessDErrorForNetworkFlag("image import", err, d.network, d.workflow)
		err = customizeErrorToDetectionResults(d.OS,
			d.workflow.GetSerialConsoleOutputValue("detected_distro"),
			d.workflow.GetSerialConsoleOutputValue("detected_major_version"),
			d.workflow.GetSerialConsoleOutputValue("detected_minor_version"), err)
	}
	return err
}

func (d bootableFinisher) serials() []string {
	if d.workflow.Logger != nil {
		return d.workflow.Logger.ReadSerialPortLogs()
	}
	return []string{}
}

func newBootableFinisher(args ImportArguments, pd pd, workflowDirectory string) (finisher, error) {
	var translateWorkflowPath string
	if args.CustomWorkflow != "" {
		translateWorkflowPath = args.CustomWorkflow
	} else {
		relPath := daisy_utils.GetTranslateWorkflowPath(args.OS)
		translateWorkflowPath = path.Join(workflowDirectory, relPath)
	}

	vars := map[string]string{
		"image_name":           args.ImageName,
		"install_gce_packages": strconv.FormatBool(!args.NoGuestEnvironment),
		"sysprep":              strconv.FormatBool(args.SysprepWindows),
		"source_disk":          pd.uri,
		"family":               args.Family,
		"description":          args.Description,
		"import_subnet":        args.Subnet,
		"import_network":       args.Network,
	}

	workflow, err := daisycommon.ParseWorkflow(translateWorkflowPath, vars,
		args.Project, args.Zone, args.ScratchBucketGcsPath, args.Oauth, args.Timeout.String(),
		args.ComputeEndpoint, args.GcsLogsDisabled, args.CloudLogsDisabled, args.StdoutLogsDisabled)

	if err != nil {
		return nil, err
	}

	return bootableFinisher{
		workflow:        workflow,
		userLabels:      args.Labels,
		storageLocation: args.StorageLocation,
		uefiCompatible:  args.UefiCompatible,
		noExternalIP:    args.NoExternalIP,
		network:         args.Network,
	}, err
}

func (d bootableFinisher) postValidateFunc() daisy.WorkflowModifier {
	return func(w *daisy.Workflow) {
		buildID := os.Getenv(daisy_utils.BuildIDOSEnvVarName)
		w.LogWorkflowInfo("Cloud Build ID: %s", buildID)
		rl := &daisy_utils.ResourceLabeler{
			BuildID:         buildID,
			UserLabels:      d.userLabels,
			BuildIDLabelKey: "gce-image-import-build-id",
			ImageLocation:   d.storageLocation,
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
		daisy_utils.UpdateAllInstanceNoExternalIP(w, d.noExternalIP)
		if d.uefiCompatible {
			daisy_utils.UpdateToUEFICompatible(w)
		}
	}
}

func (d bootableFinisher) preValidateFunc() daisy.WorkflowModifier {
	return func(w *daisy.Workflow) {
		w.SetLogProcessHook(daisy_utils.RemovePrivacyLogTag)
	}
}
