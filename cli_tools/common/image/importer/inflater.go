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
	"regexp"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

// Inflater constructs a new persistentDisk, typically starting from a
// frozen representation of a disk, such as a VMDK file or a GCP disk image.
type Inflater interface {
	Inflate() (persistentDisk, inflationInfo, error)
	Cancel(reason string) bool
}

type persistentDisk struct {
	uri        string
	sizeGb     int64
	sourceGb   int64
	sourceType string
}

type inflationInfo struct {
	// Below fields are for inflation metrics
	checksum      string
	inflationTime time.Duration
	inflationType string
}

func newInflater(request ImageImportRequest, computeClient daisyCompute.Client, storageClient domain.StorageClientInterface,
	inspector imagefile.Inspector, logger logging.Logger) (Inflater, error) {

	var fileMetadata = imagefile.Metadata{}
	if !isImage(request.Source) {
		// 1. To reduce the runtime permissions used on the inflation worker, we pre-allocate
		// disks sufficient to hold the disk file and the inflated disk. If inspection fails,
		// then the default values in the daisy workflow will be used. The scratch disk gets
		// a padding factor to account for filesystem overhead.
		// 2. Inspection also returns checksum of the image file for sanitary check. If it's
		// failed to get the checksum, the following sanitary check will be skipped.
		deadline, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(inspectionTimeout))
		defer cancelFunc()
		logger.User("Inspecting the image file...")
		fileMetadata, _ = inspector.Inspect(deadline, request.Source.Path())
	}

	di, err := newDaisyInflater(request, fileMetadata, logger)
	if err != nil {
		return nil, err
	}

	// This boolean switch controls whether native PD inflation is used, either
	// as the primary inflation method or in a shadow test mode
	tryNativePDInflation := true
	if isImage(request.Source) || !tryNativePDInflation {
		return di, nil
	}

	if isShadowTestFormat(request) {
		return &shadowTestInflaterFacade{
			mainInflater:   di,
			shadowInflater: createAPIInflater(&apiInflaterProperties{request, computeClient, storageClient, logger, true, true}),
			logger:         logger,
			qemuChecksum:   fileMetadata.Checksum,
		}, nil
	}

	return &inflaterFacade{
		apiInflater:   createAPIInflater(&apiInflaterProperties{request, computeClient, storageClient, logger, false, fileMetadata.Checksum != ""}),
		daisyInflater: di,
		logger:        logger,
		qemuChecksum:  fileMetadata.Checksum,
		computeClient: computeClient,
		request:       request,
	}, nil
}

func isShadowTestFormat(request ImageImportRequest) bool {
	// TODO: process VHD/VPC differently
	return true
}

// inflaterFacade implements an inflater using other concrete implementations.
type inflaterFacade struct {
	apiInflater   Inflater
	daisyInflater Inflater
	logger        logging.Logger
	qemuChecksum  string
	computeClient daisyCompute.Client
	request       ImageImportRequest
}

func (facade *inflaterFacade) Inflate() (persistentDisk, inflationInfo, error) {
	var pd persistentDisk
	var ii inflationInfo
	var err error
	var fallbackReason string

	if facade.qemuChecksum == "" {
		fallbackReason = "qemu_checksum_missing"
	} else {
		//  Run API inflater as the primary inflation method.
		pd, ii, err = facade.apiInflater.Inflate()

		if err != nil {
			// If the inflation is failed but not due to unsupported format, don't rerun inflation.
			if !isCausedByUnsupportedFormat(err) {
				facade.logger.Metric(&pb.OutputInfo{
					InflationType:   "api_failed",
					InflationTimeMs: []int64{ii.inflationTime.Milliseconds()},
				})
				return pd, ii, err
			}
			fallbackReason = "unsupported_format"
		} else {
			if isChecksumMatch(ii.checksum, facade.qemuChecksum) {
				facade.logger.Metric(&pb.OutputInfo{
					InflationType:   "api_success",
					InflationTimeMs: []int64{ii.inflationTime.Milliseconds()},
				})
				return pd, ii, err
			}

			// If checksum mismatches , delete the corrupted disk.
			err = facade.computeClient.DeleteDisk(facade.request.Project, facade.request.Zone, getDiskName(facade.request.ExecutionID))
			if err != nil {
				return pd, ii, daisy.Errf("Tried to delete the disk after checksum mismatch is detected, but failed on: %v", err)
			}
			facade.logger.User("Disk checksum mismatch, recreating...")
			fallbackReason = "checksum_mismatch"
		}
	}

	// Now fallback to daisy inflater. As described above, it is due to one of the below reasons:
	// 1. The API failed because it doesn't support the format of the image file.
	// 2. Checksum mismatch, which means the API produced a corrupted disk.
	// 3. QEMU checksum is missing, which means we have no way to compare the checksum.
	return facade.fallbackToDaisyInflater(fallbackReason)
}

