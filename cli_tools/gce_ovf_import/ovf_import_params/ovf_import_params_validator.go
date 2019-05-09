package ovfimportparams

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/parse"
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
	if err := validationutils.ValidateStringFlagNotEmpty(params.InstanceNames, InstanceNameFlagKey); err != nil {
		return err
	}

	instanceNameSplits := strings.Split(params.InstanceNames, ",")
	if len(instanceNameSplits) > 1 {
		return fmt.Errorf("OVF import doesn't support multi instance import at this time")
	}

	if err := validationutils.ValidateStringFlagNotEmpty(params.OvfOvaGcsPath, OvfGcsPathFlagKey); err != nil {
		return err
	}

	if err := validationutils.ValidateStringFlagNotEmpty(params.ClientID, ClientIDFlagKey); err != nil {
		return err
	}

	if _, _, err := storageutils.SplitGCSPath(params.OvfOvaGcsPath); err != nil {
		return fmt.Errorf("%v should be a path to OVF or OVA package in GCS", OvfGcsPathFlagKey)
	}

	if params.Labels != "" {
		var err error
		params.UserLabels, err = parseutils.ParseKeyValues(params.Labels)
		if err != nil {
			return err
		}
	}

	if params.NodeAffinityLabelsFlag != nil {
		var err error
		params.NodeAffinities, err = computeutils.ParseNodeAffinityLabels(params.NodeAffinityLabelsFlag)
		if err != nil {
			return err
		}
	}

	return nil
}
