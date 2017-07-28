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
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

func TestWaitForInstanceStopped(t *testing.T) {
	w := testWorkflow()

	svr, c, err := compute.NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if err := waitForInstanceStopped(w, testProject, testZone, "foo", 1*time.Microsecond); err != nil {
		t.Fatalf("error running waitForInstanceStopped: %v", err)
	}
}

func TestWaitForInstancesSignalPopulate(t *testing.T) {
	got := &WaitForInstancesSignal{&InstanceSignal{Name: "test"}}
	if err := got.populate(context.Background(), &Step{}); err != nil {
		t.Fatalf("error running populate: %v", err)
	}

	want := &WaitForInstancesSignal{&InstanceSignal{Name: "test", Interval: "5s", interval: 5 * time.Second}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got != want:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestWaitForInstancesSignalRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()

	svr, c, err := compute.NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.String(), "serialPort?alt=json&port=1") {
			fmt.Fprintln(w, `{"Contents":"failsuccess","Start":"0"}`)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), "serialPort?alt=json&port=2") {
			fmt.Fprintln(w, `{"Contents":"successfail","Start":"0"}`)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), "serialPort?alt=json&port=3") {
			w.WriteHeader(http.StatusBadRequest)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), "serialPort?alt=json&port=4") {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "test error")
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), fmt.Sprintf("/%s/zones/%s/instances/i1", testProject, testZone)) {
			fmt.Fprintln(w, `{"Status":"TERMINATED","SelfLink":"link"}`)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), fmt.Sprintf("/%s/zones/%s/instances/i2", testProject, testZone)) {
			fmt.Fprintln(w, `{"Status":"RUNNING","SelfLink":"link"}`)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), fmt.Sprintf("/%s/zones/%s/instances/i3", testProject, testZone)) {
			fmt.Fprintln(w, `{"Status":"TERMINATED","SelfLink":"link"}`)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), fmt.Sprintf("/%s/zones/%s/instances/i4", testProject, testZone)) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "test error")
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "bad request: %+v", r)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svr.Close()
	w.ComputeClient = c

	s := &Step{w: w}
	instances[w].m = map[string]*resource{
		"i1": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i1"))},
		"i2": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i2"))},
		"i3": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i3"))},
		"i4": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, w.genName("i4"))},
	}

	// Normal run, no error.
	ws := &WaitForInstancesSignal{
		{Name: "i1", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{Port: 1, SuccessMatch: "success"}},
		{Name: "i2", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{Port: 2, SuccessMatch: "success", FailureMatch: "fail"}},
		{Name: "i3", interval: 1 * time.Microsecond, Stopped: true},
	}
	if err := ws.run(ctx, s); err != nil {
		t.Errorf("error running WaitForInstancesSignal.run(): %v", err)
	}

	// Failure match error.
	ws = &WaitForInstancesSignal{
		{Name: "i1", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{Port: 1, FailureMatch: "fail", SuccessMatch: "success"}},
		{Name: "i2", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{Port: 2, FailureMatch: "fail"}},
	}
	if err := ws.run(ctx, s); err == nil {
		t.Error("expected error")
	}

	// 400 from GetSerialPortOutput but instance is terminated so no error.
	ws = &WaitForInstancesSignal{
		{Name: "i1", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{Port: 3, SuccessMatch: "success"}},
	}
	if err := ws.run(ctx, s); err != nil {
		t.Errorf("error running WaitForInstancesSignal.run(): %v", err)
	}

	// 500 from GetSerialPortOutput but instance is not terminated error.
	ws = &WaitForInstancesSignal{
		{Name: "i2", interval: 1 * time.Microsecond, SerialOutput: &SerialOutput{Port: 4, SuccessMatch: "success"}},
	}
	want := "WaitForInstancesSignal: instance \"i2-test-wf-abcdef\": error getting serial port: googleapi: got HTTP response code 500 with body: test error\n, InstanceStatus: \"RUNNING\""
	if err := ws.run(ctx, s); err.Error() != want {
		t.Errorf("did not get expected error, got: %q, want: %q", err.Error(), want)
	}

	// 500 from WaitForInstanceStopped error.
	ws = &WaitForInstancesSignal{
		{Name: "i4", interval: 1 * time.Microsecond, Stopped: true},
	}
	want = "googleapi: got HTTP response code 400 with body: test error\n"
	if err := ws.run(ctx, s); err.Error() != want {
		t.Errorf("did not get expected error, got: %q, want: %q", err.Error(), want)
	}

	// Unresolved instance error.
	ws = &WaitForInstancesSignal{
		{Name: "i5", interval: 1 * time.Microsecond, Stopped: true},
	}
	want = "unresolved instance \"i5\""
	if err := ws.run(ctx, s); err.Error() != want {
		t.Errorf("did not get expected error, got: %q, want: %q", err.Error(), want)
	}

}

func TestWaitForInstancesSignalValidate(t *testing.T) {
	// Set up.
	w := testWorkflow()
	s, _ := w.NewStep("s")
	iCreator, _ := w.NewStep("iCreator")
	w.AddDependency("s", "iCreator")
	instances[w].registerCreation("instance1", &resource{}, iCreator)

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

	for _, tt := range tests {
		if err := tt.step.validate(context.Background(), s); (err != nil) != tt.shouldErr {
			t.Errorf("fail: %s; step: %+v; error result: %s", tt.desc, tt.step, err)
		}
	}
}
