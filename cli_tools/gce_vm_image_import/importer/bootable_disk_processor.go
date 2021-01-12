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
	"strconv"
	"strings"

	daisy_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

type bootableDiskProcessor struct {
	request  ImageImportRequest
	workflow *daisy.Workflow
	logger   logging.Logger
}

func (b *bootableDiskProcessor) process(pd persistentDisk) (persistentDisk, error) {

	b.workflow.AddVar("source_disk", pd.uri)
	var err error
	err = b.workflow.RunWithModifiers(context.Background(), b.preValidateFunc(), b.postValidateFunc())
	if err != nil {
		daisy_utils.PostProcessDErrorForNetworkFlag("image import", err, b.request.Network, b.workflow)
		err = customizeErrorToDetectionResults(b.request.OS,
			b.workflow.GetSerialConsoleOutputValue("detected_distro"),
			b.workflow.GetSerialConsoleOutputValue("detected_major_version"),
			b.workflow.GetSerialConsoleOutputValue("detected_minor_version"), err)
	}
	if b.workflow.Logger != nil {
		for _, trace := range b.workflow.Logger.ReadSerialPortLogs() {
			b.logger.Trace(trace)
		}
	}
	return pd, err
}

func (b *bootableDiskProcessor) cancel(reason string) bool {
	b.workflow.CancelWithReason(reason)
	return true
}

func newBootableDiskProcessor(request ImageImportRequest, wfPath string, logger logging.Logger) (processor, error) {
	vars := map[string]string{
		"image_name":           request.ImageName,
		"install_gce_packages": strconv.FormatBool(!request.NoGuestEnvironment),
		"sysprep":              strconv.FormatBool(request.SysprepWindows),
		"family":               request.Family,
		"description":          request.Description,
		"import_subnet":        request.Subnet,
		"import_network":       request.Network,
	}

	if request.ComputeServiceAccount != "" {
		vars["compute_service_account"] = request.ComputeServiceAccount
	}

	workflow, err := daisycommon.ParseWorkflow(wfPath, vars,
		request.Project, request.Zone, request.ScratchBucketGcsPath, request.Oauth, request.Timeout.String(),
		request.ComputeEndpoint, request.GcsLogsDisabled, request.CloudLogsDisabled, request.StdoutLogsDisabled)

	if err != nil {
		return nil, err
	}

	// Temporary fix to ensure gcloud shows daisy's output.
	// A less fragile approach is tracked in b/161567644.
	workflow.Name = LogPrefix

	return &bootableDiskProcessor{
		request:  request,
		workflow: workflow,
		logger:   logger,
	}, err
}

func (b *bootableDiskProcessor) postValidateFunc() daisy.WorkflowModifier {
	return func(w *daisy.Workflow) {
		buildID := os.Getenv(daisy_utils.BuildIDOSEnvVarName)
		w.LogWorkflowInfo("Cloud Build ID: %s", buildID)
		rl := &daisy_utils.ResourceLabeler{
			BuildID:         buildID,
			UserLabels:      b.request.Labels,
			BuildIDLabelKey: "gce-image-import-build-id",
			ImageLocation:   b.request.StorageLocation,
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
		daisy_utils.UpdateAllInstanceNoExternalIP(w, b.request.NoExternalIP)
	}
}

func (b *bootableDiskProcessor) preValidateFunc() daisy.WorkflowModifier {
	return func(w *daisy.Workflow) {
		w.SetLogProcessHook(daisy_utils.RemovePrivacyLogTag)
	}
}
