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
	"github.com/stretchr/testify/assert"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func TestWaitForInstanceStopped(t *testing.T) {
	w := testWorkflow()

	svr, c, err := daisyCompute.NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/instances/%s?alt=json&prettyPrint=false", testProject, testZone, "foo") {
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

func TestWaitForSerialOutput(t *testing.T) {
	var cases = []struct {
		name                          string
		serialOutput                  []string
		expectedRemainingSerialOutput []string
		step                          InstanceSignal
		expectedErr                   string
		expectedMatchedValues         map[string]string
	}{
		{
			name: "on success, stop reading from channel",
			serialOutput: []string{
				"one", "two", "other", "success", "rest",
			},
			expectedRemainingSerialOutput: []string{"rest"},
			step: InstanceSignal{
				SerialOutput: &SerialOutput{
					SuccessMatch: "success",
				},
			},
		},
		{
			name: "on failure, stop reading from channel",
			serialOutput: []string{
				"one", "two", "other", "failure match", "rest",
			},
			expectedRemainingSerialOutput: []string{"rest"},
			step: InstanceSignal{
				SerialOutput: &SerialOutput{
					FailureMatch: []string{"failure match"},
				},
			},
			expectedErr: "failure match",
		},
		{
			name: "match failure before success",
			serialOutput: []string{
				"one", "two", "other", "Error: success", "rest",
			},
			expectedRemainingSerialOutput: []string{"rest"},
			step: InstanceSignal{
				SerialOutput: &SerialOutput{
					FailureMatch: []string{"Error"},
				},
			},
			expectedErr: "Error: success",
		},
		{
			name: "key value extraction",
			serialOutput: []string{
				"status: <serial-output key:'size' value:'10'>", "success", "rest",
			},
			expectedMatchedValues:         map[string]string{"size": "10"},
			expectedRemainingSerialOutput: []string{"rest"},
			step: InstanceSignal{
				SerialOutput: &SerialOutput{
					StatusMatch:  "status:",
					SuccessMatch: "success",
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			c := make(chan string, 1)

			go func() {
				for _, element := range tt.serialOutput {
					c <- element
				}
				close(c)
			}()

			ctx := context.Background()
			s := &Step{w: testWorkflow()}
			s.w.instances.m = map[string]*Resource{
				"i1": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, s.w.genName("i1"))},
			}
			tt.step.SerialOutput.outputChannel = c
			tt.step.Name = "i1"
			err := getStep(false, []*InstanceSignal{&tt.step}).run(ctx, s)

			var remaining []string

			for element := range c {
				fmt.Println(element)
				remaining = append(remaining, element)
			}

			if tt.expectedErr != "" {
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedRemainingSerialOutput, remaining)
			if tt.expectedMatchedValues != nil {
				for key, value := range tt.expectedMatchedValues {
					assert.Equal(t, value, s.w.GetSerialConsoleOutputValue(key))
				}
			}
		})
	}
}

func TestWaitForInstanceSignal(t *testing.T) {
	cases := []bool{true, false}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("waitForAny: %v", tt), func(t *testing.T) {

			c := make(chan string, 1)

			serialOutput := []string{
				"status", "success",
			}
			go func() {
				for _, element := range serialOutput {
					c <- element
				}
				close(c)
			}()

			ctx := context.Background()
			s := &Step{w: testWorkflow()}
			s.w.instances.m = map[string]*Resource{
				"i1": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, s.w.genName("i1"))},
				"i2": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, s.w.genName("i2"))},
			}
			s.w.ComputeClient.(*daisyCompute.TestClient).InstanceStatusFn = func(_, _, _ string) (string, error) {
				return "STOPPED", nil
			}
			s.w.ComputeClient.(*daisyCompute.TestClient).RetryFn = func(_ func(_ ...googleapi.CallOption) (*compute.Operation, error), _ ...googleapi.CallOption) (*compute.Operation, error) {
				return nil, nil
			}
			err := getStep(tt, []*InstanceSignal{{
				Name: "i1",
				SerialOutput: &SerialOutput{
					SuccessMatch:  "success",
					outputChannel: c,
				},
			},{
				Name: "i2",
				Stopped: true,
				interval: time.Nanosecond,
			},
			}).run(ctx, s)

			assert.NoError(t, err)
		})
	}
}

func TestWaitForStopped(t *testing.T) {
	testWaitForStopped(t, false)
}

func TestWaitForAnyStopped(t *testing.T) {
	testWaitForStopped(t, true)
}

func testWaitForStopped(t *testing.T, waitAny bool) {
	ctx := context.Background()
	w := testWorkflow()
	i6Counter := 0
	w.ComputeClient.(*daisyCompute.TestClient).InstanceStatusFn = func(_, _, n string) (string, error) {
		if n == w.genName("i5") {
			return "", errors.New("failed to get i5 status")
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
		{Name: "i2", Stopped: true, interval: time.Nanosecond},
		{Name: "i6", Stopped: true, interval: time.Nanosecond},
	})
	assert.NoError(t, ws.run(ctx, s))
	ws = getStep(waitAny, []*InstanceSignal{
		{Name: "i5", Stopped: true, interval: time.Nanosecond},
	})
	assert.EqualError(t, ws.run(ctx, s), "APIError: failed to get i5 status")
	ws = getStep(waitAny, []*InstanceSignal{
		{Name: "i7", Stopped: true, interval: time.Nanosecond},
	})
	assert.EqualError(t, ws.run(ctx, s), "unresolved instance \"i7\"")
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
