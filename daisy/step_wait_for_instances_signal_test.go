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
		if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s/zones/%s/instances/%s?alt=json&prettyPrint=false", testProject, testZone, "foo") {
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
	testWaitForSignalPopulate(t, false)
}

func TestWaitForAnyInstancesSignalPopulate(t *testing.T) {
	testWaitForSignalPopulate(t, true)
}

func testWaitForSignalPopulate(t *testing.T, waitAny bool) {
	got := getStep(waitAny, []*InstanceSignal{{Name: "test"}})
	if err := got.populate(context.Background(), &Step{}); err != nil {
		t.Fatalf("error running populate: %v", err)
	}

	want := getStep(waitAny, []*InstanceSignal{{Name: "test", Interval: "10s", interval: 10 * time.Second}})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got != want:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestWaitForInstancesSignalRun(t *testing.T) {
	testWaitForSignalRun(t, false)
}

func TestWaitForAnyInstancesSignalRun(t *testing.T) {
	testWaitForSignalRun(t, true)
}

func testWaitForSignalRun(t *testing.T, waitAny bool) {
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
	ws := getStep(waitAny, []*InstanceSignal{
		{Name: "i1", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{StatusMatch: "success", SuccessMatch: "success"}},
		{Name: "i1", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{SuccessMatch: "success", FailureMatch: []string{"fail"}}},
		{Name: "i1", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{SuccessMatch: "success", FailureMatch: []string{"fail", "fail2"}}},
		{Name: "i3", interval: 1 * time.Microsecond, Stopped: true},
	})
	if err := ws.run(ctx, s); err != nil {
		t.Errorf("error running stepImpl.run(): %v", err)
	}
	// Failure match error.
	ws = getStep(waitAny, []*InstanceSignal{
		{Name: "i2", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{FailureMatch: []string{"fail"}, SuccessMatch: "success"}},
		{Name: "i3", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{FailureMatch: []string{"fail"}}},
	})
	if err := ws.run(ctx, s); err == nil {
		t.Error("expected error")
	}
	// Failure matches error.
	ws = getStep(waitAny, []*InstanceSignal{
		{Name: "i2", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{FailureMatch: []string{"fail", "fail2"}, SuccessMatch: "success"}},
		{Name: "i3", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{FailureMatch: []string{"fail", "fail2"}}},
	})
	if err := ws.run(ctx, s); err == nil {
		t.Error("expected error")
	}
	// Error from GetSerialPortOutput but instance is running.
	ws = getStep(waitAny, []*InstanceSignal{
		{Name: "i4", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{SuccessMatch: "success"}},
	})
	if err := ws.run(ctx, s); err == nil {
		t.Error("expected error")
	}
	// Error from GetSerialPortOutput, error from InstanceStatus.
	ws = getStep(waitAny, []*InstanceSignal{
		{Name: "i5", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{SuccessMatch: "success"}},
	})
	if err := ws.run(ctx, s); err == nil {
		t.Error("expected error")
	}
	// Error from GetSerialPortOutput but only after instance starts (kind of "i4")
	ws = getStep(waitAny, []*InstanceSignal{
		{Name: "i6", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{SuccessMatch: "success"}},
	})
	if err := ws.run(ctx, s); err == nil {
		t.Errorf("expected error")
	}
	// Unresolved instance error.
	ws = getStep(waitAny, []*InstanceSignal{
		{Name: "i7", interval: 1 * time.Microsecond, Stopped: true},
	})
	want := "unresolved instance \"i7\""
	if err := ws.run(ctx, s); err.Error() != want {
		t.Errorf("did not get expected error, got: %q, want: %q", err.Error(), want)
	}
}

func TestWaitForInstancesSignalValidate(t *testing.T) {
	testWaitForSignalValidate(t, false)
}

func TestWaitForAnyInstancesSignalValidate(t *testing.T) {
	testWaitForSignalValidate(t, true)
}

func testWaitForSignalValidate(t *testing.T, waitAny bool) {
	// Set up.
	w := testWorkflow()
	s, _ := w.NewStep("s")
	iCreator, _ := w.NewStep("iCreator")
	iCreator.CreateInstances = &CreateInstances{Instances: []*Instance{&Instance{}}}
	w.AddDependency(s, iCreator)
	if err := w.instances.regCreate("instance1", &Resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d", testProject, testZone)}, false, iCreator); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc      string
		step      stepImpl
		shouldErr bool
	}{
		{"normal case Stopped", getStep(waitAny, []*InstanceSignal{{Name: "instance1", Stopped: true, interval: 1 * time.Second}}), false},
		{"normal SerialOutput SuccessMatch", getStep(waitAny, []*InstanceSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1, StatusMatch: "test", SuccessMatch: "test"}, interval: 1 * time.Second}}), false},
		{"normal SerialOutput FailureMatch", getStep(waitAny, []*InstanceSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1, FailureMatch: []string{"fail"}}, interval: 1 * time.Second}}), false},
		{"normal SerialOutput SuccessMatch FailureMatch", getStep(waitAny, []*InstanceSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1, SuccessMatch: "test", FailureMatch: []string{"fail"}}, interval: 1 * time.Second}}), false},
		{"normal SerialOutput SuccessMatch FailureMatch-es", getStep(waitAny, []*InstanceSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1, SuccessMatch: "test", FailureMatch: []string{"fail", "fail2"}}, interval: 1 * time.Second}}), false},
		{"SerialOutput no port", getStep(waitAny, []*InstanceSignal{{Name: "instance1", SerialOutput: &SerialOutput{SuccessMatch: "test"}, interval: 1 * time.Second}}), true},
		{"SerialOutput no SuccessMatch or FailureMatch or FailureMatches", getStep(waitAny, []*InstanceSignal{{Name: "instance1", SerialOutput: &SerialOutput{Port: 1}, interval: 1 * time.Second}}), true},
		{"instance DNE error check", getStep(waitAny, []*InstanceSignal{{Name: "instance1", Stopped: true, interval: 1 * time.Second}, {Name: "instance2", Stopped: true, interval: 1 * time.Second}}), true},
		{"no interval", getStep(waitAny, []*InstanceSignal{{Name: "instance1", Stopped: true, Interval: "0s"}}), true},
		{"no signal", getStep(waitAny, []*InstanceSignal{{Name: "instance1", interval: 1 * time.Second}}), true},
	}

	for _, tt := range tests {
		if err := tt.step.validate(context.Background(), s); (err != nil) != tt.shouldErr {
			t.Errorf("fail: %s; step: %+v; error result: %s", tt.desc, tt.step, err)
		}
	}
}

