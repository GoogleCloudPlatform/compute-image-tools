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
	"fmt"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
)

// awsImportArguments holds the structured results of parsing CLI arguments,
// and optionally allows for validating and populating the arguments.
type awsImportArguments struct {
	// Passed in by user
	accessKeyID        string
	executablePath     string
	exportLocation     string
	exportedAMIPath    string
	gcsComputeEndpoint string
	gcsProject         string
	gcsZone            string
	gcsRegion          string
	gcsScratchBucket   string
	gcsStorageLocation string
	imageID            string
	region             string
	resumeExportedAMI  bool
	secretAccessKey    string
	sessionToken       string

	// Not passed in
	exportBucket   string
	exportFolder   string
	exportKey      string
	exportFileSize int64
}

// Flags
const (
	awsImageIDFlag           = "aws_image_id"
	awsExportLocationFlag    = "aws_export_location"
	awsAccessKeyIDFlag       = "aws_access_key_id"
	awsSecretAccessKeyFlag   = "aws_secret_access_key"
	awsSessionTokenFlag      = "aws_session_token"
	awsRegionFlag            = "aws_region"
	awsExportedAMIPathFlag   = "aws_exported_ami_path"
	awsResumeExportedAMIFlag = "resume_exported_ami"
)

var (
	bucketNameRegex = `[a-z0-9][-_.a-z0-9]*`
	s3PathRegex     = regexp.MustCompile(fmt.Sprintf(`^s3://(%s)(\/.*)?$`, bucketNameRegex))
)

// newAWSImportArguments creates a new AWSImportArgument instance.
func newAWSImportArguments(args *OneStepImportArguments) *awsImportArguments {
	return &awsImportArguments{
		accessKeyID:        args.AWSAccessKeyID,
		executablePath:     args.ExecutablePath,
		exportLocation:     args.AWSExportLocation,
		exportedAMIPath:    args.AWSExportedAMIPath,
		gcsComputeEndpoint: args.ComputeEndpoint,
		gcsProject:         args.Project,
		gcsZone:            args.Zone,
		gcsRegion:          args.Region,
		gcsScratchBucket:   args.ScratchBucketGcsPath,
		gcsStorageLocation: args.StorageLocation,
		imageID:            args.AWSImageID,
		region:             args.AWSRegion,
		resumeExportedAMI:  args.AWSResumeExportedAMI,
		secretAccessKey:    args.AWSSecretAccessKey,
		sessionToken:       args.AWSSessionToken,
	}
}

// ValidateAndPopulate validates args related to import from AWS, and populates
// any missing parameters.
func (args *awsImportArguments) validateAndPopulate(populator param.Populator) error {
	err := args.validate()
	if err != nil {
		return err
	}

	err = populator.PopulateMissingParameters(&args.gcsProject, &args.gcsZone, &args.gcsRegion,
		&args.gcsScratchBucket, "", &args.gcsStorageLocation)
	if err != nil {
		return err
	}

	return args.getS3PathElements()
}

func (args *awsImportArguments) validate() error {
	if err := validation.ValidateStringFlagNotEmpty(args.accessKeyID, awsAccessKeyIDFlag); err != nil {
		return err
	}
	if err := validation.ValidateStringFlagNotEmpty(args.secretAccessKey, awsSecretAccessKeyFlag); err != nil {
		return err
	}
	if err := validation.ValidateStringFlagNotEmpty(args.region, awsRegionFlag); err != nil {
		return err
	}

	if args.resumeExportedAMI {
		if args.exportedAMIPath == "" {
			return fmt.Errorf("To resume exported AMI, flag -%v must be provided", awsExportedAMIPathFlag)
		}
	} else {
		if args.imageID == "" || args.exportLocation == "" {
			return fmt.Errorf("To export AMI, flags -%v and -%v must be provided", awsImageIDFlag, awsExportLocationFlag)
		}
	}

	return nil
}

// getS3PathElements gets bucket name, and folder or object key depending on if
// AMI has been exported, for a valid object path. Error is returned otherwise.
func (args *awsImportArguments) getS3PathElements() error {
	var err error

	// AMI already exported, should provide object path
	if args.resumeExportedAMI {
		args.exportBucket, args.exportKey, err = splitS3Path(args.exportedAMIPath)
		if err != nil {
			return err
		}
		if args.exportBucket == "" || args.exportKey == "" {
			return fmt.Errorf("%q is not a valid S3 file path", args.exportedAMIPath)
		}
		// Not exported, should provide export location
	} else {
		args.exportBucket, args.exportFolder, err = splitS3Path(args.exportLocation)
		if err != nil {
			return err
		}
		if args.exportBucket == "" {
			return fmt.Errorf("%q is not a valid S3 path", args.exportLocation)
		}
		if args.exportFolder != "" && !strings.HasSuffix(args.exportFolder, "/") {
			args.exportFolder += "/"
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
