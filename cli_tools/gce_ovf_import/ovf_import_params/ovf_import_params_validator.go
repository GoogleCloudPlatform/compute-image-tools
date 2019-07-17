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
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
)

const (
	// InstanceNameFlagKey is key for instance name CLI flag
	InstanceNameFlagKey = "instance-names"

	// ClientIDFlagKey is key for client ID CLI flag
	ClientIDFlagKey = "client-id"

	// OvfGcsPathFlagKey is key for OVF/OVA GCS path CLI flag
	OvfGcsPathFlagKey = "ovf-gcs-path"
)

// ValidateAndParseParams validates and parses OVFImportParams. It returns an error if params are
// invalid. If params are valid, additional fields in OVFImportParams will be populated with
// parsed values
func ValidateAndParseParams(params *OVFImportParams) error {
	if err := validation.ValidateStringFlagNotEmpty(params.InstanceNames, InstanceNameFlagKey); err != nil {
		return err
	}

	instanceNameSplits := strings.Split(params.InstanceNames, ",")
	if len(instanceNameSplits) > 1 {
		return fmt.Errorf("OVF import doesn't support multi instance import at this time")
	}

	if err := validation.ValidateStringFlagNotEmpty(params.OvfOvaGcsPath, OvfGcsPathFlagKey); err != nil {
		return err
	}

	if err := validation.ValidateStringFlagNotEmpty(params.ClientID, ClientIDFlagKey); err != nil {
		return err
	}

	if _, _, err := storage.SplitGCSPath(params.OvfOvaGcsPath); err != nil {
		return fmt.Errorf("%v should be a path to OVF or OVA package in GCS", OvfGcsPathFlagKey)
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
