package ovfimportparams

import (
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

	UserLabels            map[string]string
	NodeAffinities        []*compute.SchedulingNodeAffinity
	CurrentExecutablePath string
}
