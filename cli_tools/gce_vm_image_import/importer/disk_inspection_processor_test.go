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

package importer

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
)

func TestDiskInspectionProcessor_ProcessUEFI(t *testing.T) {
	tests := []struct {
		isUEFIDisk               bool
		isInputArgUEFICompatible bool
	}{
		{isUEFIDisk: true, isInputArgUEFICompatible: false},
		{isUEFIDisk: false, isInputArgUEFICompatible: false},
		{isUEFIDisk: true, isInputArgUEFICompatible: true},
		{isUEFIDisk: false, isInputArgUEFICompatible: true},
	}

	for i, tt := range tests {
		name := fmt.Sprintf("%v. inspect disk: disk is UEFI: %v, input arg UEFI compatible: %v", i+1, tt.isUEFIDisk, tt.isInputArgUEFICompatible)
		t.Run(name, func(t *testing.T) {
			args := ImportArguments{
				UefiCompatible: tt.isInputArgUEFICompatible,
			}
			p := newDiskInspectionProcessor(mockDiskInspector{tt.isUEFIDisk, &daisy.Workflow{}}, args)
			pd, err := p.process(persistentDisk{}, service.NewSingleImageImportLoggableBuilder())
			assert.NoError(t, err)
			assert.Equal(t, tt.isInputArgUEFICompatible || tt.isUEFIDisk, pd.isUEFICompatible)
		})
	}
}

type mockDiskInspector struct {
	hasEFIPartition bool
	wf              *daisy.Workflow
}

func (m mockDiskInspector) Inspect(reference string, inspectOS bool) (ir disk.InspectionResult, err error) {
	ir.HasEFIPartition = m.hasEFIPartition
	return
}

func (m mockDiskInspector) Cancel(reason string) bool {
	return false
}

func (m mockDiskInspector) TraceLogs() []string {
	return []string{}
}
