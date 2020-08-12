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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/stretchr/testify/assert"
	v1 "google.golang.org/api/compute/v1"
)

func TestProcessorProvider_InspectDataDisk(t *testing.T) {
	processorProvider := defaultProcessorProvider{
		ImportArguments{
			Inspect:     true,
			WorkflowDir: "testdata",
			DataDisk:    true,
		},
		mockCreateDiskSuccessClient{},
		mockDiskInspector{true},
	}

	processor, err := processorProvider.provide(persistentDisk{})
	assert.NoError(t, err)
	_, ok := processor.(*dataDiskProcessor)
	assert.True(t, ok, "processor is not dataDiskProcessor")
}

func TestProcessorProvider_InspectUEFI(t *testing.T) {
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
			processorProvider := defaultProcessorProvider{
				ImportArguments{
					Inspect:        true,
					WorkflowDir:    "testdata",
					OS:             "ubuntu-1804",
					UefiCompatible: tt.isInputArgUEFICompatible,
				},
				mockCreateDiskSuccessClient{},
				mockDiskInspector{tt.isUEFIDisk},
			}

			processor, err := processorProvider.provide(persistentDisk{})
			assert.NoError(t, err)
			actualProcessor, ok := processor.(*bootableDiskProcessor)
			assert.True(t, ok, "processor is not bootableDiskProcessor")
			pd, err := actualProcessor.inspectAndPreProcess()
			assert.NoError(t, err)

			if tt.isUEFIDisk && !tt.isInputArgUEFICompatible {
				assert.Truef(t, strings.Contains(pd.uri, "uefi"), "UEFI Disk URI should contains 'uefi', actual: %v", pd.uri)
			} else {
				assert.Falsef(t, strings.Contains(pd.uri, "uefi"), "Disk URI shouldn't contain 'uefi', actual: %v", pd.uri)
			}
			assert.Equal(t, tt.isUEFIDisk, pd.isUEFIDetected)
			assert.Equal(t, tt.isInputArgUEFICompatible || tt.isUEFIDisk, pd.isUEFICompatible)
		})
	}
}

type mockDiskInspector struct {
	hasEFIPartition bool
}

func (m mockDiskInspector) Inspect(reference string) (ir disk.InspectionResult, err error) {
	ir.HasEFIPartition = m.hasEFIPartition
	return
}

type mockCreateDiskSuccessClient struct {
	*mocks.MockClient
}

func (m mockCreateDiskSuccessClient) CreateDisk(arg0, arg1 string, arg2 *v1.Disk) error {
	return nil
}
