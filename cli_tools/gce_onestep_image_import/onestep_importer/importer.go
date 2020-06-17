package importer

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	daisyUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	awsImporter "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_onestep_image_import/onestep_importer/aws_importer"
	onestepUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_onestep_image_import/onestep_utils"
)

// ImportArguments holds the structured results of parsing CLI arguments,
// and optionally allows for validating and populating the arguments.
type ImportArguments struct {
	ClientID             string
	CloudLogsDisabled    bool
	CloudProvider        string
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
	Project              string
	Region               string
	ScratchBucketGcsPath string
	StdoutLogsDisabled   bool
	StorageLocation      string
	Subnet               string
	SysprepWindows       bool
	Timeout              time.Duration
	UefiCompatible       bool
	Zone                 string

	AWSAccessKeyID       string
	AWSSecretAccessKey   string
	AWSSessionToken      string
	AWSRegion            string
	AWSImageID           string
	AWSExportLocation    string
	AWSExportedAMIPath   string
	AWSResumeExportedAMI bool
}

// Flags that are validated.
const (
	cloudProviderFlag = "cloud_provider"
	clientFlag        = "client_id"
	imageNameFlag     = "image_name"
	osFlag            = "os"
)

// NewImportArguments parse the provides cli arguments and creates a new ImportArguments instance.
func NewImportArguments(args []string) (*ImportArguments, error) {
	importArgs := &ImportArguments{}
	importArgs.ExecutablePath = os.Args[0]
	flagSet := importArgs.getFlagSet()
	if err := flagSet.Parse(args); err != nil {
		return nil, err
	}
	return importArgs, nil
}

// getFlagSet gets the FlagSet used to parse arguments.
func (args *ImportArguments) getFlagSet() *flag.FlagSet {
	flagSet := flag.NewFlagSet("onestep-image-import", flag.ContinueOnError)
	flagSet.SetOutput(ioutil.Discard)
	args.registerFlags(flagSet)
	return flagSet
}

