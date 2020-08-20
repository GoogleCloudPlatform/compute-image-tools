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

// GCE OVF import tool
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/flags"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_import_params"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_importer"
)

var (
	instanceNames               = flag.String(ovfimportparams.InstanceNameFlagKey, "", "VM Instance names to be created, separated by commas.")
	machineImageName            = flag.String(ovfimportparams.MachineImageNameFlagKey, "", "Name of the machine image to create.")
	clientID                    = flag.String(ovfimportparams.ClientIDFlagKey, "", "Identifies the client of the importer, e.g. `gcloud` or `pantheon`")
	clientVersion               = flag.String("client-version", "", "Identifies the version of the client of the importer")
	ovfOvaGcsPath               = flag.String(ovfimportparams.OvfGcsPathFlagKey, "", " Google Cloud Storage URI of the OVF or OVA file to import. For example: gs://my-bucket/my-vm.ovf.")
	noGuestEnvironment          = flag.Bool("no-guest-environment", false, "Google Guest Environment will not be installed on the image.")
	canIPForward                = flag.Bool("can-ip-forward", false, "If provided, allows the instances to send and receive packets with non-matching destination or source IP addresses.")
	deletionProtection          = flag.Bool("deletion-protection", false, "Enables deletion protection for the instance.")
	description                 = flag.String("description", "", "Specifies a textual description of the instances.")
	labels                      = flag.String("labels", "", "List of label KEY=VALUE pairs to add. Keys must start with a lowercase character and contain only hyphens (-), underscores (_), lowercase characters, and numbers. Values must contain only hyphens (-), underscores (_), lowercase characters, and numbers.")
	machineType                 = flag.String("machine-type", "", "Specifies the machine type used for the instances. To get a list of available machine types, run 'gcloud compute machine-types list'. If unspecified, the default type is n1-standard-1.")
	network                     = flag.String("network", "", "Name of the network in your project to use for the image import. The network must have access to Google Cloud Storage. If not specified, the network named default is used. If -subnet is also specified subnet must be a subnetwork of network specified by -network.")
	networkTier                 = flag.String("network-tier", "", "Specifies the network tier that will be used to configure the instance. NETWORK_TIER must be one of: PREMIUM, STANDARD. The default value is PREMIUM.")
	subnet                      = flag.String("subnet", "", "Name of the subnetwork in your project to use for the image import. If	the network resource is in legacy mode, do not provide this property. If the network is in auto subnet mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this field should be specified. Zone should be specified if this field is specified.")
	privateNetworkIP            = flag.String("private-network-ip", "", "Specifies the RFC1918 IP to assign to the instance. The IP should be in the subnet or legacy network IP range.")
	noExternalIP                = flag.Bool("no-external-ip", false, "Specifies that VPC into which instances is being imported doesn't allow external IPs.")
	noRestartOnFailure          = flag.Bool("no-restart-on-failure", false, "the instance will not be restarted if it’s terminated by Compute Engine. This does not affect terminations performed by the user.")
	osID                        = flag.String("os", "", "Specifies the OS of the image being imported. OS must be one of: centos-6, centos-7, debian-8, debian-9, rhel-6, rhel-6-byol, rhel-7, rhel-7-byol, ubuntu-1404, ubuntu-1604, ubuntu-1804, ubuntu-2004, windows-10-byol, windows-2008r2, windows-2008r2-byol, windows-2012, windows-2012-byol, windows-2012r2, windows-2012r2-byol, windows-2016, windows-2016-byol, windows-7-byol, windows-2019, windows-2019-byol, windows-8-1-x64-byol.")
	shieldedIntegrityMonitoring = flag.Bool("shielded-integrity-monitoring", false, "Enables monitoring and attestation of the boot integrity of the instance. The attestation is performed against the integrity policy baseline. This baseline is initially derived from the implicitly trusted boot image when the instance is created. This baseline can be updated by using --shielded-vm-learn-integrity-policy.")
	shieldedSecureBoot          = flag.Bool("shielded-secure-boot", false, "The instance will boot with secure boot enabled.")
	shieldedVtpm                = flag.Bool("shielded-vtpm", false, "The instance will boot with the TPM (Trusted Platform Module) enabled. A TPM is a hardware module that can be used for different security operations such as remote attestation, encryption and sealing of keys.")
	tags                        = flag.String("tags", "", "Specifies a list of tags to apply to the instance. These tags allow network firewall rules and routes to be applied to specified VM instances. See `gcloud compute firewall-rules create` for more details.")
	zoneFlag                    = flag.String("zone", "", "Zone of the image to import. The zone in which to do the work of importing the image. Overrides the default compute/zone property value for this command invocation")
	bootDiskKmskey              = flag.String("boot-disk-kms-key", "", "The Cloud KMS (Key Management Service) cryptokey that will be used to protect the disk. The arguments in this group can be used to specify the attributes of this resource. ID of the key or fully qualified identifier for the key. This flag must be specified if any of the other arguments in this group are specified.")
	bootDiskKmsKeyring          = flag.String("boot-disk-kms-keyring", "", "The KMS keyring of the key.")
	bootDiskKmsLocation         = flag.String("boot-disk-kms-location", "", "The Cloud location for the key.")
	bootDiskKmsProject          = flag.String("boot-disk-kms-project", "", "The Cloud project for the key.")
	timeout                     = flag.String("timeout", "", "Maximum time a build can last before it is failed as TIMEOUT. For example, specifying 2h will fail the process after 2 hours. See `gcloud topic datetimes` for information on duration formats")
	project                     = flag.String("project", "", "project to run in, overrides what is set in workflow")
	scratchBucketGcsPath        = flag.String("scratch-bucket-gcs-path", "", "GCS scratch bucket to use, overrides what is set in workflow")
	oauth                       = flag.String("oauth", "", "path to oauth json file, overrides what is set in workflow")
	ce                          = flag.String("compute-endpoint-override", "", "API endpoint to override default")
	gcsLogsDisabled             = flag.Bool("disable-gcs-logging", false, "do not stream logs to GCS")
	cloudLogsDisabled           = flag.Bool("disable-cloud-logging", false, "do not stream logs to Cloud Logging")
	stdoutLogsDisabled          = flag.Bool("disable-stdout-logging", false, "do not display individual workflow logs on stdout")
	releaseTrack                = flag.String("release-track", ovfimporter.GA, fmt.Sprintf("Release track of OVF import. One of: %s, %s or %s. Impacts which compute API release track is used by the import tool.", ovfimporter.Alpha, ovfimporter.Beta, ovfimporter.GA))
	uefiCompatible              = flag.Bool("uefi-compatible", false, "Enables UEFI booting, which is an alternative system boot method. Most public images use the GRUB bootloader as their primary boot method.")
	hostname                    = flag.String(ovfimportparams.HostnameFlagKey, "", "Specify the hostname of the instance to be created. The specified hostname must be RFC1035 compliant.")
	machineImageStorageLocation = flag.String(ovfimportparams.MachineImageStorageLocationFlagKey, "", "GCS bucket storage location of the machine image being imported (regional or multi-regional)")
	buildID                     = flag.String("build-id", "", "Cloud Build ID override. This flag should be used if auto-generated or build ID provided by Cloud Build is not appropriate. For example, if running multiple imports in parallel in a single Cloud Build run, sharing build ID could cause premature temporary resource clean-up resulting in import failures.")

	nodeAffinityLabelsFlag flags.StringArrayFlag
	currentExecutablePath  string
)