func (facade *inflaterFacade) fallbackToDaisyInflater(reason string) (persistentDisk, inflationInfo, error) {
	pd, ii, err := facade.daisyInflater.Inflate()
	if err == nil {
		facade.logger.Metric(&pb.OutputInfo{
			InflationType:           "qemu_success",
			InflationTimeMs:         []int64{ii.inflationTime.Milliseconds()},
			InflationFallbackReason: reason,
		})
		return pd, ii, err
	}

	facade.logger.Metric(&pb.OutputInfo{
		InflationType:           "qemu_failed",
		InflationTimeMs:         []int64{ii.inflationTime.Milliseconds()},
		InflationFallbackReason: reason,
	})
	return pd, ii, err
}

func (facade *inflaterFacade) Cancel(reason string) bool {
	// No need to cancel apiInflater.
	return facade.daisyInflater.Cancel(reason)
}

// shadowTestInflaterFacade implements an inflater with shadow test support.
type shadowTestInflaterFacade struct {
	mainInflater   Inflater
	shadowInflater Inflater
	logger         logging.Logger

	// Runtime-populated fields
	qemuChecksum string
}

// signals to control the verification towards shadow inflater
const (
	sigMainInflaterDone   = "main done"
	sigMainInflaterErr    = "main err"
	sigShadowInflaterDone = "shadow done"
	sigShadowInflaterErr  = "shadow err"
)

func (facade *shadowTestInflaterFacade) Inflate() (persistentDisk, inflationInfo, error) {
	inflaterChan := make(chan string)

	// Launch main inflater.
	var pd persistentDisk
	var ii inflationInfo
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
	var shadowIi inflationInfo
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

func (facade *shadowTestInflaterFacade) Cancel(reason string) bool {
	facade.shadowInflater.Cancel(reason)
	return facade.mainInflater.Cancel(reason)
}

const matchFormat = "sizeGb-%v,sourceGb-%v,content-%v,qemuchecksum-%v"

func (facade *shadowTestInflaterFacade) compareWithShadowInflater(mainPd, shadowPd *persistentDisk, mainIi, shadowIi *inflationInfo) string {
	sizeGbMatch := shadowPd.sizeGb == mainPd.sizeGb
	sourceGbMatch := shadowPd.sourceGb == mainPd.sourceGb
	contentMatch := isChecksumMatch(shadowIi.checksum, mainIi.checksum)
	qemuChecksumMatch := "false"
	if facade.qemuChecksum == "" {
		qemuChecksumMatch = "skipped"
	} else if isChecksumMatch(facade.qemuChecksum, mainIi.checksum) {
		qemuChecksumMatch = "true"
	}

	match := sizeGbMatch && sourceGbMatch && contentMatch && (qemuChecksumMatch == "true")

	var result string
	if match {
		result = "true"
	} else {
		result = fmt.Sprintf(matchFormat, sizeGbMatch, sourceGbMatch, contentMatch, qemuChecksumMatch)
	}
	return result
}

func getDiskName(executionID string) string {
	return fmt.Sprintf("disk-%v", executionID)
}

// isChecksumMatch verifies whether checksum matches, excluded useless characters.
func isChecksumMatch(checksum1, checksum2 string) bool {
	reg, _ := regexp.Compile("[^a-zA-Z0-9]+")
	processedChecksum1 := reg.ReplaceAllString(checksum1, "")
	processedChecksum2 := reg.ReplaceAllString(checksum2, "")
	return processedChecksum1 == processedChecksum2
}
