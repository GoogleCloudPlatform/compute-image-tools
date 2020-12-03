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
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/stretchr/testify/assert"
)

func TestDefaultProcessorProvider_InspectDataDisk(t *testing.T) {
	processorProvider := defaultProcessorProvider{
		ImportArguments: ImportArguments{
			WorkflowDir: "../../../daisy_workflows",
			DataDisk:    true,
		},
	}

	processors, err := processorProvider.provide(persistentDisk{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(processors), "there should be 1 processor, got %v", len(processors))
	_, ok := processors[0].(*dataDiskProcessor)
	assert.True(t, ok, "processor is not dataDiskProcessor")
}

func TestDefaultProcessorProvider_InspectOS(t *testing.T) {
	processorProvider := defaultProcessorProvider{
		ImportArguments: ImportArguments{
			Inspect:     true,
			WorkflowDir: "../../../daisy_workflows",
			OS:          "ubuntu-1804",
		},
	}

	pd := persistentDisk{}
	processors, err := processorProvider.provide(pd)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(processors), "there should be 3 processors, got %v", len(processors))
	p, ok := processors[0].(*diskInspectionProcessor)
	assert.True(t, ok, "the 1st processor is not diskInspectionDiskProcessor")

	p.process(pd, service.NewSingleImageImportLoggableBuilder())
}

func TestDefaultProcessorProvider_InspectUEFI(t *testing.T) {
	processorProvider := defaultProcessorProvider{
		ImportArguments: ImportArguments{
			WorkflowDir: "../../../daisy_workflows",
			OS:          "ubuntu-1804",
			Inspect:     true,
		},
	}

	pd := persistentDisk{uri: "old-uri"}
	processors, err := processorProvider.provide(pd)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(processors), "there should be 3 processors, got %v", len(processors))
	_, ok := processors[0].(*diskInspectionProcessor)
	assert.True(t, ok, "the 1st processor is not diskInspectionDiskProcessor")
	_, ok = processors[1].(*metadataProcessor)
	assert.True(t, ok, "the 2nd processor is not uefiProcessor")
	_, ok = processors[2].(*bootableDiskProcessor)
	assert.True(t, ok, "the 3rd processor is not bootableDiskProcessor")
}