func init() {
	currentExecutablePath = string(os.Args[0])
	flag.Var(&nodeAffinityLabelsFlag, "node-affinity-label", "Node affinity label used to determine sole tenant node to schedule this instance on. Label is of the format: <key>,<operator>,<value>,<value2>... where <operator> can be one of: IN, NOT. For example: workload,IN,prod,test is a label with key 'workload' and values 'prod' and 'test'. This flag can be specified multiple times for multiple labels.")
}

func buildOVFImportParams() *ovfimportparams.OVFImportParams {
	flag.Parse()
	return &ovfimportparams.OVFImportParams{InstanceNames: *instanceNames,
		MachineImageName: *machineImageName, ClientID: *clientID,
		OvfOvaGcsPath: *ovfOvaGcsPath, NoGuestEnvironment: *noGuestEnvironment,
		CanIPForward: *canIPForward, DeletionProtection: *deletionProtection, Description: *description,
		Labels: *labels, MachineType: *machineType, Network: *network, NetworkTier: *networkTier,
		Subnet: *subnet, PrivateNetworkIP: *privateNetworkIP, NoExternalIP: *noExternalIP,
		NoRestartOnFailure: *noRestartOnFailure, OsID: *osID,
		ShieldedIntegrityMonitoring: *shieldedIntegrityMonitoring, ShieldedSecureBoot: *shieldedSecureBoot,
		ShieldedVtpm: *shieldedVtpm, Tags: *tags, Zone: *zoneFlag, BootDiskKmskey: *bootDiskKmskey,
		BootDiskKmsKeyring: *bootDiskKmsKeyring, BootDiskKmsLocation: *bootDiskKmsLocation,
		BootDiskKmsProject: *bootDiskKmsProject, Timeout: *timeout, Project: project,
		ScratchBucketGcsPath: *scratchBucketGcsPath, Oauth: *oauth, Ce: *ce,
		GcsLogsDisabled: *gcsLogsDisabled, CloudLogsDisabled: *cloudLogsDisabled,
		StdoutLogsDisabled: *stdoutLogsDisabled, NodeAffinityLabelsFlag: nodeAffinityLabelsFlag,
		CurrentExecutablePath: currentExecutablePath, ReleaseTrack: *releaseTrack,
		UefiCompatible: *uefiCompatible, Hostname: *hostname,
		MachineImageStorageLocation: *machineImageStorageLocation, BuildID: *buildID,
	}
}

