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

// GCE one-step image export tool
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	awsImporter "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_onestep_image_import/aws_importer"
	imageImporter "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/importer"
	"google.golang.org/api/option"
)

var (
	cloudProvider 			 = flag.String("cloud_provider", "", "The cloud provider to import image.")
	clientID             = flag.String("client_id", "", "Identifies the client of the importer, e.g. `gcloud` or `pantheon`")
	imageName            = flag.String("image_name", "", "Image name to be imported.")
	osID                   = flag.String("os", "", "Specifies the OS of the image being imported. OS must be one of: centos-6, centos-7, debian-8, debian-9, opensuse-15, sles-12-byol, sles-15-byol, rhel-6, rhel-6-byol, rhel-7, rhel-7-byol, ubuntu-1404, ubuntu-1604, ubuntu-1804, windows-10-byol, windows-2008r2, windows-2008r2-byol, windows-2012, windows-2012-byol, windows-2012r2, windows-2012r2-byol, windows-2016, windows-2016-byol, windows-7-byol.")
	customTranWorkflow   = flag.String("custom_translate_workflow", "", "Specifies the custom workflow used to do translation")
	noGuestEnvironment   = flag.Bool("no_guest_environment", false, "Google Guest Environment will not be installed on the image.")
	family               = flag.String("family", "", "Family to set for the translated image")
	description          = flag.String("description", "", "Description to set for the translated image")
	network              = flag.String("network", "", "Name of the network in your project to use for the image import. The network must have access to Google Cloud Storage. If not specified, the network named default is used.")
	subnet               = flag.String("subnet", "", "Name of the subnetwork in your project to use for the image import. If	the network resource is in legacy mode, do not provide this property. If the network is in auto subnet mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this field should be specified. Zone should be specified if this field is specified.")
	zone                 = flag.String("zone", "", "Zone of the image to import. The zone in which to do the work of importing the image. Overrides the default compute/zone property value for this command invocation.")
	timeout              = flag.Duration("timeout", time.Hour*2, "Maximum time a build can last before it is failed as TIMEOUT. For example, specifying 2h will fail the process after 2 hours. See $ gcloud topic datetimes for information on duration formats.")
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

	awsImageId          = flag.String("aws_image_id", "", ".")
	awsExportLocation   = flag.String("aws_export_location", "", ".")
	awsRegion     			= flag.String("aws_region", "", ".")
	awsSessionToken     = flag.String("aws_session_token", "", ".")
	awsAccessKeyId      = flag.String("aws_access_key_id", "", ".")
	awsSecrectAccessKey = flag.String("aws_secret_access_key", "", ".")
	awsExportedAMIPath  = flag.String("aws_exported_ami_path", "", ".")
	resumeExportedAMI   = flag.Bool("resume_exported_ami", false, ".")

	imageImportArgs *imageImporter.ImportArguments
)

const (
	// TODO: add comment for flag key
	ImageNameFlagKay = "image_name"
	CloudProviderFlagKey = "cloud_provider"
	ClientIdFlagKey = "client_id"
	OsFlagKey = "os"
)

func awsImportEntry() (service.Loggable, error) {
	var importer *awsImporter.AwsImporter
	awsArgs, err := awsImporter.Parse(*awsImageId, *awsExportLocation, *awsAccessKeyId, *awsSecrectAccessKey, *awsSessionToken, *awsRegion, *awsExportedAMIPath, *resumeExportedAMI)
	if err != nil {
		return nil, err
	}
	if importer, err = awsImporter.NewImporter(*oauth, awsArgs); err != nil {
		return nil, err
	}
	exportedGCSPath, err := importer.Run(imageImportArgs)
	if err != nil {
		return nil, err
	}
	return runImageImport(exportedGCSPath)
}

