package awsimporter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
)

// AWSImportArguments holds the structured results of parsing CLI arguments,
// and optionally allows for validating and populating the arguments.
type AWSImportArguments struct {
	// Passed in by user
	AccessKeyID        string
	ExecutablePath     string
	ExportLocation     string
	ExportedAMIPath    string
	GcsComputeEndpoint string
	GcsProject         string
	GcsZone            string
	GcsRegion          string
	GcsScratchBucket   string
	GcsStorageLocation string
	ImageID            string
	Region             string
	ResumeExportedAMI  bool
	SecretAccessKey    string
	SessionToken       string

	// Not passed in
	ExportBucket   string
	ExportFolder   string
	ExportKey      string
	ExportFileSize int64
}

// Flags
const (
	ImageIDFlag           = "aws_image_id"
	ExportLocationFlag    = "aws_export_location"
	AccessKeyIDFlag       = "aws_access_key_id"
	SecretAccessKeyFlag   = "aws_secret_access_key"
	SessionTokenFlag      = "aws_session_token"
	RegionFlag            = "aws_region"
	ExportedAMIPathFlag   = "aws_exported_ami_path"
	ResumeExportedAMIFlag = "resume_exported_ami"
)

var (
	bucketNameRegex = `[a-z0-9][-_.a-z0-9]*`
	s3PathRegex     = regexp.MustCompile(fmt.Sprintf(`^s3://(%s)(\/.*)?$`, bucketNameRegex))
)

// ValidateAndPopulate validates args related to import from AWS, and populates
// any missing parameters.
func (args *AWSImportArguments) ValidateAndPopulate(populator param.Populator) error {
	err := args.validate()
	if err != nil {
		return err
	}

	err = populator.PopulateMissingParameters(&args.GcsProject, &args.GcsZone, &args.GcsRegion,
		&args.GcsScratchBucket, "", &args.GcsStorageLocation)
	if err != nil {
		return err
	}

	return args.getS3PathElements()
}

func (args *AWSImportArguments) validate() error {
	if err := validation.ValidateStringFlagNotEmpty(args.AccessKeyID, AccessKeyIDFlag); err != nil {
		return err
	}
	if err := validation.ValidateStringFlagNotEmpty(args.SecretAccessKey, SecretAccessKeyFlag); err != nil {
		return err
	}
	if err := validation.ValidateStringFlagNotEmpty(args.Region, RegionFlag); err != nil {
		return err
	}

	if args.ResumeExportedAMI {
		if args.ExportedAMIPath == "" {
			return fmt.Errorf("To resume exported AMI, flag -%v must be provided", ExportedAMIPathFlag)
		}
	} else {
		if args.ImageID == "" || args.ExportLocation == "" {
			return fmt.Errorf("To export AMI, flags -%v and -%v must be provided", ImageIDFlag, ExportLocationFlag)
		}
	}

	return nil
}

// getS3PathElements gets bucket name, and folder or object key depending on if
// AMI has been exported, for a valid object path. Error is returned otherwise.
func (args *AWSImportArguments) getS3PathElements() error {
	var err error

	// AMI already exported, should provide object path
	if args.ResumeExportedAMI {
		args.ExportBucket, args.ExportKey, err = splitS3Path(args.ExportedAMIPath)
		if err != nil {
			return err
		}
		if args.ExportBucket == "" || args.ExportKey == "" {
			return fmt.Errorf("%q is not a valid S3 file path", args.ExportedAMIPath)
		}
		// Not exported, should provide export location
	} else {
		args.ExportBucket, args.ExportFolder, err = splitS3Path(args.ExportLocation)
		if err != nil {
			return err
		}
		if args.ExportBucket == "" {
			return fmt.Errorf("%q is not a valid S3 path", args.ExportLocation)
		}
		if args.ExportFolder != "" && !strings.HasSuffix(args.ExportFolder, "/") {
			args.ExportFolder += "/"
		}
	}
	return nil
}

// splitS3Path splits S3 path into bucket and object path portions
func splitS3Path(path string) (string, string, error) {
	matches := s3PathRegex.FindStringSubmatch(path)
	if matches != nil {
		return matches[1], strings.TrimLeft(matches[2], "/"), nil
	}
	return "", "", fmt.Errorf("%q is not a valid AWS S3 path", path)
}
