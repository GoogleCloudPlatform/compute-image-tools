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
	"context"
	"fmt"
	"io"
	"log"
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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dustin/go-humanize"
	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
)

const (
	downloadBufSize = "100MB"
	downloadBufNum  = 3
	uploadBufSize   = "200MB"
	logPrefix       = "[onestep-import-image-aws]"
)

// awsImporter is responsible for importing image from AWS
type awsImporter struct {
	args           *awsImportArguments
	gcsClient      domain.StorageClientInterface
	ctx            context.Context
	oauth          string
	paramPopulator param.Populator

	// AWS clients for SDK
	ec2Client *ec2.EC2
	s3Client  *s3.S3
}

// newAWSImporter creates an new awsImporter instance.
// Automatically populating dependencies, such as compute/storage clients.
func newAWSImporter(oauth string, args *awsImportArguments) (*awsImporter, error) {
	ctx := context.Background()
	client, err := createGCSClient(ctx, oauth)
	if err != nil {
		return nil, err
	}

	computeClient, err := param.CreateComputeClient(&ctx, oauth, args.gcsComputeEndpoint)
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

	awsSession, err := createAWSSession(args.region, args.accessKeyID, args.secretAccessKey, args.sessionToken)
	if err != nil {
		return nil, err
	}

	importer := &awsImporter{
		args:           args,
		gcsClient:      client,
		s3Client:       s3.New(awsSession),
		ec2Client:      ec2.New(awsSession),
		ctx:            ctx,
		oauth:          oauth,
		paramPopulator: paramPopulator,
	}

	return importer, nil
}

// createGCSClient creates a new GCS client.
func createGCSClient(ctx context.Context, oauth string) (domain.StorageClientInterface, error) {
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

// createAWSSession creates a new Session for AWS SDK.
func createAWSSession(region, accessKeyID, secretAccessKey, sessionToken string) (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			accessKeyID,
			secretAccessKey,
			sessionToken),
	})
}

// Run runs aws import workflow.
func (importer *awsImporter) run() (string, error) {
	//0. validate AWS args
	err := importer.args.validateAndPopulate(importer.paramPopulator)
	if err != nil {
		return "", err
	}

	// 1. export AMI to AWS S3 if user did not specify an exported AMI path.
	if importer.args.exportedAMIPath == "" {
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
	gcsFilePath, err := importer.copyFromS3ToGCS()
	if err != nil {
		return "", err
	}
	log.Printf("Successfully copied to %v in %v.\n", gcsFilePath, time.Since(start))

	return gcsFilePath, nil
}

// getAWSFileSize gets the size of the file to copy from S3 to GCS.
func (importer *awsImporter) getAWSFileSize() error {
	resp, err := importer.s3Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(importer.args.exportBucket),
		Key:    aws.String(importer.args.exportKey),
	})
	if err != nil {
		return err
	}
	fileSize := *resp.ContentLength
	if fileSize <= 0 {
		return fmt.Errorf("file is empty")
	}

	importer.args.exportFileSize = fileSize
	return nil
}

// getExportImageTask creates a awsExportImageTasks from an ec2.ExportImageTask
func getExportImageTask(task *ec2.ExportImageTask) (string, string, string) {
	return aws.StringValue(task.Status), aws.StringValue(task.StatusMessage), aws.StringValue(task.Progress)
}

