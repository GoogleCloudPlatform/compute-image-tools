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
	traceLogs []string
}

func (w literalLoggable) GetValue(key string) string { return w.strings[key] }

func (w literalLoggable) GetValueAsInt64Slice(key string) []int64 { return w.int64s[key] }

func (w literalLoggable) ReadSerialPortLogs() []string { return w.traceLogs }

// SingleImageImportLoggable returns a Loggable that is pre-initialized with the metadata
// fields that are relevant when importing a single image file.
func SingleImageImportLoggable(fileFormat string, sourceSize, resultSize int64, matchResult string, inflationTypeStr string, inflationTimeInt64 int64, shadowInflationTimeInt64 int64, traceLogs []string) Loggable {
	return literalLoggable{
		strings: map[string]string{
			importFileFormat:      fileFormat,
			inflationType:         inflationTypeStr,
			shadowDiskMatchResult: matchResult,
		},
		int64s: map[string][]int64{
			sourceSizeGb:        {sourceSize},
			targetSizeGb:        {resultSize},
			inflationTime:       {inflationTimeInt64},
			shadowInflationTime: {shadowInflationTimeInt64},
		},
		traceLogs: traceLogs,
	}
}
