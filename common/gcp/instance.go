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

package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiBeta "google.golang.org/api/compute/v0.beta"
	api "google.golang.org/api/compute/v1"

	daisyCompute "github.com/GoogleCloudPlatform/compute-daisy/compute"
)

// Instance is a compute instance.
type Instance struct {
	*api.Instance
	Client        daisyCompute.Client
	Project, Zone string
	IsWindows     bool
}

// InstanceBeta is a compute instance using Beta API.
type InstanceBeta struct {
	*apiBeta.Instance
	Client        daisyCompute.Client
	Project, Zone string
	IsWindows     bool
}

// Cleanup deletes the Instance.
func (i *Instance) Cleanup() error {
	return i.Client.DeleteInstance(i.Project, i.Zone, i.Name)
}

// RestartWithScriptCode restarts the instance with given startup script.
func (i *Instance) RestartWithScriptCode(script string) error {
	err := i.Client.StopInstance(i.Project, i.Zone, i.Name)
	if err != nil {
		return err
	}
	return i.StartWithScriptCode(script, nil)
}

// StartWithScriptCode starts the instance with given startup script and metadata
func (i *Instance) StartWithScriptCode(script string, instanceMetadata map[string]string) error {
	if instanceMetadata == nil {
		instanceMetadata = make(map[string]string)
	}

	if i.IsWindows {
		instanceMetadata["windows-startup-script-ps1"] = script
	} else {
		instanceMetadata["startup-script"] = script
	}

	var metadataItems []*api.MetadataItems
	for k, v := range instanceMetadata {
		metadataItems = append(metadataItems, BuildInstanceMetadataItem(k, v))
	}
	err := i.Client.SetInstanceMetadata(i.Project, i.Zone,
		i.Name, &api.Metadata{Items: metadataItems, Fingerprint: i.Metadata.Fingerprint})

	if err != nil {
		return err
	}

	if err = i.Client.StartInstance(i.Project, i.Zone, i.Name); err != nil {
		return err
	}
	return nil
}

// WaitForSerialOutput waits to a string match on a serial port.
func (i *Instance) WaitForSerialOutput(match string, failureMatches []string, port int64, interval, timeout time.Duration) error {
	return WaitForSerialOutput(match, failureMatches, port, interval, timeout, i.Project, i.Zone, i.Name, i.Client)
}

// WaitForSerialOutput waits to a string match on a serial port.
func (i *InstanceBeta) WaitForSerialOutput(match string, failureMatches []string, port int64, interval, timeout time.Duration) error {
	return WaitForSerialOutput(match, failureMatches, port, interval, timeout, i.Project, i.Zone, i.Name, i.Client)
}

// WaitForSerialOutput waits to a string match on a serial port.
func WaitForSerialOutput(successMatch string, failureMatches []string, port int64, interval, timeout time.Duration, project, zone, instanceName string, client daisyCompute.Client) error {
	var start int64
	var errs int
	tick := time.Tick(interval)
	timedout := time.Tick(timeout)
	for {
		select {
		case <-timedout:
			return fmt.Errorf("timed out waiting for %q", successMatch)
		case <-tick:
			resp, err := client.GetSerialPortOutput(project, zone, instanceName, port, start)
			if err != nil {
				status, sErr := client.InstanceStatus(project, zone, instanceName)
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
				if len(failureMatches) > 0 {
					for _, failureMatch := range failureMatches {
						if i := strings.Index(ln, failureMatch); i != -1 {
							errMsg := strings.TrimSpace(ln[i:])
							format := "WaitForSerialOutput FailureMatch found for %q: %q"
							return fmt.Errorf(format, instanceName, errMsg)
						}
					}
				}
				if successMatch != "" {
					if i := strings.Index(strings.ToLower(ln), strings.ToLower(successMatch)); i != -1 {
						return nil
					}
				}
			}
			errs = 0
		}
	}
}

// SetMetadata sets metadata for the given instance.
func SetMetadata(ctx context.Context, project, zone, name, key, value string, isWindows bool) (*Instance, error) {
	i, err := CreateInstanceObject(ctx, project, zone, name, isWindows)
	if err != nil {
		return nil, err
	}
	err = i.Client.SetInstanceMetadata(i.Project, i.Zone,
		i.Name, &api.Metadata{Items: []*api.MetadataItems{BuildInstanceMetadataItem(
			key, value)},
			Fingerprint: i.Metadata.Fingerprint})
	return i, err
}

// CreateInstanceObject creates an instance object to be operated by GA API client
func CreateInstanceObject(ctx context.Context, project string, zone string, name string, isWindows bool) (*Instance, error) {
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	var apiInstance *api.Instance
	apiInstance, err = client.GetInstance(project, zone, name)
	return &Instance{apiInstance, client, project, zone, isWindows}, err
}

// CreateInstanceBetaObject creates an instance object to be operated by Beta API client
func CreateInstanceBetaObject(ctx context.Context, project string, zone string, name string, isWindows bool) (*InstanceBeta, error) {
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	var apiInstance *apiBeta.Instance
	apiInstance, err = client.GetInstanceBeta(project, zone, name)
	return &InstanceBeta{apiInstance, client, project, zone, isWindows}, err
}

// CreateMachineImageObject creates a machine image object
func CreateMachineImageObject(ctx context.Context, project string, name string) (*api.MachineImage, error) {
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetMachineImage(project, name)
}

// CreateInstanceBeta creates a VM instance (not just an object representing an existing VM) using Beta API
func CreateInstanceBeta(ctx context.Context, project string, zone string, name string,
	isWindows bool, machineImageName string) (*InstanceBeta, error) {
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	apiBetaInstance := &apiBeta.Instance{
		SourceMachineImage: fmt.Sprintf("projects/%s/global/machineImages/%s", project, machineImageName),
		Name:               name,
		Zone:               zone,
	}
	i := &InstanceBeta{apiBetaInstance, client, project, zone, isWindows}

	if err := client.CreateInstanceBeta(i.Project, i.Zone, i.Instance); err != nil {
		return i, err
	}
	return i, nil
}

// BuildInstanceMetadataItem create an metadata item
func BuildInstanceMetadataItem(key, value string) *api.MetadataItems {
	return &api.MetadataItems{
		Key:   key,
		Value: func() *string { v := value; return &v }(),
	}
}

// StartWithScriptCode starts the instance with given startup script and metadata.
func (i *InstanceBeta) StartWithScriptCode(script string, instanceMetadata map[string]string) error {
	if instanceMetadata == nil {
		instanceMetadata = make(map[string]string)
	}

	if i.IsWindows {
		instanceMetadata["windows-startup-script-ps1"] = script
	} else {
		instanceMetadata["startup-script"] = script
	}

	var metadataItems []*api.MetadataItems
	for k, v := range instanceMetadata {
		metadataItems = append(metadataItems, BuildInstanceMetadataItem(k, v))
	}
	err := i.Client.SetInstanceMetadata(i.Project, i.Zone,
		i.Name, &api.Metadata{Items: metadataItems, Fingerprint: i.Metadata.Fingerprint})

	if err != nil {
		return err
	}

	if err = i.Client.StartInstance(i.Project, i.Zone, i.Name); err != nil {
		return err
	}
	return nil
}

// Cleanup deletes the InstanceBeta.
func (i *InstanceBeta) Cleanup() error {
	return i.Client.DeleteInstance(i.Project, i.Zone, i.Name)
}
