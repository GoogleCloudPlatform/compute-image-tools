//  Copyright 2018 Google Inc. All Rights Reserved.
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

// GCE VM image import tool
package main

import (
	"flag"
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

var (
	clientID             = flag.String(importer.ClientIDFlagKey, "", "Identifies the client of the importer, e.g. `gcloud` or `pantheon`")
	imageName            = flag.String(importer.ImageNameFlagKey, "", "Image name to be imported.")
	dataDisk             = flag.Bool("data_disk", false, "Specifies that the disk has no bootable OS installed on it.	Imports the disk without making it bootable or installing Google tools on it. ")
	osID                 = flag.String("os", "", "Specifies the OS of the image being imported. OS must be one of: centos-6, centos-7, debian-8, debian-9, opensuse-15, sles-12-byol, sles-15-byol, rhel-6, rhel-6-byol, rhel-7, rhel-7-byol, ubuntu-1404, ubuntu-1604, ubuntu-1804, windows-10-byol, windows-2008r2, windows-2008r2-byol, windows-2012, windows-2012-byol, windows-2012r2, windows-2012r2-byol, windows-2016, windows-2016-byol, windows-7-byol.")
	customTranWorkflow   = flag.String("custom_translate_workflow", "", "Specifies the custom workflow used to do translation")
	sourceFile           = flag.String("source_file", "", "Google Cloud Storage URI of the virtual disk file	to import. For example: gs://my-bucket/my-image.vmdk")
	sourceImage          = flag.String("source_image", "", "Compute Engine image from which to import")
	noGuestEnvironment   = flag.Bool("no_guest_environment", false, "Google Guest Environment will not be installed on the image.")
	family               = flag.String("family", "", "Family to set for the translated image")
	description          = flag.String("description", "", "Description to set for the translated image")
	network              = flag.String("network", "", "Name of the network in your project to use for the image import. The network must have access to Google Cloud Storage. If not specified, the network named default is used.")
	subnet               = flag.String("subnet", "", "Name of the subnetwork in your project to use for the image import. If	the network resource is in legacy mode, do not provide this property. If the network is in auto subnet mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this field should be specified. Zone should be specified if this field is specified.")
	zone                 = flag.String("zone", "", "Zone of the image to import. The zone in which to do the work of importing the image. Overrides the default compute/zone property value for this command invocation.")
	timeout              = flag.String("timeout", "", "Maximum time a build can last before it is failed as TIMEOUT. For example, specifying 2h will fail the process after 2 hours. See $ gcloud topic datetimes for information on duration formats.")
	project              = flag.String("project", "", "project to run in, overrides what is set in workflow")
	scratchBucketGcsPath = flag.String("scratch_bucket_gcs_path", "", "GCS scratch bucket to use, overrides what is set in workflow")
	oauth                = flag.String("oauth", "", "path to oauth json file, overrides what is set in workflow")
	ce                   = flag.String("compute_endpoint_override", "", "API endpoint to override default")
	gcsLogsDisabled      = flag.Bool("disable_gcs_logging", false, "do not stream logs to GCS")
	cloudLogsDisabled    = flag.Bool("disable_cloud_logging", false, "do not stream logs to Cloud Logging")
	stdoutLogsDisabled   = flag.Bool("disable_stdout_logging", false, "do not display individual workflow logs on stdout")
	kmsKey               = flag.String("kms_key", "", "ID of the key or fully qualified identifier for the key. This flag must be specified if any of the other arguments below are specified.")
	kmsKeyring           = flag.String("kms_keyring", "", "The KMS keyring of the key.")
	kmsLocation          = flag.String("kms_location", "", "The Cloud location for the key.")
	kmsProject           = flag.String("kms_project", "", "The Cloud project for the key")
	noExternalIP         = flag.Bool("no_external_ip", false, "VPC doesn't allow external IPs")
	labels               = flag.String("labels", "", "List of label KEY=VALUE pairs to add. Keys must start with a lowercase character and contain only hyphens (-), underscores (_), lowercase characters, and numbers. Values must contain only hyphens (-), underscores (_), lowercase characters, and numbers.")
	storageLocation      = flag.String("storage_location", "", "Location for the imported image which can be any GCS location. If the location parameter is not included, images are created in the multi-region associated with the source disk, image, snapshot or GCS bucket.")
	uefiCompatible       = flag.Bool("uefi_compatible", false, "Enables UEFI booting, which is an alternative system boot method. Most public images use the GRUB bootloader as their primary boot method.")
	sysprepWindows       = flag.Bool("sysprep_windows", false, "Whether to generalize image using Windows Sysprep.")
)

func importEntry() (*daisy.Workflow, error) {
	currentExecutablePath := string(os.Args[0])
	return importer.Run(*clientID, *imageName, *dataDisk, *osID, *customTranWorkflow, *sourceFile,
		*sourceImage, *noGuestEnvironment, *family, *description, *network, *subnet, *zone, *timeout,
		project, *scratchBucketGcsPath, *oauth, *ce, *gcsLogsDisabled, *cloudLogsDisabled,
		*stdoutLogsDisabled, *kmsKey, *kmsKeyring, *kmsLocation, *kmsProject, *noExternalIP,
		*labels, currentExecutablePath, *storageLocation, *uefiCompatible, *sysprepWindows)
}

func main() {
	flag.Parse()

	paramLog := service.InputParams{
		ImageImportParams: &service.ImageImportParams{
			CommonParams: &service.CommonParams{
				ClientID:                *clientID,
				Network:                 *network,
				Subnet:                  *subnet,
				Zone:                    *zone,
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
			},
			ImageName:          *imageName,
			DataDisk:           *dataDisk,
			OS:                 *osID,
			SourceFile:         *sourceFile,
			SourceImage:        *sourceImage,
			NoGuestEnvironment: *noGuestEnvironment,
			Family:             *family,
			Description:        *description,
			NoExternalIP:       *noExternalIP,
			HasKmsKey:          *kmsKey != "",
			HasKmsKeyring:      *kmsKeyring != "",
			HasKmsLocation:     *kmsLocation != "",
			HasKmsProject:      *kmsProject != "",
			StorageLocation:    *storageLocation,
		},
	}

	if err := service.RunWithServerLogging(service.ImageImportAction, paramLog, project, importEntry); err != nil {
		os.Exit(1)
	}
}
