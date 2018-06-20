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

package daisy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func TestWaitForInstanceStopped(t *testing.T) {
	w := testWorkflow()

	svr, c, err := daisyCompute.NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/instances/%s?alt=json", testProject, testZone, "foo") {
			fmt.Fprint(w, `{"Status":"TERMINATED"}`)
		} else {
			w.WriteHeader(500)
			fmt.Fprintln(w, "URL and Method not recognized:", r.Method, r.URL)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svr.Close()

	w.ComputeClient = c
	s := &Step{name: "foo", w: w}
	if err := waitForInstanceStopped(s, testProject, testZone, "foo", 1*time.Microsecond); err != nil {
		t.Fatalf("error running waitForInstanceStopped: %v", err)
	}
}

func TestWaitForInstancesSignalPopulate(t *testing.T) {
	got := &WaitForInstancesSignal{&InstanceSignal{Name: "test"}}
	if err := got.populate(context.Background(), &Step{}); err != nil {
		t.Fatalf("error running populate: %v", err)
	}

	want := &WaitForInstancesSignal{&InstanceSignal{Name: "test", Interval: "10s", interval: 10 * time.Second}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got != want:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestWaitForInstancesSignalRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.ComputeClient.(*daisyCompute.TestClient).GetSerialPortOutputFn = func(_, _, n string, _, _ int64) (*compute.SerialPortOutput, error) {
		ret := &compute.SerialPortOutput{Next: 20}
		switch n {
		case w.genName("i1"):
			ret.Contents = "success"
		case w.genName("i2"):
			ret.Contents = "success fail"
		case w.genName("i3"):
			ret.Contents = "fail success"
		case w.genName("i4"):
			return nil, errors.New("fail")
		case w.genName("i5"):
			return nil, errors.New("fail")
		case w.genName("i6"):
			return nil, errors.New("fail")
		}
		return ret, nil
	}

	i6Counter := 0
	w.ComputeClient.(*daisyCompute.TestClient).InstanceStatusFn = func(_, _, n string) (string, error) {
		if n == w.genName("i5") {
			return "", errors.New("fail")
		}
		if n == w.genName("i4") {
			return "RUNNING", nil
		}
		if n == w.genName("i6") {
			i6Counter++
			switch i6Counter {
			case 1:
				return "STOPPING", nil
			case 2:
				return "STOPPED", nil
			case 3:
				return "TERMINATED", nil
			default:
				return "RUNNING", nil
			}
		}
		return "STOPPED", nil
	}
	w.ComputeClient.(*daisyCompute.TestClient).RetryFn = func(_ func(_ ...googleapi.CallOption) (*compute.Operation, error), _ ...googleapi.CallOption) (*compute.Operation, error) {
		return nil, nil
	}

	s := &Step{w: w}
	w.instances.m = map[string]*Resource{
		"i1": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i1"))},
		"i2": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i2"))},
		"i3": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i3"))},
		"i4": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i4"))},
		"i5": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i5"))},
		"i6": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i6"))},
	}

	// Normal run, no error.
	ws := &WaitForInstancesSignal{
		{Name: "i1", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{StatusMatch: "success", SuccessMatch: "success"}},
		{Name: "i1", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{SuccessMatch: "success", FailureMatch: "fail"}},
		{Name: "i3", interval: 1 * time.Microsecond, Stopped: true},
	}
	if err := ws.run(ctx, s); err != nil {
		t.Errorf("error running WaitForInstancesSignal.run(): %v", err)
	}

	// Failure match error.
	ws = &WaitForInstancesSignal{
		{Name: "i2", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{FailureMatch: "fail", SuccessMatch: "success"}},
		{Name: "i3", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{FailureMatch: "fail"}},
	}
	if err := ws.run(ctx, s); err == nil {
		t.Error("expected error")
	}

	// Error from GetSerialPortOutput but instance is running.
	ws = &WaitForInstancesSignal{
		{Name: "i4", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{SuccessMatch: "success"}},
	}
	if err := ws.run(ctx, s); err == nil {
		t.Error("expected error")
	}

	// Error from GetSerialPortOutput, error from InstanceStatus.
	ws = &WaitForInstancesSignal{
		{Name: "i5", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{SuccessMatch: "success"}},
	}
	if err := ws.run(ctx, s); err == nil {
		t.Error("expected error")
	}

	// Error from GetSerialPortOutput but only after instance starts (kind of "i4")
	ws = &WaitForInstancesSignal{
		{Name: "i6", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{SuccessMatch: "success"}},
	}
	if err := ws.run(ctx, s); err == nil {
		t.Errorf("expected error")
	}

	// Unresolved instance error.
	ws = &WaitForInstancesSignal{
		{Name: "i7", interval: 1 * time.Microsecond, Stopped: true},
	}
	want := "unresolved instance \"i7\""
	if err := ws.run(ctx, s); err.Error() != want {
		t.Errorf("did not get expected error, got: %q, want: %q", err.Error(), want)
	}

}

func TestWaitForInstancesSignalValidate(t *testing.T) {
	// Set up.
	w := testWorkflow()
	s, _ := w.NewStep("s")
	iCreator, _ := w.NewStep("iCreator")
	iCreator.CreateInstances = &CreateInstances{&Instance{}}
	w.AddDependency(s, iCreator)
	if err := w.instances.regCreate("instance1", &Resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d", testProject, testZone)}, iCreator); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc      string
		step      WaitForInstancesSignal
		shouldErr bool
	}{
		{"normal case Stopped", WaitForInstancesSignal{{Name: "instance1", Stopped: true, interval: 1 * time.Second}}, false},
		{"normal SerialOutput SuccessMatch", WaitForInstancesSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1, StatusMatch: "test", SuccessMatch: "test"}, interval: 1 * time.Second}}, false},
		{"normal SerialOutput FailureMatch", WaitForInstancesSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1, FailureMatch: "fail"}, interval: 1 * time.Second}}, false},
		{"normal SerialOutput FailureMatch", WaitForInstancesSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1, SuccessMatch: "test", FailureMatch: "fail"}, interval: 1 * time.Second}}, false},
		{"SerialOutput no port", WaitForInstancesSignal{{Name: "instance1", SerialOutput: &SerialOutput{SuccessMatch: "test"}, interval: 1 * time.Second}}, true},
		{"SerialOutput no SuccessMatch or FailureMatch", WaitForInstancesSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1}, interval: 1 * time.Second}}, true},
		{"instance DNE error check", WaitForInstancesSignal{{Name: "instance1", Stopped: true, interval: 1 * time.Second}, {Name: "instance2", Stopped: true, interval: 1 * time.Second}}, true},
		{"no interval", WaitForInstancesSignal{{Name: "instance1", Stopped: true, Interval: "0s"}}, true},
		{"no signal", WaitForInstancesSignal{{Name: "instance1", interval: 1 * time.Second}}, true},
	}

	for _, tt := range tests {
		if err := tt.step.validate(context.Background(), s); (err != nil) != tt.shouldErr {
			t.Errorf("fail: %s; step: %+v; error result: %s", tt.desc, tt.step, err)
		}
	}
}
