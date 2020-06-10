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

package daisy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

const (
	port     = int64(2)
	project  = "test-project"
	zone     = "test-zone"
	instance = "test-instance"
)

func TestSerialPortWatcher_AllowsMultipleSubscribers(t *testing.T) {
	c1 := make(chan string, 10)
	c2 := make(chan string, 10)

	client := &mockSerialPortClient{
		expectedProject:  project,
		expectedZone:     zone,
		expectedInstance: instance,
		expectedPort:     port,
		t:                t,
		responses: []serialPortResponse{
			{200, "one"},
			{200, "two"},
			{404, ""},
			{404, ""},
			{404, ""},
		},
	}
	watcher := NewSerialOutputWatcher(client, project, zone)
	watcher.Watch(instance, port, c1, time.Nanosecond)
	watcher.Watch(instance, port, c2, time.Hour)
	watcher.start(instance)

	// Regardless of the poll threshold, the first message is read immediately.
	assert.Equal(t, "one", <-c1)
	assert.Equal(t, "one", <-c2)

	// Sleeping occurs after the first poll. If we didn't pick the smallest poll threshold,
	// then this test would require an hour to pass.
	assert.Equal(t, "two", <-c1)
	assert.Equal(t, "two", <-c2)
}

func TestSerialPortWatcher_AllSubscribedPortsPolled_WhenStartCalled(t *testing.T) {
	c1 := make(chan string, 10)
	port1 := int64(1)
	client1 := &mockSerialPortClient{
		expectedProject:  project,
		expectedZone:     zone,
		expectedInstance: instance,
		expectedPort:     port1,
		t:                t,
		responses: []serialPortResponse{
			{200, "1_one"},
			{200, "1_two"},
			{404, ""},
			{404, ""},
			{404, ""},
		},
	}

	c2 := make(chan string, 10)
	port2 := int64(2)
	client2 := &mockSerialPortClient{
		expectedProject:  project,
		expectedZone:     zone,
		expectedInstance: instance,
		expectedPort:     port2,
		t:                t,
		responses: []serialPortResponse{
			{200, "2_one"},
			{200, "2_two"},
			{404, ""},
			{404, ""},
			{404, ""},
		},
	}

	watcher := NewSerialOutputWatcher(mockMultiSerialPortClient{map[int64]*mockSerialPortClient{
		port1: client1, port2: client2,
	}}, project, zone)
	watcher.Watch(instance, 1, c1, time.Nanosecond)
	watcher.Watch(instance, 2, c2, time.Nanosecond)
	watcher.start(instance)

	assert.Equal(t, "1_one", <-c1)
	assert.Equal(t, "2_one", <-c2)
	assert.Equal(t, "1_two", <-c1)
	assert.Equal(t, "2_two", <-c2)
}

func TestSerialPortWatcher_StopsWhenInstanceDeleted(t *testing.T) {
	assert.Equal(t, 3, deletionRetry404s, "These tests assume that deletionRetry404s is 3.")
	type testCase struct {
		name           string
		expectedOutput string
		responses      []serialPortResponse
	}
	for _, tt := range []testCase{
		{
			name:           "Tolerate intermittent 404s, but stop when deletionRetry404s reached.",
			expectedOutput: "one two three",
			responses: []serialPortResponse{
				{200, "one "},
				{404, ""},
				{200, "two "},
				{404, ""},
				{404, ""},
				{200, "three"},
				{404, ""},
				{404, ""},
				{404, ""},
				{200, "four"},
			},
		},
		{
			name:           "Allow unlimited 404s prior to reading first log",
			expectedOutput: "log",
			responses: []serialPortResponse{
				{404, ""},
				{404, ""},
				{404, ""},
				{404, ""},
				{404, ""},
				{404, ""},
				{200, "log"},
				{404, ""},
				{404, ""},
				{404, ""},
			},
		},
		{
			name:           "Ignore non-404s",
			expectedOutput: "one two",
			responses: func() []serialPortResponse {
				// This assembles a slice of serialPortResponses with the following:
				//  - real content
				//  - all HTTP codes, other than 200 and 404
				//  - real content
				//  - three 404s
				resp := []serialPortResponse{
					{200, "one "},
				}
				for i := 100; i < 600; i++ {
					if i == 404 || i == 200 {
						continue
					}
					resp = append(resp, serialPortResponse{i, ""},
						serialPortResponse{i, ""}, serialPortResponse{i, ""})
				}
				// Finally end with a single message, plus sufficient 404s to
				// signify the instance is deleted.
				resp = append(resp, serialPortResponse{200, "two"})
				return append(resp, serialPortResponse{404, ""},
					serialPortResponse{404, ""}, serialPortResponse{404, ""})
			}(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {

			c := make(chan string, 100)
			pollingFrequency := time.Nanosecond

			client := &mockSerialPortClient{
				expectedProject:  project,
				expectedZone:     zone,
				expectedInstance: instance,
				expectedPort:     port,
				t:                t,
				responses:        tt.responses,
			}

			watcher := NewSerialOutputWatcher(client, project, zone)
			watcher.Watch(instance, port, c, pollingFrequency)
			watcher.start(instance)

			actual := ""
			read := <-c
			for read != "" {
				actual += read
				read = <-c
			}

			assert.Equal(t, tt.expectedOutput, actual)
		})
	}
}

type serialPortResponse struct {
	code    int
	content string
}

type mockSerialPortClient struct {
	expectedProject  string
	expectedZone     string
	expectedInstance string
	expectedPort     int64
	expectedStart    int64
	t                *testing.T
	responses        []serialPortResponse
}

func (m *mockSerialPortClient) GetSerialPortOutput(
	project, zone, instanceName string, port, start int64) (*compute.SerialPortOutput, error) {
	assert.Equal(m.t, m.expectedProject, project)
	assert.Equal(m.t, m.expectedZone, zone)
	assert.Equal(m.t, m.expectedInstance, instanceName)
	assert.Equal(m.t, m.expectedPort, port)
	assert.Equal(m.t, m.expectedStart, start)
	var curr serialPortResponse
	curr, m.responses = m.responses[0], m.responses[1:]
	if curr.code/100 == 2 {
		thisStart := m.expectedStart
		m.expectedStart += int64(len(curr.content))
		return &compute.SerialPortOutput{
			Contents: curr.content,
			Next:     m.expectedStart,
			SelfLink: "",
			Start:    thisStart,
			ServerResponse: googleapi.ServerResponse{
				HTTPStatusCode: curr.code,
			},
		}, nil
	}
	return nil, &googleapi.Error{Code: curr.code}
}

type mockMultiSerialPortClient struct {
	delegates map[int64]*mockSerialPortClient
}

func (m mockMultiSerialPortClient) GetSerialPortOutput(
	project, zone, instanceName string, port, start int64) (*compute.SerialPortOutput, error) {
	return m.delegates[port].GetSerialPortOutput(project, zone, instanceName, port, start)
}
