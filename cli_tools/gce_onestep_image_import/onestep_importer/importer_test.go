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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFailWhenImageNameNotProvided(t *testing.T) {
	args := setUpArgs(imageNameFlag)
	assert.EqualError(t, expectFailedValidation(t, args), "The flag -image_name must be provided")
}

func TestTrimAndLowerImageName(t *testing.T) {
	args := setUpArgs(imageNameFlag, "-image_name=   IMAGEname   ")
	assert.Equal(t, "imagename", expectSuccessfulParse(t, args...).ImageName)
}

func TestFailWhenClientIDNotProvided(t *testing.T) {
	args := setUpArgs(clientFlag)
	assert.EqualError(t, expectFailedValidation(t, args), "The flag -client_id must be provided")
}

func TestTrimAndLowerClientID(t *testing.T) {
	args := setUpArgs(clientFlag, "-client_id=      GCloud    ")
	assert.Equal(t, "gcloud", expectSuccessfulParse(t, args...).ClientID)
}

func TestFailWhenOSNotProvided(t *testing.T) {
	args := setUpArgs(osFlag)
	assert.EqualError(t, expectFailedValidation(t, args), "The flag -os must be provided")
}

func TestTrimAndLowerOS(t *testing.T) {
	args := setUpArgs(osFlag, "-os=    UbUntu-1804    ")
	assert.Equal(t, "ubuntu-1804", expectSuccessfulParse(t, args...).OS)
}

func TestOSNotRegistered(t *testing.T) {
	args := setUpArgs(osFlag, "-os=android")
	assert.Contains(t, expectFailedValidation(t, args).Error(),
		"os `android` is invalid. Allowed values:")
}

func TestTrimAccessKeyID(t *testing.T) {
	assert.Equal(t, "my-access-key-id", expectSuccessfulParse(t, "-aws_access_key_id=   my-access-key-id   ").AWSAccessKeyID)
}

func TestTrimSecretAccessKey(t *testing.T) {
	assert.Equal(t, "my-secret_access-key", expectSuccessfulParse(t, "-aws_secret_access_key=   my-secret_access-key   ").AWSSecretAccessKey)
}

func TestTrimRegion(t *testing.T) {
	assert.Equal(t, "my-region", expectSuccessfulParse(t, "-aws_region=   my-region   ").AWSRegion)
}

func TestTrimSessionToken(t *testing.T) {
	assert.Equal(t, "my-token", expectSuccessfulParse(t, "-aws_session_token=   my-token  ").AWSSessionToken)
}

func TestTrimAMIID(t *testing.T) {
	assert.Equal(t, "my-ami-id", expectSuccessfulParse(t, "-aws_ami_id=   my-ami-id  ").AWSAMIID)
}

func TestTrimSourceAMIFilePath(t *testing.T) {
	assert.Equal(t, "my-source-file-path", expectSuccessfulParse(t, "-aws_source_ami_file_path=   my-source-file-path  ").AWSSourceAMIFilePath)
}

func TestTrimAMIExportLocation(t *testing.T) {
	args := setUpAWSArgs(awsAMIExportLocationFlag, true, "-aws_ami_export_location=   my-ami_export-location  ")
	assert.Equal(t, "my-ami_export-location", expectSuccessfulParse(t, args...).AWSAMIExportLocation)
}

func TestTrimFamily(t *testing.T) {
	assert.Equal(t, "ubuntu", expectSuccessfulParse(t, "-family=  ubuntu  ").Family)
}

func TestTrimDescription(t *testing.T) {
	assert.Equal(t, "ubuntu", expectSuccessfulParse(t, "-description=  ubuntu  ").Description)
}

func TestTrimAndLowerStorageLocation(t *testing.T) {
	assert.Equal(t, "ubuntu", expectSuccessfulParse(t, "-storage_location=  ubUntu  ").StorageLocation)
}

func TestTrimProject(t *testing.T) {
	assert.Equal(t, "ubuntu", *expectSuccessfulParse(t, "-project=  ubuntu  ").ProjectPtr)
}

func TestTrimNetwork(t *testing.T) {
	assert.Equal(t, "id", expectSuccessfulParse(t, "-network=   id  ").Network)
}

func TestTrimSubnet(t *testing.T) {
	assert.Equal(t, "sub-id", expectSuccessfulParse(t, "-subnet=   sub-id  ").Subnet)
}

func TestTrimAndLowerZone(t *testing.T) {
	assert.Equal(t, "us-central4-a", expectSuccessfulParse(t, "-zone=   us-centrAl4-a  ").Zone)
}

