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

// Package onestepimporttestsuites contains e2e tests for gce_onestep_image_import
package onestepimporttestsuites

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
)

const (
	awsCredFlag     = "aws_cred_file_path"
	awsRegionFlag   = "aws_region"
	awsBucketFlag   = "aws_bucket"
	ubuntuAMIFlag   = "ubuntu_ami_id"
	windowsAMIFlag  = "windows_ami_id"
	ubuntuVMDKFlag  = "ubuntu_vmdk"
	windowsVMDKFlag = "windows_vmdk"
)

var (
	awsCredFilePath, awsAccessKeyID, awsSecretAccessKey, awsSessionToken, awsRegion, awsBucket,
	ubuntuAMIID, windowsAMIID, ubuntuVMDKFilePath, windowsVMDKFilePath string
)

type onestepImportAWSTestProperties struct {
	imageName             string
	amiID                 string
	sourceAMIFilePath     string
	os                    string
	timeout               string
	startupScript         string
	skipOSConfig          string
	computeServiceAccount string
}

// setAWSAuth downloads AWS credentials and sets access keys.
func setAWSAuth(logger *log.Logger, testCase *junitxml.TestCase) error {
	cmd := "gsutil"
	args := []string{"cp", awsCredFilePath, "."}
	if err := e2e.RunCliTool(logger, testCase, cmd, args); err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Error running cmd: %v\n", err))
		return err
	}
	return getAWSTemporaryCredentials()
}

// getAWSTemporaryCredentials calls AWS API to get temporary access keys.
func getAWSTemporaryCredentials() error {
	_, credFileName := path.Split(awsCredFilePath)
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credFileName)
	mySession := session.Must(session.NewSession())
	svc := sts.New(mySession)
	sessionDuration := int64((time.Hour * 6).Seconds())
	output, err := svc.GetSessionToken(&sts.GetSessionTokenInput{DurationSeconds: aws.Int64(sessionDuration)})
	if err != nil {
		return err
	}

	if output.Credentials == nil {
		return daisy.Errf("empty credentials")
	}

	awsAccessKeyID = aws.StringValue(output.Credentials.AccessKeyId)
	awsSecretAccessKey = aws.StringValue(output.Credentials.SecretAccessKey)
	awsSessionToken = aws.StringValue(output.Credentials.SessionToken)
	return nil
}

// getAWSTestArgs assigns aws test variables from input variable map.
func getAWSTestArgs() bool {
	for key, val := range argMap {
		switch key {
		case awsCredFlag:
			awsCredFilePath = val
		case awsRegionFlag:
			awsRegion = val
		case awsBucketFlag:
			awsBucket = val
		case ubuntuAMIFlag:
			ubuntuAMIID = val
		case windowsAMIFlag:
			windowsAMIID = val
		case ubuntuVMDKFlag:
			ubuntuVMDKFilePath = val
		case windowsVMDKFlag:
			windowsVMDKFilePath = val
		default:
			// args not related to onestep import aws tests
		}
	}

	if awsCredFilePath == "" || awsRegion == "" || awsBucket == "" ||
		ubuntuAMIID == "" || windowsAMIID == "" || ubuntuVMDKFilePath == "" ||
		windowsVMDKFilePath == "" {
		return false
	}

	return true
}
