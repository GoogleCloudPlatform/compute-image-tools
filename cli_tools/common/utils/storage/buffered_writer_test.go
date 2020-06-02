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

package storage

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	bufferSize, worker_num int64
	prefix, bkt, obj, oauth string
)

func TestCreateNewChunkOnFirstWrite(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	data := []byte("This is a sample data to write")

	buf := NewBuffer(ctx, bufferSize, worker_num, oauth, prefix, bkt, obj)
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
	data := []byte("This is a sample data to write")

	buf := NewBuffer(ctx, bufferSize, worker_num, oauth, prefix, bkt, obj)
	// make buffer size to max size
	buf.bytes = buf.cSize
	_, err := buf.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, int64(len(data)), buf.bytes)
}

func TestUseSameFileWhenCurrentChunkNotFull(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	data := []byte("This is a sample data to write")

	buf := NewBuffer(ctx, bufferSize, worker_num, oauth, prefix, bkt, obj)
	_, err := buf.Write(data)
	assert.Nil(t, err)

	fileName := buf.file.Name()
	_, err = buf.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, fileName, buf.file.Name())
}

func TestFlushErrorWhenInvalidFile(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	prefix = "//"
	buf := NewBuffer(ctx, bufferSize, worker_num, oauth, prefix, bkt, obj)
	err := buf.Close()
	assert.NotNil(t, err)
}

func TestWriteErrorWhenInvalidFile(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	data := []byte("This is a sample data to write")
	prefix = "//"
	buf := NewBuffer(ctx, bufferSize, worker_num, oauth, prefix, bkt, obj)
	_, err := buf.Write(data)
	assert.NotNil(t, err)
}

func TestFileHasDataAfterWrite(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBuffer(ctx, bufferSize, worker_num, oauth, prefix, bkt, obj)
	_, err := buf.Write(data)
	assert.Nil(t, err)

	fileData, err := ioutil.ReadFile(buf.file.Name())
	assert.Equal(t, fileData, data)
}

func TestCopyObjectWhenOneChunk(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	data := []byte("This is a sample data to write")
	buf := NewBuffer(ctx, bufferSize, worker_num, oauth, prefix, bkt, obj)
	_, err := buf.Write(data)
	assert.Nil(t, err)

	err = buf.Close()
	assert.Nil(t, err)
	client, err := gcsClient(ctx, oauth)
	reader, err := client.Bucket(bkt).Object(obj).NewReader(ctx)
	objBuf := new(bytes.Buffer)
	objBuf.ReadFrom(reader)
	assert.Equal(t, data, objBuf.Bytes())
}

func TestCopyObjectWithLargeFile(t *testing.T) {
	resetArgs()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()
	data, err := ioutil.ReadFile("../../../test_data/test_buffered_writer.txt")
	assert.Nil(t, err)

	buf := NewBuffer(ctx, bufferSize, worker_num, oauth, prefix, bkt, obj)
	io.Copy(buf, bytes.NewReader(data))
	assert.Nil(t, err)

	err = buf.Close()
	assert.Nil(t, err)
	client, err := gcsClient(ctx, oauth)
	reader, err := client.Bucket(bkt).Object(obj).NewReader(ctx)
	objBuf := new(bytes.Buffer)
	objBuf.ReadFrom(reader)
	assert.Equal(t, data, objBuf.Bytes())
}

func resetArgs() {
	bufferSize  = 100 * 1024 * 1024
	worker_num  = 4
	prefix = "/tmp"
	bkt = "fionaliu-daisy-bkt-us-east1"
	obj = "obj"
	oauth = ""
}





