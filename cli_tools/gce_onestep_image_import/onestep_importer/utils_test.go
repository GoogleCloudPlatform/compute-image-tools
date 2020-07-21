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
	"io/ioutil"
	"log"
	"os"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHandleTimeout(t *testing.T) {
	testTimeoutChan := make(chan struct{})
	const sleepTime = 5

	start := time.Now()
	handleTimeout(sleepTime*time.Second, testTimeoutChan)
	sec := time.Since(start).Seconds()

	if sec < sleepTime || sec > sleepTime*1.05 {
		t.Error("Incorrect sleep function")
	}
}

func TestNewImporter(t *testing.T) {
	_, err := newImporterForCloudProvider(expectSuccessfulParse(t))
	assert.Nil(t, err)
}

func TestImportReturnOnTimeoutLessThan3Minutes(t *testing.T) {
	args := expectSuccessfulParse(t, "-timeout=0s")
	err := importFromCloudProvider(args)
	assert.EqualError(t, err, "timeout exceeded: timeout must be at least 3 minutes")
}

func TestImportReturnOnTimeoutExceeded(t *testing.T) {
	args := expectSuccessfulParse(t)
	close(args.TimeoutChan)
	err := importFromCloudProvider(args)
	assert.EqualError(t, err, "timeout exceeded")
}

func TestImportReturnOnError(t *testing.T) {
	// missing AWS related args
	args := expectSuccessfulParse(t)
	err := importFromCloudProvider(args)
	assert.NotNil(t, err)
	assert.Regexp(t, ".*must be provided", err.Error())
}

func TestUploaderCopiesFile(t *testing.T) {
	const data = "test"
	// writer
	var output bytes.Buffer
	writerCloser := testWriteCloser{Writer: bufio.NewWriter(&output)}
	// reader
	r := ioutil.NopCloser(bytes.NewReader([]byte(data)))

	uploader := getTestUploader(writerCloser)
	uploader.Add(1)
	go uploader.uploadFile()

	// uploader should copy from reader to writer
	uploader.readerChan <- r
	close(uploader.readerChan)
	uploader.Wait()
	assert.Equal(t, output.String(), data)
}

func TestUploaderOutputsProgress(t *testing.T) {
	// Set log output destination
	var buf bytes.Buffer
	log.SetOutput(&buf)

	var output bytes.Buffer
	writerCloser := testWriteCloser{Writer: bufio.NewWriter(&output)}
	r := ioutil.NopCloser(bytes.NewReader([]byte("test")))

	uploader := getTestUploader(writerCloser)
	uploader.Add(1)
	go uploader.uploadFile()

	uploader.readerChan <- r
	close(uploader.readerChan)
	uploader.Wait()
	assert.Contains(t, buf.String(), "Total written size: 4 B of 100 B.")
}

func TestUploaderHasErrorWhenCopyFail(t *testing.T) {
	var output bytes.Buffer
	writerCloser := testWriteCloser{Writer: bufio.NewWriter(&output)}

	// iotest.TimeoutReader will return error on second empty read.
	r := ioutil.NopCloser(iotest.TimeoutReader(bytes.NewReader([]byte(""))))

	uploader := getTestUploader(writerCloser)
	uploader.Add(1)
	go uploader.uploadFile()

	// Try to upload from TimeoutReader twice.
	uploader.readerChan <- r
	uploader.readerChan <- r

	close(uploader.readerChan)
	assert.Equal(t, <-uploader.uploadErrChan, iotest.ErrTimeout)
}

func TestUploaderCleanUp(t *testing.T) {
	var output bytes.Buffer
	writerCloser := testWriteCloser{Writer: bufio.NewWriter(&output)}
	r := ioutil.NopCloser(bytes.NewReader([]byte("test")))

	uploader := getTestUploader(writerCloser)
	uploader.Add(1)
	uploader.readerChan <- r
	assert.Len(t, uploader.readerChan, 1)

	uploader.cleanup()
	assert.Len(t, uploader.readerChan, 0)
}

func TestUploderRunsImplementation(t *testing.T) {
	isUploadFileFuncCalled := false
	isCleanupFuncCalled := false
	u := uploader{
		uploadFileFn: func() {
			isUploadFileFuncCalled = true
		},
		cleanupFn: func() {
			isCleanupFuncCalled = true
		}}
	u.uploadFile()
	u.cleanup()
	assert.True(t, isUploadFileFuncCalled)
	assert.True(t, isCleanupFuncCalled)
}

func TestRunImageImportFailedWhenCmdError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	args, _ := NewOneStepImportArguments([]string{})
	err := runImageImport(args)
	assert.Regexp(t, "Running command: .*gce_vm_image_import", buf.String())
	assert.Contains(t, err.Error(), "failed to import image")
}

func TestRunCmd(t *testing.T) {
	actual := "testingRunCmd"
	oldOut := os.Stdout
	rout, wout, _ := os.Pipe()
	os.Stdout = wout

	err := runCmd("echo", []string{actual})
	assert.Nil(t, err)
	err = wout.Close()
	assert.Nil(t, err)
	expected, err := ioutil.ReadAll(rout)
	assert.Nil(t, err)
	os.Stdout = oldOut

	assert.Contains(t, string(expected), actual)
}
