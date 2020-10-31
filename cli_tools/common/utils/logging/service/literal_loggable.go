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

type literalLoggable struct {
	strings           map[string]string
	int64s            map[string][]int64
	bools             map[string]bool
	traceLogs         []string
	inspectionResults *pb.InspectionResults
}

func (w literalLoggable) GetInspectionResults() *pb.InspectionResults {
	return w.inspectionResults
}

func (w literalLoggable) GetValue(key string) string { return w.strings[key] }

func (w literalLoggable) GetValueAsBool(key string) bool { return w.bools[key] }

func (w literalLoggable) GetValueAsInt64Slice(key string) []int64 { return w.int64s[key] }

func (w literalLoggable) ReadSerialPortLogs() []string { return w.traceLogs }

// SingleImageImportLoggableBuilder initializes and builds a Loggable with the metadata
// fields that are relevant when importing a single image.
type SingleImageImportLoggableBuilder struct {
	literalLoggable
}

// NewSingleImageImportLoggableBuilder creates and initializes a SingleImageImportLoggableBuilder.
func NewSingleImageImportLoggableBuilder() *SingleImageImportLoggableBuilder {
	return &SingleImageImportLoggableBuilder{literalLoggable{
		strings: map[string]string{},
		int64s:  map[string][]int64{},
		bools:   map[string]bool{},
	}}
}

// SetInspectionResults sets inspection results.
func (b *SingleImageImportLoggableBuilder) SetInspectionResults(inspectionResults *pb.InspectionResults) *SingleImageImportLoggableBuilder {
	b.inspectionResults = inspectionResults
	return b
}

// SetUEFIMetrics sets UEFI related metrics.
func (b *SingleImageImportLoggableBuilder) SetUEFIMetrics(isUEFICompatibleImageBool bool, isUEFIDetectedBool bool,
	biosBootableBool bool, rootFSString string) *SingleImageImportLoggableBuilder {

	b.bools[isUEFICompatibleImage] = isUEFICompatibleImageBool
	b.bools[isUEFIDetected] = isUEFIDetectedBool
	b.bools[uefiBootable] = isUEFIDetectedBool
	b.bools[biosBootable] = biosBootableBool
	b.strings[rootFS] = rootFSString
	return b
}

// SetDiskAttributes sets disk related attributes.
func (b *SingleImageImportLoggableBuilder) SetDiskAttributes(fileFormat string, sourceSize int64,
	targetSize int64) *SingleImageImportLoggableBuilder {

	b.strings[importFileFormat] = fileFormat
	b.int64s[sourceSizeGb] = []int64{sourceSize}
	b.int64s[targetSizeGb] = []int64{targetSize}
	return b
}

// SetInflationAttributes sets inflation related attributes.
func (b *SingleImageImportLoggableBuilder) SetInflationAttributes(matchResult string, inflationTypeStr string,
	inflationTimeInt64 int64, shadowInflationTimeInt64 int64) *SingleImageImportLoggableBuilder {

	b.strings[inflationType] = inflationTypeStr
	b.strings[shadowDiskMatchResult] = matchResult
	b.int64s[inflationTime] = []int64{inflationTimeInt64}
	b.int64s[shadowInflationTime] = []int64{shadowInflationTimeInt64}
	return b
}

// AppendTraceLogs sets trace logs during the import.
func (b *SingleImageImportLoggableBuilder) AppendTraceLogs(traceLogs []string) *SingleImageImportLoggableBuilder {
	if b.traceLogs != nil {
		b.traceLogs = append(b.traceLogs, traceLogs...)
	} else {
		b.traceLogs = traceLogs
	}
	return b
}

// Build builds the actual Loggable object.
func (b *SingleImageImportLoggableBuilder) Build() Loggable {
	return b.literalLoggable
}

// OvfExportLoggableBuilder initializes and builds a Loggable with the metadata
// fields that are relevant when exporting OVF.
type OvfExportLoggableBuilder struct {
	literalLoggable
}

// NewOvfExportLoggableBuilder creates and initializes a OvfExportLoggableBuilder.
func NewOvfExportLoggableBuilder() *OvfExportLoggableBuilder {
	return &OvfExportLoggableBuilder{literalLoggable{
		strings: map[string]string{},
		int64s:  map[string][]int64{},
		bools:   map[string]bool{},
	}}
}

// SetDiskAttributes sets disk related attributes.
func (b *OvfExportLoggableBuilder) SetDiskSizes(sourceSizes []int64,
	targetSizes []int64) *OvfExportLoggableBuilder {

	b.int64s[sourceSizeGb] = sourceSizes
	b.int64s[targetSizeGb] = targetSizes
	return b
}

// AppendTraceLogs sets trace logs during OVF export.
func (b *OvfExportLoggableBuilder) AppendTraceLogs(traceLogs []string) *OvfExportLoggableBuilder {
	if b.traceLogs != nil {
		b.traceLogs = append(b.traceLogs, traceLogs...)
	} else {
		b.traceLogs = traceLogs
	}
	return b
}

// Build builds the actual Loggable object.
func (b *OvfExportLoggableBuilder) Build() Loggable {
	return b.literalLoggable
}
