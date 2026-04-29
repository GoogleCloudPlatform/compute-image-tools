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

package service

import "github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"

// outputInfoLoggable is an implementation of Loggable that exposes fields
// from an OutputInfo object.
type outputInfoLoggable struct {
	outputInfo *pb.OutputInfo
}

// NewOutputInfoLoggable returns a Loggable that is bacaked by a concrete instance
// of pb.OutputInfo. It's intended as a temporary shim while we
// transition tools to use the ToolLogger type.
func NewOutputInfoLoggable(outputInfo *pb.OutputInfo) Loggable {
	return &outputInfoLoggable{outputInfo}
}

func (o *outputInfoLoggable) GetValue(key string) string {
	switch key {
	case importFileFormat:
		return o.outputInfo.GetImportFileFormat()
	case inflationType:
		return o.outputInfo.GetInflationType()
	case inflationFallbackReason:
		return o.outputInfo.GetInflationFallbackReason()
	case shadowDiskMatchResult:
		return o.outputInfo.GetShadowDiskMatchResult()
	}
	return ""
}

func (o *outputInfoLoggable) GetValueAsBool(key string) bool {
	switch key {
	case isUEFICompatibleImage:
		return o.outputInfo.GetIsUefiCompatibleImage()
	case isUEFIDetected:
		return o.outputInfo.GetIsUefiDetected()
	}
	return false
}

func (o *outputInfoLoggable) GetValueAsInt64Slice(key string) []int64 {
	switch key {
	case targetSizeGb:
		return o.outputInfo.GetTargetsSizeGb()
	case sourceSizeGb:
		return o.outputInfo.GetSourcesSizeGb()
	case inflationTime:
		return o.outputInfo.GetInflationTimeMs()
	case shadowInflationTime:
		return o.outputInfo.GetShadowInflationTimeMs()
	}
	return nil
}

func (o *outputInfoLoggable) GetInspectionResults() *pb.InspectionResults {
	return o.outputInfo.GetInspectionResults()
}

func (o *outputInfoLoggable) ReadSerialPortLogs() []string {
	return o.outputInfo.SerialOutputs
}
