//  Copyright 2017 Google Inc. All Rights Reserved.
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

package workflow

import (
	"testing"
)

func TestWaitForInstancesStoppedRun(t *testing.T) {
	wf := testWorkflow()
	wf.createdInstances = []string{namer("instance1", testWf, testSuffix), namer("instance2", testWf, testSuffix), namer("instance3", testWf, testSuffix)}
	ws := &WaitForInstancesStopped{"instance1", "instance2", "instance3"}
	if err := ws.run(wf); err != nil {
		t.Fatalf("error running WaitForInstancesStopped.run(): %v", err)
	}
}

func TestWaitForInstancesStoppedValidate(t *testing.T) {
	instanceNames = nameSet{"instance1"}

	tests := []struct {
		desc      string
		step      WaitForInstancesStopped
		shouldErr bool
	}{
		{"normal case", WaitForInstancesStopped{"instance1"}, false},
		{"instance DNE error check", WaitForInstancesStopped{"instance1", "instance2"}, true},
	}

	for _, test := range tests {
		if err := test.step.validate(); (err != nil) != test.shouldErr {
			t.Errorf("fail: %s; step: %s; error result: %s", test.desc, test.step, err)
		}
	}
}
