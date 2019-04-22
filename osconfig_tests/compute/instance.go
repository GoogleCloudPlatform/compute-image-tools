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
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"path"
	"strings"
	"sync"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	api "google.golang.org/api/compute/v1"
)

// Instance is a compute instance.
type Instance struct {
	*api.Instance
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
				if status == "TERMINATED" || status == "STOPPED" || status == "STOPPING" {
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

// StreamSerialOutput stores the serial output of an instance to GCS bucket
func (i *Instance)StreamSerialOutput(ctx context.Context, storageClient *storage.Client, logsPath, bucket string, wg *sync.WaitGroup, port int64, interval time.Duration) {
	defer wg.Done()

	logsObj := path.Join(logsPath, fmt.Sprintf("%s-serial-port%d.log", i.Name, port))
	logger.Infof("Streaming instance %q serial port %d output to https: //storage.cloud.google.com/%s/%s", i.Name, port, bucket, logsObj)
	var start int64
	var buf bytes.Buffer
	var gcsErr bool
	tick := time.Tick(interval)

Loop:
	for {
		select {
		case <-tick:
			resp, err := i.client.GetSerialPortOutput(path.Base(i.Project), path.Base(i.Zone), i.Name, port, start)
			if err != nil {
				// Instance is stopped or stopping.
				status, sErr := i.client.InstanceStatus(path.Base(i.Project), path.Base(i.Zone), i.Name)
				switch status {
				case "TERMINATED", "STOPPED", "STOPPING":
					if sErr == nil {
						break Loop
					}
				}
				logger.Errorf("Instance %q: error getting serial port: %v", i.Name, err)
				break Loop
			}
			start = resp.Next
			buf.WriteString(resp.Contents)
			wc := storageClient.Bucket(bucket).Object(logsObj).NewWriter(ctx)
			wc.ContentType = "text/plain"
			if _, err := wc.Write(buf.Bytes()); err != nil && !gcsErr {
				gcsErr = true
				logger.Errorf("Instance %q: error writing log to GCS: %v", i.Name, err)
				continue
			}
			if err := wc.Close(); err != nil && !gcsErr {
				gcsErr = true
				logger.Errorf("Instance %q: error saving log to GCS: %v", i.Name, err)
				continue
			}
		}
	}
}

// CreateInstance creates a compute instance.
func CreateInstance(client daisyCompute.Client, project, zone string, i *api.Instance) (*Instance, error) {
	if err := client.CreateInstance(project, zone, i); err != nil {
		return nil, err
	}
	return &Instance{Instance: i, client: client, Project: project, Zone: zone}, nil
}

// BuildInstanceMetadataItem create an metadata item
func BuildInstanceMetadataItem(key, value string) *api.MetadataItems {
	return &api.MetadataItems{
		Key:   key,
		Value: func() *string { v := value; return &v }(),
	}
}
