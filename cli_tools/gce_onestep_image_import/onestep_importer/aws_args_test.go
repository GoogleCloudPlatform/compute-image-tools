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
	"testing"

	"github.com/stretchr/testify/assert"
)

const exportFlagErrorMsg = "specify -aws_source_ami_file_path to import from exported image file, or both -aws_ami_id and -aws_ami_export_location to import from AMI"

func TestSplitS3PathObjectInFolder(t *testing.T) {
	bucket, object, err := splitS3Path("s3://bucket_name/folder_name/object_name")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "folder_name/object_name", object)
}

func TestSplitS3PathObjectDirectlyInBucket(t *testing.T) {
	bucket, object, err := splitS3Path("s3://bucket_name/object_name")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "object_name", object)
}

func TestSplitS3PathBucketOnlyTrailingSlash(t *testing.T) {
	bucket, object, err := splitS3Path("s3://bucket_name/")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "", object)
}

func TestSplitS3PathBucketOnlyNoTrailingSlash(t *testing.T) {
	bucket, object, err := splitS3Path("s3://bucket_name")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "", object)
}

func TestSplitS3PathObjectNameNonLetters(t *testing.T) {
	bucket, object, err := splitS3Path("s3://bucket_name/|||")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "|||", object)
}

func TestSplitS3PathErrorOnNoBucket(t *testing.T) {
	_, _, err := splitS3Path("s3://")
	assert.NotNil(t, err)
}

func TestSplitS3PathErrorOnNoBucketButObjectPath(t *testing.T) {
	_, _, err := splitS3Path("s3:///object_name")
	assert.NotNil(t, err)
}

func TestSplitS3PathErrorOnInvalidPath(t *testing.T) {
	_, _, err := splitS3Path("NOT_A_S3_PATH")
	assert.NotNil(t, err)
}

func TestGenerateS3PathElementsErrorWhenExportedAMIPathInvalid(t *testing.T) {
	args := setUpAWSArgs(awsSourceAMIFilePathFlag, false, "-aws_source_ami_file_path=s3://")
	awsArgs := getAWSImportArgs(args)
	assert.Error(t, awsArgs.generateS3PathElements())
}

func TestGenerateS3PathElementsErrorWhenExportedAMIPathEmptyKay(t *testing.T) {
	args := setUpAWSArgs(awsSourceAMIFilePathFlag, false, "-aws_source_ami_file_path=s3://bucket")
	awsArgs := getAWSImportArgs(args)
	assert.Error(t, awsArgs.generateS3PathElements())
}

func TestGenerateS3PathElementsErrorWhenExportLocationInvalid(t *testing.T) {
	args := setUpAWSArgs(awsAMIExportLocationFlag, true, "-aws_ami_export_location=s3://")
	awsArgs := getAWSImportArgs(args)
	assert.Error(t, awsArgs.generateS3PathElements())
}

func TestGenerateS3PathElementsAppendSlashOnExportFolder(t *testing.T) {
	args := setUpAWSArgs(awsAMIExportLocationFlag, true, "-aws_ami_export_location=s3://bucket/folder")
	awsArgs := getAWSImportArgs(args)
	assert.NoError(t, awsArgs.generateS3PathElements())
	assert.Equal(t, awsArgs.exportFolder, "folder/")
}

func TestGenerateS3PathElementsDoesNotAppendExtraSlashOnExportFolder(t *testing.T) {
	args := setUpAWSArgs(awsAMIExportLocationFlag, true, "-aws_ami_export_location=s3://bucket/folder/")
	awsArgs := getAWSImportArgs(args)
	assert.NoError(t, awsArgs.generateS3PathElements())
	assert.Equal(t, awsArgs.exportFolder, "folder/")
}

func TestValidateExportAMI(t *testing.T) {
	args := setUpAWSArgs("", true)
	importerArgs, err := NewOneStepImportArguments(args)
	awsArgs := newAWSImportArguments(importerArgs)
	err = awsArgs.validate()
	assert.Nil(t, err)
}

