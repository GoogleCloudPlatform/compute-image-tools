package aws_importer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
)

// AWSImportArguments holds the structured results of parsing CLI arguments,
// and optionally allows for validating and populating the arguments.
type AwsImportArguments struct {
	AccessKeyId      string
	SecretAccessKey  string
	SessionToken			string
	Region           string
	ImageId		      string
	ExportLocation		string
	ExportedAMIPath  string
	ResumeExportedAMI   bool

	ExportBucket     string
	ExportPrefix     string
	ExportKey				 string
	ExportFileSize   int64
}

const (
	// TODO: add comment for flag key
	ImageIdFlagKey = "aws-image-id"
	ExportLocationFlagKey = "aws-export-location"
	AccessKeyIdFlagKey = "aws-access-key-id"
	SecretAccessKeyFlagKey = "aws-secret-access-key"
	SessionTokenFlagKey = "aws-session-token"
	RegionFlagKey = "aws-region"
	ExportedAMIPathFlagKey = "aws-exported-ami-path"
	ResumeExportedAMIFlagKey = "resume-exported-ami"
)

var (
	bucketNameRegex   = `[a-z0-9][-_.a-z0-9]*`
	s3PathRegex       = regexp.MustCompile(fmt.Sprintf(`^s3://(%s)(\/.*)?$`, bucketNameRegex))
)

func Parse(imageId, exportLocation, accessKeyId, secrectAccessKey, sessionToken, region, exportedAMIPath string, resumeExportedAMI bool) (*AwsImportArguments, error){
	args := &AwsImportArguments{accessKeyId, secrectAccessKey, sessionToken, region, imageId, exportLocation, exportedAMIPath, resumeExportedAMI,"", "", "", 0}
	err := args.validate()
	if err != nil {
		return nil, err
	}

	err = args.getMetadata()
	if err != nil {
		return nil, err
	}
	return args, nil
}

func(args *AwsImportArguments) validate() error {
	if err := validation.ValidateStringFlagNotEmpty(args.AccessKeyId, AccessKeyIdFlagKey); err != nil {
		return err
	}
	if err := validation.ValidateStringFlagNotEmpty(args.SecretAccessKey, SecretAccessKeyFlagKey); err != nil {
		return err
	}
	if err := validation.ValidateStringFlagNotEmpty(args.Region, RegionFlagKey); err != nil {
		return err
	}

	if args.ResumeExportedAMI {
		if args.ExportedAMIPath == "" {
			return fmt.Errorf("To resume exported AMI, flag -%v must be provided", ExportedAMIPathFlagKey)
		}
	} else {
		if args.ImageId == "" || args.ExportLocation == ""{
			return fmt.Errorf("To export AMI, flags -%v and -%v must be provided", ImageIdFlagKey, ExportLocationFlagKey)
		}
	}

	return nil
}

func (args *AwsImportArguments)getMetadata() error {
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