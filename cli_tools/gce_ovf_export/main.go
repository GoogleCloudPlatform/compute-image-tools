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

// GCE OVF export tool
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/exporter"
)

var (
	instanceName         = flag.String(ovfexporter.InstanceNameFlagKey, "", "VM Instance names to be created, separated by commas.")
	machineImageName     = flag.String(ovfexporter.MachineImageNameFlagKey, "", "Name of the machine image to create.")
	clientID             = flag.String(ovfexporter.ClientIDFlagKey, "", "Identifies the client of the exporter, e.g. `gcloud` or `pantheon`")
	destinationURI       = flag.String(ovfexporter.DestinationURIFlagKey, "", "Google Cloud Storage URI of the OVF or OVA file to export. For example: gs://my-bucket/my-vm.ovf.")
	ovfFormat            = flag.String(ovfexporter.OvfFormatFlagKey, "", "One of: `ovf` or `ova`. Defaults to `ovf`. If `ova` is specified, exported OVF package will be packed as an OVA archive and individual files will be removed from GCS.")
	diskExportFormat     = flag.String("disk-export-format", "vmdk", "format for disks in OVF, such as vmdk, vhdx, vpc, or qcow2. Any format supported by qemu-img is supported by OVF export. Defaults to `vmdk`.")
	network              = flag.String("network", "", "Name of the network in your project to use for the image export. The network must have access to Google Cloud Storage. If not specified, the network named default is used. If -subnet is also specified subnet must be a subnetwork of network specified by -network.")
	subnet               = flag.String("subnet", "", "Name of the subnetwork in your project to use for the image export. If	the network resource is in legacy mode, do not provide this property. If the network is in auto subnet mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this field should be specified. zone should be specified if this field is specified.")
	noExternalIP         = flag.Bool("no-external-ip", false, "Specifies that VPC used for OVF export doesn't allow external IPs.")
	osID                 = flag.String("os", "", "Specifies the OS of the image being exported. OS must be one of: centos-6, centos-7, debian-8, debian-9, rhel-6, rhel-6-byol, rhel-7, rhel-7-byol, ubuntu-1404, ubuntu-1604, ubuntu-1804, windows-10-byol, windows-2008r2, windows-2008r2-byol, windows-2012, windows-2012-byol, windows-2012r2, windows-2012r2-byol, windows-2016, windows-2016-byol, windows-7-byol, windows-2019, windows-2019-byol, windows-8-1-x64-byol.")
	zoneFlag             = flag.String("zone", "", "zone of the image to export. The zone in which to do the work of exporting the image. Overrides the default compute/zone property value for this command invocation")
	bootDiskKmskey       = flag.String("boot-disk-kms-key", "", "The Cloud KMS (Key Management Service) cryptokey that will be used to protect the disk. The arguments in this group can be used to specify the attributes of this resource. ID of the key or fully qualified identifier for the key. This flag must be specified if any of the other arguments in this group are specified.")
	bootDiskKmsKeyring   = flag.String("boot-disk-kms-keyring", "", "The KMS keyring of the key.")
	bootDiskKmsLocation  = flag.String("boot-disk-kms-location", "", "The Cloud location for the key.")
	bootDiskKmsProject   = flag.String("boot-disk-kms-project", "", "The Cloud project for the key.")
	timeout              = flag.String("timeout", "2h", "Maximum time a build can last before it is failed as TIMEOUT. For example, specifying 2h will fail the process after 2 hours. See `gcloud topic datetimes` for information on duration formats")
	project              = flag.String("project", "", "project to run in, overrides what is set in workflow")
	scratchBucketGcsPath = flag.String("scratch-bucket-gcs-path", "", "GCS scratch bucket to use, overrides what is set in workflow")
	oauth                = flag.String("oauth", "", "path to oauth json file, overrides what is set in workflow")
	ce                   = flag.String("compute-endpoint-override", "", "API endpoint to override default")
	gcsLogsDisabled      = flag.Bool("disable-gcs-logging", false, "do not stream logs to GCS")
	cloudLogsDisabled    = flag.Bool("disable-cloud-logging", false, "do not stream logs to Cloud Logging")
	stdoutLogsDisabled   = flag.Bool("disable-stdout-logging", false, "do not display individual workflow logs on stdout")
	releaseTrack         = flag.String(ovfexporter.ReleaseTrackFlagKey, ovfexporter.GA, fmt.Sprintf("Release track of OVF export. One of: %s, %s or %s. Impacts which compute API release track is used by the export tool.", ovfexporter.Alpha, ovfexporter.Beta, ovfexporter.GA))
	buildID              = flag.String("build-id", "", "Cloud Build ID override. This flag should be used if auto-generated or build ID provided by Cloud Build is not appropriate. For example, if running multiple exports in parallel in a single Cloud Build run, sharing build ID could cause premature temporary resource clean-up resulting in export failures.")

	currentExecutablePath string
)

func init() {
	//TODO uncomment
	currentExecutablePath = string(os.Args[0])
	currentExecutablePath = "/usr/local/google/home/zoranl/go/src/github.com/GoogleCloudPlatform/compute-image-tools/"
}

func buildExportParams() *ovfexporter.OVFExportParams {
	flag.Parse()
	return &ovfexporter.OVFExportParams{InstanceName: *instanceName,
		MachineImageName: *machineImageName, ClientID: *clientID,
		DestinationURI: *destinationURI, OvfFormat: *ovfFormat,
		DiskExportFormat: *diskExportFormat, Network: *network,
		Subnet: *subnet, NoExternalIP: *noExternalIP,
		OsID: *osID, Zone: *zoneFlag, BootDiskKmskey: *bootDiskKmskey,
		BootDiskKmsKeyring:  *bootDiskKmsKeyring,
		BootDiskKmsLocation: *bootDiskKmsLocation,
		BootDiskKmsProject:  *bootDiskKmsProject, Timeout: *timeout,
		Project: project, ScratchBucketGcsPath: *scratchBucketGcsPath,
		Oauth: *oauth, Ce: *ce, GcsLogsDisabled: *gcsLogsDisabled,
		CloudLogsDisabled:     *cloudLogsDisabled,
		StdoutLogsDisabled:    *stdoutLogsDisabled,
		CurrentExecutablePath: currentExecutablePath, ReleaseTrack: *releaseTrack,
		BuildID: *buildID,
	}
}

func runExport() (service.Loggable, error) {
	//TODO: service loggable
	return ovfexporter.Run(buildExportParams())

}

func main() {
	flag.Parse()

	//TODO: logging

	_, err := runExport()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
