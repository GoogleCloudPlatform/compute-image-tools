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

	//Alpha represents alpha release track
	Alpha = "alpha"

	//Beta represents beta release track
	Beta = "beta"

	//GA represents GA release track
	GA = "ga"
)

// OVFExportArgs holds args for OVF export
type OVFExportArgs struct {

	// Instance export specific args
	InstanceName string

	// Machine image specific args
	MachineImageName string

	// Common args
	ClientID             string
	ClientVersion        string
	DestinationURI       string
	OvfFormat            string
	DiskExportFormat     string
	OsID                 string
	Network              string
	Subnet               string
	NoExternalIP         bool
	Zone                 string
	Timeout              time.Duration
	Project              string
	ScratchBucketGcsPath string
	Oauth                string
	Ce                   string
	GcsLogsDisabled      bool
	CloudLogsDisabled    bool
	StdoutLogsDisabled   bool
	ReleaseTrack         string
	BuildID              string

	// Non-args
	WorkflowDir string
	Region      string
	Started     time.Time
}

// NewOVFExportArgs parses args to create an NewOVFExportArgs instance.
// No validation is done here.
func NewOVFExportArgs(args []string) *OVFExportArgs {
	currentExecutablePath := filepath.Dir(os.Args[0])
	ovfExportArgs := &OVFExportArgs{
		WorkflowDir:      toWorkingDir(currentExecutablePath, mainWorkflowDir),
		Started:          time.Now(),
		DiskExportFormat: "vmdk",
		ReleaseTrack:     GA,
	}
	return ovfExportArgs
}

func (args *OVFExportArgs) String() string {
	return fmt.Sprintf("%#v", args)
}

// IsInstanceExport returns true if export represented by these args is
// instance export. False otherwise.
func (args *OVFExportArgs) IsInstanceExport() bool {
	return args.InstanceName != ""
}

// IsMachineImageExport returns true if export represented by these args is
// a machine image export. False otherwise.
func (args *OVFExportArgs) IsMachineImageExport() bool {
	return !args.IsInstanceExport()
}

//TODO: consolidate with ovf_importer.toWorkingDir
func toWorkingDir(currentDir, workflowDir string) string {
	wd, err := filepath.Abs(filepath.Dir(currentDir))
	if err == nil {
		return path.Join(wd, workflowDir)
	}
	return workflowDir
}

// DaisyAttrs returns the subset of DaisyAttrs that are required to instantiate
// a daisy workflow.
func (args *OVFExportArgs) DaisyAttrs() daisycommon.WorkflowAttributes {
	return daisycommon.WorkflowAttributes{
		Project:           args.Project,
		Zone:              args.Zone,
		GCSPath:           args.ScratchBucketGcsPath,
		OAuth:             args.Oauth,
		Timeout:           args.Timeout.String(),
		ComputeEndpoint:   args.Ce,
		DisableGCSLogs:    args.GcsLogsDisabled,
		DisableCloudLogs:  args.CloudLogsDisabled,
		DisableStdoutLogs: args.StdoutLogsDisabled,
		NoExternalIP:      args.NoExternalIP,
		WorkflowDirectory: args.WorkflowDir,
	}
}
