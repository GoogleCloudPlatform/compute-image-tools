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

package aws_importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	imageImporter "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/importer"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dustin/go-humanize"
	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
)

// Make file paths mutable
var (
	WorkflowDir                = "daisy_workflows/image_import/"
	ImportWorkflow             = "import_image.wf.json"
	ImportFromImageWorkflow    = "import_from_image.wf.json"
	ImportAndTranslateWorkflow = "import_and_translate.wf.json"

	workers                  = runtime.NumCPU()
	gcsPermissionErrorRegExp = regexp.MustCompile(".*does not have storage.objects.create access to .*")

)

const (
	downloadBufSize = "100MB"
	downloadBufNum  = 3
	uploadBufSize   = "500MB"
	logPrefix       = "[aws-import-image]"
)

type AwsImporter struct {
	ctx 		context.Context
	oauth 	string
	client 	domain.StorageClientInterface
	args 		*AwsImportArguments
}

func NewImporter(oauth string, args *AwsImportArguments) (*AwsImporter, error) {
	ctx := context.Background()
	client, err := gcsClient(ctx, oauth)
	if err != nil {
		return nil, err
	}

	importer := &AwsImporter{
		ctx:ctx,
		oauth: oauth,
		client: client,
		args: args}

	return importer, nil
}

func (importer *AwsImporter) configure() error {
	args := importer.args
	if err := runCmd("aws", []string{"configure", "set", "aws_access_key_id", args.AccessKeyId}); err != nil {
		return err
	}
	if err := runCmd("aws", []string{"configure", "set", "aws_secret_access_key", args.SecretAccessKey}); err != nil {
		return err
	}
	if err := runCmd("aws", []string{"configure", "set", "region", args.Region}); err != nil {
		return err
	}
	if args.SessionToken != "" {
		if err := runCmd("aws", []string{"configure", "set", "session_token", args.SessionToken}); err != nil {
			return err
		}
	}
	if err := runCmd("aws", []string{"configure", "set", "output", "json"}); err != nil {
		return err
	}
	return nil
}

