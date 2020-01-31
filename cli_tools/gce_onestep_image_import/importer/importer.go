//  Copyright 2019 Google Inc. All Rights Reserved.
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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	imageImporter "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/importer"
	"golang.org/x/exp/rand"
)

// Make file paths mutable
var (
	WorkflowDir                = "daisy_workflows/image_import/"
	ImportWorkflow             = "import_image.wf.json"
	ImportFromImageWorkflow    = "import_from_image.wf.json"
	ImportAndTranslateWorkflow = "import_and_translate.wf.json"
)

// Parameter key shared with other packages
const (
	ImageNameFlagKey = "image_name"
	ClientIDFlagKey  = "client_id"
)

const (
	logPrefix = "[import-image]"
)

func validateAndParseFlags(clientID string, imageName string, sourceFile string, sourceImage string, dataDisk bool, osID string, customTranWorkflow string, labels string) (
		string, string, map[string]string, error) {

	// call original validate, then do own validation
	return "", "", nil, nil
}

func runImport(zone string,
		timeout string, project string, scratchBucketGcsPath string, oauth string, ce string,
		gcsLogsDisabled bool, cloudLogsDisabled bool, stdoutLogsDisabled bool, kmsKey string,
		kmsKeyring string, kmsLocation string, kmsProject string, noExternalIP bool,
		userLabels map[string]string, storageLocation string, uefiCompatible bool) (*daisy.Workflow, error) {

	// call original importer
	return nil, nil
}

// Run runs import workflow.
func Run(clientID string, imageName string, dataDisk bool, osID string, customTranWorkflow string,
		noGuestEnvironment bool, family string, description string,
		network string, subnet string, zone string, timeout string, project *string,
		scratchBucketGcsPath string, oauth string, ce string, gcsLogsDisabled bool, cloudLogsDisabled bool,
		stdoutLogsDisabled bool, kmsKey string, kmsKeyring string, kmsLocation string, kmsProject string,
		noExternalIP bool, labels string, currentExecutablePath string, storageLocation string,
		uefiCompatible bool, awsImageId string, awsExportBucket string, awsExportFolder string,
		awsAccessKeyId string, awsSecrectAccessKey string, awsRegion string) (*daisy.Workflow, error) {

	log.SetPrefix(logPrefix + " ")

	tmpFilePath := fmt.Sprintf("gs://tzz-noogler-3-daisy-bkt/onestep-import/tmp-%v/tmp-%v.vmdk", rand.Int() % 1000000)

	// 0. aws2 configure
	err := configure(awsAccessKeyId, awsSecrectAccessKey, awsRegion)
	if err != nil {
		return nil, err
	}

	// 1. export: aws2 ec2 export-image --image-id ami-0bdc89ef2ef39dd0a --disk-image-format VMDK --s3-export-location S3Bucket=dntczdx,S3Prefix=exports/
	//runCliTool("./gce_onestep_image_import", []string{""})
	exportedFilePath, err := exportAwsImage(awsImageId, awsExportBucket, awsExportFolder)
	if err != nil {
		return nil, err
	}

	// 2. copy: gsutil cp s3://dntczdx/exports/export-ami-0b768c1d619f93184.vmdk gs://tzz-noogler-3-daisy-bkt/amazon1.vmdk
	if err := copyToGcs(exportedFilePath, tmpFilePath); err != nil {
		return nil, err
	}

	// 3. call image importer
	log.Println("Starting image import to GCE...")
	return imageImporter.Run(clientID, imageName, dataDisk, osID, customTranWorkflow, tmpFilePath,
		"", noGuestEnvironment, family, description, network, subnet, zone, timeout,
		project, scratchBucketGcsPath, oauth, ce, gcsLogsDisabled, cloudLogsDisabled,
		stdoutLogsDisabled, kmsKey, kmsKeyring, kmsLocation, kmsProject, noExternalIP,
		labels, currentExecutablePath, storageLocation, uefiCompatible)
}

