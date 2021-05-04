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
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

// Inflater constructs a new persistentDisk, typically starting from a
// frozen representation of a disk, such as a VMDK file or a GCP disk image.
type Inflater interface {
	Inflate() (persistentDisk, shadowTestFields, error)
	Cancel(reason string) bool
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

	di, err := newDaisyInflater(request, inspector, logger)
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
			shadowInflater: createAPIInflater(request, computeClient, storageClient, logger, true),
			logger:         logger,
		}, nil
	}

	return &inflaterFacade{
		apiInflater:   createAPIInflater(request, computeClient, storageClient, logger, false),
		daisyInflater: di,
		logger:        logger,
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
}

func (facade *inflaterFacade) Inflate() (persistentDisk, shadowTestFields, error) {
	var pd persistentDisk
	var tf shadowTestFields
	var err error

	//  Try with API inflater.
	pd, tf, err = facade.apiInflater.Inflate()
	if err == nil {
		facade.logger.Metric(&pb.OutputInfo{
			InflationType:   "api_success",
			InflationTimeMs: []int64{tf.inflationTime.Milliseconds()},
		})
		return pd, tf, err
	}

	if !isCausedByUnsupportedFormat(err) {
		facade.logger.Metric(&pb.OutputInfo{
			InflationType:   "api_failed",
			InflationTimeMs: []int64{tf.inflationTime.Milliseconds()},
		})
		return pd, tf, err
	}

	// Retry with daisy inflater.
	pd, tf, err = facade.daisyInflater.Inflate()
	if err == nil {
		facade.logger.Metric(&pb.OutputInfo{
			InflationType:   "qemu_success",
			InflationTimeMs: []int64{tf.inflationTime.Milliseconds()},
		})
		return pd, tf, err
	}

	facade.logger.Metric(&pb.OutputInfo{
		InflationType:   "qemu_failed",
		InflationTimeMs: []int64{tf.inflationTime.Milliseconds()},
	})
	return pd, tf, err
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
}

// signals to control the verification towards shadow inflater
const (
	sigMainInflaterDone   = "main done"
	sigMainInflaterErr    = "main err"
	sigShadowInflaterDone = "shadow done"
	sigShadowInflaterErr  = "shadow err"
)

func (facade *shadowTestInflaterFacade) Inflate() (persistentDisk, shadowTestFields, error) {
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

func (facade *shadowTestInflaterFacade) Cancel(reason string) bool {
	facade.shadowInflater.Cancel(reason)
	return facade.mainInflater.Cancel(reason)
}

func (facade *shadowTestInflaterFacade) compareWithShadowInflater(mainPd, shadowPd *persistentDisk, mainIi, shadowIi *shadowTestFields) string {
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
