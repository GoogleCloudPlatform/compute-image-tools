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
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
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

// awsImporter is responsible for importing image from AWS.
type awsImporter struct {
	args           *awsImportArguments
	gcsClient      domain.StorageClientInterface
	ctx            context.Context
	oauth          string
	paramPopulator param.Populator
	timeoutChan    chan struct{}
	uploader       *uploader

	// AWS clients for SDK
	ec2Client ec2iface.EC2API
	s3Client  s3iface.S3API

	// Impl of the functions
	exportAWSImageFn            func() (string, error)
	monitorAWSExportImageTaskFn func() error
	getAWSFileSizeFn            func() error
	copyFromS3ToGCSFn           func() (string, error)
	transferFileFn              func() error
	getUploaderFn               func() *uploader
	importImageFn               func() error
	cleanUpFn                   func()
}

// newAWSImporter creates an new awsImporter instance.
// Automatically populating dependencies, such as compute/storage clients.
func newAWSImporter(oauth string, timeoutChan chan struct{}, args *awsImportArguments) (*awsImporter, error) {
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
		timeoutChan:    timeoutChan,
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
		return nil, daisy.Errf("failed to create Cloud Storage client: %v", err)
	}

	storage, err := storageutils.NewStorageClient(ctx, logger, option.WithHTTPClient(&http.Client{Transport: transport}),
		option.WithCredentialsFile(oauth))
	if err != nil {
		return nil, daisy.Errf("failed to create Cloud Storage client: %v", err)
	}
	return storage, nil
}

// createAWSSession creates a new Session for AWS SDK.
func createAWSSession(region, accessKeyID, secretAccessKey, sessionToken string) (*session.Session, error) {
	session, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			accessKeyID,
			secretAccessKey,
			sessionToken),
	})
	if err != nil {
		return nil, daisy.Errf("failed to create AWS session: %v", err)
	}
	return session, nil
}

// run runs the aws importer to import AMI.
func (importer *awsImporter) run(importArgs *OneStepImportArguments) error {
	startTime := time.Now()
	//1. validate AWS args
	err := importer.args.validateAndPopulate(importer.paramPopulator)
	if err != nil {
		return err
	}

	// 2. export AMI to AWS S3 if user did not specify an exported AMI path.
	var s3FilePath string
	if importer.args.isExportRequired() {
		log.Println("Starting to export image ...")
		s3FilePath, err = importer.exportAWSImage()
		if err != nil {
			return err
		}
	}
	if err := importer.getAWSFileSize(); err != nil {
		return err
	}

	// 3. copy from S3 to GCS
	log.Println("Starting to copy ...")
	gcsFilePath, err := importer.copyFromS3ToGCS()
	if err != nil {
		return err
	}

	// 4. run image import
	log.Println("Starting to import image ...")
	err = importer.importImage(importArgs, startTime, gcsFilePath)
	if err != nil {
		return err
	}
	log.Println("Image import from AWS finished successfully!")

	// 5. clean up
	log.Println("Cleaning up ...")
	importer.cleanUp(gcsFilePath, s3FilePath)

	return nil
}

// cleanUp deletes temporary files created during image import, and closes GCS client.
func (importer *awsImporter) cleanUp(gcsFilePath, s3FilePath string) {
	if importer.cleanUpFn != nil {
		importer.cleanUpFn()
		return
	}

	err := importer.gcsClient.DeleteGcsPath(gcsFilePath)
	if err != nil {
		log.Printf("Could not delete image file %v: %v. To avoid being charged, "+
			"please manually delete the file.\n", gcsFilePath, err.Error())
	}

	importer.gcsClient.Close()

	_, err = importer.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(importer.args.exportBucket),
		Key:    aws.String(importer.args.exportKey),
	})
	if err != nil {
		log.Printf("Could not delete image file %v: %v. To avoid being charged, "+
			"please manually delete the file.\n", s3FilePath, err.Error())
	}
}

// importImage updates importArgs to contain the image source file and updated timeout duration.
// It runs image import to import from gcsFilePath to Compute Engine.
func (importer *awsImporter) importImage(importArgs *OneStepImportArguments, startTime time.Time, gcsFilePath string) error {
	if importer.importImageFn != nil {
		return importer.importImageFn()
	}

	// update source file flag to copied GCS destination
	importArgs.SourceFile = gcsFilePath

	// adjust timeout to pass into image import
	importArgs.Timeout = importArgs.Timeout - time.Since(startTime)
	if importArgs.Timeout <= 0 {
		return daisy.Errf("timeout exceeded")
	}

	// add label to indicate the image import is run from onestep import
	if importArgs.Labels == nil {
		importArgs.Labels = make(map[string]string)
	}
	importArgs.Labels["onestep-image-import"] = "aws"

	err := runImageImport(importArgs)
	if err != nil {
		log.Printf("Failed to import image. "+
			"The image file is copied to Cloud Storage, located at %v. "+
			"To resume the import process, please directly use image import from Cloud Storage.\n", gcsFilePath)
		return err
	}
	return nil
}

