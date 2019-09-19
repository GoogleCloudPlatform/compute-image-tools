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

package ovfimportparams

import (
	"fmt"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/flags"
	"google.golang.org/api/compute/v1"
)

// OVFImportParams holds flags for OVF import as well as derived (parsed) params
type OVFImportParams struct {
	InstanceNames               string
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
	Project                     string
	ScratchBucketGcsPath        string
	Oauth                       string
	Ce                          string
	GcsLogsDisabled             bool
	CloudLogsDisabled           bool
	StdoutLogsDisabled          bool
	NodeAffinityLabelsFlag      flags.StringArrayFlag
	ReleaseTrack                string

	UserLabels            map[string]string
	NodeAffinities        []*compute.SchedulingNodeAffinity
	CurrentExecutablePath string
}

func (oip *OVFImportParams) String() string {
	return fmt.Sprintf("%#v", oip)
}