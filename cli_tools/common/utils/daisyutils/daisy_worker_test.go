//  Copyright 2021 Google Inc. All Rights Reserved.
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

package daisyutils

import (
	"errors"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func Test_NewDaisyWorker_IncludesStandardModifiers(t *testing.T) {
	wf := daisy.New()
	env := EnvironmentSettings{}
	worker := NewDaisyWorker(wf, env, logging.NewToolLogger("test"))
	assert.Equal(t, appliedModifiers{
		applyEnvToWorkflow:       true,
		configureDaisyLogging:    true,
		removeExternalIPModifier: false,
	}, findWhichModifiersApplied(worker))
}

func Test_NewDaisyWorker_IncludesNoExternalIPModifier_WhenRequestedByUser(t *testing.T) {
	wf := daisy.New()
	env := EnvironmentSettings{NoExternalIP: true}
	worker := NewDaisyWorker(wf, env, logging.NewToolLogger("test"))
	assert.Equal(t, appliedModifiers{
		applyEnvToWorkflow:       true,
		configureDaisyLogging:    true,
		removeExternalIPModifier: true,
	}, findWhichModifiersApplied(worker))
}

func Test_DaisyWorkerRun_RunsCustomModifiers(t *testing.T) {
	wf := daisy.New()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	modifier1 := mocks.NewMockWorkflowModifier(mockCtrl)
	modifier1.EXPECT().Modify(wf).Return(nil)
	modifier2 := mocks.NewMockWorkflowModifier(mockCtrl)
	modifier2.EXPECT().Modify(wf).Return(nil)

	env := EnvironmentSettings{
		Project:            "lucky-lemur",
		Zone:               "us-west1-c",
		GCSPath:            "gs://test",
		Timeout:            "60s",
		DaisyLogLinePrefix: "import-image",
	}
	configWorkflowForUnitTesting(t, wf, mockCtrl, env)

	worker := NewDaisyWorker(wf, env, logging.NewToolLogger("test"), modifier1, modifier2)
	assert.NoError(t, worker.Run(map[string]string{}))
}

func Test_DaisyWorkerRun_CapturesDaisyLogs(t *testing.T) {
	wf := daisy.New()

	serialLogs := []string{"serial logs"}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockLogger := mocks.NewMockDaisyLogger(mockCtrl)
	mockLogger.EXPECT().WriteLogEntry(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().ReadSerialPortLogs().Return(serialLogs)
	wf.Logger = mockLogger

	env := EnvironmentSettings{
		Project:            "lucky-lemur",
		Zone:               "us-west1-c",
		GCSPath:            "gs://test",
		Timeout:            "60s",
		DaisyLogLinePrefix: "import-image",
	}
	configWorkflowForUnitTesting(t, wf, mockCtrl, env)

	toolLogger := logging.NewToolLogger("test")
	worker := NewDaisyWorker(wf, env, toolLogger)
	assert.NoError(t, worker.Run(map[string]string{}))
	assert.Equal(t, serialLogs, toolLogger.ReadOutputInfo().SerialOutputs)
}

func Test_DaisyWorkerRun_FailsIfModifierFails(t *testing.T) {
	wf := daisy.New()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	modifier := mocks.NewMockWorkflowModifier(mockCtrl)
	modifier.EXPECT().Modify(wf).Return(errors.New("modifier failed"))

	worker := NewDaisyWorker(wf, EnvironmentSettings{}, logging.NewToolLogger("test"), modifier)
	assert.EqualError(t, worker.Run(map[string]string{}), "modifier failed")
}

func Test_DaisyWorkerRun_FailsIfWorkflowFails(t *testing.T) {
	wf := daisy.New()
	worker := NewDaisyWorker(wf, EnvironmentSettings{}, logging.NewToolLogger("test"))
	assert.EqualError(t, worker.Run(map[string]string{}),
		"error validating workflow: must provide workflow field 'Name'")
}

func Test_DaisyWorkerRun_AppliesEnvToWorkflow(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	wf := daisy.New()
	wf.Project = "old-project"
	wf.Zone = "old-zone"
	wf.GCSPath = "old-gcs-path"
	wf.DefaultTimeout = "old-timeout"
	wf.ComputeEndpoint = "old-endpoint"

	env := EnvironmentSettings{
		Project:            "lucky-lemur",
		Zone:               "us-west1-c",
		GCSPath:            "gs://test",
		Timeout:            "60s",
		ComputeEndpoint:    "new-endpoint",
		DaisyLogLinePrefix: "import-image",
	}
	configWorkflowForUnitTesting(t, wf, mockCtrl, env)

	worker := NewDaisyWorker(wf, env, logging.NewToolLogger("test"))
	if err := worker.Run(map[string]string{}); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, wf.Project, "lucky-lemur")
	assert.Equal(t, wf.Zone, "us-west1-c")
	assert.Equal(t, wf.GCSPath, "gs://test")
	assert.Equal(t, wf.DefaultTimeout, "60s")
	assert.Equal(t, wf.ComputeEndpoint, "new-endpoint")
	assert.Equal(t, wf.Name, "import-image")
}

func Test_DaisyWorkerRun_AppliesVariables(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	wf := daisy.New()
	wf.Vars = map[string]daisy.Var{
		"v": {Required: true},
	}
	env := EnvironmentSettings{
		Project:            "lucky-lemur",
		Zone:               "us-west1-c",
		GCSPath:            "gs://test",
		Timeout:            "60s",
		ComputeEndpoint:    "new-endpoint",
		DaisyLogLinePrefix: "import-image",
	}
	configWorkflowForUnitTesting(t, wf, mockCtrl, env)

	worker := NewDaisyWorker(wf, env, logging.NewToolLogger("test"))
	assert.NoError(t, worker.Run(map[string]string{"v": "value"}))
}

func Test_DaisyWorkerRunAndReadSerialValue_HappyCase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	wf := daisy.New()
	wf.AddSerialConsoleOutputValue("serial-key", "serial-output-value")
	env := EnvironmentSettings{
		Project:            "lucky-lemur",
		Zone:               "us-west1-c",
		GCSPath:            "gs://test",
		Timeout:            "60s",
		ComputeEndpoint:    "new-endpoint",
		DaisyLogLinePrefix: "import-image",
	}
	configWorkflowForUnitTesting(t, wf, mockCtrl, env)

	worker := NewDaisyWorker(wf, env, logging.NewToolLogger("test"))
	actualValue, err := worker.RunAndReadSerialValue("serial-key", map[string]string{})
	assert.NoError(t, err)
	assert.Equal(t, "serial-output-value", actualValue)
}

