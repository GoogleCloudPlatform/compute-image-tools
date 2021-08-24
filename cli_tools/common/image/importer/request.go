//  Copyright 2021 Google Inc. All Rights Reserved.
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
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
)

// Flags that are validated.
const (
	ImageFlag          = "image_name"
	ClientFlag         = "client_id"
	BYOLFlag           = "byol"
	DataDiskFlag       = "data_disk"
	OSFlag             = "os"
	CustomWorkflowFlag = "custom_translate_workflow"
)

func (args *ImageImportRequest) validate() error {
	if err := args.checkRequiredArguments(); err != nil {
		return err
	}

	if args.BYOL && (args.DataDisk || args.OS != "" || args.CustomWorkflow != "") {
		return fmt.Errorf("when -%s is specified, -%s, -%s, and -%s have to be empty",
			BYOLFlag, DataDiskFlag, OSFlag, CustomWorkflowFlag)
	}
	if args.DataDisk && (args.OS != "" || args.CustomWorkflow != "") {
		return fmt.Errorf("when -%s is specified, -%s and -%s should be empty",
			DataDiskFlag, OSFlag, CustomWorkflowFlag)
	}
	if args.OS != "" && args.CustomWorkflow != "" {
		return fmt.Errorf("-%s and -%s can't be both specified",
			OSFlag, CustomWorkflowFlag)
	}
	if !strings.HasSuffix(args.ScratchBucketGcsPath, args.ExecutionID) {
		return fmt.Errorf("Scratch bucket should have been namespaced with execution ID")
	}
	if args.OS != "" {
		if err := daisyutils.ValidateOS(args.OS); err != nil {
			return err
		}
	}
	return nil
}

func (args *ImageImportRequest) checkRequiredArguments() error {
	return validation.ValidateStruct(args)
}

// ImageImportRequest includes the parameters required to perform an image import.
//
// DaisyLogLinePrefix configures Daisy's stdout to include this prefix. During inflation,
// for example, a prefix of `ovf` would create a log line of `[ovf-inflate]`.
//
// Tags define validations; see validation.ValidateStruct for more info.
type ImageImportRequest struct {
	ExecutionID           string `name:"execution_id" validate:"required"`
	CloudLogsDisabled     bool
	ComputeEndpoint       string
	ComputeServiceAccount string
	WorkflowDir           string `name:"workflow_dir" validate:"required"`
	CustomWorkflow        string
	DataDisk              bool
	DaisyLogLinePrefix    string
	Description           string
	Family                string
	GcsLogsDisabled       bool
	ImageName             string `name:"image_name" validate:"required,gce_disk_image_name"`
	Inspect               bool
	Labels                map[string]string
	Network               string
	NoExternalIP          bool
	NoGuestEnvironment    bool
	Oauth                 string
	BYOL                  bool
	OS                    string
	Project               string `name:"project" validate:"required"`
	ScratchBucketGcsPath  string `name:"scratch_bucket_gcs_path" validate:"required"`
	Source                Source `name:"source" validate:"required"`
	StdoutLogsDisabled    bool
	StorageLocation       string
	Subnet                string
	SysprepWindows        bool
	Timeout               time.Duration `name:"timeout" validate:"required"`
	UefiCompatible        bool
	Zone                  string `name:"zone" validate:"required"`
}

// FixBYOLAndOSArguments fixes the user's arguments for the --os and --byol flags
// to follow the invariants from ImageImportRequest's validation where --byol
// may only be specified when --os is empty (implying that detection is being used).
//
// For example, `--byol --os=rhel-8` will be changed to `--os=rhel-8-byol`.
func FixBYOLAndOSArguments(osIDArgument *string, byolArgument *bool) {
	if *byolArgument && *osIDArgument != "" {
		// User wants OS override. Clear `-byol`, and ensure `--os` has the
		// suffix "-byol" if that's what the user intended.
		*byolArgument = false
		if !strings.HasSuffix(*osIDArgument, "byol") {
			*osIDArgument += "-byol"
		}
	}
}

// EnvironmentSettings returns the subset of EnvironmentSettings that are required to instantiate
// a daisy workflow.
func (args ImageImportRequest) EnvironmentSettings() daisyutils.EnvironmentSettings {
	return daisyutils.EnvironmentSettings{
		Project:               args.Project,
		Zone:                  args.Zone,
		GCSPath:               args.ScratchBucketGcsPath,
		OAuth:                 args.Oauth,
		Timeout:               args.Timeout.String(),
		ComputeEndpoint:       args.ComputeEndpoint,
		DaisyLogLinePrefix:    args.DaisyLogLinePrefix,
		DisableGCSLogs:        args.GcsLogsDisabled,
		DisableCloudLogs:      args.CloudLogsDisabled,
		DisableStdoutLogs:     args.StdoutLogsDisabled,
		Network:               args.Network,
		Subnet:                args.Subnet,
		ComputeServiceAccount: args.ComputeServiceAccount,
		NoExternalIP:          args.NoExternalIP,
		WorkflowDirectory:     args.WorkflowDir,
		Labels:                args.Labels,
		ExecutionID:           args.ExecutionID,
		StorageLocation:       args.StorageLocation,
	}
}
