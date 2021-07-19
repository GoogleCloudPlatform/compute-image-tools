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

package ovfexporter

import (
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	computeutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

type ovfExportParamValidatorImpl struct {
	validReleaseTracks []string
	zoneValidator      domain.ZoneValidatorInterface
}

// NewOvfExportParamValidator creates a new OVF export params validator
func NewOvfExportParamValidator(computeClient daisycompute.Client) ovfexportdomain.OvfExportParamValidator {
	return &ovfExportParamValidatorImpl{
		validReleaseTracks: []string{ovfexportdomain.GA, ovfexportdomain.Beta, ovfexportdomain.Alpha},
		zoneValidator:      &computeutils.ZoneValidator{ComputeClient: computeClient},
	}
}

// ValidateAndParseParams validates and parses OVFExportArgs. It returns an
// error if params are invalid. If params are valid, additional fields in
// OVFExportArgs will be populated with parsed values
func (validator *ovfExportParamValidatorImpl) ValidateAndParseParams(params *ovfexportdomain.OVFExportArgs) error {
	if params.InstanceName == "" && params.MachineImageName == "" {
		return daisy.Errf("Either the flag -%v or -%v must be provided", ovfexportdomain.InstanceNameFlagKey, ovfexportdomain.MachineImageNameFlagKey)
	}

	if params.InstanceName != "" && params.MachineImageName != "" {
		return daisy.Errf("-%v and -%v can't be provided at the same time", ovfexportdomain.InstanceNameFlagKey, ovfexportdomain.MachineImageNameFlagKey)
	}

	if err := validation.ValidateStringFlagNotEmpty(params.DestinationURI, ovfexportdomain.DestinationURIFlagKey); err != nil {
		return err
	}
	if _, err := storage.GetBucketNameFromGCSPath(params.DestinationURI); err != nil {
		return daisy.Errf("%v should be a path a Cloud Storage directory", ovfexportdomain.DestinationURIFlagKey)
	}
	if strings.HasSuffix(strings.ToLower(params.DestinationURI), ".ova") {
		return daisy.Errf("Export to OVA is currently not supported")
	}

	if params.ReleaseTrack != "" {
		isValidReleaseTrack := false
		for _, vrt := range validator.validReleaseTracks {
			if params.ReleaseTrack == vrt {
				isValidReleaseTrack = true
			}
		}

		if !isValidReleaseTrack {
			return daisy.Errf("%v should have one of the following values: %v", ovfexportdomain.ReleaseTrackFlagKey, validator.validReleaseTracks)
		}
	}

	if err := validator.zoneValidator.ZoneValid(params.Project, params.Zone); err != nil {
		return err
	}

	return nil
}