func gcsClient(ctx context.Context, oauth string) (domain.StorageClientInterface, error) {
	log.SetPrefix(logPrefix + " ")
	logger := logging.NewStdoutLogger(logPrefix)

	baseTransport := &http.Transport{
		DisableKeepAlives:     false,
		MaxIdleConns:          0,
		MaxIdleConnsPerHost:   1000,
		MaxConnsPerHost:       0,
		IdleConnTimeout:       60 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	transport, err := htransport.NewTransport(ctx, baseTransport)
	if err != nil {
		return nil, err
	}

	return storageutils.NewStorageClient(ctx, logger, option.WithHTTPClient(&http.Client{Transport: transport}),
		option.WithCredentialsFile(oauth))
}


// Run runs import workflow.
func (importer *AwsImporter) Run(imageImportArgs *imageImporter.ImportArguments) (string, error) {

	awsRand := string(rand.Int() % 1000000)
	tmpFilePath := fmt.Sprintf("gs://fionaliu-daisy-bkt-us-east1/onestep-import/tmp-%v/tmp-%v.vmdk", awsRand, awsRand)

	// 0. configure AWS SDK
	err := importer.configure()
	if err != nil {
		return "", err
	}

	fmt.Println("", importer.args.ResumeExportedAMI)

	if !importer.args.ResumeExportedAMI {
		// 1. export: aws ec2 export-image --image-id ami-0bdc89ef2ef39dd0a --disk-image-format VMDK --s3-export-location S3Bucket=dntczdx,S3Prefix=exports/
		//runCliTool("./gce_onestep_image_import", []string{""})
		err = importer.exportAwsImage()
		if err != nil {
			return "", err
		}
	}
	err = importer.getAwsFileSize()
	if err != nil {
		return "", err
	}
	// 2. copy: gsutil cp s3://dntczdx/exports/export-ami-0b768c1d619f93184.vmdk gs://tzz-noogler-3-daisy-bkt/amazon1.vmdk
	if err := importer.copyToGcs(tmpFilePath); err != nil {
		return "", err
	}

	// 3. call image importer
	log.Println("Copied to GCS %v", tmpFilePath)
	return tmpFilePath, nil
}

func (importer *AwsImporter) getAwsFileSize() error {
	output, err := runCmdAndGetOutput("aws", []string{"s3api", "head-object", "--bucket", fmt.Sprintf("%v", importer.args.ExportBucket), "--key", fmt.Sprintf("%v", importer.args.ExportKey)})
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(output, &resp); err != nil {
		return err
	}
	fileSize := int64(resp["ContentLength"].(float64))
	if fileSize == 0 {
		return fmt.Errorf("File is empty")
	}
	importer.args.ExportFileSize = fileSize
	return nil
}


type AwsTaskResponse struct {
	ExportImageTasks []AwsExportImageTasks `json:omitempty`
}

type AwsExportImageTasks struct {
	Status        string `json:omitempty`
	StatusMessage string `json:omitempty`
	Progress      string `json:omitempty`
}



func (importer *AwsImporter)exportAwsImage() error {
	output, err := runCmdAndGetOutput("aws", []string{"ec2", "export-image", "--image-id", importer.args.ImageId, "--disk-image-format", "VMDK", "--s3-export-location",
		fmt.Sprintf("S3Bucket=%v,S3Prefix=%v", importer.args.ExportBucket, importer.args.ExportPrefix)})
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(output, &resp); err != nil {
		return err
	}
	tid := resp["ExportImageTaskId"]
	if tid == "" {
		return fmt.Errorf("Empty task id returned.")
	}
	log.Println("export image task id: ", tid)

	//repeat query
	for {
		// aws ec2 describe-export-image-tasks --export-image-task-ids export-ami-0f7900141ff1f3caa
		output, err = runCmdAndGetOutputWithoutLog("aws", []string{"ec2", "describe-export-image-tasks", "--export-image-task-ids", fmt.Sprintf("%v", tid)})
		if err != nil {
			return err
		}
		var taskResp AwsTaskResponse
		if err := json.Unmarshal(output, &taskResp); err != nil {
			return err
		}
		if len(taskResp.ExportImageTasks) != 1 {
			return fmt.Errorf("Unexpected response of describe-export-image-tasks.")
		}
		log.Println(fmt.Sprintf("AWS export task status: %v, status message: %v, progress: %v", taskResp.ExportImageTasks[0].Status, taskResp.ExportImageTasks[0].StatusMessage, taskResp.ExportImageTasks[0].Progress))

		if taskResp.ExportImageTasks[0].Status != "active" {
			if taskResp.ExportImageTasks[0].Status != "completed" {
				return fmt.Errorf("AWS export task wasn't completed successfully.")
			}
			break
		}
		time.Sleep(time.Millisecond * 3000)
	}
	log.Println("AWS export task is completed!")

	importer.args.ExportKey = importer.args.ExportPrefix + tid.(string)+".vmdk"
	return nil
}

func (importer *AwsImporter)stream(writer *storageutils.BufferedWriter) (error) {
	log.Println("Downloading from s3 ...")
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	client := s3.New(sess)
	wg := new(sync.WaitGroup)
	sem := make(chan struct{}, downloadBufNum)
	var dmutex sync.Mutex
	readSize, err := humanize.ParseBytes(downloadBufSize)
	if err != nil {
		return err
	}
	readers := int(math.Ceil(float64(importer.args.ExportFileSize) / float64(readSize)))

	log.Println("readers:", readers)
	start := time.Now()
	for i := 0; i < readers; i++ {
		sem <- struct{}{}
		wg.Add(1)
		offset := i * int(readSize)
		readRange := strconv.Itoa(offset) + "-" + strconv.Itoa(offset+int(readSize)-1)

		res, err := client.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(importer.args.ExportBucket),
			Key:    aws.String(importer.args.ExportKey),
			Range:  aws.String("bytes=" + readRange),
		})

		if err != nil {
			log.Println(fmt.Sprintf("Error in downloading from bucket %v, key %v: %v \n", importer.args.ExportBucket, importer.args.ExportKey, err))
			return err
		}
		log.Println("downloaded")
		go func(writer *storageutils.BufferedWriter, res *s3.GetObjectOutput) {
			defer wg.Done()
			dmutex.Lock()
			defer dmutex.Unlock()
			defer res.Body.Close()
			io.Copy(writer, res.Body)
			<-sem
			log.Println("uploaded")
		}(writer, res)
	}

	wg.Wait()
	if err := writer.Close(); err != nil {
		return err
	}
	since := time.Since(start)
	log.Printf("Finished transferring in %s.", since)
	return nil
}

func (importer *AwsImporter) copyToGcs(gcsFilePath string) error {
	log.Println("Copying from ec2 to s3...")
	bs, err := humanize.ParseBytes(uploadBufSize)
	if err != nil {
		return err
	}

	ctx := context.Background()
	bkt, obj, err := storageutils.GetGCSObjectPathElements(gcsFilePath)
	if err != nil {
		log.Fatal(err)
	}
	writer := storageutils.NewBufferedWriter(ctx, int64(bs), int64(workers),gcsClient, importer.oauth, "/tmp", bkt, obj)
	err = importer.stream(writer)
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