func runImageImport(exportedGCSPath string) (service.Loggable, error) {
	ctx := context.Background()
	// 1. Fill in args for image import
	imageImportArgs = getImageImportArgs()
	imageImportArgs.SourceFile = exportedGCSPath

	// 2. Setup dependencies.
	storageClient, err := storage.NewStorageClient(
		ctx, logging.NewDefaultLogger(), option.WithCredentialsFile(*oauth))
	if err != nil {
		terminate(err)
	}
	computeClient, err := param.CreateComputeClient(
		&ctx, *oauth, *ce)
	if err != nil {
		terminate(err)
	}
	metadataGCE := &compute.MetadataGCE{}
	paramPopulator := param.NewPopulator(
		metadataGCE,
		storageClient,
		storage.NewResourceLocationRetriever(metadataGCE, computeClient),
		storage.NewScratchBucketCreator(ctx, storageClient),
	)

	// 3. Parse, validate, and populate arguments.
	if err = imageImportArgs.ValidateAndPopulate(
		paramPopulator, imageImporter.NewSourceFactory(storageClient)); err != nil {
		terminate(err)
	}

	importRunner, err := imageImporter.NewImporter(*imageImportArgs, computeClient)
	if err != nil {
		terminate(err)
	}

	return importRunner.Run(ctx)

}

func main() {
	if err := parse(); err != nil {
		terminate(err)
		os.Exit(1)
	}

	if *cloudProvider == "aws" {
		paramLog := service.InputParams{}

		if err := service.RunWithServerLogging(service.OneStepImageImportAction, paramLog, project, awsImportEntry); err != nil {
			os.Exit(1)
		}
	} else {
		terminate(fmt.Errorf("Only onestep image import from AWS is supported at the time."))
	}

}

func parse() error {
	flag.Parse()
	if err := validation.ValidateStringFlagNotEmpty(*imageName, ImageNameFlagKay); err != nil {
		return err
	}

	if err := validation.ValidateStringFlagNotEmpty(*cloudProvider, CloudProviderFlagKey); err != nil {
		return err
	}
	if err := validation.ValidateStringFlagNotEmpty(*clientID, ClientIdFlagKey); err != nil {
		return err
	}
	if err := validation.ValidateStringFlagNotEmpty(*osID, OsFlagKey); err != nil {
		return err
	}
	return nil
}

func getImageImportArgs() (*imageImporter.ImportArguments) {
	args := &imageImporter.ImportArguments{
		path.RandString(5),
		*clientID,
		*cloudLogsDisabled,
		*ce,
		os.Args[0],
		*customTranWorkflow,
		false,
		*description,
		*family,
		*gcsLogsDisabled,
		*imageName,
		make(map[string]string),
		*network,
		*noExternalIP,
		*noGuestEnvironment,
		*oauth,
		*osID,
		*project,
		"",
		*scratchBucketGcsPath,
		nil,
		"",
		"",
		*stdoutLogsDisabled,
		*storageLocation,
		*subnet,
		false,
		*timeout,
		*uefiCompatible,
		*zone}
	return args
}

func createImageImportLoggingParams() *service.ImageImportParams{
	return &service.ImageImportParams{
			CommonParams: &service.CommonParams{
				ClientID:                *clientID,
				Network:                 *network,
				Subnet:                  *subnet,
				Zone:                    *zone,
				Timeout:                 timeout.String(),
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
			OS:                 *osID,
			NoGuestEnvironment: *noGuestEnvironment,
			Family:             *family,
			Description:        *description,
			NoExternalIP:       *noExternalIP,
			StorageLocation:    *storageLocation,
	}
}

func createAWSLoggingParams() service.InputParams {
	return service.InputParams{
		OnestepImageImportAWSParams: &service.OnestepImageImportAWSParams{
			ImageImportParams: createImageImportLoggingParams(),
			AccessKeyId:       *awsAccessKeyId,
			SecretAccessKey:   *awsSecrectAccessKey,
			SessionToken:      *awsSessionToken,
			Region:            *awsRegion,
			ImageId:           *awsImageId,
			ExportLocation:    *awsExportLocation,
			ExportedAMIPath:   *awsExportedAMIPath,
			ResumeExportedAMI: *resumeExportedAMI,
			OS:                *osID,
			CloudProvider:     *cloudProvider,
		},
	}
}


// terminate is used when there is a failure prior to running import. It sends
// a message to the logging framework, and then executes os.Exit(1).
func terminate(cause error) {
	noOpCallback := func() (service.Loggable, error) {
		return nil, cause
	}
	// Ignoring the returned error since its a copy of
	// the return value from the callback.
	_ = service.RunWithServerLogging(
		service.OneStepImageImportAction, createAWSLoggingParams(), nil, noOpCallback)
	os.Exit(1)
}
