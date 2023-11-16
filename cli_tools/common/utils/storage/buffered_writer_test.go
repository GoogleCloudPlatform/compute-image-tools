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

package storage

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
)

var (
	bufferSize, workerNum   int64
	prefix, bkt, obj, oauth string
	mockStorageClient       *mocks.MockStorageClientInterface
	errClient               = fmt.Errorf("Cannot create client")
)

func TestCreateNewChunkOnFirstWrite(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)

	data := []byte("This is a sample data to write")

	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	_, err := buf.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, 1, buf.part)
	assert.NotEmpty(t, buf.file)
}

func TestCreateNewChunkWhenCurrentChunkFull(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)

	data := []byte("This is a sample data to write")

	// passing in mock error client so upload file behavior is not tested
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClientError, oauth, prefix, bkt, obj, "GCEExport")
	err := buf.newChunk()
	assert.Nil(t, err)
	curPart := buf.part
	// make buffer size to max size
	buf.bytes = buf.cSize
	// write should make buffer create new chunk
	_, err = buf.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, int64(len(data)), buf.bytes)
	expectedPart := curPart + 1
	assert.Equal(t, expectedPart, buf.part)
}

func TestUseSameFileWhenCurrentChunkNotFull(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	data := []byte("This is a sample data to write")
	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	_, err := buf.Write(data)
	assert.Nil(t, err)

	expectedPart := buf.part
	fileName := buf.file.Name()
	_, err = buf.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, expectedPart, buf.part)
	assert.Equal(t, fileName, buf.file.Name())
}

func TestFlushErrorWhenInvalidFile(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)

	ctx := context.Background()
	prefix = "//"
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	err := buf.flush()
	assert.NotNil(t, err)
}

func TestWriteErrorWhenFlushError(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	buf.newChunk()
	buf.bytes = buf.cSize
	buf.file, _ = os.Open("//")
	buf.file.Close()
	_, err := buf.Write(data)
	assert.NotNil(t, err)
}

func TestWriteErrorWhenChunkError(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClientError, oauth, prefix, bkt, obj, "GCEExport")
	err := buf.newChunk()
	assert.Nil(t, err)
	buf.bytes = buf.cSize
	buf.prefix = "non_existant_directory/test"
	_, err = buf.Write(data)
	assert.NotNil(t, err)
}

func TestWriteErrorWhenInvalidFilePermission(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	buf.newChunk()
	buf.file, _ = os.OpenFile("//", os.O_RDONLY, 0666)
	_, err := buf.Write(data)
	assert.NotNil(t, err)
}

func TestWriteErrorWhenInvalidFilePrefix(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClientError, oauth, prefix, bkt, obj, "GCEExport")
	buf.prefix = "non_existant_directory/test"
	_, err := buf.Write(data)
	assert.NotNil(t, err)
}

func TestUploadErrorWhenInvalidFile(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	ctx := context.Background()
	prefix = "not_an_existing_file.go"
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	rescueStdout := os.Stdout
	defer func(){ os.Stdout = rescueStdout }()
	r, w, _ := os.Pipe()
	os.Stdout = w
	buf.upload <- prefix
	time.Sleep(time.Second * 2)
	err := w.Close()
	assert.Nil(t, err)
	out, _ := ioutil.ReadAll(r)
	assert.Contains(t, string(out), "no such file or directory")
}

func TestUploadErrorWhenCopyError(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().NewWriter().Return(nil).AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockStorageObject).AnyTimes()
	ctx := context.Background()
	// using this as file name will succeed in os.Open() and fail in io.Copy
	prefix = "//"
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	rescueStdout := os.Stdout
	defer func(){ os.Stdout = rescueStdout }()
	r, w, _ := os.Pipe()
	os.Stdout = w
	buf.upload <- prefix
	time.Sleep(time.Second * 2)
	err := w.Close()
	assert.Nil(t, err)
	out, _ := ioutil.ReadAll(r)
	assert.Contains(t, string(out), "read //: is a directory")
}

