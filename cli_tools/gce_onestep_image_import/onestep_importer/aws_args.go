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
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// awsImportArguments holds the structured results of parsing CLI arguments,
// and optionally allows for validating and populating the arguments.
type awsImportArguments struct {
	// Passed in by user
	accessKeyID        string
	amiID              string
	executablePath     string
	exportLocation     string
	sourceFilePath     string
	gcsComputeEndpoint string
	gcsProjectPtr      *string
	gcsZone            string
	gcsRegion          string
	gcsScratchBucket   string
	gcsStorageLocation string
	region             string
	secretAccessKey    string
	sessionToken       string

	// Internal generated
	exportBucket   string
	exportFolder   string
	exportKey      string
	exportFileSize int64
}

// Flags
const (
	awsAMIIDFlag             = "aws_ami_id"
	awsAMIExportLocationFlag = "aws_ami_export_location"
	awsAccessKeyIDFlag       = "aws_access_key_id"
	awsSecretAccessKeyFlag   = "aws_secret_access_key"
	awsSessionTokenFlag      = "aws_session_token"
	awsRegionFlag            = "aws_region"
	awsSourceAMIFilePathFlag = "aws_source_ami_file_path"
)

var (
	bucketNameRegex = `[a-z0-9][-_.a-z0-9]*`
	s3PathRegex     = regexp.MustCompile(fmt.Sprintf(`^s3://(%s)(\/.*)?$`, bucketNameRegex))
)

// newAWSImportArguments creates a new AWSImportArgument instance.
func newAWSImportArguments(args *OneStepImportArguments) *awsImportArguments {
	return &awsImportArguments{
		accessKeyID:        args.AWSAccessKeyID,
		amiID:              args.AWSAMIID,
		executablePath:     args.ExecutablePath,
		exportLocation:     args.AWSAMIExportLocation,
		sourceFilePath:     args.AWSSourceAMIFilePath,
		gcsComputeEndpoint: args.ComputeEndpoint,
		gcsProjectPtr:      args.ProjectPtr,
		gcsZone:            args.Zone,
		gcsRegion:          args.Region,
		gcsScratchBucket:   args.ScratchBucketGcsPath,
		gcsStorageLocation: args.StorageLocation,
		region:             args.AWSRegion,
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

	err = populator.PopulateMissingParameters(args.gcsProjectPtr, &args.gcsZone, &args.gcsRegion,
		&args.gcsScratchBucket, "", &args.gcsStorageLocation)
	if err != nil {
		return err
	}

	return args.generateS3PathElements()
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
	if err := validation.ValidateStringFlagNotEmpty(args.sessionToken, awsSessionTokenFlag); err != nil {
		return err
	}

	needsExport := args.amiID != "" && args.exportLocation != "" && args.sourceFilePath == ""
	isResumeExported := args.amiID == "" && args.exportLocation == "" && args.sourceFilePath != ""

	if !(needsExport || isResumeExported) {
		return daisy.Errf("specify -%v to import from "+
			"exported image file, or both -%v and -%v to "+
			"import from AMI", awsSourceAMIFilePathFlag, awsAMIIDFlag, awsAMIExportLocationFlag)
	}

	return nil
}

// isExportRequired returns true if AMI needs to be exported, false otherwise.
func (args *awsImportArguments) isExportRequired() bool {
	return args.sourceFilePath == ""
}

// generateS3PathElements gets bucket name, and folder or object key depending on if
// AMI has been exported, for a valid object path. Error is returned otherwise.
func (args *awsImportArguments) generateS3PathElements() error {
	var err error

	if args.isExportRequired() {
		// Export required, get metadata from provided export location.
		args.exportBucket, args.exportFolder, err = splitS3Path(args.exportLocation)
		if err != nil {
			return err
		}

		if args.exportFolder != "" && !strings.HasSuffix(args.exportFolder, "/") {
			args.exportFolder += "/"
		}
	} else {
		// AMI already exported, get metadata from provide object path.
		args.exportBucket, args.exportKey, err = splitS3Path(args.sourceFilePath)
		if err != nil {
			return err
		}
		if args.exportKey == "" {
			return daisy.Errf("%v is not a valid S3 file path", args.sourceFilePath)
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
	return "", "", daisy.Errf("%v is not a valid AWS S3 path", path)
}
