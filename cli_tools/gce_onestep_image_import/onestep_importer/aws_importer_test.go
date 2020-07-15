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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

var (
	shouldSetGetObjectDelay = false

	getObjectResp struct {
		output *s3.GetObjectOutput
		err    error
	}
	headObjectResp struct {
		output *s3.HeadObjectOutput
		err    error
	}
	deleteObjectResp struct {
		output *s3.DeleteObjectOutput
		err    error
	}
	exportImageResp struct {
		output *ec2.ExportImageOutput
		err    error
	}
	describeExportTaskResp struct {
		output *ec2.DescribeExportImageTasksOutput
		err    error
	}
)

func resetAPIOutput() {
	getObjectResp.output, getObjectResp.err = &s3.GetObjectOutput{}, nil
	headObjectResp.output, headObjectResp.err = &s3.HeadObjectOutput{ContentLength: aws.Int64(10)}, nil
	deleteObjectResp.output, deleteObjectResp.err = &s3.DeleteObjectOutput{}, nil
	exportImageResp.output, exportImageResp.err = &ec2.ExportImageOutput{}, nil
	describeExportTaskResp.output, describeExportTaskResp.err = &ec2.DescribeExportImageTasksOutput{}, nil
}

type mockS3Client struct {
	s3iface.S3API
}

func (m *mockS3Client) HeadObject(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	return headObjectResp.output, headObjectResp.err
}

func (m *mockS3Client) GetObject(*s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	// set delay so uploader has time to set values for tests
	if shouldSetGetObjectDelay {
		time.Sleep(time.Second * 3)
	}
	return getObjectResp.output, getObjectResp.err
}

func (m *mockS3Client) DeleteObject(*s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	return deleteObjectResp.output, deleteObjectResp.err
}

type mockEC2Client struct {
	ec2iface.EC2API
}

func (m *mockEC2Client) ExportImage(input *ec2.ExportImageInput) (*ec2.ExportImageOutput, error) {
	return exportImageResp.output, exportImageResp.err
}

func (m *mockEC2Client) DescribeExportImageTasks(*ec2.DescribeExportImageTasksInput) (*ec2.DescribeExportImageTasksOutput, error) {
	return describeExportTaskResp.output, describeExportTaskResp.err
}
func (m *mockEC2Client) CancelExportTask(*ec2.CancelExportTaskInput) (*ec2.CancelExportTaskOutput, error) {
	return nil, nil
}

func TestNewImporterPopulates(t *testing.T) {
	args := setUpAWSArgs(awsAccessKeyIDFlag, true)
	awsImporter := getAWSImporter(t, args)

	assert.NotNil(t, awsImporter.ctx)
	assert.NotNil(t, awsImporter.args)
	assert.NotNil(t, awsImporter.gcsClient)
	assert.NotNil(t, awsImporter.paramPopulator)
	assert.NotNil(t, awsImporter.timeoutChan)
	assert.NotNil(t, awsImporter.s3Client)
	assert.NotNil(t, awsImporter.ec2Client)
}

func TestNewImporterFailWhenBadOauth(t *testing.T) {
	args := setUpAWSArgs("", true)
	awsArgs := getAWSImportArgs(args)
	_, err := newAWSImporter("bad-oauth", nil, awsArgs)
	assert.Regexp(t, "failed to create compute client: .* open bad-oauth: no such file or directory", err)
}

func TestRunImporterFailWhenValidateFail(t *testing.T) {
	args := setUpAWSArgs(awsAccessKeyIDFlag, true)
	awsImporter := getAWSImporter(t, args)
	importer, err := NewOneStepImportArguments(args)
	assert.Nil(t, err)
	err = awsImporter.run(importer)
	assert.Error(t, err)
}

func TestRunImporterExportAMI(t *testing.T) {
	args := setUpAWSArgs("", true)
	awsImporter := getAWSImporter(t, args)
	importer, err := NewOneStepImportArguments(args)
	assert.Nil(t, err)

	awsImporter.exportAWSImageFn = func() (string, error) {
		return "", fmt.Errorf("failed")
	}
	err = awsImporter.run(importer)
	assert.EqualError(t, err, "failed")
}

