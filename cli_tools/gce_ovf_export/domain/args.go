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

package ovfexportdomain

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/flags"
)

const (
	// InstanceNameFlagKey is key for instance name CLI flag
	InstanceNameFlagKey = "instance-name"

	// MachineImageNameFlagKey is key for machine image name CLI flag
	MachineImageNameFlagKey = "machine-image-name"

	// ClientIDFlagKey is key for client ID CLI flag
	ClientIDFlagKey = "client-id"

	// DestinationURIFlagKey is key for OVF/OVA GCS path CLI flag
	DestinationURIFlagKey = "destination-uri"

	// ReleaseTrackFlagKey is key for release track flag
	ReleaseTrackFlagKey = "release-track"

	// OvfFormatFlagKey is key for OVF format flag
	OvfFormatFlagKey = "ovf-format"

	mainWorkflowDir = "daisy_workflows/"

	//Alpha represents alpha release track
	Alpha = "alpha"

	//Beta represents beta release track
	Beta = "beta"

	//GA represents GA release track
	GA = "ga"
)

// OVFExportArgs holds args for OVF export
type OVFExportArgs struct {

	// Instance export specific args
	InstanceName string

	// Machine image specific args
	MachineImageName string

	// Common args
	ClientID              string
	ClientVersion         string
	DestinationURI        string
	DiskExportFormat      string
	OsID                  string
	Network               string
	Subnet                string
	NoExternalIP          bool
	Zone                  string
	Timeout               time.Duration
	Project               string
	ScratchBucketGcsPath  string
	Oauth                 string
	Ce                    string
	GcsLogsDisabled       bool
	CloudLogsDisabled     bool
	StdoutLogsDisabled    bool
	ReleaseTrack          string
	BuildID               string
	ComputeServiceAccount string

	// Non-args
	WorkflowDir string
	Region      string
	Started     time.Time
	//DestinationDirectory holds a full path to GCS directory where OVF will be saved after export. E.g. `gs://my-bucket/my-folder/`
	DestinationDirectory string
	// OvfName is a name of OVF being exported. E.g. if DestinationURI is
	//`gs://my-bucket/my-folder/vm.ovf`, OvfName will be `vm`. If a directory is
	// provided for DestinationURI, instance name will be used for OVF name
	OvfName string
}

// NewOVFExportArgs parses args to create an NewOVFExportArgs instance.
// No validation is done here.
func NewOVFExportArgs(args []string) (*OVFExportArgs, error) {
	currentExecutablePath := filepath.Dir(os.Args[0])
	ovfExportArgs := &OVFExportArgs{
		WorkflowDir:      toWorkingDir(currentExecutablePath, mainWorkflowDir),
		Started:          time.Now(),
		DiskExportFormat: "vmdk",
		ReleaseTrack:     GA,
	}
	err := ovfExportArgs.registerFlags(args)
	return ovfExportArgs, err
}

func (args *OVFExportArgs) String() string {
	return fmt.Sprintf("%#v", args)
}

// IsInstanceExport returns true if export represented by these args is
// instance export. False otherwise.
func (args *OVFExportArgs) IsInstanceExport() bool {
	return args.InstanceName != ""
}

// IsMachineImageExport returns true if export represented by these args is
// a machine image export. False otherwise.
func (args *OVFExportArgs) IsMachineImageExport() bool {
	return !args.IsInstanceExport()
}

// GetResourceName returns name of the resource to be exported, either instance name or machine image name
func (args *OVFExportArgs) GetResourceName() string {
	if args.IsInstanceExport() {
		return args.InstanceName
	}
	return args.MachineImageName
}

//TODO: consolidate with ovf_importer.toWorkingDir
func toWorkingDir(currentDir, workflowDir string) string {
	wd, err := filepath.Abs(filepath.Dir(currentDir))
	if err == nil {
		return path.Join(wd, workflowDir)
	}
	return workflowDir
}

// EnvironmentSettings creates an EnvironmentSettings instance from the fields
// in this struct.
func (args *OVFExportArgs) EnvironmentSettings() daisyutils.EnvironmentSettings {
	return daisyutils.EnvironmentSettings{
		Project:               args.Project,
		Zone:                  args.Zone,
		GCSPath:               args.ScratchBucketGcsPath,
		OAuth:                 args.Oauth,
		Timeout:               args.Timeout.String(),
		ComputeEndpoint:       args.Ce,
		DisableGCSLogs:        args.GcsLogsDisabled,
		DisableCloudLogs:      args.CloudLogsDisabled,
		DisableStdoutLogs:     args.StdoutLogsDisabled,
		NoExternalIP:          args.NoExternalIP,
		WorkflowDirectory:     args.WorkflowDir,
		Network:               args.Network,
		Subnet:                args.Subnet,
		ComputeServiceAccount: args.ComputeServiceAccount,
		Labels:                map[string]string{},
		ExecutionID:           args.BuildID,
		StorageLocation:       "",
	}
}