func TestAddObjectWhenWorkerUploaded(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	var output bytes.Buffer
	writerCloser := testWriteCloser{Writer: bufio.NewWriter(&output)}
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().NewWriter().Return(writerCloser).AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockStorageObject).AnyTimes()

	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	_, err := buf.Write(data)
	assert.Nil(t, err)
	buf.flush()
	time.Sleep(time.Second * 2)
	assert.NotEmpty(t, buf.tmpObjs)
}

func TestWriteToGCS(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	output := bytes.NewBuffer([]byte{})
	writerCloser := testWriteCloser{Writer: output}
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().NewWriter().Return(writerCloser).AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockStorageObject).AnyTimes()

	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	_, err := buf.Write(data)
	assert.Nil(t, err)
	err = buf.flush()
	assert.Nil(t, err)
	time.Sleep(time.Second * 2)
	assert.Equal(t, output.Bytes(), data)
}

func TestClientError(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClientError, oauth, prefix, bkt, obj, "GCEExport")
	err := buf.Close()
	assert.NotNil(t, err)
}

func TestCopyObjectWhenOneChunk(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().Delete().Return(nil).AnyTimes()
	mockStorageObject.EXPECT().NewWriter().Return(testWriteCloser{ioutil.Discard, nil}).AnyTimes()

	mockStorageObject.EXPECT().ObjectName().Return("").AnyTimes()
	mockStorageObject.EXPECT().CopyFrom(gomock.Any()).Return(nil, nil).AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockStorageObject).AnyTimes()

	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	_, err := buf.Write(data)
	assert.Nil(t, err)

	err = buf.Close()
	assert.Nil(t, err)
}

func TestCopyObjectWithMultipleIterations(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().Delete().Return(nil).AnyTimes()
	mockStorageObject.EXPECT().NewWriter().Return(testWriteCloser{ioutil.Discard, nil}).AnyTimes()

	mockStorageObject.EXPECT().ObjectName().Return("").Times(2)
	mockStorageObject.EXPECT().Compose(gomock.Any()).Return(nil, nil).Times(2)
	mockStorageObject.EXPECT().CopyFrom(gomock.Any()).Return(nil, nil)

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockStorageObject).AnyTimes()

	ctx := context.Background()
	var err error
	data := []byte("This is a sample data to write")
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	for i := 0; i < 33; i++ {
		_, err = buf.Write(data)
		assert.Nil(t, err)
		err = buf.flush()
		assert.Nil(t, err)
		err = buf.newChunk()
		assert.Nil(t, err)
	}
	time.Sleep(time.Second * 2)
	assert.Len(t, buf.tmpObjs, 33)
	err = buf.Close()
	assert.Nil(t, err)
}

func TestErrorWhenCopyFails(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().Delete().Return(nil).AnyTimes()
	mockStorageObject.EXPECT().NewWriter().Return(testWriteCloser{ioutil.Discard, nil}).AnyTimes()
	mockStorageObject.EXPECT().ObjectName().Return("").AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockStorageObject).AnyTimes()

	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	_, err := buf.Write(data)
	assert.Nil(t, err)

	mockStorageObject.EXPECT().CopyFrom(gomock.Any()).Return(nil, fmt.Errorf("Fail to copy")).AnyTimes()

	err = buf.Close()
	assert.NotNil(t, err)
	assert.Equal(t, "Fail to copy", err.Error())
}

