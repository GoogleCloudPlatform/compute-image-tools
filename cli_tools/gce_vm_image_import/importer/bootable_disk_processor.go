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

type bootableDiskProcessor struct {
	workflow        *daisy.Workflow
	userLabels      map[string]string
	storageLocation string
	uefiCompatible  bool
	noExternalIP    bool
	network         string
	OS              string
}

func (b bootableDiskProcessor) process() (err error) {
	err = b.workflow.RunWithModifiers(context.Background(), b.preValidateFunc(), b.postValidateFunc())
	if err != nil {
		daisy_utils.PostProcessDErrorForNetworkFlag("image import", err, b.network, b.workflow)
		err = customizeErrorToDetectionResults(b.OS,
			b.workflow.GetSerialConsoleOutputValue("detected_distro"),
			b.workflow.GetSerialConsoleOutputValue("detected_major_version"),
			b.workflow.GetSerialConsoleOutputValue("detected_minor_version"), err)
	}
	return err
}

func (b bootableDiskProcessor) cancel(reason string) bool {
	b.workflow.CancelWithReason(reason)
	return true
}

func (b bootableDiskProcessor) traceLogs() []string {
	if b.workflow.Logger != nil {
		return b.workflow.Logger.ReadSerialPortLogs()
	}
	return []string{}
}

func newBootableDiskProcessor(args ImportArguments, pd persistentDisk) (processor, error) {
	var translateWorkflowPath string
	if args.CustomWorkflow != "" {
		translateWorkflowPath = args.CustomWorkflow
	} else {
		relPath := daisy_utils.GetTranslateWorkflowPath(args.OS)
		translateWorkflowPath = path.Join(args.WorkflowDir, "image_import", relPath)
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

	// Temporary fix to ensure gcloud shows daisy's output.
	// A less fragile approach is tracked in b/161567644.
	workflow.Name = LogPrefix

	return &bootableDiskProcessor{
		workflow:        workflow,
		userLabels:      args.Labels,
		storageLocation: args.StorageLocation,
		uefiCompatible:  args.UefiCompatible,
		noExternalIP:    args.NoExternalIP,
		network:         args.Network,
		OS:              args.OS,
	}, err
}

func (b bootableDiskProcessor) postValidateFunc() daisy.WorkflowModifier {
	return func(w *daisy.Workflow) {
		buildID := os.Getenv(daisy_utils.BuildIDOSEnvVarName)
		w.LogWorkflowInfo("Cloud Build ID: %s", buildID)
		rl := &daisy_utils.ResourceLabeler{
			BuildID:         buildID,
			UserLabels:      b.userLabels,
			BuildIDLabelKey: "gce-image-import-build-id",
			ImageLocation:   b.storageLocation,
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
		daisy_utils.UpdateAllInstanceNoExternalIP(w, b.noExternalIP)
		if b.uefiCompatible {
			daisy_utils.UpdateToUEFICompatible(w)
		}
	}
}

func (b bootableDiskProcessor) preValidateFunc() daisy.WorkflowModifier {
	return func(w *daisy.Workflow) {
		w.SetLogProcessHook(daisy_utils.RemovePrivacyLogTag)
	}
}
