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
	"fmt"
	"net/http"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

// SerialOutputWatcher provides a pubsub mechanism for serial port output. Subscribers
// call Watch with a channel and requested duration. Logs are relayed to
// the channel c as they are received from the instance.
//
// The SerialOutputWatcher signal will not block sending to c: the caller must ensure
// that c has sufficient buffer space to keep up with the expected
// signal rate. For a channel that is immediately consumed, a buffer of size 1 is
// sufficient.
//
// Watch needs to be called prior to the instance's CreateInstance step being run.
// Invocations after that point will be ignored.
//
// Polling occurs using the shortest interval of all subscribers,
// and continues until the instance is deleted. It is guaranteed
// to continue through a reboot. For a reference of instance lifecycles, see:
//
// https://cloud.google.com/compute/docs/instances/instance-life-cycle
type SerialOutputWatcher interface {
	Watch(instanceName string, port int64, c chan<- string, interval time.Duration)
	start(instanceName string)
}

// NewSerialOutputWatcher constructs a new instance of SerialOutputWatcher using the GCP
// compute API to make periodic polling requests to check for new serial output.
func NewSerialOutputWatcher(client serialOutputClient, project, zone string) SerialOutputWatcher {
	return &serialOutputWatcher{
		subscriptions: map[serialPort][]watch{},
		client:        client,
		project:       project,
		zone:          zone,
	}
}

type serialOutputWatcher struct {
	subscriptions map[serialPort][]watch
	client        serialOutputClient
	project, zone string
}

// serialOutputWatcher assumes that 404s are returned when an instance doesn't
// exist. The value `deletionRetry404s` determines how many subsequent HTTP 404 responses to
// see prior to deciding the instance has been deleted (assuming the
// instance was seen earlier).
//
// Although a value of 1 is theoretically sufficient, a higher value allows withstanding
// unexpected intermittent failures.
//
// At the time of writing, the following HTTP error codes were seen:
//  - Prior to creating instance: 404
//  - Once deletion finished: 404
//  - Instance stopped, but not deleted: 400
//  - Serial output successfully returned: 200
const deletionRetry404s = 3

type serialPort struct {
	instance   string
	portNumber int64
}

type watch struct {
	channel         chan<- string
	pollingInterval time.Duration
}

func (s *serialOutputWatcher) Watch(
	instanceName string, port int64, c chan<- string, interval time.Duration) {
	sp := serialPort{instanceName, port}
	s.subscriptions[sp] = append(s.subscriptions[sp], watch{c, interval})
}

func (s *serialOutputWatcher) start(instanceName string) {
	for serialPort := range s.subscriptions {
		if serialPort.instance != instanceName {
			continue
		}
		callbacks := s.subscriptions[serialPort]
		if len(callbacks) == 0 {
			panic(fmt.Sprintf("Start called without subscribers for %v", serialPort))
		}
		var subscribers []chan<- string
		pollingFrequency := callbacks[0].pollingInterval
		for _, cb := range callbacks {
			subscribers = append(subscribers, cb.channel)
			if cb.pollingInterval < pollingFrequency {
				cb.pollingInterval = pollingFrequency
			}
		}
		watchOnePort(s.client, pollingFrequency, s.project, s.zone, serialPort.instance, serialPort.portNumber, subscribers)
	}
}

// watchOnePort starts a goroutine that polls for an instance's serial output. Output is written to the
// subscribing channels. Channels are closed when the instance is deleted.
func watchOnePort(client serialOutputClient, pollingFrequency time.Duration,
	project, zone, instanceName string, port int64, subscribers []chan<- string) {
	go func() {
		seenInstance := false
		consecutive404s := 0
		// The GCE serial port API implements paging using a cursor of byte offsets:
		// https://cloud.google.com/compute/docs/reference/rest/v1/instances/getSerialPortOutput
		readPosition := int64(0)
		ticker := time.NewTicker(pollingFrequency)
		for {
			resp, err := client.GetSerialPortOutput(project, zone, instanceName, port, readPosition)
			var httpCode int
			if err != nil {
				realErr := err.(*googleapi.Error)
				httpCode = realErr.Code
			} else {
				readPosition = resp.Next
				httpCode = resp.HTTPStatusCode
				seenInstance = true
				if len(resp.Contents) > 0 {
					for _, c := range subscribers {
						select {
						case c <- resp.Contents:
						default:
						}
					}
				}
			}
			if seenInstance && httpCode == http.StatusNotFound {
				consecutive404s++
				if consecutive404s >= deletionRetry404s {
					for _, c := range subscribers {
						close(c)
					}
					return
				}
			} else {
				consecutive404s = 0
			}
			// Sleep at the end to ensure the first read occurs immediately.
			<-ticker.C
		}
	}()
}

// serialOutputClient is the subset of the compute API used by the reader's daemon.
type serialOutputClient interface {
	GetSerialPortOutput(project, zone, instanceName string, port, start int64) (*compute.SerialPortOutput, error)
}
