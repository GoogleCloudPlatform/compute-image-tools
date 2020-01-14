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
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const (
	// InstanceNameFlagKey is key for instance name CLI flag
	InstanceNameFlagKey = "instance-names"

	// MachineImageNameFlagKey is key for machine image name CLI flag
	MachineImageNameFlagKey = "machine-image-name"

	// MachineImageStorageLocationFlagKey is key for machine image storage location CLI flag
	MachineImageStorageLocationFlagKey = "machine-image-storage-location"

	// ClientIDFlagKey is key for client ID CLI flag
	ClientIDFlagKey = "client-id"

	// OvfGcsPathFlagKey is key for OVF/OVA GCS path CLI flag
	OvfGcsPathFlagKey = "ovf-gcs-path"
)

// ValidateAndParseParams validates and parses OVFImportParams. It returns an error if params are
// invalid. If params are valid, additional fields in OVFImportParams will be populated with
// parsed values
func ValidateAndParseParams(params *OVFImportParams) error {
	if params.InstanceNames == "" && params.MachineImageName == "" {
		return daisy.Errf("Either the flag -%v or -%v must be provided", InstanceNameFlagKey, MachineImageNameFlagKey)
	}

	if params.InstanceNames != "" && params.MachineImageName != "" {
		return daisy.Errf("-%v and -%v can't be provided at the same time", InstanceNameFlagKey, MachineImageNameFlagKey)
	}

	if params.IsInstanceImport() {
		// instance import specific validation
		instanceNameSplits := strings.Split(params.InstanceNames, ",")
		if len(instanceNameSplits) > 1 {
			return daisy.Errf("OVF import doesn't support multi instance import at this time")
		}

		if params.MachineImageStorageLocation != "" {
			return daisy.Errf("-%v can't be provided when importing an instance", MachineImageStorageLocationFlagKey)
		}
	}

	if err := validation.ValidateStringFlagNotEmpty(params.OvfOvaGcsPath, OvfGcsPathFlagKey); err != nil {
		return err
	}

	if err := validation.ValidateStringFlagNotEmpty(params.ClientID, ClientIDFlagKey); err != nil {
		return err
	}

	if _, err := storage.GetBucketNameFromGCSPath(params.OvfOvaGcsPath); err != nil {
		return daisy.Errf("%v should be a path to OVF or OVA package in Cloud Storage", OvfGcsPathFlagKey)
	}

	if params.Labels != "" {
		var err error
		params.UserLabels, err = param.ParseKeyValues(params.Labels)
		if err != nil {
			return err
		}
	}

	if params.NodeAffinityLabelsFlag != nil {
		var err error
		params.NodeAffinities, err = compute.ParseNodeAffinityLabels(params.NodeAffinityLabelsFlag)
		if err != nil {
			return err
		}
	}

	return nil
}
