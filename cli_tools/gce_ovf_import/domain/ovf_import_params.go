//  Copyright 2019 Google Inc. All Rights Reserved.
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

package domain

import (
	"fmt"
	"time"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/flags"
)

const (
	//Alpha represents alpha release track
	Alpha = "alpha"

	//Beta represents beta release track
	Beta = "beta"

	//GA represents GA release track
	GA = "ga"
)

// OVFImportParams holds flags for OVF import as well as derived (parsed) params
type OVFImportParams struct {

	// Instance import specific flags
	InstanceNames string

	// Machine image specific flags
	MachineImageName            string
	MachineImageStorageLocation string

	// Common flags
	ClientID                    string
	OvfOvaGcsPath               string
	NoGuestEnvironment          bool
	CanIPForward                bool
	DeletionProtection          bool
	Description                 string
	Labels                      string
	MachineType                 string
	Network                     string
	NetworkTier                 string
	Subnet                      string
	PrivateNetworkIP            string
	NoExternalIP                bool
	NoRestartOnFailure          bool
	OsID                        string
	BYOL                        bool
	ShieldedIntegrityMonitoring bool
	ShieldedSecureBoot          bool
	ShieldedVtpm                bool
	Tags                        string
	Zone                        string
	BootDiskKmskey              string
	BootDiskKmsKeyring          string
	BootDiskKmsLocation         string
	BootDiskKmsProject          string
	Timeout                     string
	Project                     *string
	ScratchBucketGcsPath        string
	Oauth                       string
	Ce                          string
	ComputeServiceAccount       string
	InstanceServiceAccount      string
	InstanceAccessScopesFlag    string
	GcsLogsDisabled             bool
	CloudLogsDisabled           bool
	StdoutLogsDisabled          bool
	NodeAffinityLabelsFlag      flags.StringArrayFlag
	ReleaseTrack                string
	UefiCompatible              bool
	Hostname                    string
	BuildID                     string

	// Non-flags

	// Deadline of when timeout will occur.
	Deadline time.Time

	UserLabels            map[string]string
	UserTags              []string
	NodeAffinities        []*compute.SchedulingNodeAffinity
	NodeAffinitiesBeta    []*computeBeta.SchedulingNodeAffinity
	InstanceAccessScopes  []string
	CurrentExecutablePath string
	Region                string

	// Path to daisy_workflows directory.
	WorkflowDir string
}

func (oip *OVFImportParams) String() string {
	return fmt.Sprintf("%#v", oip)
}

// IsInstanceImport returns true if import represented by these params is
// instance import. False otherwise.
func (oip *OVFImportParams) IsInstanceImport() bool {
	return oip.InstanceNames != ""
}

// IsMachineImageImport returns true if import represented by these params is
// a machine image import. False otherwise.
func (oip *OVFImportParams) IsMachineImageImport() bool {
	return !oip.IsInstanceImport()
}

// GetTool returns a description of the tool being run that can be used for logging and messaging.
func (oip *OVFImportParams) GetTool() daisyutils.Tool {
	if oip.IsInstanceImport() {
		return daisyutils.Tool{
			HumanReadableName: "instance import",
			URISafeName:       "instance-import",
		}
	}
	return daisyutils.Tool{
		HumanReadableName: "machine image import",
		URISafeName:       "machine-image-import",
	}
}
