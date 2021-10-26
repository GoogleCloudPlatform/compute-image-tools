package domain

import (
	"testing"
	"time"

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

func TestEnvironmentSettings_InstanceImport(t *testing.T) {
	checkEnvironmentSettings(t, func(params *OVFImportParams) {
		params.InstanceNames = "new-instance"
	}, func(settings *daisyutils.EnvironmentSettings) {
		settings.Tool = daisyutils.Tool{HumanReadableName: "instance import", ResourceLabelName: "instance-import"}
		settings.DaisyLogLinePrefix = "instance-import"
	})
}

func TestEnvironmentSettings_MachineImageImport(t *testing.T) {
	checkEnvironmentSettings(t, func(params *OVFImportParams) {
		params.MachineImageName = "machine-image-name"
	}, func(settings *daisyutils.EnvironmentSettings) {
		settings.Tool = daisyutils.Tool{HumanReadableName: "machine image import", ResourceLabelName: "machine-image-import"}
		settings.DaisyLogLinePrefix = "machine-image-import"
	})
}

func checkEnvironmentSettings(t *testing.T,
	paramModifier func(*OVFImportParams),
	expectedEnvModifier func(settings *daisyutils.EnvironmentSettings)) {

	timeout, err := time.ParseDuration("3h")
	assert.NoError(t, err)
	project := "project"
	params := &OVFImportParams{
		Project:               &project,
		Zone:                  "zone",
		ScratchBucketGcsPath:  "gs://bucket",
		Oauth:                 "auth-key",
		Timeout:               timeout.String(),
		Deadline:              time.Now().Add(timeout),
		Ce:                    "https://endpoint.com",
		GcsLogsDisabled:       true,
		CloudLogsDisabled:     true,
		NoExternalIP:          true,
		WorkflowDir:           "~/workflows",
		Network:               "non-default-network",
		Subnet:                "non-default-subnet",
		ComputeServiceAccount: "user@domain.com",
		UserLabels:            map[string]string{"k": "v"},
		BuildID:               "a123",
		Region:                "us-west2",
	}
	expectedEnv := daisyutils.EnvironmentSettings{
		Project:               *params.Project,
		Zone:                  params.Zone,
		GCSPath:               params.ScratchBucketGcsPath,
		OAuth:                 params.Oauth,
		ComputeEndpoint:       params.Ce,
		WorkflowDirectory:     params.WorkflowDir,
		DisableGCSLogs:        params.GcsLogsDisabled,
		DisableCloudLogs:      params.CloudLogsDisabled,
		DisableStdoutLogs:     params.StdoutLogsDisabled,
		NoExternalIP:          params.NoExternalIP,
		Network:               params.Network,
		Subnet:                params.Subnet,
		ComputeServiceAccount: params.ComputeServiceAccount,
		Labels:                params.UserLabels,
		ExecutionID:           params.BuildID,
		StorageLocation:       params.Region,
	}
	paramModifier(params)
	expectedEnvModifier(&expectedEnv)

	actualEnv := params.EnvironmentSettings()

	// The timeout is calculated dynamically using deadline and the current time.
	// As long as the unit test runs within an hour this check will pass.
	assert.Regexp(t, "[2|3]h.*", actualEnv.Timeout)
	actualEnv.Timeout = ""
	assert.Equal(t,
		expectedEnv,
		actualEnv)
}
