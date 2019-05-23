//  Copyright 2018 Google Inc. All Rights Reserved.
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

// Package compute contains wrappers around the GCE compute API.
package compute

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/guest-logging-go/logger"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	computeBeta "google.golang.org/api/compute/v0.beta"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

// Instance is a compute instance.
type Instance struct {
	*compute.Instance
	client        daisyCompute.Client
	Project, Zone string
}

// Cleanup deletes the Instance.
func (i *Instance) Cleanup() {
	if err := i.client.DeleteInstance(i.Project, i.Zone, i.Name); err != nil {
		fmt.Printf("Error deleting instance: %v\n", err)
	}
}

// WaitForSerialOutput waits to a string match on a serial port.
func (i *Instance) WaitForSerialOutput(match string, port int64, interval, timeout time.Duration) error {
	var start int64
	var errs int
	tick := time.Tick(interval)
	timedout := time.Tick(timeout)
	for {
		select {
		case <-timedout:
			return fmt.Errorf("timed out waiting for %q", match)
		case <-tick:
			resp, err := i.client.GetSerialPortOutput(i.Project, i.Zone, i.Name, port, start)
			if err != nil {
				status, sErr := i.client.InstanceStatus(i.Project, i.Zone, i.Name)
				if sErr != nil {
					err = fmt.Errorf("%v, error getting InstanceStatus: %v", err, sErr)
				} else {
					err = fmt.Errorf("%v, InstanceStatus: %q", err, status)
				}

				// Wait until machine restarts to evaluate SerialOutput.
				if isTerminal(status) {
					continue
				}

				// Retry up to 3 times in a row on any error if we successfully got InstanceStatus.
				if errs < 3 {
					errs++
					continue
				}

				return err
			}
			start = resp.Next
			for _, ln := range strings.Split(resp.Contents, "\n") {
				if i := strings.Index(ln, match); i != -1 {
					return nil
				}
			}
			errs = 0
		}
	}
}

// WaitForGuestAttributes waits for guest attribute (queryPath, variableKey) to appear.
func (i *Instance) WaitForGuestAttributes(queryPath, variableKey string, interval, timeout time.Duration) ([]*computeBeta.GuestAttributesEntry, error) {
	tick := time.Tick(interval)
	timedout := time.Tick(timeout)
	for {
		select {
		case <-timedout:
			return nil, fmt.Errorf("timed out waiting for guest attribute %q", path.Join(queryPath, variableKey))
		case <-tick:
			attr, err := i.GetGuestAttributes(queryPath, variableKey)
			if err != nil {
				apiErr, ok := err.(*googleapi.Error)
				if ok && apiErr.Code == http.StatusNotFound {
					continue
				}
				return nil, err
			}
			return attr, nil
		}
	}
}

// GetGuestAttributes gets guest attributes for an instance.
func (i *Instance) GetGuestAttributes(queryPath, variableKey string) ([]*computeBeta.GuestAttributesEntry, error) {
	resp, err := i.client.GetGuestAttributes(i.Project, i.Zone, i.Name, queryPath, variableKey)
	if err != nil {
		return nil, err
	}
	if resp.QueryValue == nil {
		return nil, nil
	}

	return resp.QueryValue.Items, nil
}

// StreamSerialOutput stores the serial output of an instance to GCS bucket
func (i *Instance) StreamSerialOutput(ctx context.Context, storageClient *storage.Client, logsPath, bucket string, logwg *sync.WaitGroup, port int64, interval time.Duration) {
	defer logwg.Done()

	logsObj := path.Join(logsPath, fmt.Sprintf("%s-serial-port%d.log", i.Name, port))
	logger.Infof("Streaming instance %q serial port %d output to https://storage.cloud.google.com/%s/%s", i.Name, port, bucket, logsObj)
	var start int64
	var buf bytes.Buffer
	tick := time.Tick(interval)

	for {
		select {
		case <-tick:
			resp, err := i.client.GetSerialPortOutput(path.Base(i.Project), path.Base(i.Zone), i.Name, port, start)
			if err != nil {
				// Instance is stopped or stopping.
				status, _ := i.client.InstanceStatus(path.Base(i.Project), path.Base(i.Zone), i.Name)
				if !isTerminal(status) {
					logger.Errorf("Instance %q: error getting serial port: %s", i.Name, err)
				}
				return
			}
			start = resp.Next
			wc := storageClient.Bucket(bucket).Object(logsObj).NewWriter(ctx)
			buf.WriteString(resp.Contents)
			wc.ContentType = "text/plain"
			if _, err := wc.Write(buf.Bytes()); err != nil {
				logger.Errorf("Instance %q: error writing log to GCS: %v", i.Name, err)
				continue
			}
			if err := wc.Close(); err != nil {
				logger.Errorf("Instance %q: error saving log to GCS: %v", i.Name, err)
				continue
			}
		}
	}
}

func isTerminal(status string) bool {
	return status == "TERMINATED" || status == "STOPPED" || status == "STOPPING"
}

// CreateInstance creates a compute instance.
func CreateInstance(client daisyCompute.Client, project, zone string, i *compute.Instance) (*Instance, error) {
	logger.Infof("Creating instance %s in zone %s", i.Name, zone)
	if err := client.CreateInstance(project, zone, i); err != nil {
		return nil, err
	}
	return &Instance{Instance: i, client: client, Project: project, Zone: zone}, nil
}

// BuildInstanceMetadataItem create an metadata item
func BuildInstanceMetadataItem(key, value string) *compute.MetadataItems {
	return &compute.MetadataItems{
		Key:   key,
		Value: func() *string { v := value; return &v }(),
	}
}
