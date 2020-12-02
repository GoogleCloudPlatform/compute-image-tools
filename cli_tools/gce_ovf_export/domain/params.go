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

package ovfexportdomain

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
)

const (
	// InstanceNameFlagKey is key for instance name CLI flag
	InstanceNameFlagKey = "instance-name"

	// MachineImageNameFlagKey is key for machine image name CLI flag
	MachineImageNameFlagKey = "machine-image-name"

	// ClientIDFlagKey is key for client ID CLI flag
	ClientIDFlagKey = "client-id"

	// DestinationURIFlagKey is key for OVF/OVA GCS path CLI flag
	DestinationURIFlagKey = "destination-uri"

	// ReleaseTrackFlagKey is key for release track flag
	ReleaseTrackFlagKey = "release-track"

	// OvfFormatFlagKey is key for OVF format flag
	OvfFormatFlagKey = "ovf-format"

	mainWorkflowDir = "daisy_workflows/"
)

// OVFExportParams holds flags for OVF export
type OVFExportParams struct {

	// Instance export specific flags
	InstanceName string

	// Machine image specific flags
	MachineImageName string

	// Common flags
	ClientID             string
	DestinationURI       string
	OvfFormat            string
	DiskExportFormat     string
	OsID                 string
	Network              string
	Subnet               string
	NoExternalIP         bool
	Zone                 string
	Timeout              time.Duration
	Project              *string
	ScratchBucketGcsPath string
	Oauth                string
	Ce                   string
	GcsLogsDisabled      bool
	CloudLogsDisabled    bool
	StdoutLogsDisabled   bool
	ReleaseTrack         string
	BuildID              string

	// Non-flags
	WorkflowDir string
	Region      string
	Started     time.Time
}

func (params *OVFExportParams) String() string {
	return fmt.Sprintf("%#v", params)
}

// IsInstanceExport returns true if export represented by these params is
// instance export. False otherwise.
func (params *OVFExportParams) IsInstanceExport() bool {
	return params.InstanceName != ""
}

// IsMachineImageExport returns true if export represented by these params is
// a machine image export. False otherwise.
func (params *OVFExportParams) IsMachineImageExport() bool {
	return !params.IsInstanceExport()
}

//TODO: consolidate with ovf_importer.toWorkingDir
func toWorkingDir(currentDir, workflowDir string) string {
	wd, err := filepath.Abs(filepath.Dir(currentDir))
	if err == nil {
		return path.Join(wd, workflowDir)
	}
	return workflowDir
}

// InitWorkflowPath initializes workflow path field
func (params *OVFExportParams) InitWorkflowPath() {
	currentExecutablePath := filepath.Dir(os.Args[0])
	params.WorkflowDir = toWorkingDir(currentExecutablePath, mainWorkflowDir)
}

// DaisyAttrs returns the subset of DaisyAttrs that are required to instantiate
// a daisy workflow.
func (params *OVFExportParams) DaisyAttrs() daisycommon.WorkflowAttributes {
	return daisycommon.WorkflowAttributes{
		Project:           *params.Project,
		Zone:              params.Zone,
		GCSPath:           params.ScratchBucketGcsPath,
		OAuth:             params.Oauth,
		Timeout:           params.Timeout.String(),
		ComputeEndpoint:   params.Ce,
		DisableGCSLogs:    params.GcsLogsDisabled,
		DisableCloudLogs:  params.CloudLogsDisabled,
		DisableStdoutLogs: params.StdoutLogsDisabled,
		NoExternalIP:      params.NoExternalIP,
		WorkflowDirectory: params.WorkflowDir,
	}
}