// getAWSFileSize gets the size of the file to copy from S3 to GCS.
func (importer *awsImporter) getAWSFileSize() error {
	if importer.getAWSFileSizeFn != nil {
		return importer.getAWSFileSizeFn()
	}

	resp, err := importer.s3Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(importer.args.exportBucket),
		Key:    aws.String(importer.args.exportKey),
	})
	if err != nil {
		return daisy.Errf("failed to get file size: %v", err)
	}
	fileSize := aws.Int64Value(resp.ContentLength)
	if fileSize <= 0 {
		return daisy.Errf("file is empty")
	}

	importer.args.exportFileSize = fileSize
	return nil
}

// getExportImageTask creates a awsExportImageTasks from an ec2.ExportImageTask
func getExportImageTask(task *ec2.ExportImageTask) (string, string, string) {
	return aws.StringValue(task.Status), aws.StringValue(task.StatusMessage), aws.StringValue(task.Progress)
}

// exportAWSImage calls 'aws ec2 export-image' command to export AMI to S3
func (importer *awsImporter) exportAWSImage() (string, error) {
	if importer.exportAWSImageFn != nil {
		return importer.exportAWSImageFn()
	}

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
		return "", daisy.Errf("failed to begin export AWS image: %v", err)
	}

	// 2. get export task id from response
	taskID := aws.StringValue(resp.ExportImageTaskId)
	if taskID == "" {
		return "", daisy.Errf("empty task id returned")
	}

	// 3. monitor export task progress
	err = importer.monitorAWSExportImageTask(taskID)
	if err != nil {
		importer.cancelAWSExportImageTask(taskID)
		return "", err
	}

	// 4. set exported file data
	importer.args.exportKey = fmt.Sprintf("%v%v.vmdk", importer.args.exportFolder, taskID)
	s3FilePath := fmt.Sprintf("s3://%v/%v", importer.args.exportBucket, importer.args.exportKey)
	importer.args.sourceFilePath = s3FilePath
	log.Printf("Image export location is %v.\n", s3FilePath)

	return s3FilePath, nil
}

// monitorAWSExportImageTask monitors the progress of the AWS export image task.
// It calls AWS SDK to get the progress of taskID, and outputs status if task is active or completed.
// Else, there is an error with the export image task, and the error is returned.
func (importer *awsImporter) monitorAWSExportImageTask(taskID string) error {
	if importer.monitorAWSExportImageTaskFn != nil {
		return importer.monitorAWSExportImageTaskFn()
	}

	describeExportInput := &ec2.DescribeExportImageTasksInput{
		ExportImageTaskIds: []*string{aws.String(taskID)},
	}
	var taskStatus, taskStatusMessage, taskProgress string

	for {
		output, err := importer.ec2Client.DescribeExportImageTasks(describeExportInput)
		if err != nil {
			return daisy.Errf("failed to get export task status: %v", err)
		}
		if len(output.ExportImageTasks) != 1 {
			return daisy.Errf("failed to get export task status: unexpected response")
		}
		taskStatus, taskStatusMessage, taskProgress = getExportImageTask(output.ExportImageTasks[0])

		if taskStatus != "active" {
			if taskStatus != "completed" {
				return daisy.Errf("AWS export task wasn't completed successfully")
			}
			log.Println("AWS export task is completed!")
			break
		}

		log.Printf("AWS export task status: %v, status message: %v, "+
			"progress: %v.\n", taskStatus, taskStatusMessage, taskProgress)

		select {
		case <-importer.timeoutChan:
			return daisy.Errf("timeout exceeded during image export")
		default:
			// Did not timeout, continue to check task status.
		}

		time.Sleep(time.Second * 10)
	}
	return nil
}

// cancelAWSExportImageTask cancels the AWS export task.
func (importer *awsImporter) cancelAWSExportImageTask(taskID string) {
	log.Printf("Cancelling export task ...")
	importer.ec2Client.CancelExportTask(&ec2.CancelExportTaskInput{ExportTaskId: aws.String(taskID)})
}