func configure(awsAccessKeyId string, awsSecrectAccessKey string, awsRegion string) error {
	if err := runCmd("aws2", []string{"configure", "set", "aws_access_key_id", awsAccessKeyId}); err != nil {
		return err
	}
	if err := runCmd("aws2", []string{"configure", "set", "aws_secret_access_key", awsSecrectAccessKey}); err != nil {
		return err
	}
	if err := runCmd("aws2", []string{"configure", "set", "region", awsRegion}); err != nil {
		return err
	}
	if err := runCmd("aws2", []string{"configure", "set", "output", "json"}); err != nil {
		return err
	}
	return nil
}

type AwsTaskResponse struct {
	ExportImageTasks []AwsExportImageTasks `json:omitempty`
}

type AwsExportImageTasks struct {
	Status string `json:omitempty`
	StatusMessage string `json:omitempty`
	Progress string `json:omitempty`
}

func exportAwsImage(awsImageId string, awsExportBucket string, awsExportFolder string) (string, error) {
	output, err := runCmdAndGetOutput("aws2", []string{"ec2", "export-image", "--image-id", awsImageId, "--disk-image-format", "VMDK", "--s3-export-location",
		fmt.Sprintf("S3Bucket=%v,S3Prefix=%v/", awsExportBucket, awsExportFolder)})
	if err != nil {
		return "", err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(output, &resp); err != nil {
		return "", err
	}
	tid := resp["ExportImageTaskId"]
	if tid == "" {
		return "", fmt.Errorf("Empty task id returned.")
	}
	log.Println("export image task id: ", tid)

	//repeat query
	for {
		// aws2 ec2 describe-export-image-tasks --export-image-task-ids export-ami-0f7900141ff1f3caa
		output, err = runCmdAndGetOutputWithoutLog("aws2", []string{"ec2", "describe-export-image-tasks", "--export-image-task-ids", fmt.Sprintf("%v", tid)})
		if err != nil {
			return "", err
		}
		var taskResp AwsTaskResponse
		if err := json.Unmarshal(output, &taskResp); err != nil {
			return "", err
		}
		if len(taskResp.ExportImageTasks) != 1 {
			return "", fmt.Errorf("Unexpected response of describe-export-image-tasks.")
		}
		log.Println(fmt.Sprintf("AWS export task status: %v, status message: %v, progress: %v", taskResp.ExportImageTasks[0].Status, taskResp.ExportImageTasks[0].StatusMessage, taskResp.ExportImageTasks[0].Progress))

		if taskResp.ExportImageTasks[0].Status !=  "active" {
			if taskResp.ExportImageTasks[0].Status != "completed" {
				return "", fmt.Errorf("AWS export task wasn't completed successfully.")
			}
			break
		}
		time.Sleep(time.Millisecond * 3000)
	}
	log.Println("AWS export task is completed!")

	return fmt.Sprintf("s3://%v/%v/%v.vmdk", awsExportBucket, awsExportFolder, tid), nil
}


func copyToGcs(awsFilePath string, gcsFilePath string) error {
	log.Println("Copying from ec2 to cache...")
	// aws2 s3 cp s3://dntczdx/exports/export-ami-0b768c1d619f93184.vmdk tmp
	err := runCmd("aws2", []string{"s3", "cp", awsFilePath, "tmp"})
	if err != nil {
		return err
	}
	// gsutil cp tmp gs://tzz-noogler-3-daisy-bkt/amazon1.vmdk
	log.Println("Copying from cache to gcs...")
	err = runCmd("gsutil", []string{"cp", "tmp", gcsFilePath})
	if err != nil {
		return err
	}
	log.Println("Copied.")

	return nil
}

func runCmd(cmdString string, args []string) error {
	log.Printf("Running command: '%s %s'", cmdString, strings.Join(args, " "))
	cmd := exec.Command(cmdString, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCmdAndGetOutput(cmdString string, args []string) ([]byte, error) {
	log.Printf("Running command: '%s %s'", cmdString, strings.Join(args, " "))
	return runCmdAndGetOutputWithoutLog(cmdString, args)
}

func runCmdAndGetOutputWithoutLog(cmdString string, args []string) ([]byte, error) {
	output, err := exec.Command(cmdString, args...).Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}