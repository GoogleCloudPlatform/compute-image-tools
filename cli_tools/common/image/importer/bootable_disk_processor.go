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
	"strconv"
	"strings"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/distro"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
)

type bootableDiskProcessor struct {
	request    ImageImportRequest
	worker     daisyutils.DaisyWorker
	vars       map[string]string
	logger     logging.Logger
	detectedOs distro.Release
}

func (b *bootableDiskProcessor) process(pd persistentDisk) (persistentDisk, error) {
	b.logger.User("Making disk bootable on Google Compute Engine")
	b.vars["source_disk"] = pd.uri
	var err error
	err = b.worker.Run(b.vars)
	if err != nil {
		err = customizeErrorToDetectionResults(b.logger, b.request.OS, b.detectedOs, err)
	} else {
		b.logger.User("Finished making disk bootable")
	}
	return pd, err
}

func (b *bootableDiskProcessor) cancel(reason string) bool {
	b.worker.Cancel(reason)
	return true
}

func newBootableDiskProcessor(request ImageImportRequest, wfPath string, logger logging.Logger, detectedOs distro.Release) processor {
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

	workflowProvider := func() (*daisy.Workflow, error) {
		wf, err := daisyutils.ParseWorkflow(wfPath, vars,
			request.Project, request.Zone, request.ScratchBucketGcsPath, request.Oauth, request.Timeout.String(),
			request.ComputeEndpoint, request.GcsLogsDisabled, request.CloudLogsDisabled, request.StdoutLogsDisabled)

		if !strings.Contains(strings.ToLower(wfPath), "windows") {
			var disks *[]*compute.AttachedDisk
			if wf.Steps["translate-disk"] != nil {
				disks = &wf.Steps["translate-disk"].IncludeWorkflow.Workflow.Steps["translate-disk-inst"].CreateInstances.Instances[0].Disks
			} else if (wf.Steps["translate-disk-inst"]) != nil {
				// case translation wf that doesn't include another workflow (e.g. opensuse_15)
				disks = &wf.Steps["translate-disk-inst"].CreateInstances.Instances[0].Disks
			}

			for _, dataAttachedDisk := range request.TanslationWorkerAttachedDisks {
				*disks = append(*disks, dataAttachedDisk)
			}
		}

		return wf, err
	}

	env := request.EnvironmentSettings()
	if env.DaisyLogLinePrefix != "" {
		env.DaisyLogLinePrefix += "-"
	}
	env.DaisyLogLinePrefix += "translate"
	diskProcessor := &bootableDiskProcessor{
		request:    request,
		worker:     daisyutils.NewDaisyWorker(workflowProvider, env, logger, createResourceLabeler(request)),
		logger:     logger,
		detectedOs: detectedOs,
		vars:       vars,
	}
	return diskProcessor
}

func createResourceLabeler(request ImageImportRequest) *daisyutils.ResourceLabeler {
	return &daisyutils.ResourceLabeler{
		BuildID:         request.ExecutionID,
		UserLabels:      request.Labels,
		BuildIDLabelKey: "gce-image-import-build-id",
		ImageLocation:   request.StorageLocation,
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
}