func TestRunImporterSkipExportAMI(t *testing.T) {
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	importer, err := NewOneStepImportArguments(args)
	assert.Nil(t, err)

	awsImporter.exportAWSImageFn = func() (string, error) {
		return "", fmt.Errorf("export image failed")
	}
	awsImporter.getAWSFileSizeFn = func() error {
		return fmt.Errorf("get file size failed")
	}
	err = awsImporter.run(importer)
	assert.EqualError(t, err, "get file size failed")
}

func TestRunImporterCopyFile(t *testing.T) {
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	importer, err := NewOneStepImportArguments(args)
	assert.Nil(t, err)

	awsImporter.copyFromS3ToGCSFn = func() (string, error) {
		return "", fmt.Errorf("failed")
	}
	err = awsImporter.run(importer)
	assert.EqualError(t, err, "failed")
}

func TestRunImporterImportImage(t *testing.T) {
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	importer, err := NewOneStepImportArguments(args)
	assert.Nil(t, err)

	awsImporter.importImageFn = func() error {
		return fmt.Errorf("failed")
	}
	err = awsImporter.run(importer)
	assert.EqualError(t, err, "failed")
}

func TestRunImporterCleanup(t *testing.T) {
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	importer, err := NewOneStepImportArguments(args)
	assert.Nil(t, err)

	isCleanUpCalled := false
	awsImporter.cleanUpFn = func() {
		isCleanUpCalled = true
	}
	err = awsImporter.run(importer)
	assert.True(t, isCleanUpCalled)
}

func TestExportImageReturnErrorWhenCallError(t *testing.T) {
	resetAPIOutput()
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)

	awsImporter.exportAWSImageFn = nil
	exportImageResp.err = fmt.Errorf("export image failed")
	_, err := awsImporter.exportAWSImage()
	assert.Contains(t, err.Error(), "export image failed")
}

func TestExportImageReturnErrorWhenEmptyTaskID(t *testing.T) {
	resetAPIOutput()
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)

	awsImporter.exportAWSImageFn = nil
	exportImageResp.output.ExportImageTaskId = aws.String("")
	_, err := awsImporter.exportAWSImage()
	assert.Contains(t, err.Error(), "empty task id returned")
}

func TestExportImageReturnErrorWhenMonitorTaskError(t *testing.T) {
	resetAPIOutput()
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.exportAWSImageFn = nil
	exportImageResp.output.ExportImageTaskId = aws.String("not empty task id")
	awsImporter.monitorAWSExportImageTaskFn = func() error {
		return fmt.Errorf("failed")
	}
	_, err := awsImporter.exportAWSImage()
	assert.Equal(t, err.Error(), "failed")
}

func TestExportImageCancelTaskWhenError(t *testing.T) {
	resetAPIOutput()
	var buf bytes.Buffer
	log.SetOutput(&buf)

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.exportAWSImageFn = nil
	exportImageResp.output.ExportImageTaskId = aws.String("not empty task id")
	awsImporter.monitorAWSExportImageTaskFn = func() error {
		return fmt.Errorf("failed")
	}

	_, err := awsImporter.exportAWSImage()
	assert.Equal(t, err.Error(), "failed")
	assert.Contains(t, buf.String(), "Cancelling export task ...")
}

func TestMonitorExportTaskReturnErrorWhenCallError(t *testing.T) {
	resetAPIOutput()
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)

	awsImporter.monitorAWSExportImageTaskFn = nil
	describeExportTaskResp.err = fmt.Errorf("get task status failed")
	err := awsImporter.monitorAWSExportImageTask("")
	assert.Contains(t, err.Error(), "get task status failed")
}

func TestMonitorExportTaskReturnErrorWhenMoreThanOneTask(t *testing.T) {
	resetAPIOutput()
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)

	awsImporter.monitorAWSExportImageTaskFn = nil
	exportTask := &ec2.ExportImageTask{}
	describeExportTaskResp.output.ExportImageTasks = []*ec2.ExportImageTask{exportTask, exportTask}
	err := awsImporter.monitorAWSExportImageTask("")
	assert.Contains(t, err.Error(), "failed to get export task status: unexpected response")
}

