//  Copyright 2021  Licensed under the Apache License, Version 2.0 (the "License");
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
	"time"

	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	string_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/string"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const (
	inflateFilePath  = "image_import/inflate_file.wf.json"
	inflateImagePath = "image_import/inflate_image.wf.json"

	// When exceeded, we use default values for PDs, rather than more accurate
	// values used by inspection. When using default values, the worker may
	// need to resize the PDs, which requires escalated privileges.
	inspectionTimeout = time.Second * 3

	// 10GB is the default disk size used in inflate_file.wf.json.
	defaultInflationDiskSizeGB = 10

	// See `daisy_workflows/image_import/import_image.sh` for generation of these values.
	targetSizeGBKey     = "target-size-gb"
	sourceSizeGBKey     = "source-size-gb"
	importFileFormatKey = "import-file-format"
	diskChecksumKey     = "disk-checksum"
)

// daisyInflater implements an inflater using daisy workflows, and is capable
// of inflating GCP disk images and qemu-img compatible disk files.
type daisyInflater struct {
	worker          daisyutils.DaisyWorker
	vars            map[string]string
	source          Source
	inflatedDiskURI string
	logger          logging.Logger
}

func (inflater *daisyInflater) Inflate() (persistentDisk, shadowTestFields, error) {
	if inflater.source != nil {
		inflater.logger.User("Creating Google Compute Engine disk from " + inflater.source.Path())
	}
	serialValues, err := inflater.worker.RunAndReadSerialValues(inflater.vars,
		targetSizeGBKey, sourceSizeGBKey, importFileFormatKey, diskChecksumKey)
	if err == nil {
		inflater.logger.User("Finished creating Google Compute Engine disk")
	}
	return persistentDisk{
			uri:        inflater.inflatedDiskURI,
			sizeGb:     enforceMinimumDiskSize(string_utils.SafeStringToInt(serialValues[targetSizeGBKey])),
			sourceGb:   string_utils.SafeStringToInt(serialValues[sourceSizeGBKey]),
			sourceType: serialValues[importFileFormatKey],
		}, shadowTestFields{
			checksum:      serialValues[diskChecksumKey],
			inflationTime: time.Since(time.Now()),
			inflationType: "qemu",
		}, err
}

// NewDaisyInflater returns an inflater that uses a Daisy workflow.
func NewDaisyInflater(request ImageImportRequest, fileInspector imagefile.Inspector, logger logging.Logger) (Inflater, error) {
	return newDaisyInflater(request, fileInspector, logger)
}

func newDaisyInflater(request ImageImportRequest, fileInspector imagefile.Inspector, logger logging.Logger) (*daisyInflater, error) {
	diskName := "disk-" + request.ExecutionID
	var wfPath string
	var vars map[string]string
	var inflationDiskIndex int
	if isImage(request.Source) {
		wfPath = inflateImagePath
		vars = map[string]string{
			"source_image": request.Source.Path(),
			"disk_name":    diskName,
		}
		inflationDiskIndex = 0 // Workflow only uses one disk.
	} else {
		wfPath = inflateFilePath
		vars = createDaisyVarsForFile(request, fileInspector, diskName)
		inflationDiskIndex = 1 // First disk is for the worker
	}

	wf, err := daisyutils.ParseWorkflow(path.Join(request.WorkflowDir, wfPath), vars,
		request.Project, request.Zone, request.ScratchBucketGcsPath, request.Oauth, request.Timeout.String(), request.ComputeEndpoint,
		request.GcsLogsDisabled, request.CloudLogsDisabled, request.StdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	if request.UefiCompatible {
		addFeatureToDisk(wf, "UEFI_COMPATIBLE", inflationDiskIndex)
	}
	if strings.Contains(request.OS, "windows") {
		addFeatureToDisk(wf, "WINDOWS", inflationDiskIndex)
	}

	env := request.EnvironmentSettings()
	if env.DaisyLogLinePrefix != "" {
		env.DaisyLogLinePrefix += "-"
	}
	env.DaisyLogLinePrefix += "inflate"
	return &daisyInflater{
		worker:          daisyutils.NewDaisyWorker(wf, env, logger),
		inflatedDiskURI: fmt.Sprintf("zones/%s/disks/%s", request.Zone, diskName),
		logger:          logger,
		source:          request.Source,
		vars:            vars,
	}, nil
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

func (inflater *daisyInflater) Cancel(reason string) bool {
	return inflater.worker.Cancel(reason)
}

func createDaisyVarsForFile(request ImageImportRequest,
	fileInspector imagefile.Inspector, diskName string) map[string]string {
	vars := map[string]string{
		"source_disk_file": request.Source.Path(),
		"import_network":   request.Network,
		"import_subnet":    request.Subnet,
		"disk_name":        diskName,
	}

	if request.ComputeServiceAccount != "" {
		vars["compute_service_account"] = request.ComputeServiceAccount
	}

	// To reduce the runtime permissions used on the inflation worker, we pre-allocate
	// disks sufficient to hold the disk file and the inflated disk. If inspection fails,
	// then the default values in the daisy workflow will be used. The scratch disk gets
	// a padding factor to account for filesystem overhead.
	deadline, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(inspectionTimeout))
	defer cancelFunc()
	metadata, err := fileInspector.Inspect(deadline, request.Source.Path())
	if err == nil {
		vars["inflated_disk_size_gb"] = fmt.Sprintf("%d", calculateInflatedSize(metadata))
		vars["scratch_disk_size_gb"] = fmt.Sprintf("%d", calculateScratchDiskSize(metadata))
	}
	return vars
}

// Allocate extra room for filesystem overhead, and
// ensure a minimum of 10GB (the minimum size of a GCP disk).
func calculateScratchDiskSize(metadata imagefile.Metadata) int64 {
	// This uses the historic padding calculation from import_image.sh: add ten percent,
	// and round up.
	padded := int64(float64(metadata.PhysicalSizeGB)*1.1) + 1
	return enforceMinimumDiskSize(padded)
}

// Ensure a minimum of 10GB (the minimum size of a GCP disk)
func calculateInflatedSize(metadata imagefile.Metadata) int64 {
	return enforceMinimumDiskSize(metadata.VirtualSizeGB)
}

func enforceMinimumDiskSize(size int64) int64 {
	if size < defaultInflationDiskSizeGB {
		return defaultInflationDiskSizeGB
	}
	return size
}
