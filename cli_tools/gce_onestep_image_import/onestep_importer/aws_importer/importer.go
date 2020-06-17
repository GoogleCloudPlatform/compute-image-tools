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
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dustin/go-humanize"
	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
)

const (
	downloadBufSize = "100MB"
	downloadBufNum  = 3
	uploadBufSize   = "500MB"
	logPrefix       = "[onestep-import-image-aws]"
)

type AwsImporter struct {
	args 						*AWSImportArguments
	client 					domain.StorageClientInterface
	ctx 						context.Context
	oauth 					string
	paramPopulator 	param.Populator
}

func NewImporter(oauth string, args *AWSImportArguments) (*AwsImporter, error) {
	ctx := context.Background()
	client, err := gcsClient(ctx, oauth)
	if err != nil {
		return nil, err
	}

	computeClient, err := param.CreateComputeClient(&ctx, oauth, args.GcsComputeEndpoint)
	if err != nil {
		return nil, err
	}

	metadataGCE := &compute.MetadataGCE{}
	paramPopulator := param.NewPopulator(
		metadataGCE,
		client,
		storageutils.NewResourceLocationRetriever(metadataGCE, computeClient),
		storageutils.NewScratchBucketCreator(ctx, client),
	)

	importer := &AwsImporter{
		args: args,
		client: client,
		ctx:ctx,
		oauth: oauth,
		paramPopulator: paramPopulator,
	}

	return importer, nil
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

// Run runs aws import workflow.
func (importer *AwsImporter) Run() (string, error) {
	//0. validate AWS args
	err := importer.args.ValidateAndPopulate(importer.paramPopulator)
	if err != nil {
		return "", err
	}

	// 1. configure AWS SDK
	err = importer.configure()
	if err != nil {
		return "", err
	}

	// 1. export AMI to AWS S3 if user did not specify to resume from exported AMI.
	if !importer.args.ResumeExportedAMI {
		err = importer.exportAwsImage()
		if err != nil {
			return "", err
		}
	}

	// 2. copy s3 vmdk to gcs
	gcsFilePath := pathutils.JoinURL(importer.args.GcsScratchBucket,
		fmt.Sprintf("onestep-image-import-aws-%v.vmdk", pathutils.RandString(5)))
	if err := importer.copyToGcs(gcsFilePath); err != nil {
		return "", err
	}
	log.Println(fmt.Sprintf("Successfully copied file to Google Cloud Storage location %v\n.", gcsFilePath))

	return gcsFilePath, nil
}

// Configure AWS CLI
func (importer *AwsImporter) configure() error {
	if err := runCmd("aws", []string{"configure", "set", "aws_access_key_id", importer.args.AccessKeyId}); err != nil {
		return err
	}
	if err := runCmd("aws", []string{"configure", "set", "aws_secret_access_key", importer.args.SecretAccessKey}); err != nil {
		return err
	}
	if err := runCmd("aws", []string{"configure", "set", "region", importer.args.Region}); err != nil {
		return err
	}
	if importer.args.SessionToken != "" {
		if err := runCmd("aws", []string{"configure", "set", "session_token", importer.args.SessionToken}); err != nil {
			return err
		}
	}
	if err := runCmd("aws", []string{"configure", "set", "output", "json"}); err != nil {
		return err
	}
	return nil
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
	fileSize := resp["ContentLength"].(int64)
	if fileSize <= 0 {
		return fmt.Errorf("file is empty")
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
		return fmt.Errorf("empty task id returned")
	}

	// Repeat query to get export task status
	for {
		// aws ec2 describe-export-image-tasks --export-image-task-ids <task id>
		output, err = runCmdAndGetOutputWithoutLog("aws", []string{"ec2", "describe-export-image-tasks", "--export-image-task-ids", fmt.Sprintf("%v", tid)})
		if err != nil {
			return err
		}
		var taskResp AwsTaskResponse
		if err := json.Unmarshal(output, &taskResp); err != nil {
			return err
		}
		if len(taskResp.ExportImageTasks) != 1 {
			return fmt.Errorf("unexpected response of describe-export-image-tasks")
		}
		log.Println(fmt.Sprintf("AWS export task status: %v, status message: %v, progress: %v", taskResp.ExportImageTasks[0].Status, taskResp.ExportImageTasks[0].StatusMessage, taskResp.ExportImageTasks[0].Progress))

		if taskResp.ExportImageTasks[0].Status != "active" {
			if taskResp.ExportImageTasks[0].Status != "completed" {
				return fmt.Errorf("AWS export task wasn't completed successfully")
			}
			break
		}
		time.Sleep(time.Millisecond * 3000)
	}

	// Set exported file data
	importer.args.ExportKey = importer.args.ExportPrefix + tid.(string)+".vmdk"
	if err := importer.getAwsFileSize(); err != nil {
		return err
	}
	log.Println("AWS export task is completed!",
		fmt.Sprintf("Exported location is s3://%v/%v", importer.args.ExportBucket, importer.args.ExportKey))

	return nil
}


func (importer *AwsImporter) copyToGcs(gcsFilePath string) error {
	log.Println("Copying from ec2 to s3...")
	bs, err := humanize.ParseBytes(uploadBufSize)
	if err != nil {
		return err
	}

	bkt, obj, err := storageutils.GetGCSObjectPathElements(gcsFilePath)
	if err != nil {
		log.Fatal(err)
	}
	workers := int64(runtime.NumCPU())
	writer := storageutils.NewBufferedWriter(importer.ctx, int64(bs), workers ,gcsClient, importer.oauth, "/tmp", bkt, obj)

	return importer.stream(writer)
}

func (importer *AwsImporter)stream(writer *storageutils.BufferedWriter) error {
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

		go func(writer *storageutils.BufferedWriter, res *s3.GetObjectOutput) {
			defer wg.Done()
			dmutex.Lock()
			defer dmutex.Unlock()
			defer res.Body.Close()
			io.Copy(writer, res.Body)
			<-sem
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
