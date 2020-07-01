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
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun_HappyCase_CollectAllLogs(t *testing.T) {
	inflaterLogs := []string{"log-a", "log-b"}
	processorLogs := []string{"log-c", "log-d"}
	expectedLogs := []string{"log-a", "log-b", "log-c", "log-d"}
	mockProcessor := mockProcessor{
		serialLogs: processorLogs,
	}
	importer := importer{
		preValidator: mockValidator{},
		inflater: &mockInflater{
			serialLogs: inflaterLogs,
			pd: persistentDisk{
				sizeGb:     100,
				sourceGb:   10,
				sourceType: "vmdk",
			},
		},
		processorProvider: &mockProcessorProvider{
			processor: &mockProcessor,
		},
	}
	loggable, actualError := importer.Run(context.Background())
	assert.Nil(t, actualError)
	assert.NotNil(t, loggable)
	assert.Equal(t, expectedLogs, loggable.ReadSerialPortLogs())
	assert.Equal(t, "vmdk", loggable.GetValue("import-file-format"))
	assert.Equal(t, []int64{10}, loggable.GetValueAsInt64Slice("source-size-gb"))
	assert.Equal(t, []int64{100}, loggable.GetValueAsInt64Slice("target-size-gb"))
	assert.Equal(t, 1, mockProcessor.interactions)
}

func TestRun_DeleteDisk(t *testing.T) {
	project := "project"
	zone := "zone"
	diskURI := "uri"
	mockDiskClient := mockDiskClient{}

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
			processor: &mockProcessor{},
		},
	}
	_, actualError := importer.Run(context.Background())
	assert.NoError(t, actualError)
	assert.Equal(t, 1, mockDiskClient.interactions)
	assert.Equal(t, project, mockDiskClient.project)
	assert.Equal(t, zone, mockDiskClient.zone)
	assert.Equal(t, diskURI, mockDiskClient.uri)
}

func TestRun_DontRunInflate_IfPreValidationFails(t *testing.T) {
	expectedError := errors.New("failed validation")
	inflater := mockInflater{}
	importer := importer{
		preValidator: mockValidator{err: errors.New("failed validation")},
		inflater:     &inflater,
	}
	loggable, actualError := importer.Run(context.Background())
	assert.NotNil(t, loggable)
	assert.Equal(t, expectedError, actualError)
	assert.Equal(t, 0, inflater.interactions)
}

func TestRun_DontRunProcessIfInflateFails(t *testing.T) {
	expectedError := errors.New("the errors")
	mockProcessorProvider := mockProcessorProvider{}
	importer := importer{
		preValidator: mockValidator{},
		inflater: &mockInflater{
			err: expectedError,
		},
		processorProvider: &mockProcessorProvider,
	}
	loggable, actualError := importer.Run(context.Background())
	assert.NotNil(t, loggable)
	assert.Equal(t, expectedError, actualError)
	assert.Equal(t, 0, mockProcessorProvider.interactions)
}

func TestRun_IncludeInflaterLogs_WhenFailureToCreateProcessor(t *testing.T) {
	mockProcessor := mockProcessor{}
	expectedError := errors.New("the errors")
	expectedLogs := []string{"log-a", "log-b"}
	importer := importer{
		preValidator: mockValidator{},
		inflater: &mockInflater{
			serialLogs: expectedLogs,
			pd: persistentDisk{
				sizeGb:     100,
				sourceGb:   10,
				sourceType: "vmdk",
			},
		},
		processorProvider: &mockProcessorProvider{
			err:       expectedError,
			processor: &mockProcessor,
		},
	}
	loggable, actualError := importer.Run(context.Background())
	assert.Equal(t, expectedError, actualError)
	assert.NotNil(t, loggable)
	assert.Equal(t, expectedLogs, loggable.ReadSerialPortLogs())
	assert.Equal(t, "vmdk", loggable.GetValue("import-file-format"))
	assert.Equal(t, []int64{10}, loggable.GetValueAsInt64Slice("source-size-gb"))
	assert.Equal(t, []int64{100}, loggable.GetValueAsInt64Slice("target-size-gb"))
	assert.Equal(t, 0, mockProcessor.interactions)
}

func TestRun_DeleteDisk_WhenFailureToCreateProcessor(t *testing.T) {
	project := "project"
	zone := "zone"
	diskURI := "uri"
	mockDiskClient := mockDiskClient{}

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
	}
	_, actualError := importer.Run(context.Background())
	assert.Equal(t, expectedError, actualError)
	assert.Equal(t, 1, mockDiskClient.interactions)
	assert.Equal(t, project, mockDiskClient.project)
	assert.Equal(t, zone, mockDiskClient.zone)
	assert.Equal(t, diskURI, mockDiskClient.uri)
}

type mockProcessorProvider struct {
	processor    processor
	err          error
	interactions int
}

func (m *mockProcessorProvider) provide(pd persistentDisk) (processor, error) {
	m.interactions++
	return m.processor, m.err
}

type mockProcessor struct {
	serialLogs   []string
	err          error
	interactions int
}

func (m *mockProcessor) process(ctx context.Context) error {
	m.interactions++
	return m.err
}

func (m mockProcessor) traceLogs() []string {
	return m.serialLogs
}

type mockInflater struct {
	serialLogs   []string
	pd           persistentDisk
	err          error
	interactions int
}

func (m *mockInflater) inflate(ctx context.Context) (persistentDisk, error) {
	m.interactions++
	return m.pd, m.err
}

func (m mockInflater) traceLogs() []string {
	return m.serialLogs
}

type mockDiskClient struct {
	interactions       int
	project, zone, uri string
}

func (m *mockDiskClient) DeleteDisk(project, zone, uri string) error {
	m.interactions++
	m.project = project
	m.zone = zone
	m.uri = uri
	return nil
}

type mockValidator struct {
	err error
}

func (m mockValidator) validate() error {
	return m.err
}