func TestValidateResumeExportedAMI(t *testing.T) {
	input := setUpAWSArgs("", false)
	awsArgs := getAWSImportArgs(input)
	err := awsArgs.validate()
	assert.Nil(t, err)
}

func TestValidateAndPopulateErrorWhenValidateFailed(t *testing.T) {
	args := setUpAWSArgs(awsRegionFlag, false)
	awsArgs := getAWSImportArgs(args)
	err := awsArgs.validateAndPopulate(mockPopulator{})
	assert.EqualError(t, err, "The flag -aws_region must be provided")
}

func TestValidateAndPopulateErrorWhenPopulateFailed(t *testing.T) {
	args := setUpAWSArgs("", false)
	awsArgs := getAWSImportArgs(args)

	errMsg := "populate failed"
	err := awsArgs.validateAndPopulate(mockPopulator{
		err: fmt.Errorf(errMsg),
	})
	assert.EqualError(t, err, errMsg)
}

func TestPopulateParam(t *testing.T) {
	args := setUpAWSArgs("", false)
	awsArgs := getAWSImportArgs(args)
	err := awsArgs.validateAndPopulate(mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	})
	assert.NoError(t, err)
}

func TestIsExportedAMIExported(t *testing.T) {
	args := setUpAWSArgs("", false)
	awsArgs := getAWSImportArgs(args)
	assert.False(t, awsArgs.isExportRequired())
}

func TestIsExportedAMINotExported(t *testing.T) {
	args := setUpAWSArgs("", true)
	awsArgs := getAWSImportArgs(args)
	assert.True(t, awsArgs.isExportRequired())
}

func TestFailWhenAccessKeyIDNotProvided(t *testing.T) {
	args := setUpAWSArgs(awsAccessKeyIDFlag, true)
	assert.EqualError(t, expectFailedAWSValidation(t, args), "The flag -aws_access_key_id must be provided")
}

func TestFailWhenSecretAccessKeyNotProvided(t *testing.T) {
	args := setUpAWSArgs(awsSecretAccessKeyFlag, true)
	assert.EqualError(t, expectFailedAWSValidation(t, args), "The flag -aws_secret_access_key must be provided")
}

func TestFailWhenRegionNotProvided(t *testing.T) {
	args := setUpAWSArgs(awsRegionFlag, true)
	assert.EqualError(t, expectFailedAWSValidation(t, args), "The flag -aws_region must be provided")
}

func TestFailWhenSessionTokenNotProvided(t *testing.T) {
	args := setUpAWSArgs(awsSessionTokenFlag, true)
	assert.EqualError(t, expectFailedAWSValidation(t, args), "The flag -aws_session_token must be provided")
}

func TestFailWhenOnlyAMIIDNotProvided(t *testing.T) {
	args := setUpAWSArgs(awsAMIIDFlag, true)
	assert.EqualError(t, expectFailedAWSValidation(t, args), exportFlagErrorMsg)
}

func TestFailWhenOnlyExportLocationNotProvided(t *testing.T) {
	args := setUpAWSArgs(awsAMIExportLocationFlag, true)
	assert.EqualError(t, expectFailedAWSValidation(t, args), exportFlagErrorMsg)
}

func TestFailWhenAllExportFlagsProvided(t *testing.T) {
	args := setUpAWSArgs("", true, "-aws_source_ami_file_path=s3://bucket/object")
	assert.EqualError(t, expectFailedAWSValidation(t, args), exportFlagErrorMsg)
}

func expectFailedAWSValidation(t *testing.T, args []string) error {
	importArgs, err := NewOneStepImportArguments(args)
	assert.NoError(t, err)

	awsImportArgs := newAWSImportArguments(importArgs)
	err = awsImportArgs.validate()
	assert.Error(t, err)
	return err
}
