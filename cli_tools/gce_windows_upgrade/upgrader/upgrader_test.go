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

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
)

type TestUpgrader struct {
	*upgrader

	initFn               func() error
	printIntroHelpTextFn func() error
	validateParamsFn     func() error
	prepareFn            func() (*daisy.Workflow, error)
	upgradeFn            func() (*daisy.Workflow, error)
	retryUpgradeFn       func() (*daisy.Workflow, error)
	rebootFn             func() (*daisy.Workflow, error)
	cleanupFn            func() (*daisy.Workflow, error)
	rollbackFn           func() (*daisy.Workflow, error)
}

func (tu *TestUpgrader) getUpgrader() *upgrader {
	return tu.upgrader
}

func (tu *TestUpgrader) init() error {
	if tu.initFn == nil {
		return tu.upgrader.init()
	}
	return tu.initFn()
}

func (tu *TestUpgrader) printIntroHelpText() error {
	if tu.printIntroHelpTextFn == nil {
		return tu.upgrader.printIntroHelpText()
	}
	return tu.printIntroHelpTextFn()
}

func (tu *TestUpgrader) validateAndDeriveParams() error {
	if tu.validateParamsFn == nil {
		return tu.upgrader.validateAndDeriveParams()
	}
	return tu.validateParamsFn()
}

func (tu *TestUpgrader) prepare() (*daisy.Workflow, error) {
	if tu.prepareFn == nil {
		return tu.upgrader.prepare()
	}
	return tu.prepareFn()
}

func (tu *TestUpgrader) upgrade() (*daisy.Workflow, error) {
	if tu.upgradeFn == nil {
		return tu.upgrader.upgrade()
	}
	return tu.upgradeFn()
}

func (tu *TestUpgrader) retryUpgrade() (*daisy.Workflow, error) {
	if tu.retryUpgradeFn == nil {
		return tu.upgrader.retryUpgrade()
	}
	return tu.retryUpgradeFn()
}

func (tu *TestUpgrader) reboot() (*daisy.Workflow, error) {
	if tu.rebootFn == nil {
		return tu.upgrader.reboot()
	}
	return tu.rebootFn()
}

func (tu *TestUpgrader) cleanup() (*daisy.Workflow, error) {
	if tu.cleanupFn == nil {
		return tu.upgrader.cleanup()
	}
	return tu.cleanupFn()
}

func (tu *TestUpgrader) rollback() (*daisy.Workflow, error) {
	if tu.rollbackFn == nil {
		return tu.upgrader.rollback()
	}
	return tu.rollbackFn()
}

func TestUpgraderRunFailedOnInit(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.initFn = nil
	tu.Oauth = "bad-oauth"

	_, err := run(tu)
	assert.EqualError(t, err, "Failed to create GCE client: error creating HTTP API client: cannot read credentials file: open bad-oauth: no such file or directory")
}

func TestUpgraderRunFailedOnValidateParams(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.validateParamsFn = func() error {
		return fmt.Errorf("failed")
	}

	_, err := run(tu)
	assert.EqualError(t, err, "failed")
}

func TestUpgraderRunFailedOnPrintUpgradeGuide(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.printIntroHelpTextFn = func() error {
		return fmt.Errorf("failed")
	}

	_, err := run(tu)
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

	_, err := run(tu)
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

	_, err := run(tu)
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

	_, err := run(tu)
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

	_, err := run(tu)
	assert.EqualError(t, err, "failed")
}

func TestUpgraderRunSuccessWithoutReboot(t *testing.T) {
	tu := initTestUpgrader(t)

	_, err := run(tu)
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

	_, err := run(tu)
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

	_, err := run(tu)
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

	_, err := run(tu)
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
	_, err := run(tu)
	assert.EqualError(t, err, "failed1")
	assert.True(t, cleanupExecuted, "Cleanup not executed.")
}

func initTestUpgrader(t *testing.T) *TestUpgrader {
	u := initTest()
	tu := &TestUpgrader{upgrader: u}
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
