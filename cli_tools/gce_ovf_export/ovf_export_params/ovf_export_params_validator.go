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

package ovfexportparams

import (
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const (
	// InstanceNameFlagKey is key for instance name CLI flag
	InstanceNameFlagKey = "instance-name"

	// MachineImageNameFlagKey is key for machine image name CLI flag
	MachineImageNameFlagKey = "machine-image-name"

	// ClientIDFlagKey is key for client ID CLI flag
	ClientIDFlagKey = "client-id"

	// DestinationUriFlagKey is key for OVF/OVA GCS path CLI flag
	DestinationUriFlagKey = "destination-uri"

	// ReleaseTrackFlagKey is key for release track flag
	ReleaseTrackFlagKey = "release-track"
)

// ValidateAndParseParams validates and parses OVFExportParams. It returns an
// error if params are invalid. If params are valid, additional fields in
// OVFExportParams will be populated with parsed values
func ValidateAndParseParams(params *OVFExportParams, validReleaseTracks []string) error {
	if params.InstanceName == "" && params.MachineImageName == "" {
		return daisy.Errf("Either the flag -%v or -%v must be provided", InstanceNameFlagKey, MachineImageNameFlagKey)
	}

	if params.InstanceName != "" && params.MachineImageName != "" {
		return daisy.Errf("-%v and -%v can't be provided at the same time", InstanceNameFlagKey, MachineImageNameFlagKey)
	}

	if err := validation.ValidateStringFlagNotEmpty(params.DestinationUri, DestinationUriFlagKey); err != nil {
		return err
	}

	if err := validation.ValidateStringFlagNotEmpty(params.ClientID, ClientIDFlagKey); err != nil {
		return err
	}

	if _, err := storage.GetBucketNameFromGCSPath(params.DestinationUri); err != nil {
		return daisy.Errf("%v should be a path to OVF or OVA package in Cloud Storage", DestinationUriFlagKey)
	}

	if params.ReleaseTrack != "" {
		isValidReleaseTrack := false
		for _, vrt := range validReleaseTracks {
			if params.ReleaseTrack == vrt {
				isValidReleaseTrack = true
			}
		}

		if !isValidReleaseTrack {
			return daisy.Errf("%v should have one of the following values: %v", ReleaseTrackFlagKey, validReleaseTracks)
		}
	}

	return nil
}