func TestMonitorExportTaskReturnErrorWhenTaskFailed(t *testing.T) {
	resetAPIOutput()
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)

	awsImporter.monitorAWSExportImageTaskFn = nil
	exportTask := &ec2.ExportImageTask{Status: aws.String("cancelling")}
	describeExportTaskResp.output.ExportImageTasks = []*ec2.ExportImageTask{exportTask}
	err := awsImporter.monitorAWSExportImageTask("")
	assert.Contains(t, err.Error(), "AWS export task wasn't completed successfully")
}

func TestMonitorExportTaskReturnNilWhenTaskCompleted(t *testing.T) {
	resetAPIOutput()
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)

	awsImporter.monitorAWSExportImageTaskFn = nil
	exportTask := &ec2.ExportImageTask{Status: aws.String("completed")}
	describeExportTaskResp.output.ExportImageTasks = []*ec2.ExportImageTask{exportTask}
	err := awsImporter.monitorAWSExportImageTask("")
	assert.Nil(t, err)
}

func TestMonitorExportTaskLogTaskStatus(t *testing.T) {
	resetAPIOutput()
	var buf bytes.Buffer
	log.SetOutput(&buf)

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	// timeout so monitorAWSExportImageTask only checks status once for this test
	close(awsImporter.timeoutChan)
	awsImporter.monitorAWSExportImageTaskFn = nil
	exportTask := &ec2.ExportImageTask{
		Status:        aws.String("active"),
		StatusMessage: aws.String("task is active"),
		Progress:      aws.String("30%"),
	}
	describeExportTaskResp.output.ExportImageTasks = []*ec2.ExportImageTask{exportTask}

	awsImporter.monitorAWSExportImageTask("")
	assert.Contains(t, buf.String(), "AWS export task status: active, status message: task is active, "+
		"progress: 30%.")
}

func TestMonitorExportTaskReturnErrorWhenTimeout(t *testing.T) {
	resetAPIOutput()
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	close(awsImporter.timeoutChan)

	awsImporter.monitorAWSExportImageTaskFn = nil
	exportTask := &ec2.ExportImageTask{
		Status:        aws.String("active"),
		StatusMessage: aws.String("task is active"),
		Progress:      aws.String("30%"),
	}
	describeExportTaskResp.output.ExportImageTasks = []*ec2.ExportImageTask{exportTask}

	err := awsImporter.monitorAWSExportImageTask("")
	assert.EqualError(t, err, "timeout exceeded during image export")
}

func TestGetExportImageTaskReturnEmptyWhenNil(t *testing.T) {
	status, message, progress := getExportImageTask(&ec2.ExportImageTask{})
	assert.Empty(t, status)
	assert.Empty(t, message)
	assert.Empty(t, progress)
}

func TestGetExportImageTaskReturnString(t *testing.T) {
	exportTask := &ec2.ExportImageTask{
		Progress:      aws.String("progress"),
		Status:        aws.String("status"),
		StatusMessage: aws.String("message"),
	}
	status, message, progress := getExportImageTask(exportTask)
	assert.Equal(t, "status", status)
	assert.Equal(t, "message", message)
	assert.Equal(t, "progress", progress)
}

func TestExportImageUpdatesImporterArgs(t *testing.T) {
	resetAPIOutput()

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.exportAWSImageFn = nil
	taskID := "my-task-id"
	exportImageResp.output.ExportImageTaskId = aws.String(taskID)

	s3FilePath, err := awsImporter.exportAWSImage()
	assert.Contains(t, awsImporter.args.exportKey, taskID)
	assert.Contains(t, s3FilePath, taskID)
	assert.NoError(t, err)
}

func TestGetFileSizeReturnErrorWhenCallError(t *testing.T) {
	resetAPIOutput()

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.getAWSFileSizeFn = nil
	headObjectResp.err = fmt.Errorf("get file size error")

	err := awsImporter.getAWSFileSize()
	assert.Contains(t, err.Error(), "get file size error")
}

func TestGetFileSizeReturnErrorWhenEmptyFile(t *testing.T) {
	resetAPIOutput()

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.getAWSFileSizeFn = nil
	headObjectResp.output.ContentLength = aws.Int64(0)

	err := awsImporter.getAWSFileSize()
	assert.Contains(t, err.Error(), "file is empty")
}

