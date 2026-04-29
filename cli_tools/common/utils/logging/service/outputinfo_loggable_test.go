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

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pbtesting"
)

func Test_OutputInfoLoggable_GetInspectionResults(t *testing.T) {
	outputInfo := &pb.OutputInfo{
		InspectionResults: &pb.InspectionResults{
			OsRelease: &pb.OsRelease{
				DistroId: pb.Distro_WINDOWS,
			},
			RootFs:  path.RandString(5),
			OsCount: 1,
		},
	}
	pbtesting.AssertEqual(t, outputInfo.InspectionResults, NewOutputInfoLoggable(outputInfo).GetInspectionResults())
}

func Test_OutputInfoLoggable_GetValue(t *testing.T) {
	outputInfo := &pb.OutputInfo{
		ImportFileFormat:      "vmdk",
		InflationType:         "api",
		ShadowDiskMatchResult: "matched",
	}

	loggable := NewOutputInfoLoggable(outputInfo)
	assert.Equal(t, "vmdk", loggable.GetValue(importFileFormat))
	assert.Equal(t, "api", loggable.GetValue(inflationType))
	assert.Equal(t, "matched", loggable.GetValue(shadowDiskMatchResult))
	assert.Equal(t, "", loggable.GetValue("not-a-key"))
}

func Test_OutputInfoLoggable_GetValueAsBool(t *testing.T) {
	for i, tt := range []struct {
		input                  *pb.OutputInfo
		expectedUefiCompatible bool
		expectedUefiDetected   bool
	}{
		{
			input:                  &pb.OutputInfo{},
			expectedUefiCompatible: false,
			expectedUefiDetected:   false,
		},
		{
			input:                  &pb.OutputInfo{IsUefiCompatibleImage: true},
			expectedUefiCompatible: true,
			expectedUefiDetected:   false,
		},
		{
			input:                  &pb.OutputInfo{IsUefiDetected: true},
			expectedUefiCompatible: false,
			expectedUefiDetected:   true,
		},
	} {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			loggable := NewOutputInfoLoggable(tt.input)
			assert.Equal(t, tt.expectedUefiCompatible, loggable.GetValueAsBool(isUEFICompatibleImage))
			assert.Equal(t, tt.expectedUefiDetected, loggable.GetValueAsBool(isUEFIDetected))
			assert.False(t, loggable.GetValueAsBool("random-key"))
		})
	}
}

func Test_OutputInfoLoggable_GetValueAsInt64Slice(t *testing.T) {
	outputInfo := &pb.OutputInfo{
		TargetsSizeGb:         []int64{1, 2},
		SourcesSizeGb:         []int64{3, 4},
		InflationTimeMs:       []int64{5, 6},
		ShadowInflationTimeMs: []int64{7, 8},
	}

	loggable := NewOutputInfoLoggable(outputInfo)
	assert.Equal(t, []int64{1, 2}, loggable.GetValueAsInt64Slice(targetSizeGb))
	assert.Equal(t, []int64{3, 4}, loggable.GetValueAsInt64Slice(sourceSizeGb))
	assert.Equal(t, []int64{5, 6}, loggable.GetValueAsInt64Slice(inflationTime))
	assert.Equal(t, []int64{7, 8}, loggable.GetValueAsInt64Slice(shadowInflationTime))
	assert.Empty(t, loggable.GetValueAsInt64Slice("not-a-key"))
}

func Test_OutputInfoLoggable_ReadSerialPortLogs(t *testing.T) {
	outputInfo := &pb.OutputInfo{
		ImportFileFormat:      "vmdk",
		InflationType:         "api",
		ShadowDiskMatchResult: "matched",
	}

	loggable := NewOutputInfoLoggable(outputInfo)
	assert.Equal(t, "vmdk", loggable.GetValue(importFileFormat))
	assert.Equal(t, "api", loggable.GetValue(inflationType))
	assert.Equal(t, "matched", loggable.GetValue(shadowDiskMatchResult))
	assert.Equal(t, "", loggable.GetValue("not-a-key"))
}