func TestErrorWhenComposeFails(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().Delete().Return(nil).AnyTimes()
	mockStorageObject.EXPECT().NewWriter().Return(testWriteCloser{ioutil.Discard, nil}).AnyTimes()

	mockStorageObject.EXPECT().ObjectName().Return("").AnyTimes()
	mockStorageObject.EXPECT().CopyFrom(gomock.Any()).Return(nil, nil).AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockStorageObject).AnyTimes()

	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")
	var err error
	for i := 0; i < 33; i++ {
		_, err = buf.Write(data)
		assert.Nil(t, err)
		err = buf.flush()
		assert.Nil(t, err)
		err = buf.newChunk()
		assert.Nil(t, err)
	}
	time.Sleep(time.Second * 2)
	assert.Len(t, buf.tmpObjs, 33)

	mockStorageObject.EXPECT().Compose(gomock.Any()).Return(nil, fmt.Errorf("Fail to compose")).AnyTimes()

	err = buf.Close()
	assert.NotNil(t, err)
	assert.Equal(t, "Fail to compose", err.Error())
}

func TestBufferedWriterGetPermissionErrorOutput(t *testing.T) {
	resetArgs()
	runTestAssertOutputContainsKeyword(func() {
		mockNewBufferedWriterWithError(t, "some account does not have storage.objects.get access to some object")
	}, t, "GCEExport")
}

func TestBufferedWriterCreatePermissionErrorOutput(t *testing.T) {
	resetArgs()
	runTestAssertOutputContainsKeyword(func() {
		mockNewBufferedWriterWithError(t, "some account does not have storage.objects.create access to some object")
	}, t, "GCEExport")
}

func runTestAssertOutputContainsKeyword(f func(), t *testing.T, keyword string) {
	// Redirect output string to collect console output
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the function that we want to test
	f()

	// Restore the original output
	w.Close()
	output, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	assert.True(t, strings.Contains(string(output), keyword))

}

func mockNewBufferedWriterWithError(t *testing.T, errorMsg string) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().Delete().Return(nil).AnyTimes()
	mockStorageObject.EXPECT().NewWriter().Return(testWriteCloser{ioutil.Discard, &googleapi.Error{Code: 403, Message: errorMsg}}).AnyTimes()
	mockStorageObject.EXPECT().ObjectName().Return("").AnyTimes()
	mockStorageObject.EXPECT().CopyFrom(gomock.Any()).Return(nil, nil).AnyTimes()
	mockStorageObject.EXPECT().Compose(gomock.Any()).Return(nil, nil).AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockStorageObject).AnyTimes()

	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj, "GCEExport")

	var err error
	for i := 0; i < 33; i++ {
		_, err = buf.Write(data)
		assert.Nil(t, err)
		err = buf.flush()
		assert.Nil(t, err)
		err = buf.newChunk()
		assert.Nil(t, err)
	}
	time.Sleep(time.Second * 2)
	assert.Len(t, buf.tmpObjs, 33)

	err = buf.Close()
}

func TestClientErrorWhenUploadFailed(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	buf := NewBufferedWriter(ctx, bufferSize, workerNum, mockGcsClientError, oauth, prefix, bkt, obj, "GCEExport")
	rescueStdout := os.Stdout
	defer func(){ os.Stdout = rescueStdout }()
	r, w, _ := os.Pipe()
	os.Stdout = w
	buf.upload <- "file"
	time.Sleep(time.Second * 2)
	err := w.Close()
	assert.Nil(t, err)
	out, _ := ioutil.ReadAll(r)
	assert.Contains(t, string(out), "Cannot create client")
}

func resetArgs() {
	bufferSize = 1 * 1024
	workerNum = 4
	prefix = "/tmp"
	bkt = "bkt"
	obj = "obj"
	oauth = ""
	errClient = fmt.Errorf("Cannot create client")
	exit = func(code int) {
		fmt.Println("exit with code ", code)
	}
}

func mockGcsClientError(ctx context.Context, oauth string) (domain.StorageClientInterface, error) {
	return nil, errClient
}

func mockGcsClient(ctx context.Context, oauth string) (domain.StorageClientInterface, error) {
	return mockStorageClient, nil
}

type testWriteCloser struct {
	io.Writer
	returnedError error
}

func (w testWriteCloser) Close() error {
	return w.returnedError
}
