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

func Test_NewDaisyWorker_IncludesStandardHooks(t *testing.T) {
	wf := daisy.New()
	env := EnvironmentSettings{
		Labels:             map[string]string{"env": "prod"},
		StorageLocation:    "us-east",
		DaisyLogLinePrefix: "import-image",
		ExecutionID:        "b1234",
		Tool:               Tool{ResourceLabelName: "unit-test"},
	}
	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, env, logging.NewToolLogger("test"))
	assert.Equal(t, appliedHooks{
		applyEnvToWorkflow:    true,
		configureDaisyLogging: true,
		removeExternalIPHook:  false,
		resourceLabeler:       true,
	}, findWhichHooksApplied(worker))
	rl := getResourceLabeler(t, worker)
	assert.Equal(t, env.Labels, rl.UserLabels)
	assert.Equal(t, env.StorageLocation, rl.ImageLocation)
	assert.Contains(t, rl.BuildIDLabelKey, env.Tool.ResourceLabelName)
	assert.Equal(t, env.ExecutionID, rl.BuildID)
}

func Test_NewDaisyWorker_Panics_IfHooksDontImplementPreOrPostInterface(t *testing.T) {
	wf := daisy.New()
	env := EnvironmentSettings{
		ExecutionID: "b1234",
		Tool:        Tool{ResourceLabelName: "unit-test"},
	}
	type notAHook struct {
	}
	assert.PanicsWithValue(t, "daisyutils.notAHook must implement WorkflowPreHook and/or WorkflowPostHook", func() {
		NewDaisyWorker(func() (*daisy.Workflow, error) {
			return wf, nil
		}, env, logging.NewToolLogger("test"), notAHook{})
	})
}

func Test_NewDaisyWorker_IncludesNoExternalIPHook_WhenRequestedByUser(t *testing.T) {
	wf := daisy.New()
	env := EnvironmentSettings{NoExternalIP: true,
		ExecutionID: "b1234",
		Tool:        Tool{ResourceLabelName: "unit-test"},
	}
	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, env, logging.NewToolLogger("test"))
	assert.Equal(t, appliedHooks{
		applyEnvToWorkflow:    true,
		configureDaisyLogging: true,
		removeExternalIPHook:  true,
		resourceLabeler:       true,
	}, findWhichHooksApplied(worker))
}

func Test_NewDaisyWorker_KeepsResourceLabelerIfSpecified(t *testing.T) {
	wf := daisy.New()
	env := EnvironmentSettings{NoExternalIP: true, ExecutionID: "b1234",
		Tool: Tool{ResourceLabelName: "unit-test"}}
	rl := NewResourceLabeler("tool", "buildid", map[string]string{}, "location")
	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, env, logging.NewToolLogger("test"), rl)
	actualResourceLabeler := getResourceLabeler(t, worker)
	assert.Equal(t, rl, actualResourceLabeler)
}

func Test_DaisyWorkerRun_RunsCustomPreHooks(t *testing.T) {
	wf := daisy.New()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	hook1 := mocks.NewMockWorkflowPreHook(mockCtrl)
	hook1.EXPECT().PreRunHook(wf).Return(nil)
	hook2 := mocks.NewMockWorkflowPreHook(mockCtrl)
	hook2.EXPECT().PreRunHook(wf).Return(nil)

	env := EnvironmentSettings{
		Project:            "lucky-lemur",
		Zone:               "us-west1-c",
		GCSPath:            "gs://test",
		Timeout:            "60s",
		DaisyLogLinePrefix: "import-image",
		ExecutionID:        "b1234",
		Tool:               Tool{ResourceLabelName: "unit-test"},
	}
	configWorkflowForUnitTesting(t, wf, mockCtrl, env)

	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, env, logging.NewToolLogger("test"), hook1, hook2)
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
		ExecutionID:        "b1234",
		Tool:               Tool{ResourceLabelName: "unit-test"},
	}
	configWorkflowForUnitTesting(t, wf, mockCtrl, env)

	toolLogger := logging.NewToolLogger("test")
	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, env, toolLogger)
	assert.NoError(t, worker.Run(map[string]string{}))
	assert.Equal(t, serialLogs, toolLogger.ReadOutputInfo().SerialOutputs)
}

func Test_DaisyWorkerRun_DoesntRunWorkflow_IfCancelWasCalled(t *testing.T) {
	wf := daisy.New()
	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, EnvironmentSettings{
		ExecutionID: "b1234",
		Tool:        Tool{ResourceLabelName: "unit-test"},
	}, logging.NewToolLogger("test"))
	worker.Cancel("don't run workflow")
	assert.EqualError(t, worker.Run(map[string]string{}), "workflow canceled: don't run workflow")
}

func Test_DaisyWorkerCancel_SafeToCallMultipleTimes(t *testing.T) {
	wf := daisy.New()
	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, EnvironmentSettings{
		ExecutionID: "b1234",
		Tool:        Tool{ResourceLabelName: "unit-test"},
	}, logging.NewToolLogger("test"))
	worker.Cancel("reason")
	worker.Cancel("reason")
	worker.Cancel("reason")
	assert.EqualError(t, worker.Run(map[string]string{}), "workflow canceled: reason")
	assert.EqualError(t, worker.Run(map[string]string{}), "workflow canceled")
}

func Test_DaisyWorkerRun_DoesntRunWorkflow_IfCancelChanelClosedAndEmpty(t *testing.T) {
	wf := daisy.New()
	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, EnvironmentSettings{
		ExecutionID: "b1234",
		Tool:        Tool{ResourceLabelName: "unit-test"},
	}, logging.NewToolLogger("test"))
	close(worker.(*defaultDaisyWorker).cancel)
	assert.EqualError(t, worker.Run(map[string]string{}), "workflow canceled")
}

