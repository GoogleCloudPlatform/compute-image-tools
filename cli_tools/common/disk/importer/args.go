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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/flags"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
)

// Flags that are validated.
const (
	imageFlag          = "image_name"
	clientFlag         = "client_id"
	byolFlag           = "byol"
	dataDiskFlag       = "data_disk"
	osFlag             = "os"
	customWorkflowFlag = "custom_translate_workflow"
	workflowDir        = "daisy_workflows"
)

// ImportArguments holds the structured results of parsing CLI arguments,
// and optionally allows for validating and populating the arguments.
type ImportArguments struct {
	ExecutionID          string
	ClientID             string
	ClientVersion        string
	CloudLogsDisabled    bool
	ComputeEndpoint      string
	WorkflowDir          string
	CustomWorkflow       string
	DataDisk             bool
	Description          string
	Family               string
	GcsLogsDisabled      bool
	ImageName            string
	Inspect              bool
	Labels               map[string]string
	Network              string
	NoExternalIP         bool
	NoGuestEnvironment   bool
	Oauth                string
	BYOL                 bool
	OS                   string
	Project              string
	Region               string
	ScratchBucketGcsPath string
	Source               Source
	SourceFile           string
	SourceImage          string
	StdoutLogsDisabled   bool
	StorageLocation      string
	Subnet               string
	SysprepWindows       bool
	Started              time.Time
	Timeout              time.Duration
	UefiCompatible       bool
	Zone                 string
}

// NewImportArguments parses args to create an ImportArguments instance.
// No validation occurs; to validate, use ValidateAndPopulate.
func NewImportArguments(args []string) (ImportArguments, error) {
	flagSet := flag.NewFlagSet("image-import", flag.ContinueOnError)
	// Don't write parse errors to stdout, instead propagate them via an
	// exception since we use flag.ContinueOnError.
	flagSet.SetOutput(ioutil.Discard)

	parsed := ImportArguments{
		WorkflowDir: filepath.Join(filepath.Dir(os.Args[0]), workflowDir),
		Started:     time.Now(),
	}

	parsed.registerFlags(flagSet)

	return parsed, flagSet.Parse(args)
}

// ValidateAndPopulate parses, validates, and populates the arguments.
func (args *ImportArguments) ValidateAndPopulate(populator param.Populator,
	sourceFactory SourceFactory) (err error) {
	args.Source, err = sourceFactory.Init(args.SourceFile, args.SourceImage)
	if err != nil {
		return err
	}

	if err := populator.PopulateMissingParameters(&args.Project, args.ClientID, &args.Zone, &args.Region,
		&args.ScratchBucketGcsPath, args.SourceFile, &args.StorageLocation); err != nil {
		return err
	}

	args.populateExecutionID()

	args.populateNamespacedScratchDirectory()
	if err := args.populateNetwork(); err != nil {
		return err
	}

	return args.validate()
}

func (args *ImportArguments) populateExecutionID() {
	if args.ExecutionID == "" {
		args.ExecutionID = path.RandString(5)
	}
}

// populateNamespacedScratchDirectory updates ScratchBucketGcsPath to include a directory
// that is specific to this import, formulated using the start timestamp and the execution ID.
// This ensures all logs and artifacts are contained in a single directory.
func (args *ImportArguments) populateNamespacedScratchDirectory() {
	if !strings.HasSuffix(args.ScratchBucketGcsPath, "/") {
		args.ScratchBucketGcsPath += "/"
	}

	args.ScratchBucketGcsPath += fmt.Sprintf(
		"gce-image-import-%s-%s", args.Started.Format(time.RFC3339), args.ExecutionID)
}

func (args ImportArguments) validate() error {
	if args.ClientID == "" {
		return fmt.Errorf("-%s has to be specified", clientFlag)
	}
	if args.ImageName == "" {
		return fmt.Errorf("-%s has to be specified", imageFlag)
	}
	if args.BYOL && (args.DataDisk || args.OS != "" || args.CustomWorkflow != "") {
		return fmt.Errorf("when -%s is specified, -%s, -%s, and -%s have to be empty",
			byolFlag, dataDiskFlag, osFlag, customWorkflowFlag)
	}
	if args.DataDisk && (args.OS != "" || args.CustomWorkflow != "") {
		return fmt.Errorf("when -%s is specified, -%s and -%s should be empty",
			dataDiskFlag, osFlag, customWorkflowFlag)
	}
	if args.OS != "" && args.CustomWorkflow != "" {
		return fmt.Errorf("-%s and -%s can't be both specified",
			osFlag, customWorkflowFlag)
	}
	if !strings.HasSuffix(args.ScratchBucketGcsPath, args.ExecutionID) {
		panic("Scratch bucket should have been namespaced with execution ID.")
	}
	if args.OS != "" {
		if err := daisy.ValidateOS(args.OS); err != nil {
			return err
		}
	}
	return nil
}

