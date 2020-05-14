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

package upgrader

import (
	"testing"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
)

func TestGeneratePrepareWorkflow(t *testing.T) {
	type testCase struct {
		testName               string
		populateFunc           func(*upgrader, *daisy.Workflow) error
		instanceName           string
		skipMachineImageBackup bool
	}

	tcs := []testCase{
		{"prepare", populatePrepareSteps, testInstance, true},
		{"prepare with original startup script", populatePrepareSteps, testInstanceWithStartupScript, false},
	}

	for _, tc := range tcs {
		u := initTest()
		u.Instance = daisyutils.GetInstanceURI(testProject, testZone, tc.instanceName)

		err := u.validateAndDeriveParams()
		if err != nil {
			t.Errorf("[%v]: validateAndDeriveParams failed: %v", tc.testName, err)
			continue
		}

		w, err := u.generateWorkflowWithSteps("test", DefaultTimeout, tc.populateFunc)
		assert.NoError(t, err, "[test name: %v] Unexpected error.", tc.testName)

		_, hasBackupMachineImageStep := w.Steps["backup-machine-image"]
		if u.SkipMachineImageBackup && hasBackupMachineImageStep {
			t.Errorf("[%v]: Skiped machine image backup but still see this step in workflow.", tc.testName)
		} else if !u.SkipMachineImageBackup && !hasBackupMachineImageStep {
			t.Errorf("[%v]: Didn't skip machine image backup but can't see this step in workflow.", tc.testName)
		}

		_, hasBackupStartupScriptStep := w.Steps["backup-script"]
		if tc.instanceName != testInstanceWithStartupScript && hasBackupStartupScriptStep {
			t.Errorf("[%v]: Original startup script doesn't exist but still see this step in workflow.", tc.testName)
		} else if tc.instanceName == testInstanceWithStartupScript && !hasBackupStartupScriptStep {
			t.Errorf("[%v]: Original startup script exists but can't see this step in workflow.", tc.testName)
		}
	}
}

func TestGenerateStaticWorkflow(t *testing.T) {
	type testCase struct {
		testName     string
		populateFunc func(*upgrader, *daisy.Workflow) error
		instanceName string
	}

	tcs := []testCase{
		{"reboot", populateRebootSteps, testInstance},
		{"cleanup", populateCleanupSteps, testInstance},
		{"rollback", populateRollbackSteps, testInstanceWithStartupScript},
	}
	for sourceOS := range supportedSourceOSVersions {
		tcs = append(tcs, testCase{
			"upgrade-" + sourceOS,
			upgradeSteps[sourceOS],
			testInstance,
		})
		tcs = append(tcs, testCase{
			"retry-upgrade-" + sourceOS,
			retryUpgradeSteps[sourceOS],
			testInstanceWithStartupScript,
		})
	}

	for _, tc := range tcs {
		u := initTest()
		u.Instance = daisyutils.GetInstanceURI(testProject, testZone, tc.instanceName)

		err := u.validateAndDeriveParams()
		if err != nil {
			t.Errorf("[%v]: validateAndDeriveParams failed: %v", tc.testName, err)
			continue
		}

		_, err = u.generateWorkflowWithSteps("test", DefaultTimeout, tc.populateFunc)
		assert.NoError(t, err, "[test name: %v] Unexpected error.", tc.testName)
	}
}

func TestRunWorkflowWithSteps(t *testing.T) {
	type testCase struct {
		testName                  string
		populateFunc              func(*upgrader, *daisy.Workflow) error
		expectExitOnPopulateSteps bool
	}

	tcs := []testCase{
		{"populate without error", func(u *upgrader, w *daisy.Workflow) error {
			w.Steps = map[string]*daisy.Step{
				"step1": {},
			}
			return nil
		}, false},
		{"populate with error", func(u *upgrader, w *daisy.Workflow) error {
			w.Steps = map[string]*daisy.Step{
				"step1": {},
			}
			return daisy.Errf("some error")
		}, true},
	}

	for _, tc := range tcs {
		u := initTest()
		err := u.validateAndDeriveParams()
		if err != nil {
			t.Errorf("[%v]: validateAndDeriveParams failed: %v", tc.testName, err)
			continue
		}

		w, err := u.runWorkflowWithSteps("test", "10m", tc.populateFunc)
		if _, ok := w.Steps["step1"]; !ok {
			t.Errorf("[%v]: missed step.", tc.testName)
		}

		if tc.expectExitOnPopulateSteps {
			if w.DefaultTimeout == u.Timeout {
				t.Errorf("[%v]: Default timeout of the workflow '%v' was overrided to '%v' unexpected.", tc.testName, w.DefaultTimeout, u.Timeout)
			}
		} else if w.DefaultTimeout != u.Timeout {
			t.Errorf("[%v]: Default timeout of the workflow '%v' should be overrided to '%v'.", tc.testName, w.DefaultTimeout, u.Timeout)
		}
	}
}

func TestRunAllWorkflowFunctions(t *testing.T) {
	u := initTest()

	type testCase struct {
		testName     string
		workflowFunc func() (*daisy.Workflow, error)
	}

	tcs := []testCase{
		{"prepare", u.prepare},
		{"upgrade", u.upgrade},
		{"reboot", u.reboot},
		{"retry-upgrade", u.retryUpgrade},
		{"rollback", u.rollback},
		{"cleanup", u.cleanup},
	}

	for _, tc := range tcs {
		err := u.validateAndDeriveParams()
		if err != nil {
			t.Errorf("[%v]: validateAndDeriveParams failed: %v", tc.testName, err)
			continue
		}

		w, err := tc.workflowFunc()

		if w.DefaultTimeout != u.Timeout {
			t.Errorf("[%v]: Default timeout of the workflow '%v' should be overrided to '%v'.", tc.testName, w.DefaultTimeout, u.Timeout)
		}
	}
}