func TestWaitForInstancesSignalGetOutputValue(t *testing.T) {
	testWaitForSignalGetOutputValue(t, false)
}

func TestWaitForAnyInstancesSignalGetOutputValue(t *testing.T) {
	testWaitForSignalGetOutputValue(t, true)
}

func testWaitForSignalGetOutputValue(t *testing.T, waitAny bool) {
	ctx := context.Background()
	w := testWorkflow()
	w.ComputeClient.(*daisyCompute.TestClient).GetSerialPortOutputFn = func(_, _, n string, _, _ int64) (*compute.SerialPortOutput, error) {
		ret := &compute.SerialPortOutput{Next: 20}
		switch n {
		case w.genName("i1"):
			ret.Contents = "status: no output value"
		case w.genName("i2"):
			ret.Contents = "status: <serial-output key:'my-key' value:'my-value'>"
		}
		return ret, nil
	}

	s := &Step{w: w}
	w.instances.m = map[string]*Resource{
		"i1": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i1"))},
		"i2": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i2"))},
	}

	// No output value.
	ws := getStep(waitAny, []*InstanceSignal{
		{Name: "i1", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{StatusMatch: "status", SuccessMatch: "status"}},
	})
	if ws.run(ctx, s); w.serialControlOutputValues != nil {
		t.Errorf("error running stepImpl.run(): there shouldn't be any output value")
	}

	// There is an output value.
	ws = getStep(waitAny, []*InstanceSignal{
		{Name: "i2", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{StatusMatch: "status", SuccessMatch: "status"}},
	})
	if ws.run(ctx, s); w.serialControlOutputValues == nil || w.serialControlOutputValues["my-key"] != "my-value" {
		t.Errorf("error running stepImpl.run(): didn't get expected output value")
	}
}

func getStep(waitAny bool, iss []*InstanceSignal) stepImpl {
	if waitAny {
		si := WaitForAnyInstancesSignal{}
		for _, is := range iss {
			si = append(si, is)
		}
		return &si
	}

	si := WaitForInstancesSignal{}
	for _, is := range iss {
		si = append(si, is)
	}
	return &si
}
