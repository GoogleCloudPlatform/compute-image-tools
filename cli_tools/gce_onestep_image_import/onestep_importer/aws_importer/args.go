package aws_importer

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
	AccessKeyId      string
	ExportLocation		string
	ExportedAMIPath  string
	GcsComputeEndpoint  string
	GcsProject          string
	GcsZone             string
	GcsRegion           string
	GcsScratchBucket    string
	GcsStorageLocation 	string
	ImageId		      string
	Region           string
	ResumeExportedAMI   bool
	SecretAccessKey  string
	SessionToken			string

	ExportBucket     string
	ExportPrefix     string
	ExportKey				 string
	ExportFileSize   int64
}

const (
	// TODO: add comment for flag key
	ImageIdFlag = "aws_image_id"
	ExportLocationFlag = "aws_export_location"
	AccessKeyIdFlag = "aws_access_key_id"
	SecretAccessKeyFlag = "aws_secret_access_key"
	SessionTokenFlag = "aws_session_token"
	RegionFlag = "aws_region"
	ExportedAMIPathFlag = "aws_exported_ami_path"
	ResumeExportedAMIFlag = "resume_exported_ami"
)

var (
	bucketNameRegex   = `[a-z0-9][-_.a-z0-9]*`
	s3PathRegex       = regexp.MustCompile(fmt.Sprintf(`^s3://(%s)(\/.*)?$`, bucketNameRegex))
)

func(args *AWSImportArguments)ValidateAndPopulate(populator param.Populator) error{
	err := args.validate()
	if err != nil {
		return err
	}

	err = populator.PopulateMissingParameters(&args.GcsProject, &args.GcsZone, &args.GcsRegion,
		&args.GcsScratchBucket, "", &args.GcsStorageLocation);
	if err != nil {
		return err
	}

	return args.getMetadata()
}

func(args *AWSImportArguments) validate() error {
	if err := validation.ValidateStringFlagNotEmpty(args.AccessKeyId, AccessKeyIdFlag); err != nil {
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
		if args.ImageId == "" || args.ExportLocation == ""{
			return fmt.Errorf("To export AMI, flags -%v and -%v must be provided", ImageIdFlag, ExportLocationFlag)
		}
	}

	return nil
}

func (args *AWSImportArguments)getMetadata() error {
	var err error
	if args.ResumeExportedAMI {
		args.ExportBucket, args.ExportKey, err = splitS3Path(args.ExportedAMIPath)
		if err != nil {
			return err
		}
		if args.ExportKey == "" {
			return fmt.Errorf("%q is not a valid S3 file path", args.ExportedAMIPath)
		}
	} else {
		args.ExportBucket, args.ExportPrefix, err = splitS3Path(args.ExportLocation)
		if err != nil {
			return err
		}
		if args.ExportPrefix != "" && !strings.HasSuffix(args.ExportPrefix, "/") {
			args.ExportPrefix += "/"
		}
	}
	return nil
}

func splitS3Path(path string) (string, string, error){
	matches := s3PathRegex.FindStringSubmatch(path)
	if matches != nil {
		return matches[1], strings.TrimLeft(matches[2], "/"), nil
	}
	return "", "", fmt.Errorf("%q is not a valid AWS S3 path", path)
}