// copyFromS3ToGCS copies VMDK file from S3 to GCS.
func (importer *awsImporter) copyFromS3ToGCS() (string, error) {
	if importer.copyFromS3ToGCSFn != nil {
		return importer.copyFromS3ToGCSFn()
	}

	start := time.Now()
	// 1. get GCS path as copy destination.
	gcsFilePath := pathutils.JoinURL(importer.args.gcsScratchBucket,
		fmt.Sprintf("onestep-image-import-aws-%v.vmdk", pathutils.RandString(5)))

	log.Printf("Copying %v to %v.\n", importer.args.sourceFilePath, gcsFilePath)

	// 2. create a new folder for local buffer
	path := filepath.Join(filepath.Dir(importer.args.executablePath), fmt.Sprint("upload", pathutils.RandString(5)))

	err := os.Mkdir(path, 0755)
	if err != nil {
		return "", daisy.ToDError(err)
	}
	defer os.RemoveAll(path)

	// 3. get writer
	bs, err := humanize.ParseBytes(uploadBufSize)
	if err != nil {
		return "", daisy.ToDError(err)
	}
	bkt, obj, err := storageutils.GetGCSObjectPathElements(gcsFilePath)
	if err != nil {
		return "", err
	}
	workers := int64(runtime.NumCPU())
	writer := storageutils.NewBufferedWriter(importer.ctx, int64(bs), workers, createGCSClient, importer.oauth, path, bkt, obj)

	// 4. Transfer file from S3 to GCS
	if err := importer.transferFile(writer); err != nil {
		return gcsFilePath, err
	}
	log.Printf("Successfully copied to %v in %v.\n", gcsFilePath, time.Since(start))

	return gcsFilePath, nil
}

func (importer *awsImporter) getUploader(writer io.WriteCloser) *uploader {
	if importer.getUploaderFn != nil {
		return importer.getUploaderFn()
	}

	return &uploader{
		readerChan:    make(chan io.ReadCloser, downloadBufNum),
		writer:        writer,
		totalUploaded: 0,
		totalFileSize: importer.args.exportFileSize,
		uploadErrChan: make(chan error),
	}
}

// transferFile downloads S3 file and uploads to GCS concurrently.
func (importer *awsImporter) transferFile(writer io.WriteCloser) error {
	if importer.transferFileFn != nil {
		return importer.transferFileFn()
	}
	// 1. Set up download size and get number of chunks to download
	output, err := humanize.ParseBytes(downloadBufSize)
	if err != nil {
		return daisy.ToDError(err)
	}
	readSize := int64(output)
	// Take ceiling to get number of chunks to download.
	readers := (importer.args.exportFileSize-1)/readSize + 1
	// Set up download retry delay interval
	delayTime := []int{1, 2, 4, 8, 8}
	maxRetryTimes := len(delayTime)

	// 2. Set up upload info
	importer.uploader = importer.getUploader(writer)
	importer.uploader.Add(1)
	go importer.uploader.uploadFile()

	// 3. Range download
	for i := int64(0); i < readers; i++ {
		startRange := i * readSize
		endRange := startRange + readSize - 1
		for retryAttempt := 0; ; retryAttempt++ {
			res, err := importer.s3Client.GetObject(&s3.GetObjectInput{
				Bucket: aws.String(importer.args.exportBucket),
				Key:    aws.String(importer.args.exportKey),
				Range:  aws.String(fmt.Sprintf("bytes=%v-%v", startRange, endRange)),
			})
			if err != nil {
				if retryAttempt >= maxRetryTimes {
					return daisy.Errf("error in downloading from %v: %v", importer.args.sourceFilePath, err)
				}
				time.Sleep(time.Duration(delayTime[retryAttempt]) * time.Second)
				continue
			}
			importer.uploader.readerChan <- res.Body
			break
		}

		// Stop downloading as soon as one of the upload fails.
		select {
		case err := <-importer.uploader.uploadErrChan:
			importer.uploader.cleanup()
			return err
		default:
			// No error, continue to download.
		}

		// Stop downloading if timeout exceeded.
		select {
		case <-importer.timeoutChan:
			importer.uploader.cleanup()
			return daisy.Errf("timeout exceeded during transfer file")
		default:
			// Did not timeout, continue to download.
		}
	}

	// All file chunks are downloaded, wait for upload to finish.
	close(importer.uploader.readerChan)
	importer.uploader.Wait()

	err = importer.uploader.writer.Close()
	if err != nil {
		return daisy.ToDError(err)
	}
	return nil
}
