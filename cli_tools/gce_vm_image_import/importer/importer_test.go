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
	finisherLogs := []string{"log-c", "log-d"}
	expectedLogs := []string{"log-a", "log-b", "log-c", "log-d"}
	mockFinisher := mockFinisher{
		serialLogs: finisherLogs,
	}
	importer := importer{
		inflater: &mockInflater{
			serialLogs: inflaterLogs,
			pd: pd{
				sizeGb:     100,
				sourceGb:   10,
				sourceType: "vmdk",
			},
		},
		finisherProvider: &mockFinisherProvider{
			finisher: &mockFinisher,
		},
	}
	loggable, actualError := importer.Run(context.Background())
	assert.Nil(t, actualError)
	assert.NotNil(t, loggable)
	assert.Equal(t, expectedLogs, loggable.ReadSerialPortLogs())
	assert.Equal(t, "vmdk", loggable.GetValue("import-file-format"))
	assert.Equal(t, []int64{10}, loggable.GetValueAsInt64Slice("source-size-gb"))
	assert.Equal(t, []int64{100}, loggable.GetValueAsInt64Slice("target-size-gb"))
	assert.Equal(t, 1, mockFinisher.interactions)
}

func TestRun_DeleteDisk(t *testing.T) {
	project := "project"
	zone := "zone"
	diskURI := "uri"
	mockDiskClient := mockDiskClient{}

	importer := importer{
		project:    project,
		zone:       zone,
		diskClient: &mockDiskClient,
		inflater: &mockInflater{
			pd: pd{
				uri: diskURI,
			},
		},
		finisherProvider: &mockFinisherProvider{
			finisher: &mockFinisher{},
		},
	}
	_, actualError := importer.Run(context.Background())
	assert.NoError(t, actualError)
	assert.Equal(t, 1, mockDiskClient.interactions)
	assert.Equal(t, project, mockDiskClient.project)
	assert.Equal(t, zone, mockDiskClient.zone)
	assert.Equal(t, diskURI, mockDiskClient.uri)
}

func TestRun_DontRunFinishIfInflateFails(t *testing.T) {
	expectedError := errors.New("the errors")
	mockFinisherProvider := mockFinisherProvider{}
	importer := importer{
		inflater: &mockInflater{
			err: expectedError,
		},
		finisherProvider: &mockFinisherProvider,
	}
	loggable, actualError := importer.Run(context.Background())
	assert.NotNil(t, loggable)
	assert.Equal(t, expectedError, actualError)
	assert.Equal(t, 0, mockFinisherProvider.interactions)
}

func TestRun_IncludeInflaterLogs_WhenFailureToCreateFinisher(t *testing.T) {
	mockFinisher := mockFinisher{}
	expectedError := errors.New("the errors")
	expectedLogs := []string{"log-a", "log-b"}
	importer := importer{
		inflater: &mockInflater{
			serialLogs: expectedLogs,
			pd: pd{
				sizeGb:     100,
				sourceGb:   10,
				sourceType: "vmdk",
			},
		},
		finisherProvider: &mockFinisherProvider{
			err:      expectedError,
			finisher: &mockFinisher,
		},
	}
	loggable, actualError := importer.Run(context.Background())
	assert.Equal(t, expectedError, actualError)
	assert.NotNil(t, loggable)
	assert.Equal(t, expectedLogs, loggable.ReadSerialPortLogs())
	assert.Equal(t, "vmdk", loggable.GetValue("import-file-format"))
	assert.Equal(t, []int64{10}, loggable.GetValueAsInt64Slice("source-size-gb"))
	assert.Equal(t, []int64{100}, loggable.GetValueAsInt64Slice("target-size-gb"))
	assert.Equal(t, 0, mockFinisher.interactions)
}

func TestRun_DeleteDisk_WhenFailureToCreateFinisher(t *testing.T) {
	project := "project"
	zone := "zone"
	diskURI := "uri"
	mockDiskClient := mockDiskClient{}

	expectedError := errors.New("the errors")
	importer := importer{
		project:    project,
		zone:       zone,
		diskClient: &mockDiskClient,
		inflater: &mockInflater{
			pd: pd{
				uri: diskURI,
			},
		},
		finisherProvider: &mockFinisherProvider{
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

type mockFinisherProvider struct {
	finisher     finisher
	err          error
	interactions int
}

func (m *mockFinisherProvider) provide(pd pd) (finisher, error) {
	m.interactions++
	return m.finisher, m.err
}

type mockFinisher struct {
	serialLogs   []string
	err          error
	interactions int
}

func (m *mockFinisher) run(ctx context.Context) error {
	m.interactions++
	return m.err
}

func (m mockFinisher) serials() []string {
	return m.serialLogs
}

type mockInflater struct {
	serialLogs   []string
	pd           pd
	err          error
	interactions int
}

func (m *mockInflater) inflate(ctx context.Context) (pd, error) {
	m.interactions++
	return m.pd, m.err
}

func (m mockInflater) serials() []string {
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