func TestGetFileSizeUpdatesImporterArg(t *testing.T) {
	resetAPIOutput()

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.getAWSFileSizeFn = nil
	headObjectResp.output.ContentLength = aws.Int64(100)

	err := awsImporter.getAWSFileSize()
	assert.NoError(t, err)
	assert.EqualValues(t, awsImporter.args.exportFileSize, 100)
}

func TestCopyS3FileReturnErrorWhenGcsPathInvalid(t *testing.T) {
	resetAPIOutput()
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.copyFromS3ToGCSFn = nil
	awsImporter.args.gcsScratchBucket = "not valid gcs"
	_, err := awsImporter.copyFromS3ToGCS()
	assert.Contains(t, err.Error(), "is not a valid Cloud Storage object path")
}

func TestCopyS3FileReturnErrorWhenTransferFileFail(t *testing.T) {
	resetAPIOutput()
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.transferFileFn = func() error {
		return fmt.Errorf("transfer file failed")
	}
	awsImporter.copyFromS3ToGCSFn = nil
	awsImporter.args.gcsScratchBucket = "gs://bucket"
	_, err := awsImporter.copyFromS3ToGCS()
	assert.Contains(t, err.Error(), "transfer file failed")
}

func TestCopyS3FileReturnGCSPath(t *testing.T) {
	resetAPIOutput()

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.copyFromS3ToGCSFn = nil
	awsImporter.args.gcsScratchBucket = "gs://bucket"
	path, err := awsImporter.copyFromS3ToGCS()
	assert.Contains(t, path, "gs://bucket")
	assert.NoError(t, err)
}

func TestTransferFileDownloadError(t *testing.T) {
	resetAPIOutput()
	var output bytes.Buffer
	writer := testWriteCloser{Writer: bufio.NewWriter(&output)}

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.transferFileFn = nil
	getObjectResp.err = fmt.Errorf("download file failed")

	err := awsImporter.transferFile(writer)
	assert.Contains(t, err.Error(), "download file failed")
}

func TestTransferFileDownloadToReaderChan(t *testing.T) {
	resetAPIOutput()

	var output bytes.Buffer
	writer := testWriteCloser{Writer: bufio.NewWriter(&output)}

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.transferFileFn = nil
	getObjectResp.output = &s3.GetObjectOutput{
		Body: ioutil.NopCloser(bytes.NewReader([]byte("file data"))),
	}

	awsImporter.getUploaderFn = func() *uploader {
		u := getTestUploader(writer)
		u.uploadFileFn = func() {
			awsImporter.uploader.Done()
		}
		return u
	}

	err := awsImporter.transferFile(writer)
	assert.NoError(t, err)
	assert.Len(t, awsImporter.uploader.readerChan, 1)

	reader := <-awsImporter.uploader.readerChan
	io.Copy(writer, reader)
	assert.Contains(t, output.String(), "file data")
}

func TestTransferFileErrorWhenUploadError(t *testing.T) {
	resetAPIOutput()

	var output bytes.Buffer
	writer := testWriteCloser{Writer: bufio.NewWriter(&output)}

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.transferFileFn = nil
	getObjectResp.output = &s3.GetObjectOutput{
		Body: ioutil.NopCloser(bytes.NewReader([]byte("file data"))),
	}

	awsImporter.getUploaderFn = func() *uploader {
		u := getTestUploader(writer)
		u.uploadFileFn = func() {
			awsImporter.uploader.uploadErrChan <- fmt.Errorf("upload error")
			awsImporter.uploader.Done()
		}
		return u
	}

	// delay during getObject so uploadFileFn has time to finish updating uploadErr
	shouldSetGetObjectDelay = true
	err := awsImporter.transferFile(writer)
	assert.NotNil(t, err)
}

func TestTransferFileCallsCleanupWhenTimeout(t *testing.T) {
	resetAPIOutput()

	var output bytes.Buffer
	writer := testWriteCloser{Writer: bufio.NewWriter(&output)}

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	close(awsImporter.timeoutChan)
	awsImporter.transferFileFn = nil
	getObjectResp.output = &s3.GetObjectOutput{
		Body: ioutil.NopCloser(bytes.NewReader([]byte("file data"))),
	}

	var isCleanupCalled = false
	awsImporter.getUploaderFn = func() *uploader {
		u := getTestUploader(writer)
		u.cleanupFn = func() {
			isCleanupCalled = true
		}
		return u
	}

	err := awsImporter.transferFile(writer)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "timeout exceeded during transfer file")
	assert.True(t, isCleanupCalled)
}

