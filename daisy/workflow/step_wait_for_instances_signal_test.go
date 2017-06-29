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
	"context"
	"reflect"
	"testing"
	"time"
)

func TestWaitForInstancesSignalPopulate(t *testing.T) {
	got := &WaitForInstancesSignal{InstanceSignal{Name: "test"}}
	if err := got.populate(context.Background(), &Step{}); err != nil {
		t.Fatalf("error running populate: %v")
	}

	want := &WaitForInstancesSignal{InstanceSignal{Name: "test", Interval: "5s", interval: 5 * time.Second}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got != want:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestWaitForInstancesSignalRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	instances[w].m = map[string]*resource{
		"i1": {real: w.genName("i1"), link: "link"},
		"i2": {real: w.genName("i2"), link: "link"},
		"i3": {real: w.genName("i3"), link: "link"}}
	ws := &WaitForInstancesSignal{
		{Name: "i1", interval: 1 * time.Second, SerialOutput: &SerialOutput{Port: 1, SuccessMatch: "success"}},
		{Name: "i2", interval: 1 * time.Second, SerialOutput: &SerialOutput{Port: 2, SuccessMatch: "success", FailureMatch: "fail"}},
		{Name: "i3", Stopped: true},
	}
	if err := ws.run(ctx, s); err != nil {
		t.Errorf("error running WaitForInstancesSignal.run(): %v", err)
	}

	ws = &WaitForInstancesSignal{
		{Name: "i1", interval: 1 * time.Second, SerialOutput: &SerialOutput{Port: 1, FailureMatch: "fail", SuccessMatch: "success"}},
		{Name: "i2", interval: 1 * time.Second, SerialOutput: &SerialOutput{Port: 2, FailureMatch: "fail"}},
	}
	if err := ws.run(ctx, s); err == nil {
		t.Error("expected error")
	}
}

func TestWaitForInstancesSignalValidate(t *testing.T) {
	// Set up.
	w := &Workflow{}
	s := &Step{w: w}
	validatedInstances = nameSet{w: {"instance1"}}

	tests := []struct {
		desc      string
		step      WaitForInstancesSignal
		shouldErr bool
	}{
		{"normal case Stopped", WaitForInstancesSignal{{Name: "instance1", Stopped: true, interval: 1 * time.Second}}, false},
		{"normal SerialOutput SuccessMatch", WaitForInstancesSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1, SuccessMatch: "test"}, interval: 1 * time.Second}}, false},
		{"normal SerialOutput FailureMatch", WaitForInstancesSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1, FailureMatch: "fail"}, interval: 1 * time.Second}}, false},
		{"normal SerialOutput FailureMatch", WaitForInstancesSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1, SuccessMatch: "test", FailureMatch: "fail"}, interval: 1 * time.Second}}, false},
		{"SerialOutput no port", WaitForInstancesSignal{{Name: "instance1", SerialOutput: &SerialOutput{SuccessMatch: "test"}, interval: 1 * time.Second}}, true},
		{"SerialOutput no SuccessMatch or FailureMatch", WaitForInstancesSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1}, interval: 1 * time.Second}}, true},
		{"instance DNE error check", WaitForInstancesSignal{{Name: "instance1", Stopped: true, interval: 1 * time.Second}, {Name: "instance2", Stopped: true, interval: 1 * time.Second}}, true},
		{"no interval", WaitForInstancesSignal{{Name: "instance1", Stopped: true, Interval: "0s"}}, true},
	}

	for _, test := range tests {
		if err := test.step.validate(context.Background(), s); (err != nil) != test.shouldErr {
			t.Errorf("fail: %s; step: %+v; error result: %s", test.desc, test.step, err)
		}
	}
}
