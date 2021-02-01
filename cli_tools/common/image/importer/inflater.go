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
	"time"
	"regexp"

	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
	daisyUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	string_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/string"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
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
)

// inflaterFacade implements an inflater using other concrete implementations.
type inflaterFacade struct {
	mainInflater   Inflater
	shadowInflater Inflater
	logger         logging.Logger
}

// signals to control the verification towards shadow inflater
const (
	sigMainInflaterDone   = "main done"
	sigMainInflaterErr    = "main err"
	sigShadowInflaterDone = "shadow done"
	sigShadowInflaterErr  = "shadow err"
)

var allowedAPIFormatsEx = regexp.MustCompile("^(vpc)$")

func (facade *inflaterFacade) Inflate() (persistentDisk, shadowTestFields, error) {
	inflaterChan := make(chan string)

	// Launch main inflater.
	var pd persistentDisk
	var ii shadowTestFields
	var err error
	go func() {
		pd, ii, err = facade.mainInflater.Inflate()
		if err != nil {
			inflaterChan <- sigMainInflaterErr
		} else {
			inflaterChan <- sigMainInflaterDone
		}
	}()

	// Launch shadow inflater.
	var shadowPd persistentDisk
	var shadowIi shadowTestFields
	var shadowErr error
	go func() {
		shadowPd, shadowIi, shadowErr = facade.shadowInflater.Inflate()
		if shadowErr != nil {
			inflaterChan <- sigShadowInflaterErr
		} else {
			inflaterChan <- sigShadowInflaterDone
		}
	}()

	var matchResult string

	// Return early if main inflater finished first.
	result := <-inflaterChan
	if result == sigMainInflaterDone || result == sigMainInflaterErr {
		if result == sigMainInflaterDone {
			matchResult = "Main inflater finished earlier"
		} else {
			matchResult = "Main inflater failed earlier"
		}

		// Wait for shadowInflater.inflate() to be canceled. Otherwise, shadowInflater.inflate() may
		// be interrupted with temporary resources left: b/169073057
		cancelResult := facade.shadowInflater.Cancel("cleanup shadow PD")
		if cancelResult == false {
			matchResult += " cleanup failed"
		}
		return pd, ii, err
	}

	// Wait for main inflater to finish, then process shadow inflater's result.
	mainResult := <-inflaterChan
	if result == sigShadowInflaterDone {
		if mainResult == sigMainInflaterErr {
			matchResult = "Main inflater failed while shadow inflater succeeded"
		} else {
			matchResult = facade.compareWithShadowInflater(&pd, &shadowPd, &ii, &shadowIi)
		}
	} else if result == sigShadowInflaterErr && mainResult == sigMainInflaterDone {
		if isCausedByUnsupportedFormat(shadowErr) {
			matchResult = "Shadow inflater doesn't support the format while main inflater supports"
		} else if isCausedByAlphaAPIAccess(shadowErr) {
			matchResult = "Shadow inflater not executed: no Alpha API access"
		} else {
			matchResult = fmt.Sprintf("Shadow inflater failed while main inflater succeeded: [%v]", shadowErr)
		}
	}

	facade.logger.Metric(&pb.OutputInfo{
		ShadowDiskMatchResult: matchResult,
		InflationType:         ii.inflationType,
		InflationTimeMs:       []int64{ii.inflationTime.Milliseconds()},
		ShadowInflationTimeMs: []int64{shadowIi.inflationTime.Milliseconds()},
	})
	return pd, ii, err
}

func (facade *inflaterFacade) Cancel(reason string) bool {
	facade.shadowInflater.Cancel(reason)
	return facade.mainInflater.Cancel(reason)
}

func (facade *inflaterFacade) compareWithShadowInflater(mainPd, shadowPd *persistentDisk, mainIi, shadowIi *shadowTestFields) string {
	matchFormat := "sizeGb-%v,sourceGb-%v,content-%v"
	sizeGbMatch := shadowPd.sizeGb == mainPd.sizeGb
	sourceGbMatch := shadowPd.sourceGb == mainPd.sourceGb
	contentMatch := shadowIi.checksum == mainIi.checksum
	match := sizeGbMatch && sourceGbMatch && contentMatch

	var result string
	if match {
		result = "true"
	} else {
		result = fmt.Sprintf(matchFormat, sizeGbMatch, sourceGbMatch, contentMatch)
	}
	return result
}

// Inflater constructs a new persistentDisk, typically starting from a
// frozen representation of a disk, such as a VMDK file or a GCP disk image.
//
// Implementers can expose detailed logs using the traceLogs() method.
type Inflater interface {
	Inflate() (persistentDisk, shadowTestFields, error)
	Cancel(reason string) bool
}

