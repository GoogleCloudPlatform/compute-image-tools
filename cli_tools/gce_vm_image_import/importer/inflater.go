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

	daisy_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	string_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/string"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
)

const (
	inflateFilePath  = "inflate_file.wf.json"
	inflateImagePath = "inflate_image.wf.json"
)

// inflater constructs a new persistentDisk, typically starting from a
// frozen representation of a disk, such as a VMDK file or a GCP disk image.
//
// Implementers can expose detailed logs using the traceLogs() method.
type inflater interface {
	inflate() (persistentDisk, error)
	traceLogs() []string
	cancel(reason string) bool
}

// daisyInflater implements an inflater using daisy workflows, and is capable
// of inflating GCP disk images and qemu-img compatible disk files.
type daisyInflater struct {
	wf              *daisy.Workflow
	inflatedDiskURI string
	serialLogs      []string
	diskClient      diskClient
	project         string
	zone            string
}

func (inflater *daisyInflater) inflate() (persistentDisk, error) {
	err := inflater.wf.Run(context.Background())

	if err != nil {
		return persistentDisk{
			uri: inflater.inflatedDiskURI,
		}, err
	}
	if inflater.wf.Logger != nil {
		inflater.serialLogs = inflater.wf.Logger.ReadSerialPortLogs()
	}
	// See `daisy_workflows/image_import/import_image.sh` for generation of these values.
	targetSizeGB := inflater.wf.GetSerialConsoleOutputValue("target-size-gb")
	sourceSizeGB := inflater.wf.GetSerialConsoleOutputValue("source-size-gb")
	importFileFormat := inflater.wf.GetSerialConsoleOutputValue("import-file-format")
	return persistentDisk{
		uri:        inflater.inflatedDiskURI,
		sizeGb:     string_utils.SafeStringToInt(targetSizeGB),
		sourceGb:   string_utils.SafeStringToInt(sourceSizeGB),
		sourceType: importFileFormat,
	}, nil
}

type persistentDisk struct {
	uri        string
	sizeGb     int64
	sourceGb   int64
	sourceType string
}

func createDaisyInflater(args ImportArguments) (inflater, error) {
	diskName := "disk-" + args.ExecutionID
	var wfPath string
	var vars map[string]string
	var inflationDiskIndex int
	if isImage(args.Source) {
		wfPath = inflateImagePath
		vars = map[string]string{
			"source_image": args.Source.Path(),
			"disk_name":    diskName,
		}
		inflationDiskIndex = 0 // Workflow only uses one disk.
	} else {
		wfPath = inflateFilePath
		vars = map[string]string{
			"source_disk_file": args.Source.Path(),
			"import_network":   args.Network,
			"import_subnet":    args.Subnet,
			"disk_name":        diskName,
		}
		inflationDiskIndex = 1 // First disk is for the worker
	}

	wf, err := daisycommon.ParseWorkflow(path.Join(args.WorkflowDir, wfPath), vars,
		args.Project, args.Zone, args.ScratchBucketGcsPath, args.Oauth, args.Timeout.String(), args.ComputeEndpoint,
		args.GcsLogsDisabled, args.CloudLogsDisabled, args.StdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	daisy_utils.UpdateAllInstanceNoExternalIP(wf, args.NoExternalIP)
	for k, v := range vars {
		wf.AddVar(k, v)
	}

	if strings.Contains(args.OS, "windows") {
		addFeatureToDisk(wf, "WINDOWS", inflationDiskIndex)
	}

	return &daisyInflater{
		wf:              wf,
		inflatedDiskURI: fmt.Sprintf("zones/%s/disks/%s", args.Zone, diskName),
		project:         args.Project,
		zone:            args.Zone,
	}, nil
}

func (inflater *daisyInflater) traceLogs() []string {
	return inflater.serialLogs
}

// addFeatureToDisk finds the first `CreateDisk` step, and adds `feature` as
// a guestOsFeature to the disk at index `diskIndex`.
func addFeatureToDisk(workflow *daisy.Workflow, feature string, diskIndex int) {
	disk := getDisk(workflow, diskIndex)
	disk.GuestOsFeatures = append(disk.GuestOsFeatures, &compute.GuestOsFeature{
		Type: feature,
	})
}

func getDisk(workflow *daisy.Workflow, diskIndex int) *daisy.Disk {
	for _, step := range workflow.Steps {
		if step.CreateDisks != nil {
			disks := *step.CreateDisks
			if diskIndex < len(disks) {
				return disks[diskIndex]
			}
			panic(fmt.Sprintf("CreateDisks step did not have disk at index %d", diskIndex))
		}
	}

	panic("Did not find CreateDisks step.")
}

func (inflater *daisyInflater) cancel(reason string) bool {
	inflater.wf.CancelWithReason(reason)
	return true
}
