package onestepimporttestsuites

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

const (
	awsRegion = "us-east-2"
	awsBucket = "s3://fionaliu-init"

	ubuntuAMIID         = "ami-0b0361fee657fe4ea"
	windowsAMIID        = "ami-03599982c8775acc9"
	ubuntuVMDKFilePath  = "s3://fionaliu-init/ubuntu1804.vmdk"
	windowsVMDKFilePath = "s3://fionaliu-init/export-ami-0ac5eaad663347858.vmdk"
)

var (
	awsAccessKeyID, awsSecretAccessKey, awsSessionToken string
)

type onestepImportAWSTestProperties struct {
	imageName         string
	amiID             string
	amiExportLocation string
	sourceAMIFilePath string
	os                string
	timeout           string
	startupScript     string
}

// setAWSAuth sets "aws auth"
func setAWSAuth(logger *log.Logger, testCase *junitxml.TestCase) error {
	credsPath := "gs://compute-image-test-pool-001-daisy-bkt-us-east1/aws_cred"
	cmd := "gsutil"
	args := []string{"cp", credsPath, "."}
	if err := utils.RunCliTool(logger, testCase, cmd, args); err != nil {
		utils.Failure(testCase, logger, fmt.Sprintf("Error running cmd: %v\n", err))
		return err
	}
	return getAWSTemporaryCredentials()
}

func getAWSTemporaryCredentials() error {
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", "aws_cred")
	mySession := session.Must(session.NewSession())
	svc := sts.New(mySession)
	sessionDuration := int64((time.Hour * 3).Seconds())
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