// daisyInflater implements an inflater using daisy workflows, and is capable
// of inflating GCP disk images and qemu-img compatible disk files.
type daisyInflater struct {
	wf              *daisy.Workflow
	source          Source
	inflatedDiskURI string
	logger          logging.Logger
}

func (inflater *daisyInflater) Inflate() (persistentDisk, shadowTestFields, error) {
	if inflater.source != nil {
		inflater.logger.User("Creating Google Compute Engine disk from " + inflater.source.Path())
	}
	startTime := time.Now()
	err := inflater.wf.Run(context.Background())
	if inflater.wf.Logger != nil {
		for _, trace := range inflater.wf.Logger.ReadSerialPortLogs() {
			inflater.logger.Trace(trace)
		}
	}
	inflater.logger.User("Finished creating Google Compute Engine disk")
	// See `daisy_workflows/image_import/import_image.sh` for generation of these values.
	targetSizeGB := inflater.wf.GetSerialConsoleOutputValue("target-size-gb")
	sourceSizeGB := inflater.wf.GetSerialConsoleOutputValue("source-size-gb")
	importFileFormat := inflater.wf.GetSerialConsoleOutputValue("import-file-format")
	checksum := inflater.wf.GetSerialConsoleOutputValue("disk-checksum")
	return persistentDisk{
			uri:        inflater.inflatedDiskURI,
			sizeGb:     string_utils.SafeStringToInt(targetSizeGB),
			sourceGb:   string_utils.SafeStringToInt(sourceSizeGB),
			sourceType: importFileFormat,
		}, shadowTestFields{
			checksum:      checksum,
			inflationTime: time.Since(startTime),
			inflationType: "qemu",
		}, err
}

type persistentDisk struct {
	uri        string
	sizeGb     int64
	sourceGb   int64
	sourceType string
}

type shadowTestFields struct {
	// Below fields are for shadow API inflation metrics
	checksum      string
	inflationTime time.Duration
	inflationType string
}

func newInflater(request ImageImportRequest, computeClient daisyCompute.Client, storageClient domain.StorageClientInterface,
	inspector imagefile.Inspector, logger logging.Logger) (Inflater, error) {

	di, err := NewDaisyInflater(request, inspector, logger)
	if err != nil {
		return nil, err
	}

	if isImage(request.Source) {
		return di, nil
	}

	deadline, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(inspectionTimeout))
	defer cancelFunc()
	metadata, inspectorErr := inspector.Inspect(deadline, request.Source.Path())
	if inspectorErr != nil {
		return nil, inspectorErr
	}

	if !allowedAPIFormatsEx.MatchString(metadata.FileFormat) {
		return di, nil
	}

	ai := createAPIInflater(request, computeClient, storageClient, logger, metadata)
	return &inflaterFacade{
		mainInflater:   di,
		shadowInflater: ai,
		logger:         logger,
	}, nil
}

// NewDaisyInflater returns an Inflater that uses a Daisy workflow.
func NewDaisyInflater(request ImageImportRequest, fileInspector imagefile.Inspector, logger logging.Logger) (Inflater, error) {
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

	wf, err := daisycommon.ParseWorkflow(path.Join(request.WorkflowDir, wfPath), vars,
		request.Project, request.Zone, request.ScratchBucketGcsPath, request.Oauth, request.Timeout.String(), request.ComputeEndpoint,
		request.GcsLogsDisabled, request.CloudLogsDisabled, request.StdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	for k, v := range vars {
		wf.AddVar(k, v)
	}
	daisyUtils.UpdateAllInstanceNoExternalIP(wf, request.NoExternalIP)
	if request.UefiCompatible {
		addFeatureToDisk(wf, "UEFI_COMPATIBLE", inflationDiskIndex)
	}
	if strings.Contains(request.OS, "windows") {
		addFeatureToDisk(wf, "WINDOWS", inflationDiskIndex)
	}

	logPrefix := request.DaisyLogLinePrefix
	if logPrefix != "" {
		logPrefix += "-"
	}
	wf.Name = logPrefix + "inflate"
	return &daisyInflater{
		wf:              wf,
		inflatedDiskURI: fmt.Sprintf("zones/%s/disks/%s", request.Zone, diskName),
		logger:          logger,
		source:          request.Source,
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
	inflater.wf.CancelWithReason(reason)
	return true
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
	if padded < defaultInflationDiskSizeGB {
		return defaultInflationDiskSizeGB
	}
	return padded
}

// Ensure a minimum of 10GB (the minimum size of a GCP disk)
func calculateInflatedSize(metadata imagefile.Metadata) int64 {
	if metadata.VirtualSizeGB < defaultInflationDiskSizeGB {
		return defaultInflationDiskSizeGB
	}
	return metadata.VirtualSizeGB
}