// registerFlags defines the flags to parse.
func (args *ImportArguments) registerFlags(flagSet *flag.FlagSet) {
	//TODO: add comment for aws flags
	flagSet.Var((*trimmedString)(&args.AWSAccessKeyID), awsImporter.AccessKeyIDFlag, ".")
	flagSet.Var((*trimmedString)(&args.AWSImageID), awsImporter.ImageIDFlag, ".")
	flagSet.Var((*trimmedString)(&args.AWSExportLocation), awsImporter.ExportLocationFlag, ".")
	flagSet.Var((*trimmedString)(&args.AWSExportedAMIPath), awsImporter.ExportedAMIPathFlag, ".")
	flagSet.Var((*trimmedString)(&args.AWSRegion), awsImporter.RegionFlag, ".")
	flagSet.BoolVar(&args.AWSResumeExportedAMI, awsImporter.ResumeExportedAMIFlag, false,
		".")
	flagSet.Var((*trimmedString)(&args.AWSSessionToken), awsImporter.SessionTokenFlag, ".")
	flagSet.Var((*trimmedString)(&args.AWSSecretAccessKey), awsImporter.SecretAccessKeyFlag, ".")

	flagSet.Var((*lowerTrimmedString)(&args.CloudProvider), cloudProviderFlag,
		"Identifies the cloud provider of the import source, e.g. 'aws'.")

	flagSet.Var((*lowerTrimmedString)(&args.ClientID), clientFlag,
		"Identifies the client of the importer, e.g. 'gcloud', 'pantheon', or 'api'.")

	flagSet.Var((*trimmedString)(&args.Project), "project",
		"The project where workflows will be run, and where the resulting image will be stored.")

	flagSet.Var((*trimmedString)(&args.Network), "network",
		"Name of the network in your project to use for the image import. "+
			"The network must have access to Google Cloud Storage. "+
			"If not specified, the network named default is used.")

	flagSet.Var((*trimmedString)(&args.Subnet), "subnet",
		"Name of the subnetwork in your project to use for the image import. "+
			"If the network resource is in legacy mode, do not provide this property. "+
			"If the network is in auto subnet mode, providing the subnetwork is optional. "+
			"If the network is in custom subnet mode, then this field should be specified. "+
			"Zone should be specified if this field is specified.")

	flagSet.Var((*lowerTrimmedString)(&args.Zone), "zone",
		"The zone where workflows will be run, and where the resulting image will be stored.")

	flagSet.Var((*trimmedString)(&args.ScratchBucketGcsPath), "scratch_bucket_gcs_path",
		"A system-generated bucket name will be used if omitted. "+
			"If the bucket doesn't exist, it will be created. If it does exist, it will be reused.")

	flagSet.Var((*trimmedString)(&args.Oauth), "oauth",
		"Path to oauth json file.")

	flagSet.Var((*trimmedString)(&args.ComputeEndpoint), "compute_endpoint_override",
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

	flagSet.Var((*lowerTrimmedString)(&args.ImageName), imageNameFlag,
		"Name of the disk image to create.")

	flagSet.Var((*trimmedString)(&args.Family), "family",
		"Family to set for the imported image.")

	flagSet.Var((*trimmedString)(&args.Description), "description",
		"Description to set for the imported image.")

	flagSet.Var((*keyValueString)(&args.Labels), "labels",
		"List of label KEY=VALUE pairs to add. "+
			"For more information, see: https://cloud.google.com/compute/docs/labeling-resources")

	flagSet.Var((*lowerTrimmedString)(&args.StorageLocation), "storage_location",
		"Specifies a Cloud Storage location, either regional or multi-regional, "+
			"where image content is to be stored. If not specified, the multi-region "+
			"location closest to the source is chosen automatically.")

	flagSet.Var((*lowerTrimmedString)(&args.OS), osFlag,
		"Specifies the OS of the image being imported. OS must be one of: "+
			"centos-6, centos-7, debian-8, debian-9, opensuse-15, sles-12-byol, "+
			"sles-15-byol, rhel-6, rhel-6-byol, rhel-7, rhel-7-byol, ubuntu-1404, "+
			"ubuntu-1604, ubuntu-1804, windows-10-byol, windows-2008r2, windows-2008r2-byol, "+
			"windows-2012, windows-2012-byol, windows-2012r2, windows-2012r2-byol, "+
			"windows-2016, windows-2016-byol, windows-7-byol.")

	flagSet.BoolVar(&args.NoGuestEnvironment, "no_guest_environment", false,
		"When enabled, the Google Guest Environment will not be installed.")

	flagSet.DurationVar(&args.Timeout, "timeout", time.Hour*2,
		"Maximum time a build can last before it is failed as TIMEOUT. For example, "+
			"specifying 2h will fail the process after 2 hours. See $ gcloud topic datetimes "+
			"for information on duration formats.")

	flagSet.Var((*trimmedString)(&args.CustomWorkflow), "custom_translate_workflow",
		"A Daisy workflow JSON file to use for translation.")

	flagSet.BoolVar(&args.UefiCompatible, "uefi_compatible", false,
		"Enables UEFI booting, which is an alternative system boot method. "+
			"Most public images use the GRUB bootloader as their primary boot method.")

	flagSet.BoolVar(&args.SysprepWindows, "sysprep_windows", false,
		"Whether to generalize image using Windows Sysprep. Only applicable to Windows.")
}

func (args *ImportArguments) validate() error {
	if err := validation.ValidateStringFlagNotEmpty(args.ImageName, imageNameFlag); err != nil {
		return err
	}
	if err := validation.ValidateStringFlagNotEmpty(args.CloudProvider, cloudProviderFlag); err != nil {
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

// buildAWSImportArgs creates a new AWSImportArgument instance.
func (args *ImportArguments) buildAWSImportArgs() *awsImporter.AWSImportArguments {
	return &awsImporter.AWSImportArguments{
		AccessKeyID:        args.AWSAccessKeyID,
		ExecutablePath:     args.ExecutablePath,
		ExportLocation:     args.AWSExportLocation,
		ExportedAMIPath:    args.AWSExportedAMIPath,
		GcsComputeEndpoint: args.ComputeEndpoint,
		GcsProject:         args.Project,
		GcsZone:            args.Zone,
		GcsRegion:          args.Region,
		GcsScratchBucket:   args.ScratchBucketGcsPath,
		GcsStorageLocation: args.StorageLocation,
		ImageID:            args.AWSImageID,
		Region:             args.AWSRegion,
		ResumeExportedAMI:  args.AWSResumeExportedAMI,
		SecretAccessKey:    args.AWSSecretAccessKey,
		SessionToken:       args.AWSSessionToken,
	}
}

// Run performs onestep image import.
func Run(args *ImportArguments) (service.Loggable, error) {
	// 1. Validate required flags that are not cloud-provider specific.
	if err := args.validate(); err != nil {
		return nil, err
	}

	if args.CloudProvider == "aws" {
		return importAMI(args)
	}

	return nil, fmt.Errorf("import from cloud provider %v is currently not supported", args.CloudProvider)
}

// importAMI imports image from AWS.
func importAMI(args *ImportArguments) (service.Loggable, error) {
	importer, err := awsImporter.NewImporter(args.Oauth, args.buildAWSImportArgs())
	if err != nil {
		return nil, err
	}
	exportedGCSPath, err := importer.Run()
	if err != nil {
		return nil, err
	}

	err = runImageImport(exportedGCSPath, args)
	if err != nil {
		log.Println("Failed to import image.",
			fmt.Sprintf("The image file has been copied to Google Cloud Storage, located at %v.", exportedGCSPath),
			"To resume the import process, please directly use image import from GCS.")
	}
	return nil, err
}

// runImageImport imports image from the provided GCS path.
func runImageImport(exportedGCSPath string, args *ImportArguments) error {
	if args.Labels == nil {
		args.Labels = make(map[string]string)
		args.Labels["onestep-image-import"] = args.CloudProvider
	}
	err := onestepUtils.RunCmd("gce_vm_image_import", []string{
		fmt.Sprintf("-image_name=%v", args.ImageName),
		fmt.Sprintf("-client_id=%v", args.ClientID),
		fmt.Sprintf("-os=%v", args.OS),
		fmt.Sprintf("-source_file=%v", exportedGCSPath),
		fmt.Sprintf("-no_guest_environment=%v", args.NoGuestEnvironment),
		fmt.Sprintf("-family=%v", args.Family),
		fmt.Sprintf("-description=%v", args.Description),
		fmt.Sprintf("-network=%v", args.Network),
		fmt.Sprintf("-subnet=%v", args.Subnet),
		fmt.Sprintf("-zone=%v", args.Zone),
		fmt.Sprintf("-timeout=%v", args.Timeout),
		fmt.Sprintf("-project=%v", args.Project),
		fmt.Sprintf("-scratch_bucket_gcs_path=%v", args.ScratchBucketGcsPath),
		fmt.Sprintf("-oauth=%v", args.Oauth),
		fmt.Sprintf("-compute_endpoint_override=%v", args.ComputeEndpoint),
		fmt.Sprintf("-disable_gcs_logging=%v", args.GcsLogsDisabled),
		fmt.Sprintf("-disable_cloud_logging=%v", args.CloudLogsDisabled),
		fmt.Sprintf("-disable_stdout_logging=%v", args.StdoutLogsDisabled),
		fmt.Sprintf("-no_external_ip=%v", args.NoExternalIP),
		fmt.Sprintf("-labels=%v", keyValueString(args.Labels).String()),
		fmt.Sprintf("-storage_location=%v", args.StorageLocation)})
	if err != nil {
		return err
	}
	return nil
}

//TODO: put these into a common utils section

// keyValueString is an implementation of flag.Value that creates a map
// from the user's argument prior to storing it. It expects the argument
// is in the form KEY1=AB,KEY2=CD. For more info on the format, see
// param.ParseKeyValues.
type keyValueString map[string]string

func (s keyValueString) String() string {
	parts := []string{}
	for k, v := range s {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}

func (s *keyValueString) Set(input string) error {
	if *s != nil {
		return fmt.Errorf("only one instance of this flag is allowed")
	}

	*s = make(map[string]string)
	if input != "" {
		var err error
		*s, err = param.ParseKeyValues(input)
		if err != nil {
			return err
		}
	}
	return nil
}

// trimmedString is an implementation of flag.Value that trims whitespace
// from the incoming argument prior to storing it.
type trimmedString string

func (s trimmedString) String() string { return (string)(s) }
func (s *trimmedString) Set(input string) error {
	*s = trimmedString(strings.TrimSpace(input))
	return nil
}

// lowerTrimmedString is an implementation of flag.Value that trims whitespace
// and converts to lowercase the incoming argument prior to storing it.
type lowerTrimmedString string

func (s lowerTrimmedString) String() string { return (string)(s) }
func (s *lowerTrimmedString) Set(input string) error {
	*s = lowerTrimmedString(strings.ToLower(strings.TrimSpace(input)))
	return nil
}
