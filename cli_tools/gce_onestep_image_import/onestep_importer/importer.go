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

package importer

import (
	"flag"
	"io/ioutil"
	"os"
	"time"

	daisyUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/flags"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// OneStepImportArguments holds the structured results of parsing CLI arguments,
// and optionally allows for validating and populating the arguments.
type OneStepImportArguments struct {
	ClientID             string
	CloudLogsDisabled    bool
	ComputeEndpoint      string
	CustomWorkflow       string
	DataDisk             bool
	Description          string
	ExecutablePath       string
	Family               string
	GcsLogsDisabled      bool
	ImageName            string
	Labels               map[string]string
	Network              string
	NoExternalIP         bool
	NoGuestEnvironment   bool
	Oauth                string
	OS                   string
	ProjectPtr           *string
	Region               string
	ScratchBucketGcsPath string
	SourceFile           string
	StdoutLogsDisabled   bool
	StorageLocation      string
	Subnet               string
	SysprepWindows       bool
	Timeout              time.Duration
	TimeoutChan          chan struct{}
	UefiCompatible       bool
	Zone                 string

	AWSAccessKeyID       string
	AWSSecretAccessKey   string
	AWSSessionToken      string
	AWSRegion            string
	AWSAMIID             string
	AWSAMIExportLocation string
	AWSSourceAMIFilePath string
}

// Flags that are validated.
const (
	clientFlag    = "client_id"
	imageNameFlag = "image_name"
	osFlag        = "os"
)

// NewOneStepImportArguments parse the provides cli arguments and creates a new ImportArguments instance.
func NewOneStepImportArguments(args []string) (*OneStepImportArguments, error) {
	importArgs := &OneStepImportArguments{}
	importArgs.ExecutablePath = os.Args[0]
	importArgs.TimeoutChan = make(chan struct{})
	importArgs.ProjectPtr = new(string)
	flagSet := importArgs.getFlagSet()
	if err := flagSet.Parse(args); err != nil {
		return nil, daisy.ToDError(err)
	}

	return importArgs, nil
}

// getFlagSet gets the FlagSet used to parse arguments.
func (args *OneStepImportArguments) getFlagSet() *flag.FlagSet {
	flagSet := flag.NewFlagSet("onestep-image-import", flag.ContinueOnError)
	flagSet.SetOutput(ioutil.Discard)
	args.registerFlags(flagSet)
	return flagSet
}

// registerFlags defines the flags to parse.
func (args *OneStepImportArguments) registerFlags(flagSet *flag.FlagSet) {
	flagSet.Var((*flags.TrimmedString)(&args.AWSAccessKeyID), awsAccessKeyIDFlag,
		"The access key ID for an AWS credential. "+
			"This credential is associated with an IAM user or role. "+
			"This IAM user must have permissions to import images.")

	flagSet.Var((*flags.TrimmedString)(&args.AWSAMIID), awsAMIIDFlag,
		"The AWS AMI ID of the image to import.")

	flagSet.Var((*flags.TrimmedString)(&args.AWSAMIExportLocation), awsAMIExportLocationFlag,
		"The AWS S3 Bucket location where you want to export the image.")

	flagSet.Var((*flags.TrimmedString)(&args.AWSSourceAMIFilePath), awsSourceAMIFilePathFlag,
		"The S3 resource path of the exported image file.")

	flagSet.Var((*flags.TrimmedString)(&args.AWSRegion), awsRegionFlag,
		"The AWS region for the image that you want to import.")

	flagSet.Var((*flags.TrimmedString)(&args.AWSSessionToken), awsSessionTokenFlag,
		"The session token for your AWS credential. "+
			"This credential is associated with an IAM user or role. "+
			"This IAM user must have permissions to import images.")

	flagSet.Var((*flags.TrimmedString)(&args.AWSSecretAccessKey), awsSecretAccessKeyFlag,
		"The secret access key for your AWS credential. "+
			"This credential is associated with an IAM user or role. "+
			"This IAM user must have permissions to import images.")

	flagSet.Var((*flags.LowerTrimmedString)(&args.ClientID), clientFlag,
		"Identifies the client of the importer, e.g. 'gcloud', 'pantheon', or 'api'.")

	flagSet.Var((*flags.TrimmedString)(args.ProjectPtr), "project",
		"The project where workflows will be run, and where the resulting image will be stored.")

	flagSet.Var((*flags.TrimmedString)(&args.Network), "network",
		"Name of the network in your project to use for the image import. "+
			"The network must have access to Google Cloud Storage. "+
			"If not specified, the network named default is used.")

	flagSet.Var((*flags.TrimmedString)(&args.Subnet), "subnet",
		"Name of the subnetwork in your project to use for the image import. "+
			"If the network resource is in legacy mode, do not provide this property. "+
			"If the network is in auto subnet mode, providing the subnetwork is optional. "+
			"If the network is in custom subnet mode, then this field should be specified. "+
			"Zone should be specified if this field is specified.")

	flagSet.Var((*flags.LowerTrimmedString)(&args.Zone), "zone",
		"The zone where workflows will be run, and where the resulting image will be stored.")

	flagSet.Var((*flags.TrimmedString)(&args.ScratchBucketGcsPath), "scratch_bucket_gcs_path",
		"A system-generated bucket name will be used if omitted. "+
			"If the bucket doesn't exist, it will be created. If it does exist, it will be reused.")

	flagSet.Var((*flags.TrimmedString)(&args.Oauth), "oauth",
		"Path to oauth json file.")

	flagSet.Var((*flags.TrimmedString)(&args.ComputeEndpoint), "compute_endpoint_override",
		"API endpoint to override default.")

	flagSet.BoolVar(&args.GcsLogsDisabled, "disable_gcs_logging", false,
		"Do not store logs in GCS.")

	flagSet.BoolVar(&args.CloudLogsDisabled, "disable_cloud_logging", false,
		"Do not store logs in Cloud Logging.")

	flagSet.BoolVar(&args.StdoutLogsDisabled, "disable_stdout_logging", false,
		"Do not write logs to stdout.")

	flagSet.BoolVar(&args.NoExternalIP, "no_external_ip", false,
		"VPC doesn't allow external IPs.")

	flagSet.Bool("kms_key", false, "Reserved for future use.")
	flagSet.Bool("kms_keyring", false, "Reserved for future use.")
	flagSet.Bool("kms_location", false, "Reserved for future use.")
	flagSet.Bool("kms_project", false, "Reserved for future use.")

	flagSet.Var((*flags.LowerTrimmedString)(&args.ImageName), imageNameFlag,
		"Name of the disk image to create.")

	flagSet.Var((*flags.TrimmedString)(&args.Family), "family",
		"Family to set for the imported image.")

	flagSet.Var((*flags.TrimmedString)(&args.Description), "description",
		"Description to set for the imported image.")

	flagSet.Var((*flags.KeyValueString)(&args.Labels), "labels",
		"List of label KEY=VALUE pairs to add. "+
			"For more information, see: https://cloud.google.com/compute/docs/labeling-resources")

	flagSet.Var((*flags.LowerTrimmedString)(&args.StorageLocation), "storage_location",
		"Specifies a Cloud Storage location, either regional or multi-regional, "+
			"where image content is to be stored. If not specified, the multi-region "+
			"location closest to the source is chosen automatically.")

	flagSet.Var((*flags.LowerTrimmedString)(&args.OS), osFlag,
		"Specifies the OS of the disk image being imported. "+
			"This must be specified if cloud provider is specified. "+
			"OS must be one of: centos-6, centos-7, debian-8, debian-9, opensuse-15, "+
			"sles-12-byol, sles-15-byol, rhel-6, rhel-6-byol, rhel-7, rhel-7-byol, "+
			"ubuntu-1404, ubuntu-1604, ubuntu-1804, windows-10-byol, windows-2008r2, "+
			"windows-2008r2-byol, windows-2012, windows-2012-byol, windows-2012r2, "+
			"windows-2012r2-byol, windows-2016, windows-2016-byol, windows-7-byol.")

	flagSet.BoolVar(&args.NoGuestEnvironment, "no_guest_environment", false,
		"When enabled, the Google Guest Environment will not be installed.")

	flagSet.DurationVar(&args.Timeout, "timeout", time.Hour*2,
		"Maximum time a build can last before it is failed as TIMEOUT. For example, "+
			"specifying 2h will fail the process after 2 hours. See $ gcloud topic datetimes "+
			"for information on duration formats.")

	flagSet.Var((*flags.TrimmedString)(&args.CustomWorkflow), "custom_translate_workflow",
		"A Daisy workflow JSON file to use for translation.")

	flagSet.BoolVar(&args.UefiCompatible, "uefi_compatible", false,
		"Enables UEFI booting, which is an alternative system boot method. "+
			"Most public images use the GRUB bootloader as their primary boot method.")

	flagSet.BoolVar(&args.SysprepWindows, "sysprep_windows", false,
		"Whether to generalize image using Windows Sysprep. Only applicable to Windows.")
}

func (args *OneStepImportArguments) validate() error {
	if err := validation.ValidateStringFlagNotEmpty(args.ImageName, imageNameFlag); err != nil {
		return err
	}
	if err := validation.ValidateStringFlagNotEmpty(args.OS, osFlag); err != nil {
		return err
	}
	if err := daisyUtils.ValidateOS(args.OS); err != nil {
		return err
	}
	if err := validation.ValidateStringFlagNotEmpty(args.ClientID, clientFlag); err != nil {
		return err
	}

	return nil
}

// Run performs onestep image import.
func Run(args *OneStepImportArguments) (service.Loggable, error) {
	// validate required flags that are not cloud-provider specific.
	if err := args.validate(); err != nil {
		return nil, err
	}

	return nil, importFromCloudProvider(args)
}
