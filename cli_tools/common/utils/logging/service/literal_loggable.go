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

type literalLoggable struct {
	strings   map[string]string
	int64s    map[string][]int64
	bools     map[string]bool
	traceLogs []string
}

func (w literalLoggable) GetValue(key string) string { return w.strings[key] }

func (w literalLoggable) GetValueAsBool(key string) bool { return w.bools[key] }

func (w literalLoggable) GetValueAsInt64Slice(key string) []int64 { return w.int64s[key] }

func (w literalLoggable) ReadSerialPortLogs() []string { return w.traceLogs }

// SingleImageImportLoggableBuilder initializes and builds a Loggable with the metadata
// fields that are relevant when importing a single image.
type SingleImageImportLoggableBuilder struct {
	fileFormat                string
	sourceSize                int64
	resultSize                int64
	matchResult               string
	inflationTypeStr          string
	inflationTimeInt64        int64
	shadowInflationTimeInt64  int64
	isUEFICompatibleImageBool bool
	isUEFIDetectedBool        bool
	traceLogs                 []string
}

// SetDiskAttributes sets disk related attributes.
func (b *SingleImageImportLoggableBuilder) SetDiskAttributes(fileFormat string, sourceSize int64,
	resultSize int64, isUEFICompatibleImageBool bool, isUEFIDetectedBool bool) *SingleImageImportLoggableBuilder {

	b.fileFormat = fileFormat
	b.sourceSize = sourceSize
	b.resultSize = resultSize
	b.isUEFICompatibleImageBool = isUEFICompatibleImageBool
	b.isUEFIDetectedBool = isUEFIDetectedBool
	return b
}

// SetInflationAttributes sets inflation related attributes.
func (b *SingleImageImportLoggableBuilder) SetInflationAttributes(matchResult string, inflationTypeStr string,
	inflationTimeInt64 int64, shadowInflationTimeInt64 int64) *SingleImageImportLoggableBuilder {

	b.matchResult = matchResult
	b.inflationTypeStr = inflationTypeStr
	b.inflationTimeInt64 = inflationTimeInt64
	b.shadowInflationTimeInt64 = shadowInflationTimeInt64
	return b
}

// SetTraceLogs sets trace logs during the import.
func (b *SingleImageImportLoggableBuilder) SetTraceLogs(traceLogs []string) *SingleImageImportLoggableBuilder {
	b.traceLogs = traceLogs
	return b
}

// Build builds the actual Loggable object.
func (b *SingleImageImportLoggableBuilder) Build() Loggable {
	return literalLoggable{
		strings: map[string]string{
			importFileFormat:      b.fileFormat,
			inflationType:         b.inflationTypeStr,
			shadowDiskMatchResult: b.matchResult,
		},
		int64s: map[string][]int64{
			sourceSizeGb:        {b.sourceSize},
			targetSizeGb:        {b.resultSize},
			inflationTime:       {b.inflationTimeInt64},
			shadowInflationTime: {b.shadowInflationTimeInt64},
		},
		bools: map[string]bool{
			isUEFICompatibleImage: b.isUEFICompatibleImageBool,
			isUEFIDetected:        b.isUEFIDetectedBool,
		},
		traceLogs: b.traceLogs,
	}
}
