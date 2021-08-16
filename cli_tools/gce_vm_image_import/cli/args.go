//  Copyright 2021 Google Inc. All Rights Reserved.
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

package cli

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/flags"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
)

// imageImportArgs receives arguments passed by the user and facilitates creating
// importer.ImageImportRequest.
type imageImportArgs struct {
	ClientID      string
	ClientVersion string
	Region        string
	SourceFile    string
	SourceImage   string
	Started       time.Time
	importer.ImageImportRequest
}

// parseArgsFromUser creates an imageImportArgs instance from the arguments
// passed by the user.
func parseArgsFromUser(argsFromUser []string) (imageImportArgs, error) {
	flagSet := flag.NewFlagSet("image-import", flag.ContinueOnError)
	// Don't write parse errors to stdout, instead propagate them via an
	// exception since we use flag.ContinueOnError.
	flagSet.SetOutput(ioutil.Discard)
	parsed := imageImportArgs{}
	parsed.registerFlags(flagSet)
	return parsed, flagSet.Parse(argsFromUser)
}

// populateAndValidate populates missing arguments, and validates specified
// arguments. We depend on the importer module to validate *its* arguments,
// so validation is limited to the fields that aren't used by that module.
func (args *imageImportArgs) populateAndValidate(populator param.Populator,
	sourceFactory importer.SourceFactory) (err error) {

	if args.Started == (time.Time{}) {
		args.Started = time.Now()
	}

	if args.ExecutionID == "" {
		args.ExecutionID = path.RandString(5)
	}

	importer.FixBYOLAndOSArguments(&args.OS, &args.BYOL)
	args.Source, err = sourceFactory.Init(args.SourceFile, args.SourceImage)
	if err != nil {
		return err
	}
	if err := populator.PopulateMissingParameters(&args.Project, args.ClientID, &args.Zone, &args.Region,
		&args.ScratchBucketGcsPath, args.SourceFile, &args.StorageLocation, &args.Network, &args.Subnet); err != nil {
		return err
	}

	// Ensure that all workflow logs are put in the same GCS directory.
	// path.join doesn't work since it converts `gs://` to `gs:/`.
	if !strings.HasSuffix(args.ScratchBucketGcsPath, "/") {
		args.ScratchBucketGcsPath += "/"
	}
	args.ScratchBucketGcsPath += fmt.Sprintf(
		"gce-image-import-%s-%s", args.Started.Format(time.RFC3339), args.ExecutionID)

	return nil
}

func (args *imageImportArgs) registerFlags(flagSet *flag.FlagSet) {
	flagSet.Var((*flags.LowerTrimmedString)(&args.ClientID), importer.ClientFlag,
		"Identifies the client of the importer, e.g. 'gcloud', 'pantheon', or 'api'.")

	flagSet.Var((*flags.TrimmedString)(&args.ClientVersion), "client_version",
		"Identifies the version of the client of the importer.")

	flagSet.Var((*flags.TrimmedString)(&args.Project), "project",
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

	flagSet.Var((*flags.TrimmedString)(&args.ComputeServiceAccount), "compute_service_account",
		"Compute service account to be used by importer Virtual Machine. When empty, the Compute Engine default service account is used.")

	flagSet.BoolVar(&args.GcsLogsDisabled, "disable_gcs_logging", false,
		"Do not store logs in GCS.")

	flagSet.BoolVar(&args.CloudLogsDisabled, "disable_cloud_logging", false,
		"Do not store logs in Cloud Logging.")

	flagSet.BoolVar(&args.StdoutLogsDisabled, "disable_stdout_logging", false,
		"Do not write logs to stdout.")

	flagSet.BoolVar(&args.NoExternalIP, "no_external_ip", false,
		"Temporary VMs are created in your project during {operation}. "+
			"Set this flag so that these temporary VMs are not assigned external IP addresses. "+
			"For more information, see: https://cloud.google.com/compute/docs/import/importing-virtual-disks#no-external-ip")

	flagSet.Var((*flags.TrimmedString)(&args.ExecutionID), "execution_id",
		"The execution ID to differentiate GCE resources of each imports.")

	flagSet.Bool("kms_key", false, "Reserved for future use.")
	flagSet.Bool("kms_keyring", false, "Reserved for future use.")
	flagSet.Bool("kms_location", false, "Reserved for future use.")
	flagSet.Bool("kms_project", false, "Reserved for future use.")

	flagSet.Var((*flags.LowerTrimmedString)(&args.ImageName), importer.ImageFlag,
		"Name of the disk image to create.")

	flagSet.Var((*flags.TrimmedString)(&args.Family), "family",
		"Family to set for the imported image.")

	flagSet.Var((*flags.TrimmedString)(&args.Description), "description",
		"Description to set for the imported image.")

	flagSet.Var((*flags.KeyValueString)(&args.Labels), "labels",
		"List of label KEY=VALUE pairs to add. "+
			"For more information, see: https://cloud.google.com/compute/docs/labeling-resources")

	flagSet.Var((*flags.LowerTrimmedString)(&args.StorageLocation), "storage_location",
		"Specifies a Cloud Storage location, either a region or a multi-region, "+
			"where image content is to be stored. If not specified, the multi-region "+
			"location closest to the source is chosen automatically.")

	flagSet.Var((*flags.TrimmedString)(&args.SourceFile), "source_file",
		"The Cloud Storage URI of the virtual disk file to import.")

	flagSet.Var((*flags.TrimmedString)(&args.SourceImage), "source_image",
		"An existing Compute Engine image from which to import.")

	flagSet.BoolVar(&args.BYOL, importer.BYOLFlag, false,
		"Import using an existing license. These are equivalent: "+
			"`-os=rhel-8 -byol`, `-os=rhel-8-byol -byol`, and `-os=rhel-8-byol`")

	flagSet.BoolVar(&args.DataDisk, importer.DataDiskFlag, false,
		"Specifies that the disk has no bootable OS installed on it. "+
			"Imports the disk without making it bootable or installing Google tools on it.")

	flagSet.Var((*flags.LowerTrimmedString)(&args.OS), importer.OSFlag,
		"Specifies the OS of the image being imported. OS must be one of: "+
			strings.Join(daisyutils.GetSortedOSIDs(), ", ")+".")

	flagSet.BoolVar(&args.NoGuestEnvironment, "no_guest_environment", false,
		"When enabled, the Google Guest Environment will not be installed.")

	flagSet.DurationVar(&args.Timeout, "timeout", time.Hour*2,
		"Maximum time a build can last before it is failed as TIMEOUT. For example, "+
			"specifying 2h will fail the process after 2 hours. See $ gcloud topic datetimes "+
			"for information on duration formats.")

	flagSet.Var((*flags.TrimmedString)(&args.CustomWorkflow), importer.CustomWorkflowFlag,
		"A Daisy workflow JSON file to use for translation.")

	flagSet.BoolVar(&args.UefiCompatible, "uefi_compatible", false,
		"Enables UEFI booting, which is an alternative system boot method. "+
			"Most public images use the GRUB bootloader as their primary boot method.")

	flagSet.BoolVar(&args.SysprepWindows, "sysprep_windows", false,
		"Generalize image using Windows Sysprep. Only applicable to Windows.")
}
