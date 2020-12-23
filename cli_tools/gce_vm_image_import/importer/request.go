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
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
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
	element := reflect.ValueOf(args).Elem()
	typeOfElement := element.Type()

	for i := 0; i < typeOfElement.NumField(); i++ {
		field := element.Field(i)
		typeOfField := typeOfElement.Field(i)
		if typeOfField.Tag.Get("name") == "" {
			continue
		}
		var present bool
		switch typeOfField.Type.Name() {
		case "string":
			regex := typeOfField.Tag.Get("regex")
			if regex == "" {
				panic("Required string field needs regex struct tag")
			}
			present = field.String() != ""
			if present && !regexp.MustCompile("^"+regex+"$").MatchString(field.String()) {
				return daisy.Errf("image_name %q does not match pattern "+regex, field.String())
			}
		case "Source":
			present = field.Interface() != nil
		case "Duration":
			present = field.Int() > 0
		}
		if !present {
			return fmt.Errorf("%s has to be specified", typeOfField.Tag.Get("name"))
		}
	}
	return nil
}

// ImageImportRequest includes the parameters required to perform an image import.
//
// Using tags for validation:
//  - All fields with the tag key 'name' will be validated.
//      - The value of 'name' is shown to the user in an error message.
//  - Validated string fields must also contain a regex pattern.
//  - Validated non-string fields are required to have a non-default value.
type ImageImportRequest struct {
	ExecutionID           string `name:"execution_id" regex:".+"`
	CloudLogsDisabled     bool
	ComputeEndpoint       string
	ComputeServiceAccount string
	WorkflowDir           string `name:"workflow_dir" regex:".+"`
	CustomWorkflow        string
	DataDisk              bool
	Description           string
	Family                string
	GcsLogsDisabled       bool
	ImageName             string `name:"image_name" regex:"[a-z]([-a-z0-9]{0,61}[a-z0-9])?"` // https://cloud.google.com/compute/docs/reference/rest/v1/images
	Inspect               bool
	Labels                map[string]string
	Network               string `name:"network" regex:".+"`
	NoExternalIP          bool
	NoGuestEnvironment    bool
	Oauth                 string
	BYOL                  bool
	OS                    string
	Project               string `name:"project" regex:".+"`
	ScratchBucketGcsPath  string `name:"scratch_bucket_gcs_path" regex:".+"`
	Source                Source `name:"source"`
	StdoutLogsDisabled    bool
	StorageLocation       string
	Subnet                string
	SysprepWindows        bool
	Timeout               time.Duration `name:"timeout"`
	UefiCompatible        bool
	Zone                  string `name:"zone" regex:".+"`
}

// EnvironmentSettings returns the subset of EnvironmentSettings that are required to instantiate
// a daisy workflow.
func (args ImageImportRequest) EnvironmentSettings() daisycommon.EnvironmentSettings {
	return daisycommon.EnvironmentSettings{
		Project:           args.Project,
		Zone:              args.Zone,
		GCSPath:           args.ScratchBucketGcsPath,
		OAuth:             args.Oauth,
		Timeout:           args.Timeout.String(),
		ComputeEndpoint:   args.ComputeEndpoint,
		DisableGCSLogs:    args.GcsLogsDisabled,
		DisableCloudLogs:  args.CloudLogsDisabled,
		DisableStdoutLogs: args.StdoutLogsDisabled,
		NoExternalIP:      args.NoExternalIP,
		WorkflowDirectory: args.WorkflowDir,
	}
}