// registerFlags registers OVF exporter CLI flags with args
func (args *OVFExportArgs) registerFlags(cliArgs []string) error {
	flagSet := flag.NewFlagSet("ovf-export", flag.ContinueOnError)
	// Don't write parse errors to stdout, instead propagate them via an
	// exception since we use flag.ContinueOnError.
	flagSet.SetOutput(ioutil.Discard)

	flagSet.Var((*flags.TrimmedString)(&args.InstanceName), InstanceNameFlagKey,
		"Name of the VM Instance to be exported.")
	flagSet.Var((*flags.TrimmedString)(&args.MachineImageName), MachineImageNameFlagKey,
		"Name of the Google machine image to be exported.")
	flagSet.Var((*flags.LowerTrimmedString)(&args.ClientID), ClientIDFlagKey,
		"Identifies the client of the exporter, e.g. `gcloud` or `pantheon`")
	flagSet.Var((*flags.TrimmedString)(&args.ClientVersion), "client-version",
		"Identifies the version of the client of the exporter.")
	flagSet.Var((*flags.TrimmedString)(&args.DestinationURI), DestinationURIFlagKey,
		"Google Cloud Storage URI of the OVF descriptor or directory to export to. For example: `gs://my-bucket/my-vm.ovf` or `gs://my-bucket/my-ovf/`.")
	flagSet.Var((*flags.LowerTrimmedString)(&args.DiskExportFormat), "disk-export-format",
		"format for disks in OVF, such as vmdk, vhdx, vpc, or qcow2. Any format supported by qemu-img is supported by OVF export. Defaults to `vmdk`.")
	flagSet.Var((*flags.TrimmedString)(&args.Network), "network",
		"Name of the network in your project to use for the image export. The network must have access to Google Cloud Storage. If not specified, the network named default is used. If -subnet is also specified subnet must be a subnetwork of network specified by -network.")
	flagSet.Var((*flags.TrimmedString)(&args.Subnet), "subnet",
		"Name of the subnetwork in your project to use for the image export. If	the network resource is in legacy mode, do not provide this property. If the network is in auto subnet mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this field should be specified. zone should be specified if this field is specified.")
	flagSet.BoolVar(&args.NoExternalIP, "no-external-ip", false,
		"Specifies that VPC used for OVF export doesn't allow external IPs.")
	flagSet.Var((*flags.TrimmedString)(&args.OsID), "os",
		"Specifies the OS of the image being exported. OS must be one of: "+strings.Join(daisyutils.GetSortedOSIDs(), ", ")+".")
	flagSet.Var((*flags.TrimmedString)(&args.Zone), "zone",
		"zone of the image to export. The zone in which to do the work of exporting the image. Overrides the default compute/zone property value for this command invocation")
	flagSet.DurationVar(&args.Timeout, "timeout", time.Hour*2,
		"Maximum time a build can last before it is failed as TIMEOUT. For example, specifying 2h will fail the process after 2 hours. See `gcloud topic datetimes` for information on duration formats")
	flagSet.Var((*flags.TrimmedString)(&args.Project), "project",
		"project to run in, overrides what is set in workflow")
	flagSet.Var((*flags.TrimmedString)(&args.ScratchBucketGcsPath), "scratch-bucket-gcs-path",
		"GCS scratch bucket to use, overrides what is set in workflow")
	flagSet.Var((*flags.TrimmedString)(&args.Oauth), "oauth",
		"path to oauth json file, overrides what is set in workflow")
	flagSet.Var((*flags.TrimmedString)(&args.Ce), "compute-endpoint-override", "API endpoint to override default")
	flagSet.BoolVar(&args.GcsLogsDisabled, "disable-gcs-logging", false, "do not stream logs to GCS")
	flagSet.BoolVar(&args.CloudLogsDisabled, "disable-cloud-logging", false, "do not stream logs to Cloud Logging")
	flagSet.BoolVar(&args.StdoutLogsDisabled, "disable-stdout-logging", false, "do not display individual workflow logs on stdout")
	flagSet.Var((*flags.TrimmedString)(&args.ReleaseTrack), ReleaseTrackFlagKey,
		fmt.Sprintf("Release track of OVF export. One of: %s, %s or %s. Impacts which compute API release track is used by the export tool.", Alpha, Beta, GA))
	flagSet.Var((*flags.TrimmedString)(&args.BuildID), "build-id",
		"Cloud Build ID override. This flag should be used if auto-generated or build ID provided by Cloud Build is not appropriate. For example, if running multiple exports in parallel in a single Cloud Build run, sharing build ID could cause premature temporary resource clean-up resulting in export failures.")
	flagSet.Var((*flags.TrimmedString)(&args.ComputeServiceAccount), "compute-service-account", "Compute service account to be used by exporter Virtual Machine. When empty, the Compute Engine default service account is used.")
	return flagSet.Parse(cliArgs)
}
