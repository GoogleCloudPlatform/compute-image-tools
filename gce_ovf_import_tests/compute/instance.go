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

// Package compute contains wrappers around the GCE compute API.
package compute

import (
	"context"
	"fmt"
	"strings"
	"time"

	computeApi "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	api "google.golang.org/api/compute/v1"
)

// Instance is a compute instance.
type Instance struct {
	*api.Instance
	Client        daisyCompute.Client
	Project, Zone string
	IsWindows     bool
}

// Cleanup deletes the Instance.
func (i *Instance) Cleanup() error {
	return i.Client.DeleteInstance(i.Project, i.Zone, i.Name)
}

func (i *Instance) StartWithScript(script string) error {
	startupScriptKey := "startup-script"
	if i.IsWindows {
		startupScriptKey = "windows-startup-script-ps1"
	}
	err := i.Client.SetInstanceMetadata(i.Project, i.Zone,
		i.Name, &api.Metadata{Items: []*api.MetadataItems{BuildInstanceMetadataItem(
			startupScriptKey, script)},
			Fingerprint: i.Metadata.Fingerprint})

	if err != nil {
		return err
	}

	if err = i.Client.StartInstance(i.Project, i.Zone, i.Name); err != nil {
		return err
	}
	return nil
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
			resp, err := i.Client.GetSerialPortOutput(i.Project, i.Zone, i.Name, port, start)
			if err != nil {
				status, sErr := i.Client.InstanceStatus(i.Project, i.Zone, i.Name)
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
				if i := strings.Index(strings.ToLower(ln), strings.ToLower(match)); i != -1 {
					return nil
				}
			}
			errs = 0
		}
	}
}

// CreateImageObject creates an image object to be operated by API client
func CreateInstanceObject(ctx context.Context, project string, zone string, name string, isWindows bool) (*Instance, error) {
	client, err := computeApi.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	var apiInstance *api.Instance
	apiInstance, err = client.GetInstance(project, zone, name)
	return &Instance{apiInstance, client, project, zone, isWindows}, err
}

// BuildInstanceMetadataItem create an metadata item
func BuildInstanceMetadataItem(key, value string) *api.MetadataItems {
	return &api.MetadataItems{
		Key:   key,
		Value: func() *string { v := value; return &v }(),
	}
}