func Test_DaisyWorkerRun_DoesntRunWorkflow_IfWorkflowAlreadyCanceled(t *testing.T) {
	wf := daisy.New()
	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, EnvironmentSettings{
		ExecutionID: "b1234",
		Tool:        Tool{ResourceLabelName: "unit-test"},
	}, logging.NewToolLogger("test"))
	close(wf.Cancel)
	assert.EqualError(t, worker.Run(map[string]string{}), "workflow canceled")
}

func Test_DaisyWorkerRun_FailsIfPreHookFails(t *testing.T) {
	wf := daisy.New()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	hook := mocks.NewMockWorkflowPreHook(mockCtrl)
	hook.EXPECT().PreRunHook(wf).Return(errors.New("hook failed"))

	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, EnvironmentSettings{
		ExecutionID: "b1234",
		Tool:        Tool{ResourceLabelName: "unit-test"},
	}, logging.NewToolLogger("test"), hook)
	assert.EqualError(t, worker.Run(map[string]string{}), "hook failed")
}

func Test_DaisyWorkerRun_FailsIfWorkflowFails(t *testing.T) {
	wf := daisy.New()
	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, EnvironmentSettings{
		ExecutionID: "b1234",
		Tool:        Tool{ResourceLabelName: "unit-test"},
	}, logging.NewToolLogger("test"))
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
		ExecutionID:        "b1234",
		Tool:               Tool{ResourceLabelName: "unit-test"},
	}
	configWorkflowForUnitTesting(t, wf, mockCtrl, env)

	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, env, logging.NewToolLogger("test"))
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

func Test_DaisyWorkerRun_DoesntRunWorkflowMoreThanTwiceEventWhenRequested(t *testing.T) {
	expectedError := "error validating workflow: must provide workflow field 'Name'"
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	numWorkflowInvocations := 0
	postHook := mocks.NewMockWorkflowPostHook(mockCtrl)
	postHook.EXPECT().PostRunHook(gomock.Any()).DoAndReturn(
		func(err error) (bool, error) {
			assert.EqualError(t, err, expectedError)
			return true, err
		}).Times(2)
	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		numWorkflowInvocations++
		return daisy.New(), nil
	}, EnvironmentSettings{
		ExecutionID: "b1234",
		Tool:        Tool{ResourceLabelName: "unit-test"},
	}, logging.NewToolLogger("test"), postHook)
	assert.EqualError(t, worker.Run(map[string]string{}),
		expectedError)
	assert.Equal(t, 2, numWorkflowInvocations)
}

func Test_DaisyWorkerRun_DoesntReRunFailedWorkflowIfNotRequested(t *testing.T) {
	expectedError := "error validating workflow: must provide workflow field 'Name'"
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	numWorkflowInvocations := 0
	postHook := mocks.NewMockWorkflowPostHook(mockCtrl)
	postHook.EXPECT().PostRunHook(gomock.Any()).DoAndReturn(
		func(err error) (bool, error) {
			assert.EqualError(t, err, expectedError)
			return false, err
		}).Times(1)
	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		numWorkflowInvocations++
		return daisy.New(), nil
	}, EnvironmentSettings{
		ExecutionID: "b1234",
		Tool:        Tool{ResourceLabelName: "unit-test"},
	}, logging.NewToolLogger("test"), postHook)
	assert.EqualError(t, worker.Run(map[string]string{}),
		expectedError)
	assert.Equal(t, 1, numWorkflowInvocations)
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
		ExecutionID:        "b1234",
		Tool:               Tool{ResourceLabelName: "unit-test"},
	}
	configWorkflowForUnitTesting(t, wf, mockCtrl, env)

	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, env, logging.NewToolLogger("test"))
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
		ExecutionID:        "b1234",
		Tool:               Tool{ResourceLabelName: "unit-test"},
	}
	configWorkflowForUnitTesting(t, wf, mockCtrl, env)

	worker := NewDaisyWorker(func() (*daisy.Workflow, error) {
		return wf, nil
	}, env, logging.NewToolLogger("test"))
	actualValue, err := worker.RunAndReadSerialValue("serial-key", map[string]string{})
	assert.NoError(t, err)
	assert.Equal(t, "serial-output-value", actualValue)
}

type appliedHooks struct {
	applyEnvToWorkflow, configureDaisyLogging, removeExternalIPHook, resourceLabeler bool
}

func findWhichHooksApplied(worker DaisyWorker) (t appliedHooks) {
	realWorker := worker.(*defaultDaisyWorker)
	for _, hook := range realWorker.hooks {
		if _, ok := hook.(*ApplyEnvToWorkflow); ok {
			t.applyEnvToWorkflow = true
		}
		if _, ok := hook.(*ConfigureDaisyLogging); ok {
			t.configureDaisyLogging = true
		}
		if _, ok := hook.(*RemoveExternalIPHook); ok {
			t.removeExternalIPHook = true
		}
		if _, ok := hook.(*ResourceLabeler); ok {
			t.resourceLabeler = true
		}
	}
	return t
}

func getResourceLabeler(t *testing.T, worker DaisyWorker) *ResourceLabeler {
	var actualResourceLabeler *ResourceLabeler
	realWorker := worker.(*defaultDaisyWorker)
	for _, hook := range realWorker.hooks {
		switch hook.(type) {
		case *ResourceLabeler:
			if actualResourceLabeler != nil {
				assert.Fail(t, "Found more than one resource labeler in hooks")
			}
			actualResourceLabeler = hook.(*ResourceLabeler)
		}
	}
	return actualResourceLabeler
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
