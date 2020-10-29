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
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
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
	BootDiskKmskey       string
	BootDiskKmsKeyring   string
	BootDiskKmsLocation  string
	BootDiskKmsProject   string
	Timeout              string
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

// ValidateAndPopulateParams validaegs and populate OVF export params
func (params *OVFExportParams) ValidateAndPopulateParams(paramValidator OvfExportParamValidator, paramPopulator param.Populator) error {
	if err := paramValidator.ValidateAndParseParams(params); err != nil {
		return err
	}
	if err := paramPopulator.PopulateMissingParameters(params.Project, params.ClientID, &params.Zone,
		&params.Region, &params.ScratchBucketGcsPath, params.DestinationURI, nil); err != nil {
		return err
	}
	params.populateNetwork()
	params.populateBuildID()
	params.populateNamespacedScratchDirectory()
	params.populateDestinationURI()
	return nil
}

func (params *OVFExportParams) populateNetwork() {
	if params.Network == "" && params.Subnet == "" {
		params.Network = "default"
	}
	if params.Subnet != "" {
		params.Subnet = param.GetRegionalResourcePath(params.Region, "subnetworks", params.Subnet)
	}
	if params.Network != "" {
		params.Network = param.GetGlobalResourcePath("networks", params.Network)
	}
}

func (params *OVFExportParams) populateBuildID() {
	if params != nil && params.BuildID != "" {
		return
	}
	params.BuildID = os.Getenv("BUILD_ID")
	if params.BuildID == "" {
		params.BuildID = pathutils.RandString(5)
	}
}

func (params *OVFExportParams) populateDestinationURI() {
	params.DestinationURI = strings.ToLower(params.DestinationURI)
	if !strings.HasSuffix(params.DestinationURI, ".ova") {
		params.DestinationURI = pathutils.ToDirectoryURL(params.DestinationURI)
	}
}

// populateNamespacedScratchDirectory updates ScratchBucketGcsPath to include a directory
// that is specific to this export, formulated using the start timestamp and the execution ID.
// This ensures all logs and artifacts are contained in a single directory.
func (params *OVFExportParams) populateNamespacedScratchDirectory() {
	if !strings.HasSuffix(params.ScratchBucketGcsPath, "/") {
		params.ScratchBucketGcsPath += "/"
	}

	params.ScratchBucketGcsPath += fmt.Sprintf(
		"gce-ovf-export-%s-%s", params.Started.Format(time.RFC3339), params.BuildID)
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
	//TODO: remove in prod
	currentExecutablePath = "/usr/local/google/home/zoranl/go/src/github.com/GoogleCloudPlatform/compute-image-tools/"
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
		Timeout:           params.Timeout,
		ComputeEndpoint:   params.Ce,
		DisableGCSLogs:    params.GcsLogsDisabled,
		DisableCloudLogs:  params.CloudLogsDisabled,
		DisableStdoutLogs: params.StdoutLogsDisabled,
		NoExternalIP:      params.NoExternalIP,
		WorkflowDirectory: params.WorkflowDir,
	}
}
