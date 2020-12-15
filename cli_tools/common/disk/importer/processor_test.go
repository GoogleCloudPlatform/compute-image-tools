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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DefaultProcessorProvider_SkipsPlanningForDataDisk(t *testing.T) {
	processorProvider := defaultProcessorProvider{
		ImportArguments: ImportArguments{
			DataDisk: true,
		},
	}

	processors, err := processorProvider.provide(persistentDisk{})
	assert.NoError(t, err)
	assert.Len(t, processors, 1)
	assert.IsType(t, &dataDiskProcessor{}, processors[0])
}

func Test_DefaultProcessorProvider_IncludesMetadataStepWhenMetadataChangesRequired(t *testing.T) {
	processorProvider := defaultProcessorProvider{
		ImportArguments: ImportArguments{
			WorkflowDir: "../../../../daisy_workflows",
		},
		planner: mockProcessPlanner{
			result: &processingPlan{
				requiredLicenses:        []string{"url/license"},
				translationWorkflowPath: opensuse15workflow,
			},
		},
	}
	processors, err := processorProvider.provide(persistentDisk{})
	assert.NoError(t, err)
	assert.Len(t, processors, 2)
	assert.IsType(t, &metadataProcessor{}, processors[0])
	assert.IsType(t, &bootableDiskProcessor{}, processors[1])
}

func Test_DefaultProcessorProvider_SkipsMetadataStepWhenNoChangesRequired(t *testing.T) {
	processorProvider := defaultProcessorProvider{
		ImportArguments: ImportArguments{
			WorkflowDir: "../../../../daisy_workflows",
		},
		planner: mockProcessPlanner{
			result: &processingPlan{
				translationWorkflowPath: opensuse15workflow,
			},
		},
	}
	processors, err := processorProvider.provide(persistentDisk{})
	assert.NoError(t, err)
	assert.Len(t, processors, 1)
	assert.IsType(t, &bootableDiskProcessor{}, processors[0])
}

func Test_DefaultProcessorProvider_FailsWhenPlanningFails(t *testing.T) {
	processorProvider := defaultProcessorProvider{
		planner: mockProcessPlanner{err: errors.New("planning failed")},
	}
	_, err := processorProvider.provide(persistentDisk{})
	assert.Error(t, err, "planning failed")
}

type mockProcessPlanner struct {
	err    error
	result *processingPlan
}

func (m mockProcessPlanner) plan(pd persistentDisk) (*processingPlan, error) {
	return m.result, m.err
}
