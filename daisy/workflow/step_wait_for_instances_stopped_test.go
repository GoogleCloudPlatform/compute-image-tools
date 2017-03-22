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
	wf.instanceRefs.m = map[string]*Resource{
		"i1": {"i1", wf.ephemeralName("i1"), "link", false},
		"i2": {"i2", wf.ephemeralName("i2"), "link", false},
		"i3": {"i3", wf.ephemeralName("i3"), "link", false}}
	ws := &WaitForInstancesStopped{"i1", "i2", "i3"}
	if err := ws.run(wf); err != nil {
		t.Fatalf("error running WaitForInstancesStopped.run(): %v", err)
	}
}

func TestWaitForInstancesStoppedValidate(t *testing.T) {
	// Set up.
	w := &Workflow{}
	validatedInstances = nameSet{w: {"instance1"}}

	tests := []struct {
		desc      string
		step      WaitForInstancesStopped
		shouldErr bool
	}{
		{"normal case", WaitForInstancesStopped{"instance1"}, false},
		{"instance DNE error check", WaitForInstancesStopped{"instance1", "instance2"}, true},
	}

	for _, test := range tests {
		if err := test.step.validate(w); (err != nil) != test.shouldErr {
			t.Errorf("fail: %s; step: %s; error result: %s", test.desc, test.step, err)
		}
	}
}