type appliedModifiers struct {
	applyEnvToWorkflow, configureDaisyLogging, removeExternalIPModifier bool
}

func findWhichModifiersApplied(worker DaisyWorker) (t appliedModifiers) {
	realWorker := worker.(*defaultDaisyWorker)
	for _, modifier := range realWorker.modifiers {
		if _, ok := modifier.(*ApplyEnvToWorkflow); ok {
			t.applyEnvToWorkflow = true
		}
		if _, ok := modifier.(*ConfigureDaisyLogging); ok {
			t.configureDaisyLogging = true
		}
		if _, ok := modifier.(*RemoveExternalIPModifier); ok {
			t.removeExternalIPModifier = true
		}
	}
	return t
}

// configWorkflowForUnitTesting adds a minimal number of steps and mocks so that the workflow can run without errors.
func configWorkflowForUnitTesting(t *testing.T, wf *daisy.Workflow, mockCtrl *gomock.Controller, env EnvironmentSettings) {
	step, err := wf.NewStep("test")
	if err != nil {
		t.Fatal(err)
	}
	step.StartInstances = &daisy.StartInstances{}
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetProject(env.Project).Return(nil, nil).AnyTimes()
	mockComputeClient.EXPECT().ListZones(env.Project).Return([]*compute.Zone{{Name: env.Zone}}, nil).AnyTimes()
	wf.ComputeClient = mockComputeClient
	wf.StorageClient = &storage.Client{}
	wf.DisableCloudLogging()
	wf.DisableCloudLogging()
	wf.DisableGCSLogging()
}
