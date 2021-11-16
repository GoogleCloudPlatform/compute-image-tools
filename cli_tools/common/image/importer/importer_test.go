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

package importer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

func TestRun_HappyCase_CollectDiskMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Metric(&pb.OutputInfo{
		SourcesSizeGb:    []int64{10},
		TargetsSizeGb:    []int64{100},
		ImportFileFormat: "vmdk",
	})

	pd := persistentDisk{
		sizeGb:     100,
		sourceGb:   10,
		sourceType: "vmdk",
	}
	mockProcessor := mockProcessor{}
	importer := importer{
		preValidator: mockValidator{},
		inflater: &mockInflater{
			pd: pd,
		},
		processorProvider: &mockProcessorProvider{
			processors: []processor{&mockProcessor},
		},
		logger: mockLogger,
	}
	actualError := importer.Run(context.Background())
	assert.Nil(t, actualError)
	assert.Equal(t, 1, mockProcessor.interactions)
}

func TestRun_DeletesDisk_AfterImportingImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	project := "project"
	zone := "zone"
	diskURI := "uri"
	pd := persistentDisk{
		uri: diskURI,
	}
	mockDiskClient := mockDiskClient{
		disk: &compute.Disk{},
	}

	importer := importer{
		project:      project,
		zone:         zone,
		diskClient:   &mockDiskClient,
		preValidator: mockValidator{},
		inflater: &mockInflater{
			pd: pd,
		},
		processorProvider: &mockProcessorProvider{
			processors: []processor{&mockProcessor{}},
		},
		logger: mockLogger,
	}
	actualError := importer.Run(context.Background())
	assert.NoError(t, actualError)
	assert.Equal(t, 1, mockDiskClient.interactions)
	assert.Equal(t, project, mockDiskClient.project)
	assert.Equal(t, zone, mockDiskClient.zone)
	assert.Equal(t, diskURI, mockDiskClient.uri)
}

func TestRun_NoErrorLoggedWhenDeletingDiskThatWasNotCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	project := "project"
	zone := "zone"
	diskURI := "uri"
	pd := persistentDisk{
		uri: diskURI,
	}
	mockDiskClient := mockDiskClient{
		disk:            nil,
		deleteDiskError: &googleapi.Error{Code: 404},
	}

	importer := importer{
		project:      project,
		zone:         zone,
		diskClient:   &mockDiskClient,
		preValidator: mockValidator{},
		inflater: &mockInflater{
			pd: pd,
		},
		processorProvider: &mockProcessorProvider{
			processors: []processor{&mockProcessor{}},
		},
		logger: mockLogger,
	}
	actualError := importer.Run(context.Background())
	assert.NoError(t, actualError)
	assert.Equal(t, 1, mockDiskClient.interactions)
	assert.Equal(t, project, mockDiskClient.project)
	assert.Equal(t, zone, mockDiskClient.zone)
	assert.Equal(t, "", buf.String())
}

func TestRun_ErrorLoggedWhenErrorDeletingDisk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	project := "project"
	zone := "zone"
	diskURI := "uri"
	pd := persistentDisk{
		uri: diskURI,
	}
	mockDiskClient := mockDiskClient{
		disk:            nil,
		deleteDiskError: &googleapi.Error{Code: 403},
	}

	importer := importer{
		project:      project,
		zone:         zone,
		diskClient:   &mockDiskClient,
		preValidator: mockValidator{},
		inflater: &mockInflater{
			pd: pd,
		},
		processorProvider: &mockProcessorProvider{
			processors: []processor{&mockProcessor{}},
		},
		logger: mockLogger,
	}
	actualError := importer.Run(context.Background())
	assert.NoError(t, actualError)
	assert.Equal(t, 1, mockDiskClient.interactions)
	assert.Equal(t, project, mockDiskClient.project)
	assert.Equal(t, zone, mockDiskClient.zone)
	assert.True(t, strings.Contains(buf.String(), "Failed to remove temporary disk"))
}

func TestRun_DontRunInflate_IfPreValidationFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	expectedError := errors.New("failed validation")
	inflater := mockInflater{}
	importer := importer{
		preValidator: mockValidator{err: errors.New("failed validation")},
		inflater:     &inflater,
		logger:       mockLogger,
	}
	actualError := importer.Run(context.Background())
	assert.Equal(t, expectedError, actualError)
	assert.Equal(t, 0, inflater.interactions)
}

func TestRun_DontRunProcessIfInflateFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	expectedError := errors.New("the errors")
	mockProcessorProvider := mockProcessorProvider{}
	importer := importer{
		preValidator: mockValidator{},
		inflater: &mockInflater{
			err: expectedError,
		},
		processorProvider: &mockProcessorProvider,
		logger:            mockLogger,
	}
	actualError := importer.Run(context.Background())
	assert.Equal(t, expectedError, actualError)
	assert.Equal(t, 0, mockProcessorProvider.interactions)
}

func TestRun_IncludeInflaterLogs_WhenFailureToCreateProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Metric(&pb.OutputInfo{
		SourcesSizeGb:    []int64{10},
		TargetsSizeGb:    []int64{100},
		ImportFileFormat: "vmdk",
	})

	mockProcessor := mockProcessor{}
	expectedError := errors.New("the errors")
	importer := importer{
		preValidator: mockValidator{},
		inflater: &mockInflater{
			serialLogs: []string{"log-a", "log-b"},
			pd: persistentDisk{
				sizeGb:     100,
				sourceGb:   10,
				sourceType: "vmdk",
			},
		},
		processorProvider: &mockProcessorProvider{
			err:        expectedError,
			processors: []processor{&mockProcessor},
		},
		logger: mockLogger,
	}
	actualError := importer.Run(context.Background())
	assert.Equal(t, expectedError, actualError)
	assert.Equal(t, 0, mockProcessor.interactions)
}

func TestRun_DeleteDisk_WhenFailureToCreateProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	project := "project"
	zone := "zone"
	diskURI := "uri"
	mockDiskClient := mockDiskClient{
		disk: &compute.Disk{},
	}

	expectedError := errors.New("the errors")
	importer := importer{
		project:      project,
		zone:         zone,
		diskClient:   &mockDiskClient,
		preValidator: mockValidator{},
		inflater: &mockInflater{
			pd: persistentDisk{
				uri: diskURI,
			},
		},
		processorProvider: &mockProcessorProvider{
			err: expectedError,
		},
		logger: mockLogger,
	}
	actualError := importer.Run(context.Background())
	assert.Equal(t, expectedError, actualError)
	assert.Equal(t, 1, mockDiskClient.interactions)
	assert.Equal(t, project, mockDiskClient.project)
	assert.Equal(t, zone, mockDiskClient.zone)
	assert.Equal(t, diskURI, mockDiskClient.uri)
}

func TestRun_DontRunProcessIfTimedOutDuringInflate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	mockProcessor := mockProcessor{
		processingTime: 10 * time.Second,
	}
	mockProcessorProvider := mockProcessorProvider{
		processors: []processor{&mockProcessor},
	}
	inflater := &mockInflater{
		inflationTime: 5 * time.Second,
	}
	importer := importer{
		preValidator:      mockValidator{},
		inflater:          inflater,
		processorProvider: &mockProcessorProvider,
		logger:            mockLogger,
		timeout:           100 * time.Millisecond,
	}
	start := time.Now()
	actualError := importer.Run(context.Background())
	duration := time.Since(start)

	assert.NotNil(t, actualError)
	assert.Equal(t, 1, inflater.interactions)
	assert.Equal(t, 0, mockProcessorProvider.interactions)
	assert.Equal(t, 0, mockProcessor.interactions)

	// to ensure inflater got interrupted and didn't run the full 10 seconds
	assert.True(t, duration < time.Duration(1)*time.Second)
}

func TestRun_ProcessInterruptedTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	mockProcessor := mockProcessor{
		processingTime: time.Duration(10) * time.Second,
	}
	mockProcessorProvider := mockProcessorProvider{
		processors: []processor{&mockProcessor},
	}
	mockInflater := &mockInflater{}
	importer := importer{
		preValidator:      mockValidator{},
		inflater:          mockInflater,
		processorProvider: &mockProcessorProvider,
		logger:            mockLogger,
		timeout:           time.Duration(100) * time.Millisecond,
	}
	start := time.Now()
	actualError := importer.Run(context.Background())
	duration := time.Since(start)

	assert.NotNil(t, actualError)
	assert.Equal(t, 1, mockInflater.interactions)
	assert.Equal(t, 1, mockProcessor.interactions)

	// to ensure processor got interrupted and didn't run the full 10 seconds
	assert.True(t, duration < time.Duration(1)*time.Second)
}

func TestRun_ProcessCantTimeoutImportSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	mockProcessor := mockProcessor{
		processingTime: time.Duration(1) * time.Second,
		cantCancel:     true,
	}
	mockProcessorProvider := mockProcessorProvider{
		processors: []processor{&mockProcessor},
	}
	mockInflater := &mockInflater{}
	importer := importer{
		preValidator:      mockValidator{},
		inflater:          mockInflater,
		processorProvider: &mockProcessorProvider,
		logger:            mockLogger,
		timeout:           time.Duration(100) * time.Millisecond,
	}
	start := time.Now()
	actualError := importer.Run(context.Background())
	duration := time.Since(start)

	assert.Nil(t, actualError)
	assert.Equal(t, 1, mockInflater.interactions)
	assert.Equal(t, 1, mockProcessor.interactions)
	assert.True(t, duration > importer.timeout)
}

func TestRunStep_VeryShortTimeout(t *testing.T) {
	// This test ensures that inflater.runStep doesn't dead lock when timeout
	// has already passed before step function is able to run.

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	doTestWithTimeOut(t, 2*time.Second, func(t *testing.T) {
		// run this test with a timeout as it might never finish if there is a bug

		importer := importer{
			preValidator:      mockValidator{},
			inflater:          &mockInflater{},
			processorProvider: &mockProcessorProvider{},
			logger:            mockLogger,
			// this simulates a very short timeout, or a situation when a timeout
			// occurs immediately after one step finishes and the next one is about
			// to start
			timeout: 0 * time.Millisecond,
		}

		ctx, cancel := context.WithTimeout(context.Background(), importer.timeout)
		defer cancel()

		cancelChan := make(chan bool)
		didStepRun := false
		importer.runStep(ctx,
			func() error {
				// step
				didStepRun = true
				select {
				case <-cancelChan:
					break
				case <-time.After(time.Second * 5):
					break
				}
				return nil
			},
			func(string) bool {
				// cancel
				select {
				case cancelChan <- true:
					break
				default:
					break
				}
				return true
			})
		assert.False(t, didStepRun)
	})
}

// doTestWithTimeOut allows a test to be run for a predefined amount of time.
// If this time passes, the test fails
func doTestWithTimeOut(t *testing.T, timeout time.Duration, test func(t *testing.T)) {
	timeoutChan := time.After(timeout)
	done := make(chan bool)
	go func() {
		test(t)
		done <- true
	}()
	select {
	case <-timeoutChan:
		t.Fatal("Test timed out")
	case <-done:
	}
}

type mockProcessorProvider struct {
	processors   []processor
	err          error
	interactions int
}

func (m *mockProcessorProvider) provide(pd persistentDisk) ([]processor, error) {
	m.interactions++
	return m.processors, m.err
}

type mockProcessor struct {
	serialLogs     []string
	err            error
	interactions   int
	processingTime time.Duration
	processingChan chan bool
	cantCancel     bool
	cancelChan     chan bool
}

func (m *mockProcessor) process(pd persistentDisk) (persistentDisk, error) {
	m.interactions++
	m.cancelChan = make(chan bool)

	if m.processingTime > 0 {
		select {
		case <-m.cancelChan:
			break
		case <-time.After(m.processingTime):
			break
		}
	}

	return pd, m.err
}

func (m *mockProcessor) cancel(reason string) bool {
	m.cancelChan <- true
	return !m.cantCancel
}

type mockInflater struct {
	serialLogs    []string
	pd            persistentDisk
	ii            inflationInfo
	err           error
	interactions  int
	inflationTime time.Duration
	cancelChan    chan bool
}

func (m *mockInflater) Inflate() (persistentDisk, inflationInfo, error) {
	m.interactions++
	m.cancelChan = make(chan bool)

	if m.inflationTime > 0 {
		select {
		case <-m.cancelChan:
			return m.pd, m.ii, fmt.Errorf("cancelled inflater")
		case <-time.After(m.inflationTime):
			break
		}
	}
	return m.pd, m.ii, m.err
}

func (m *mockInflater) Cancel(reason string) bool {
	m.cancelChan <- true
	return true
}

type mockDiskClient struct {
	interactions                 int
	project, zone, uri, diskName string
	disk                         *compute.Disk
	deleteDiskError              error
}

func (m *mockDiskClient) DeleteDisk(project, zone, uri string) error {
	m.interactions++
	m.project = project
	m.zone = zone
	m.uri = uri
	return m.deleteDiskError
}

type mockValidator struct {
	err error
}

func (m mockValidator) validate() error {
	return m.err
}
