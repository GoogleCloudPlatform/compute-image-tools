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

package awsimporter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	onestepUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_onestep_image_import/onestep_utils"
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

// AWSImporter is responsible for importing image from AWS
type AWSImporter struct {
	args           *AWSImportArguments
	client         domain.StorageClientInterface
	ctx            context.Context
	oauth          string
	paramPopulator param.Populator
}

// NewImporter creates an new AWSImporter instance.
// Automatically populating dependencies, such as compute/storage clients.
func NewImporter(oauth string, args *AWSImportArguments) (*AWSImporter, error) {
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

	importer := &AWSImporter{
		args:           args,
		client:         client,
		ctx:            ctx,
		oauth:          oauth,
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
func (importer *AWSImporter) Run() (string, error) {
	//0. validate AWS args
	err := importer.args.ValidateAndPopulate(importer.paramPopulator)
	if err != nil {
		return "", err
	}

	// 1. configure AWS CLI
	err = importer.configure()
	if err != nil {
		return "", err
	}

	// 1. export AMI to AWS S3 if user did not specify to resume from exported AMI.
	if !importer.args.ResumeExportedAMI {
		err = importer.exportAWSImage()
		if err != nil {
			return "", err
		}
	}

	// 2. copy from S3 to GCS
	if err := importer.getAWSFileSize(); err != nil {
		return "", err
	}

	log.Println("Starting to copy ...")
	start := time.Now()
	gcsFilePath, err := importer.copyToGCS()
	if err != nil {
		return "", err
	}
	log.Println(fmt.Sprintf("Successfully copied to %v in %v.", gcsFilePath, time.Since(start)))

	return gcsFilePath, nil
}

// configure configures AWS CLI options.
func (importer *AWSImporter) configure() error {
	if err := onestepUtils.RunCmd("aws", []string{"configure", "set", "aws_access_key_id", importer.args.AccessKeyID}); err != nil {
		return err
	}
	if err := onestepUtils.RunCmd("aws", []string{"configure", "set", "aws_secret_access_key", importer.args.SecretAccessKey}); err != nil {
		return err
	}
	if err := onestepUtils.RunCmd("aws", []string{"configure", "set", "region", importer.args.Region}); err != nil {
		return err
	}
	if importer.args.SessionToken != "" {
		if err := onestepUtils.RunCmd("aws", []string{"configure", "set", "session_token", importer.args.SessionToken}); err != nil {
			return err
		}
	}
	if err := onestepUtils.RunCmd("aws", []string{"configure", "set", "output", "json"}); err != nil {
		return err
	}
	return nil
}

// getAWSFileSize gets the size of the file to copy from S3 to GCS.
func (importer *AWSImporter) getAWSFileSize() error {
	output, err := onestepUtils.RunCmdAndGetOutput("aws", []string{"s3api", "head-object", "--bucket", fmt.Sprintf("%v", importer.args.ExportBucket), "--key", fmt.Sprintf("%v", importer.args.ExportKey)})
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(output, &resp); err != nil {
		return err
	}
	fileSize := int64(resp["ContentLength"].(float64))
	if fileSize <= 0 {
		return fmt.Errorf("file is empty")
	}
	importer.args.ExportFileSize = fileSize
	return nil
}

type awsTaskResponse struct {
	ExportImageTasks []awsExportImageTasks `json:"export_image_task,omitempty"`
}

type awsExportImageTasks struct {
	Status        string `json:"status,omitempty"`
	StatusMessage string `json:"status_message,omitempty"`
	Progress      string `json:"progress,omitempty"`
}

// exportAWSImage calls 'aws ec2 export-image' command to export AMI to S3
func (importer *AWSImporter) exportAWSImage() error {
	// 1. call export command
	output, err := onestepUtils.RunCmdAndGetOutput("aws", []string{"ec2", "export-image", "--image-id", importer.args.ImageID, "--disk-image-format", "VMDK", "--s3-export-location",
		fmt.Sprintf("S3Bucket=%v,S3Prefix=%v", importer.args.ExportBucket, importer.args.ExportFolder)})
	if err != nil {
		return err
	}

	// 2. get export task id from response
	var resp map[string]interface{}
	if err := json.Unmarshal(output, &resp); err != nil {
		return err
	}
	tid := resp["ExportImageTaskId"]
	if tid == "" {
		return fmt.Errorf("empty task id returned")
	}

	// 3. monitor export task progress
	for {
		// aws ec2 describe-export-image-tasks --export-image-task-ids <task id>
		output, err = onestepUtils.RunCmdAndGetOutputWithoutLog("aws", []string{"ec2", "describe-export-image-tasks", "--export-image-task-ids", fmt.Sprintf("%v", tid)})
		if err != nil {
			return err
		}
		var taskResp awsTaskResponse
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
			log.Println("AWS export task is completed!")
			break
		}
		time.Sleep(time.Millisecond * 3000)
	}

	// 4. set exported file data
	importer.args.ExportKey = fmt.Sprintf("%v%v.vmdk", importer.args.ExportFolder, tid)
	importer.args.ExportedAMIPath = fmt.Sprintf("s3://%v/%v", importer.args.ExportBucket, importer.args.ExportKey)
	log.Println(fmt.Sprintf("Exported location is %v.", importer.args.ExportedAMIPath))

	return nil
}

// copyToGCS copies vmdk file from S3 to GCS
func (importer *AWSImporter) copyToGCS() (string, error) {
	// 1. get GCS path as copy destination.
	gcsFilePath := pathutils.JoinURL(importer.args.GcsScratchBucket,
		fmt.Sprintf("onestep-image-import-aws-%v.vmdk", pathutils.RandString(5)))

	log.Println(fmt.Sprintf("Copying %v to %v.",
		importer.args.ExportedAMIPath, gcsFilePath))

	// 2. create a new folder for local buffer
	path := filepath.Join(filepath.Dir(importer.args.ExecutablePath), fmt.Sprint("upload", pathutils.RandString(5)))

	err := os.Mkdir(path, 0755)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(path)

	// 3. get writer
	bs, err := humanize.ParseBytes(uploadBufSize)
	if err != nil {
		return "", err
	}
	bkt, obj, err := storageutils.GetGCSObjectPathElements(gcsFilePath)
	if err != nil {
		return "", err
	}
	workers := int64(runtime.NumCPU())
	writer := storageutils.NewBufferedWriter(importer.ctx, int64(bs), workers, gcsClient, importer.oauth, path, bkt, obj)

	return gcsFilePath, importer.stream(writer)
}

// stream downloads S3 file and uploads to GCS concurrently
func (importer *AWSImporter) stream(writer io.WriteCloser) error {
	totalSize := humanize.IBytes(uint64(importer.args.ExportFileSize))

	// allow max 3 download chunks waiting to be uploaded
	downloadSem := make(chan struct{}, downloadBufNum)
	// used to synchronize upload chunks
	var uploadMutex sync.Mutex

	// create S3 client
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	s3Client := s3.New(sess)

	// setup download
	readSize, err := humanize.ParseBytes(downloadBufSize)
	if err != nil {
		return err
	}
	readers := int64(math.Ceil(float64(importer.args.ExportFileSize) / float64(readSize)))

	wg := new(sync.WaitGroup)
	for i := int64(0); i < readers; i++ {
		startRange := i * int64(readSize)
		endRange := startRange + int64(readSize) - 1
		res, err := s3Client.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(importer.args.ExportBucket),
			Key:    aws.String(importer.args.ExportKey),
			Range:  aws.String(fmt.Sprintf("bytes=%v-%v", startRange, endRange)),
		})
		if err != nil {
			log.Println(fmt.Sprintf("Error in downloading from bucket %v, key %v: %v",
				importer.args.ExportBucket, importer.args.ExportKey, err))
			return err
		}

		downloadSem <- struct{}{}
		wg.Add(1)

		go func(writer io.WriteCloser, reader io.ReadCloser) {
			defer wg.Done()
			defer reader.Close()
			uploadMutex.Lock()
			io.Copy(writer, reader)
			uploadMutex.Unlock()
			<-downloadSem
		}(writer, res.Body)

		uploadTotal := humanize.IBytes(uint64(math.Min(float64(endRange), float64(importer.args.ExportFileSize))))
		log.Println(fmt.Sprintf("Total written size: %v of %v.", uploadTotal, totalSize))
	}

	wg.Wait()
	if err := writer.Close(); err != nil {
		return err
	}
	return nil
}
