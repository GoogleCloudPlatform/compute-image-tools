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
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/flags"
	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
)

// RegisterFlags registers OVF exporter CLI flags with args
func RegisterFlags(ovfExportArgs *ovfexportdomain.OVFExportArgs, args []string) error {
	flagSet := flag.NewFlagSet("ovf-export", flag.ContinueOnError)
	// Don't write parse errors to stdout, instead propagate them via an
	// exception since we use flag.ContinueOnError.
	flagSet.SetOutput(ioutil.Discard)

	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.InstanceName), ovfexportdomain.InstanceNameFlagKey,
		"Name of the VM Instance to be exported.")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.MachineImageName), ovfexportdomain.MachineImageNameFlagKey,
		"Name of the Google machine image to be exported.")
	flagSet.Var((*flags.LowerTrimmedString)(&ovfExportArgs.ClientID), ovfexportdomain.ClientIDFlagKey,
		"Identifies the client of the exporter, e.g. `gcloud` or `pantheon`")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.ClientVersion), "client-version",
		"Identifies the version of the client of the exporter.")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.DestinationURI), ovfexportdomain.DestinationURIFlagKey,
		"Google Cloud Storage URI of the OVF or OVA file to export to. For example: gs://my-bucket/my-vm.ovf.")
	flagSet.Var((*flags.LowerTrimmedString)(&ovfExportArgs.OvfFormat), ovfexportdomain.OvfFormatFlagKey,
		"One of: `ovf` or `ova`. Defaults to `ovf`. If `ova` is specified, exported OVF package will be packed as an OVA archive and individual files will be removed from GCS.")
	flagSet.Var((*flags.LowerTrimmedString)(&ovfExportArgs.DiskExportFormat), "disk-export-format",
		"format for disks in OVF, such as vmdk, vhdx, vpc, or qcow2. Any format supported by qemu-img is supported by OVF export. Defaults to `vmdk`.")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.Network), "network",
		"Name of the network in your project to use for the image export. The network must have access to Google Cloud Storage. If not specified, the network named default is used. If -subnet is also specified subnet must be a subnetwork of network specified by -network.")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.Subnet), "subnet",
		"Name of the subnetwork in your project to use for the image export. If	the network resource is in legacy mode, do not provide this property. If the network is in auto subnet mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this field should be specified. zone should be specified if this field is specified.")
	flagSet.BoolVar(&ovfExportArgs.NoExternalIP, "no-external-ip", false,
		"Specifies that VPC used for OVF export doesn't allow external IPs.")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.OsID), "os",
		"Specifies the OS of the image being exported. OS must be one of: "+strings.Join(daisy.GetSortedOSIDs(), ", ")+".")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.Zone), "zone",
		"zone of the image to export. The zone in which to do the work of exporting the image. Overrides the default compute/zone property value for this command invocation")
	flagSet.DurationVar(&ovfExportArgs.Timeout, "timeout", time.Hour*2,
		"Maximum time a build can last before it is failed as TIMEOUT. For example, specifying 2h will fail the process after 2 hours. See `gcloud topic datetimes` for information on duration formats")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.Project), "project",
		"project to run in, overrides what is set in workflow")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.ScratchBucketGcsPath), "scratch-bucket-gcs-path",
		"GCS scratch bucket to use, overrides what is set in workflow")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.Oauth), "oauth",
		"path to oauth json file, overrides what is set in workflow")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.Ce), "compute-endpoint-override", "API endpoint to override default")
	flagSet.BoolVar(&ovfExportArgs.GcsLogsDisabled, "disable-gcs-logging", false, "do not stream logs to GCS")
	flagSet.BoolVar(&ovfExportArgs.CloudLogsDisabled, "disable-cloud-logging", false, "do not stream logs to Cloud Logging")
	flagSet.BoolVar(&ovfExportArgs.StdoutLogsDisabled, "disable-stdout-logging", false, "do not display individual workflow logs on stdout")
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.ReleaseTrack), ovfexportdomain.ReleaseTrackFlagKey,
		fmt.Sprintf("Release track of OVF export. One of: %s, %s or %s. Impacts which compute API release track is used by the export tool.", ovfexportdomain.Alpha, ovfexportdomain.Beta, ovfexportdomain.GA))
	flagSet.Var((*flags.TrimmedString)(&ovfExportArgs.BuildID), "build-id",
		"Cloud Build ID override. This flag should be used if auto-generated or build ID provided by Cloud Build is not appropriate. For example, if running multiple exports in parallel in a single Cloud Build run, sharing build ID could cause premature temporary resource clean-up resulting in export failures.")
	return flagSet.Parse(args)
}