// exportAWSImage calls 'aws ec2 export-image' command to export AMI to S3
func (importer *awsImporter) exportAWSImage() error {
	// 1. send export request
	s3ExportLocation := &ec2.ExportTaskS3LocationRequest{
		S3Bucket: aws.String(importer.args.exportBucket),
		S3Prefix: aws.String(importer.args.exportKey),
	}

	resp, err := importer.ec2Client.ExportImage(&ec2.ExportImageInput{
		DiskImageFormat:  aws.String("VMDK"),
		ImageId:          aws.String(importer.args.amiID),
		S3ExportLocation: s3ExportLocation,
	})

	if err != nil {
		return err
	}
	// 2. get export task id from response
	taskID := aws.StringValue(resp.ExportImageTaskId)
	if taskID == "" {
		return fmt.Errorf("empty task id returned")
	}

	// 3. monitor export task progress
	describeExportInput := ec2.DescribeExportImageTasksInput{
		ExportImageTaskIds: []*string{resp.ExportImageTaskId},
	}
	var (
		taskStatus, taskStatusMessage, taskProgress string
	)
	for {
		err := importer.ec2Client.DescribeExportImageTasksPages(&describeExportInput,
			func(page *ec2.DescribeExportImageTasksOutput, lastPage bool) bool {
				if len(page.ExportImageTasks) != 1 {
					err = fmt.Errorf("unexpected response of describe-export-image-tasks")
				}
				taskStatus, taskStatusMessage, taskProgress = getExportImageTask(page.ExportImageTasks[0])
				return false
			})
		if err != nil {
			return err
		}

		if taskStatus != "active" {
			if taskStatus != "completed" {
				return fmt.Errorf("AWS export task wasn't completed successfully")
			}
			log.Println("AWS export task is completed!")
			break
		}

		log.Printf("AWS export task status: %v, status message: %v, "+
			"progress: %v.\n", taskStatus, taskStatusMessage, taskProgress)

		time.Sleep(time.Millisecond * 3000)
	}

	// 4. set exported file data
	importer.args.exportKey = fmt.Sprintf("%v%v.vmdk", importer.args.exportFolder, taskID)
	importer.args.exportedAMIPath = fmt.Sprintf("s3://%v/%v", importer.args.exportBucket, importer.args.exportKey)
	log.Printf("Exported location is %v.\n", importer.args.exportedAMIPath)

	return nil
}

// copyToGCS copies vmdk file from S3 to GCS
func (importer *awsImporter) copyFromS3ToGCS() (string, error) {
	// 1. get GCS path as copy destination.
	gcsFilePath := pathutils.JoinURL(importer.args.gcsScratchBucket,
		fmt.Sprintf("onestep-image-import-aws-%v.vmdk", pathutils.RandString(5)))

	log.Printf("Copying %v to %v.\n", importer.args.exportedAMIPath, gcsFilePath)

	// 2. create a new folder for local buffer
	path := filepath.Join(filepath.Dir(importer.args.executablePath), fmt.Sprint("upload", pathutils.RandString(5)))

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
	writer := storageutils.NewBufferedWriter(importer.ctx, int64(bs), workers, createGCSClient, importer.oauth, path, bkt, obj)

	return gcsFilePath, importer.transferFile(writer)
}

// transferFile downloads S3 file and uploads to GCS concurrently
func (importer *awsImporter) transferFile(writer io.WriteCloser) error {
	// 1. Set up download size and get number of chunks to download
	output, err := humanize.ParseBytes(downloadBufSize)
	if err != nil {
		return err
	}
	readSize := int64(output)
	// take ceiling
	readers := (importer.args.exportFileSize-1)/readSize + 1

	// 2. Set up upload info
	uploader := &uploader{
		readerChan:    make(chan io.ReadCloser, downloadBufNum),
		writer:        writer,
		totalFileSize: importer.args.exportFileSize,
		totalUploaded: 0,
		err:           nil,
	}
	uploader.Add(1)
	go uploader.uploadFile()

	// 3. Range download
	retryer := &retryer{}
	for i := int64(0); i < readers; i++ {
		startRange := i * readSize
		endRange := startRange + readSize - 1
		req, res := importer.s3Client.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(importer.args.exportBucket),
			Key:    aws.String(importer.args.exportKey),
			Range:  aws.String(fmt.Sprintf("bytes=%v-%v", startRange, endRange)),
		})
		req.Retryer = retryer
		err := req.Send()
		if err != nil {
			log.Printf("Error in downloading from %v.\n",
				importer.args.exportedAMIPath)
			uploader.cleanup()
			return err
		}

		uploader.readerChan <- res.Body

		// Stop downloading as soon as one of the upload fails.
		if uploader.err != nil {
			uploader.cleanup()
			return uploader.err
		}
	}

	// all file chunks are downloaded, wait for upload to finish
	close(uploader.readerChan)
	uploader.Wait()

	return uploader.writer.Close()
}
