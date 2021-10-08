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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func init() {
	initTest()
}

type TestUpgrader struct {
	*upgrader
}

func TestUpgraderRunFailedOnInit(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.initFn = nil
	tu.Oauth = "bad-oauth"

	err := tu.run()
	assert.EqualError(t, err, "Failed to create GCE client: error creating HTTP API client: cannot read credentials file: open bad-oauth: no such file or directory")
}

func TestUpgraderRunFailedOnValidateParams(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.validateAndDeriveParamsFn = func() error {
		return fmt.Errorf("failed")
	}

	err := tu.run()
	assert.EqualError(t, err, "failed")
}

func TestUpgraderRunFailedOnPrintUpgradeGuide(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.printIntroHelpTextFn = func() error {
		return fmt.Errorf("failed")
	}

	err := tu.run()
	assert.EqualError(t, err, "failed")
}

func TestUpgraderRunFailedOnPrepare(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.prepareFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("failed")
	}
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}

	err := tu.run()
	assert.EqualError(t, err, "failed")
}

func TestUpgraderRunFailedOnUpgrade(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.upgradeFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("failed")
	}
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}

	err := tu.run()
	assert.EqualError(t, err, "failed")
}

func TestUpgraderRunFailedOnReboot(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.upgradeFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("Windows needs to be restarted")
	}
	tu.rebootFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("failed")
	}
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}

	err := tu.run()
	assert.EqualError(t, err, "failed")
}

func TestUpgraderRunFailedOnRetryUpgrade(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.upgradeFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("Windows needs to be restarted")
	}
	tu.rebootFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}
	tu.retryUpgradeFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("failed")
	}
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}

	err := tu.run()
	assert.EqualError(t, err, "failed")
}

func TestUpgraderRunSuccessWithoutReboot(t *testing.T) {
	tu := initTestUpgrader(t)

	err := tu.run()
	assert.NoError(t, err)
}

func TestUpgraderRunSuccessWithReboot(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.upgradeFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("Windows needs to be restarted")
	}
	tu.rebootFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}
	tu.retryUpgradeFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}

	err := tu.run()
	assert.NoError(t, err)
}

func TestUpgraderRunFailedWithAutoRollback(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.prepareFn = func() (*daisy.Workflow, error) {
		// Test workaround: let newOSDiskName to be the same as current disk name
		// in order to trigger auto rollback.
		tu.newOSDiskName = testDisk
		return nil, fmt.Errorf("failed")
	}
	tu.AutoRollback = true
	rollbackExecuted := false
	tu.rollbackFn = func() (*daisy.Workflow, error) {
		rollbackExecuted = true
		return nil, nil
	}
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		t.Errorf("Unexpected cleanup.")
		return nil, nil
	}

	err := tu.run()
	assert.EqualError(t, err, "failed")
	assert.True(t, rollbackExecuted, "Rollback not executed.")
}

func TestUpgraderRunFailedWithAutoRollbackFailed(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.prepareFn = func() (*daisy.Workflow, error) {
		// Test workaround: let newOSDiskName to be the same as current disk name
		// in order to trigger auto rollback.
		tu.newOSDiskName = testDisk
		return nil, fmt.Errorf("failed1")
	}
	tu.AutoRollback = true
	rollbackExecuted := false
	tu.rollbackFn = func() (*daisy.Workflow, error) {
		rollbackExecuted = true
		return nil, fmt.Errorf("failed2")
	}

	err := tu.run()
	assert.EqualError(t, err, "failed1")
	assert.True(t, rollbackExecuted, "Rollback not executed.")
}

func TestUpgraderRunFailedWithAutoRollbackWithoutNewOSDiskAttached(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.prepareFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("failed1")
	}
	tu.AutoRollback = true
	cleanupExecuted := false
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		cleanupExecuted = true
		return nil, fmt.Errorf("failed2")
	}
	err := tu.run()
	assert.EqualError(t, err, "failed1")
	assert.True(t, cleanupExecuted, "Cleanup not executed.")
}

func initTestUpgrader(t *testing.T) *TestUpgrader {
	tu := newTestUpgrader()
	tu.initFn = func() error {
		computeClient = newTestGCEClient()
		return nil
	}
	tu.prepareFn = func() (workflow *daisy.Workflow, e error) {
		// Test workaround: let newOSDiskName to be the same as current disk name
		// in order to pretend the enw OS disk has been attached.
		tu.newOSDiskName = testDisk
		return nil, nil
	}
	tu.upgradeFn = func() (workflow *daisy.Workflow, e error) {
		return nil, nil
	}
	tu.rebootFn = func() (workflow *daisy.Workflow, e error) {
		t.Errorf("Unexpected reboot.")
		return nil, nil
	}
	tu.retryUpgradeFn = func() (workflow *daisy.Workflow, e error) {
		t.Errorf("Unexpected retryUpgrade.")
		return nil, nil
	}
	tu.cleanupFn = func() (workflow *daisy.Workflow, e error) {
		t.Errorf("Unexpected cleanup.")
		return nil, nil
	}
	tu.rollbackFn = func() (workflow *daisy.Workflow, e error) {
		t.Errorf("Unexpected rollback.")
		return nil, nil
	}
	return tu
}
