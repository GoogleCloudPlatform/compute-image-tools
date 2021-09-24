package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
)

func TestOVFImportParams_GetTool_InstanceImport(t *testing.T) {
	assert.Equal(t, daisyutils.Tool{
		HumanReadableName: "instance import",
		ResourceLabelName: "instance-import",
	}, (&OVFImportParams{InstanceNames: "instance"}).GetTool())
}

func TestOVFImportParams_GetTool_MachineImageImport(t *testing.T) {
	assert.Equal(t, daisyutils.Tool{
		HumanReadableName: "machine image import",
		ResourceLabelName: "machine-image-import",
	}, (&OVFImportParams{}).GetTool())
}
