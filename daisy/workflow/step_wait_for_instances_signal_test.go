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

func TestWaitForInstancesSignalRun(t *testing.T) {
	wf := testWorkflow()
	wf.instanceRefs.m = map[string]*resource{
		"i1": {"i1", wf.genName("i1"), "link", false},
		"i2": {"i2", wf.genName("i2"), "link", false},
		"i3": {"i3", wf.genName("i3"), "link", false}}
	ws := &WaitForInstancesSignal{{Name: "i1", Stopped: true}, {Name: "i2", Stopped: true}, {Name: "i3", Stopped: true}}
	if err := ws.run(wf); err != nil {
		t.Fatalf("error running WaitForInstancesStopped.run(): %v", err)
	}
}

func TestWaitForInstancesSignalValidate(t *testing.T) {
	// Set up.
	w := &Workflow{}
	validatedInstances = nameSet{w: {"instance1"}}

	tests := []struct {
		desc      string
		step      WaitForInstancesSignal
		shouldErr bool
	}{
		{"normal case", WaitForInstancesSignal{{Name: "instance1", Stopped: true}}, false},
		{"instance DNE error check", WaitForInstancesSignal{{Name: "instance1", Stopped: true}, {Name: "instance2", Stopped: true}}, true},
	}

	for _, test := range tests {
		if err := test.step.validate(w); (err != nil) != test.shouldErr {
			t.Errorf("fail: %s; step: %s; error result: %s", test.desc, test.step, err)
		}
	}
}