func TestTrimScratchBucket(t *testing.T) {
	assert.Equal(t, "gcs://bucket", expectSuccessfulParse(t, "-scratch_bucket_gcs_path=   gcs://bucket  ").ScratchBucketGcsPath)
}

func TestTrimOauth(t *testing.T) {
	assert.Equal(t, "file.json", expectSuccessfulParse(t, "-oauth=   file.json  ").Oauth)
}

func TestTrimComputeEndpoint(t *testing.T) {
	assert.Equal(t, "http://endpoint",
		expectSuccessfulParse(t, "-compute_endpoint_override=   http://endpoint  ").ComputeEndpoint)
}

func TestTrimCustomWorkflow(t *testing.T) {
	assert.Equal(t, "workflow.json", expectSuccessfulParse(t, "-custom_translate_workflow=  workflow.json  ").CustomWorkflow)
}

func TestParseLabelsToMap(t *testing.T) {
	expected := map[string]string{"internal": "true", "private": "false"}
	assert.Equal(t, expected, expectSuccessfulParse(t, "-labels=internal=true,private=false").Labels)
}

func TestFailWhenLabelSyntaxError(t *testing.T) {
	args := setUpArgs("", "-labels=internal:true")
	_, err := NewOneStepImportArguments(args)
	assert.NotNil(t, err)
	assert.Errorf(t, err,
		"invalid value \"internal:true\" for flag -labels")
}

func TestGcsLogsDisabled(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-disable_gcs_logging=false").GcsLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_gcs_logging=true").GcsLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_gcs_logging").GcsLogsDisabled)
}

func TestCloudLogsDisabled(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-disable_cloud_logging=false").CloudLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_cloud_logging=true").CloudLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_cloud_logging").CloudLogsDisabled)
}

func TestStdoutLogsDisabled(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-disable_stdout_logging=false").StdoutLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_stdout_logging=true").StdoutLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_stdout_logging").StdoutLogsDisabled)
}

func TestNoExternalIp(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-no_external_ip=false").NoExternalIP)
	assert.True(t, expectSuccessfulParse(t, "-no_external_ip=true").NoExternalIP)
	assert.True(t, expectSuccessfulParse(t, "-no_external_ip").NoExternalIP)
}

func TestDurationHasDefaultValue(t *testing.T) {
	assert.Equal(t, time.Hour*2, expectSuccessfulParse(t).Timeout)
}

func TestDurationIsSettable(t *testing.T) {
	assert.Equal(t, time.Hour*5, expectSuccessfulParse(t, "-timeout=5h").Timeout)
}

func TestUEFISettable(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-uefi_compatible=false").UefiCompatible)
	assert.True(t, expectSuccessfulParse(t, "-uefi_compatible=true").UefiCompatible)
	assert.True(t, expectSuccessfulParse(t, "-uefi_compatible").UefiCompatible)
}

func TestSysprepSettable(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-sysprep_windows=false").SysprepWindows)
	assert.True(t, expectSuccessfulParse(t, "-sysprep_windows=true").SysprepWindows)
	assert.True(t, expectSuccessfulParse(t, "-sysprep_windows").SysprepWindows)
}

func TestRunReturnErrorWhenInvalidArgs(t *testing.T) {
	args := setUpArgs(osFlag)
	importArgs, _ := NewOneStepImportArguments(args)

	_, err := Run(importArgs)
	assert.EqualError(t, err, "The flag -os must be provided")
}

func TestRunReturnErrorWhenImporterFail(t *testing.T) {
	args := setUpArgs("", "-timeout=0s")
	importArgs, _ := NewOneStepImportArguments(args)

	_, err := Run(importArgs)
	assert.EqualError(t, err, "timeout exceeded: timeout must be at least 3 minutes")
}

func expectFailedValidation(t *testing.T, args []string) error {
	importArgs, err := NewOneStepImportArguments(args)
	assert.NoError(t, err)

	err = importArgs.validate()
	assert.Error(t, err)
	return err
}

func setUpArgs(requiredFlagToTest string, args ...string) []string {
	var (
		appendClientID  = requiredFlagToTest != clientFlag
		appendImageName = requiredFlagToTest != imageNameFlag
		appendOS        = requiredFlagToTest != osFlag
	)

	for _, arg := range args {
		if strings.HasPrefix(arg, "-client_id") {
			appendClientID = false
		} else if strings.HasPrefix(arg, "-image_name") {
			appendImageName = false
		} else if strings.HasPrefix(arg, "-os") {
			appendOS = false
		}
	}

	if appendClientID {
		args = append(args, "-client_id=gcloud")
	}

	if appendImageName {
		args = append(args, "-image_name=name")
	}

	if appendOS {
		args = append(args, "-os=ubuntu-1804")
	}

	return args
}
