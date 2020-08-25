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
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	daisy_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

type bootableDiskProcessor struct {
	args              ImportArguments
	workflow          *daisy.Workflow
	computeDiskClient daisyCompute.Client
}

func (b *bootableDiskProcessor) process(pd persistentDisk) (persistentDisk, error) {
	// Due to GuestOS features limitations, a new disk needs to be created to add the additional "UEFI_COMPATIBLE"
	// and the old disk will be deleted.
	// If UEFI_COMPATIBLE is enforced in user input args (b.uefiCompatible),
	// then it has been honored in inflation stage, so no need to recreate a new disk here.
	if !b.args.UefiCompatible && pd.isUEFIDetected {
		diskName := fmt.Sprintf("disk-%v-uefi", b.args.ExecutionID)
		err := b.computeDiskClient.CreateDisk(b.args.Project, b.args.Zone, &compute.Disk{
			Name:            diskName,
			SourceDisk:      pd.uri,
			GuestOsFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
		})
		if err != nil {
			return pd, daisy.Errf("Failed to create UEFI disk: %v", err)
		}
		log.Println("UEFI disk created: ", diskName)

		// Cleanup the old disk after the new disk is created.
		cleanupDisk(b.computeDiskClient, b.args.Project, b.args.Zone, pd)

		// Update the new disk URI
		pd.uri = fmt.Sprintf("zones/%v/disks/%v", b.args.Zone, diskName)
	}

	b.workflow.AddVar("source_disk", pd.uri)
	var err error
	err = b.workflow.RunWithModifiers(context.Background(), b.preValidateFunc(), b.postValidateFunc())
	if err != nil {
		daisy_utils.PostProcessDErrorForNetworkFlag("image import", err, b.args.Network, b.workflow)
		err = customizeErrorToDetectionResults(b.args.OS,
			b.workflow.GetSerialConsoleOutputValue("detected_distro"),
			b.workflow.GetSerialConsoleOutputValue("detected_major_version"),
			b.workflow.GetSerialConsoleOutputValue("detected_minor_version"), err)
	}
	return pd, err
}

func (b *bootableDiskProcessor) cancel(reason string) bool {
	b.workflow.CancelWithReason(reason)
	return true
}

func (b *bootableDiskProcessor) traceLogs() []string {
	if b.workflow.Logger != nil {
		return b.workflow.Logger.ReadSerialPortLogs()
	}
	return []string{}
}

func newBootableDiskProcessor(computeDiskClient daisyCompute.Client, args ImportArguments) (processor, error) {
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
		args:              args,
		workflow:          workflow,
		computeDiskClient: computeDiskClient,
	}, err
}

func (b *bootableDiskProcessor) postValidateFunc() daisy.WorkflowModifier {
	return func(w *daisy.Workflow) {
		buildID := os.Getenv(daisy_utils.BuildIDOSEnvVarName)
		w.LogWorkflowInfo("Cloud Build ID: %s", buildID)
		rl := &daisy_utils.ResourceLabeler{
			BuildID:         buildID,
			UserLabels:      b.args.Labels,
			BuildIDLabelKey: "gce-image-import-build-id",
			ImageLocation:   b.args.StorageLocation,
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
		daisy_utils.UpdateAllInstanceNoExternalIP(w, b.args.NoExternalIP)
	}
}

func (b *bootableDiskProcessor) preValidateFunc() daisy.WorkflowModifier {
	return func(w *daisy.Workflow) {
		w.SetLogProcessHook(daisy_utils.RemovePrivacyLogTag)
	}
}
