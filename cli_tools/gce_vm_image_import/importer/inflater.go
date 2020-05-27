//  Copyright 2020  Licensed under the Apache License, Version 2.0 (the "License");
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
	"fmt"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
)

const (
	inflateFilePath  = "import_disk.wf.json"
	inflateImagePath = "inflate_image.wf.json"
)

type inflater interface {
	inflate(ctx context.Context) (pd, error)
	serials() []string
}

type daisyInflater struct {
	wf         *daisy.Workflow
	uri        string
	serialLogs []string
}

func (d daisyInflater) inflate(ctx context.Context) (pd, error) {
	err := d.wf.Run(ctx)
	if err != nil {
		return pd{}, err
	}
	if d.wf.Logger != nil {
		d.serialLogs = d.wf.Logger.ReadSerialPortLogs()
	}
	return pd{
		uri: d.uri,
	}, nil
}

type pd struct {
	uri        string
	sizeGb     int64
	sourceGb   int64
	sourceType string
}

func createDaisyInflater(t ImportArguments, workflowDirectory string) (inflater, error) {
	diskName := "disk-" + t.ExecutionID
	var wfPath string
	var vars map[string]string
	var inflationDiskIndex int
	if isImage(t.Source) {
		wfPath = inflateImagePath
		vars = map[string]string{
			"source_image": t.Source.Path(),
			"disk_name":    diskName,
		}
		inflationDiskIndex = 0 // Workflow only uses one disk.
	} else {
		wfPath = inflateFilePath
		vars = map[string]string{
			"source_disk_file": t.Source.Path(),
			"import_network":   t.Network,
			"import_subnet":    t.Subnet,
			"disk_name":        diskName,
		}
		inflationDiskIndex = 1 // First disk is for the worker
	}

	wf, err := daisycommon.ParseWorkflow(path.Join(workflowDirectory, wfPath), vars,
		t.Project, t.Zone, t.ScratchBucketGcsPath, t.Oauth, t.Timeout.String(), t.ComputeEndpoint,
		t.GcsLogsDisabled, t.CloudLogsDisabled, t.StdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	for k, v := range vars {
		wf.AddVar(k, v)
	}

	if strings.Contains(t.OS, "windows") {
		addFeatureToDisk(wf, "WINDOWS", inflationDiskIndex)
	}

	return daisyInflater{
		wf:  wf,
		uri: fmt.Sprintf("zones/%s/disks/%s", t.Zone, diskName),
	}, nil
}

func (d daisyInflater) serials() []string {
	return d.serialLogs
}

// If the inflated disk hold Windows, then it needs the WINDOWS GuestOSFeature
// in order to boot during the subsequent translate step.
func addFeatureToDisk(workflow *daisy.Workflow, feature string, diskIndex int) {
	disk := getDisk(workflow, diskIndex)
	disk.GuestOsFeatures = append(disk.GuestOsFeatures, &compute.GuestOsFeature{
		Type: feature,
	})
}

func getDisk(workflow *daisy.Workflow, diskIndex int) *daisy.Disk {
	for _, step := range workflow.Steps {
		if step.CreateDisks != nil {
			return (*step.CreateDisks)[diskIndex]
		}
	}

	panic(fmt.Sprintf("Expected disk at index %d", diskIndex))
}