func TestTransferFileReturnErrorWhenWriterCloseFail(t *testing.T) {
	resetAPIOutput()

	var output bytes.Buffer
	writer := testWriteCloser{Writer: bufio.NewWriter(&output)}
	writer.closeReturnVal = fmt.Errorf("close writer failed")

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.transferFileFn = nil
	getObjectResp.output = &s3.GetObjectOutput{
		Body: ioutil.NopCloser(bytes.NewReader([]byte("file data"))),
	}
	err := awsImporter.transferFile(writer)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "close writer failed")
}

func TestImportImageUpdateImporterArgs(t *testing.T) {
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	importer, err := NewOneStepImportArguments(args)
	assert.Nil(t, err)

	awsImporter.copyFromS3ToGCSFn = func() (string, error) {
		return "gs://test-file-path", nil
	}
	awsImporter.importImageFn = nil

	originalTimeout := importer.Timeout
	err = awsImporter.run(importer)

	assert.Equal(t, importer.SourceFile, "gs://test-file-path")
	assert.NotEqual(t, importer.Timeout, originalTimeout)
}

func TestImportImageReturnTimeoutError(t *testing.T) {
	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	importer, err := NewOneStepImportArguments(args)
	assert.Nil(t, err)

	importer.Timeout = 0
	awsImporter.importImageFn = nil
	err = awsImporter.run(importer)

	assert.EqualError(t, err, "timeout exceeded")
}

func TestImportImageReturnsError(t *testing.T) {
	args := setUpAWSArgs("", false)
	importer, err := NewOneStepImportArguments(args)
	awsImporter := getAWSImporter(t, args)
	awsImporter.importImageFn = func() error {
		return fmt.Errorf("image import failed")
	}
	err = awsImporter.importImage(importer, time.Now(), "")
	assert.EqualError(t, err, "image import failed")
}

func TestCleanupDeleteGCSPath(t *testing.T) {
	gcsPath := "gcsPath"
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().DeleteGcsPath("gcsPath")
	mockStorageClient.EXPECT().Close()

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.gcsClient = mockStorageClient
	awsImporter.cleanUpFn = nil
	awsImporter.cleanUp(gcsPath, "")
}

func TestCleanupDeleteGCSPathError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	gcsPath := "gcsPath"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().DeleteGcsPath("gcsPath").Return(fmt.Errorf("delete error"))
	mockStorageClient.EXPECT().Close()

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.gcsClient = mockStorageClient
	awsImporter.cleanUpFn = nil
	awsImporter.cleanUp(gcsPath, "")

	assert.Contains(t, buf.String(), "delete error")
}

func TestCleanupDeleteS3PathError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().DeleteGcsPath("")
	mockStorageClient.EXPECT().Close()
	deleteObjectResp.err = fmt.Errorf("delete error")

	args := setUpAWSArgs("", false)
	awsImporter := getAWSImporter(t, args)
	awsImporter.gcsClient = mockStorageClient
	awsImporter.cleanUpFn = nil
	awsImporter.cleanUp("", "")

	assert.Contains(t, buf.String(), "delete error")
}

func getAWSImporter(t *testing.T, args []string) *awsImporter {
	awsArgs := getAWSImportArgs(args)
	awsImporter, err := newAWSImporter("", make(chan struct{}), awsArgs)
	assert.Nil(t, err)

	awsImporter.ec2Client = &mockEC2Client{}
	awsImporter.s3Client = &mockS3Client{}
	awsImporter.paramPopulator = mockPopulator{}

	awsImporter.exportAWSImageFn = func() (string, error) { return "", nil }
	awsImporter.monitorAWSExportImageTaskFn = func() error { return nil }
	awsImporter.getAWSFileSizeFn = func() error { return nil }
	awsImporter.copyFromS3ToGCSFn = func() (string, error) { return "", nil }
	awsImporter.transferFileFn = func() error { return nil }
	awsImporter.importImageFn = func() error { return nil }
	awsImporter.cleanUpFn = func() {}

	return awsImporter
}
