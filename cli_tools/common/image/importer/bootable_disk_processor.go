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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/distro"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

type bootableDiskProcessor struct {
	request    ImageImportRequest
	workflow   *daisy.Workflow
	logger     logging.Logger
	detectedOs distro.Release
}

func (b *bootableDiskProcessor) process(pd persistentDisk) (persistentDisk, error) {
	b.logger.User("Making disk bootable on Google Compute Engine")
	b.workflow.AddVar("source_disk", pd.uri)
	var err error
	err = b.workflow.Run(context.Background())
	if err != nil {
		b.logger.User("Finished making disk bootable")
		daisyutils.PostProcessDErrorForNetworkFlag("image import", err, b.request.Network, b.workflow)
		err = customizeErrorToDetectionResults(b.logger, b.request.OS, b.detectedOs, err)
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

func newBootableDiskProcessor(request ImageImportRequest, wfPath string, logger logging.Logger, detectedOs distro.Release) (processor, error) {
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

	workflow, err := daisyutils.ParseWorkflow(wfPath, vars,
		request.Project, request.Zone, request.ScratchBucketGcsPath, request.Oauth, request.Timeout.String(),
		request.ComputeEndpoint, request.GcsLogsDisabled, request.CloudLogsDisabled, request.StdoutLogsDisabled)

	daisyutils.UpdateAllInstanceNoExternalIP(workflow, request.NoExternalIP)
	workflow.SetLogProcessHook(daisyutils.RemovePrivacyLogTag)
	if err != nil {
		return nil, err
	}

	// Daisy uses the workflow name as the prefix for log lines.
	logPrefix := request.DaisyLogLinePrefix
	if logPrefix != "" {
		logPrefix += "-"
	}
	workflow.Name = logPrefix + "translate"

	diskProcessor := &bootableDiskProcessor{
		request:    request,
		workflow:   workflow,
		logger:     logger,
		detectedOs: detectedOs,
	}
	diskProcessor.labelResources()
	return diskProcessor, err
}

func (b *bootableDiskProcessor) labelResources() {
	buildID := os.Getenv(daisyutils.BuildIDOSEnvVarName)
	b.logger.User("Cloud Build ID: " + buildID)
	rl := &daisyutils.ResourceLabeler{
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
	rl.LabelResources(b.workflow)
}