func (args *ImportArguments) populateNetwork() error {
	// Populate Network and Subnet. Two goals:
	//
	// a. Explicitly use the 'default' network only when
	//    network is omitted and subnet is empty.
	// b. Convert bare identifiers to URIs.
	//
	// Rules: https://cloud.google.com/vpc/docs/vpc
	if args.Network == "" && args.Subnet == "" {
		args.Network = "default"
	}
	if args.Subnet != "" {
		args.Subnet = param.GetRegionalResourcePath(args.Region, "subnetworks", args.Subnet)
	}
	if args.Network != "" {
		args.Network = param.GetGlobalResourcePath("networks", args.Network)
	}

	return nil
}

func (args *ImportArguments) registerFlags(flagSet *flag.FlagSet) {
	flagSet.Var((*flags.LowerTrimmedString)(&args.ClientID), clientFlag,
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

	flagSet.BoolVar(&args.GcsLogsDisabled, "disable_gcs_logging", false,
		"Do not store logs in GCS.")

	flagSet.BoolVar(&args.CloudLogsDisabled, "disable_cloud_logging", false,
		"Do not store logs in Cloud Logging.")

	flagSet.BoolVar(&args.StdoutLogsDisabled, "disable_stdout_logging", false,
		"Do not write logs to stdout.")

	flagSet.BoolVar(&args.NoExternalIP, "no_external_ip", false,
		"VPC doesn't allow external IPs.")

	flagSet.BoolVar(&args.Inspect, "inspect", true, "Run disk inspections.")

	flagSet.Var((*flags.TrimmedString)(&args.ExecutionID), "execution_id",
		"The execution ID to differentiate GCE resources of each imports.")

	flagSet.Bool("kms_key", false, "Reserved for future use.")
	flagSet.Bool("kms_keyring", false, "Reserved for future use.")
	flagSet.Bool("kms_location", false, "Reserved for future use.")
	flagSet.Bool("kms_project", false, "Reserved for future use.")

	flagSet.Var((*flags.LowerTrimmedString)(&args.ImageName), imageFlag,
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

	flagSet.Var((*flags.TrimmedString)(&args.SourceFile), "source_file",
		"The Cloud Storage URI of the virtual disk file to import.")

	flagSet.Var((*flags.TrimmedString)(&args.SourceImage), "source_image",
		"An existing Compute Engine image from which to import.")

	flagSet.BoolVar(&args.BYOL, byolFlag, false,
		"Specifies that a BYOL license should be applied.")

	flagSet.BoolVar(&args.DataDisk, dataDiskFlag, false,
		"Specifies that the disk has no bootable OS installed on it. "+
			"Imports the disk without making it bootable or installing Google tools on it.")

	flagSet.Var((*flags.LowerTrimmedString)(&args.OS), osFlag,
		"Specifies the OS of the image being imported. OS must be one of: "+
			strings.Join(daisy.GetSortedOSIDs(), ", ")+".")

	flagSet.BoolVar(&args.NoGuestEnvironment, "no_guest_environment", false,
		"When enabled, the Google Guest Environment will not be installed.")

	flagSet.DurationVar(&args.Timeout, "timeout", time.Hour*2,
		"Maximum time a build can last before it is failed as TIMEOUT. For example, "+
			"specifying 2h will fail the process after 2 hours. See $ gcloud topic datetimes "+
			"for information on duration formats.")

	flagSet.Var((*flags.TrimmedString)(&args.CustomWorkflow), customWorkflowFlag,
		"A Daisy workflow JSON file to use for translation.")

	flagSet.BoolVar(&args.UefiCompatible, "uefi_compatible", false,
		"Enables UEFI booting, which is an alternative system boot method. "+
			"Most public images use the GRUB bootloader as their primary boot method.")

	flagSet.BoolVar(&args.SysprepWindows, "sysprep_windows", false,
		"Whether to generalize image using Windows Sysprep. Only applicable to Windows.")
}

// DaisyAttrs returns the subset of DaisyAttrs that are required to instantiate
// a daisy workflow.
func (args ImportArguments) DaisyAttrs() daisycommon.WorkflowAttributes {
	return daisycommon.WorkflowAttributes{
		Project:           args.Project,
		Zone:              args.Zone,
		GCSPath:           args.ScratchBucketGcsPath,
		OAuth:             args.Oauth,
		Timeout:           args.Timeout.String(),
		ComputeEndpoint:   args.ComputeEndpoint,
		DisableGCSLogs:    args.GcsLogsDisabled,
		DisableCloudLogs:  args.CloudLogsDisabled,
		DisableStdoutLogs: args.StdoutLogsDisabled,
		NoExternalIP:      args.NoExternalIP,
		WorkflowDirectory: args.WorkflowDir,
	}
}
