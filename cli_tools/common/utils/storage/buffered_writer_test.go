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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	bufferSize, workerNum   int64
	prefix, bkt, obj, oauth string
	mockStorageClient       *mocks.MockStorageClientInterface
)

func TestCreateNewChunkOnFirstWrite(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)

	data := []byte("This is a sample data to write")

	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj)
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
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClientError, oauth, prefix, bkt, obj)
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
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj)
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
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj)
	err := buf.flush()
	assert.NotNil(t, err)
}

func TestWriteErrorWhenInvalidFile(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	ctx := context.Background()
	data := []byte("This is a sample data to write")
	prefix = "//"
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj)
	_, err := buf.Write(data)
	assert.NotNil(t, err)
}

func TestAddObjectWhenWorkerUploaded(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	var output bytes.Buffer
	writerCloser := testWriteCloser{Writer: bufio.NewWriter(&output)}
	mockObjectHandle := mocks.NewMockStorageObjectInterface(mockCtrl)
	mockObjectHandle.EXPECT().NewWriter().Return(writerCloser).AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockObjectHandle).AnyTimes()

	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj)
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
	mockObjectHandle := mocks.NewMockStorageObjectInterface(mockCtrl)
	mockObjectHandle.EXPECT().NewWriter().Return(writerCloser).AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockObjectHandle).AnyTimes()

	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj)
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
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClientError, oauth, prefix, bkt, obj)
	err := buf.Close()
	assert.NotNil(t, err)
}

func TestCopyObjectWhenOneChunk(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockObjectHandle := mocks.NewMockStorageObjectInterface(mockCtrl)
	mockObjectHandle.EXPECT().Delete().Return(nil).AnyTimes()
	mockObjectHandle.EXPECT().NewWriter().Return(testWriteCloser{ioutil.Discard}).AnyTimes()

	mockObjectHandle.EXPECT().ObjectName().Return("").AnyTimes()
	//mockObjectHandle.EXPECT().RunComposer(gomock.Any()).Return(nil, nil).AnyTimes()
	mockObjectHandle.EXPECT().RunCopier(gomock.Any()).Return(nil, nil).AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockObjectHandle).AnyTimes()

	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj)
	_, err := buf.Write(data)
	assert.Nil(t, err)

	err = buf.Close()
	assert.Nil(t, err)
}

func TestCopyObjectWithMultipleIterations(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockObjectHandle := mocks.NewMockStorageObjectInterface(mockCtrl)
	mockObjectHandle.EXPECT().Delete().Return(nil).AnyTimes()
	mockObjectHandle.EXPECT().NewWriter().Return(testWriteCloser{ioutil.Discard}).AnyTimes()

	mockObjectHandle.EXPECT().ObjectName().Return("").Times(2)
	mockObjectHandle.EXPECT().RunComposer(gomock.Any()).Return(nil, nil).Times(2)
	mockObjectHandle.EXPECT().RunCopier(gomock.Any()).Return(nil, nil)

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockObjectHandle).AnyTimes()

	ctx := context.Background()
	var err error
	data := []byte("This is a sample data to write")
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj)
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
	mockObjectHandle := mocks.NewMockStorageObjectInterface(mockCtrl)
	mockObjectHandle.EXPECT().Delete().Return(nil).AnyTimes()
	mockObjectHandle.EXPECT().NewWriter().Return(testWriteCloser{ioutil.Discard}).AnyTimes()
	mockObjectHandle.EXPECT().ObjectName().Return("").AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockObjectHandle).AnyTimes()

	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj)
	_, err := buf.Write(data)
	assert.Nil(t, err)

	mockObjectHandle.EXPECT().RunCopier(gomock.Any()).Return(nil, fmt.Errorf("Fail to copy")).AnyTimes()

	err = buf.Close()
	assert.NotNil(t, err)
	assert.Equal(t, "Fail to copy", err.Error())
}

func TestErrorWhenComposeFails(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockObjectHandle := mocks.NewMockStorageObjectInterface(mockCtrl)
	mockObjectHandle.EXPECT().Delete().Return(nil).AnyTimes()
	mockObjectHandle.EXPECT().NewWriter().Return(testWriteCloser{ioutil.Discard}).AnyTimes()

	mockObjectHandle.EXPECT().ObjectName().Return("").AnyTimes()
	mockObjectHandle.EXPECT().RunCopier(gomock.Any()).Return(nil, nil).AnyTimes()

	mockStorageClient = mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().Close().Return(nil).AnyTimes()
	mockStorageClient.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(mockObjectHandle).AnyTimes()

	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClient, oauth, prefix, bkt, obj)
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

	mockObjectHandle.EXPECT().RunComposer(gomock.Any()).Return(nil, fmt.Errorf("Fail to compose")).AnyTimes()

	err = buf.Close()
	assert.NotNil(t, err)
	assert.Equal(t, "Fail to compose", err.Error())
}

func TestClientErrorWhenUploadFailed(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	buf := NewBuffer(ctx, bufferSize, workerNum, mockGcsClientError, oauth, prefix, bkt, obj)
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
	bkt = "fionaliu-daisy-bkt-us-east1"
	obj = "obj"
	oauth = ""
}

func mockGcsClientError(ctx context.Context, oauth string) (domain.StorageClientInterface, error) {
	return nil, fmt.Errorf("Cannot create client")
}

func mockGcsClient(ctx context.Context, oauth string) (domain.StorageClientInterface, error) {
	return mockStorageClient, nil
}

type testWriteCloser struct {
	io.Writer
}

func (testWriteCloser) Close() error {
	return nil
}