func runImport() (service.Loggable, error) {
	var ovfImporter *ovfimporter.OVFImporter
	var err error
	defer func() {
		if ovfImporter != nil {
			ovfImporter.CleanUp()
		}
	}()

	if ovfImporter, err = ovfimporter.NewOVFImporter(buildOVFImportParams()); err != nil {
		return nil, err
	}

	wf, err := ovfImporter.Import()
	return service.NewLoggableFromWorkflow(wf), err
}

func main() {
	flag.Parse()

	var paramLog service.InputParams
	var action string

	isInstanceImport := *instanceNames != ""
	if isInstanceImport {
		paramLog = createInstanceImportInputParams()
		action = service.InstanceImportAction
	} else {
		paramLog = createMachineImageImportInputParams()
		action = service.MachineImageImportAction
	}

	if err := service.RunWithServerLogging(action, paramLog, project, runImport); err != nil {
		os.Exit(1)
	}
}

func createInstanceImportInputParams() service.InputParams {
	return service.InputParams{
		InstanceImportParams: &service.InstanceImportParams{
			CommonParams: createCommonInputParams(),

			InstanceName:                *instanceNames,
			OvfGcsPath:                  *ovfOvaGcsPath,
			CanIPForward:                *canIPForward,
			DeletionProtection:          *deletionProtection,
			MachineType:                 *machineType,
			NetworkInterface:            *network,
			NetworkTier:                 *networkTier,
			PrivateNetworkIP:            *privateNetworkIP,
			NoExternalIP:                *noExternalIP,
			NoRestartOnFailure:          *noRestartOnFailure,
			OS:                          *osID,
			ShieldedIntegrityMonitoring: *shieldedIntegrityMonitoring,
			ShieldedSecureBoot:          *shieldedSecureBoot,
			ShieldedVtpm:                *shieldedVtpm,
			Tags:                        *tags,
			HasBootDiskKmsKey:           *bootDiskKmskey != "",
			HasBootDiskKmsKeyring:       *bootDiskKmsKeyring != "",
			HasBootDiskKmsLocation:      *bootDiskKmsLocation != "",
			HasBootDiskKmsProject:       *bootDiskKmsProject != "",
			NoGuestEnvironment:          *noGuestEnvironment,
			NodeAffinityLabel:           nodeAffinityLabelsFlag.String(),
		},
	}
}

func createMachineImageImportInputParams() service.InputParams {
	return service.InputParams{
		MachineImageImportParams: &service.MachineImageImportParams{
			CommonParams: createCommonInputParams(),

			MachineImageName:            *machineImageName,
			OvfGcsPath:                  *ovfOvaGcsPath,
			CanIPForward:                *canIPForward,
			DeletionProtection:          *deletionProtection,
			MachineType:                 *machineType,
			NetworkInterface:            *network,
			NetworkTier:                 *networkTier,
			PrivateNetworkIP:            *privateNetworkIP,
			NoExternalIP:                *noExternalIP,
			NoRestartOnFailure:          *noRestartOnFailure,
			OS:                          *osID,
			ShieldedIntegrityMonitoring: *shieldedIntegrityMonitoring,
			ShieldedSecureBoot:          *shieldedSecureBoot,
			ShieldedVtpm:                *shieldedVtpm,
			Tags:                        *tags,
			HasBootDiskKmsKey:           *bootDiskKmskey != "",
			HasBootDiskKmsKeyring:       *bootDiskKmsKeyring != "",
			HasBootDiskKmsLocation:      *bootDiskKmsLocation != "",
			HasBootDiskKmsProject:       *bootDiskKmsProject != "",
			NoGuestEnvironment:          *noGuestEnvironment,
			NodeAffinityLabel:           nodeAffinityLabelsFlag.String(),
			Hostname:                    *hostname,
			MachineImageStorageLocation: *machineImageStorageLocation,
		},
	}
}

func createCommonInputParams() *service.CommonParams {
	return &service.CommonParams{
		ClientID:                *clientID,
		ClientVersion:           *clientVersion,
		Network:                 *network,
		Subnet:                  *subnet,
		Zone:                    *zoneFlag,
		Timeout:                 *timeout,
		Project:                 *project,
		ObfuscatedProject:       service.Hash(*project),
		Labels:                  *labels,
		ScratchBucketGcsPath:    *scratchBucketGcsPath,
		Oauth:                   *oauth,
		ComputeEndpointOverride: *ce,
		DisableGcsLogging:       *gcsLogsDisabled,
		DisableCloudLogging:     *cloudLogsDisabled,
		DisableStdoutLogging:    *stdoutLogsDisabled,
	}
}
