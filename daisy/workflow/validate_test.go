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
	"errors"
	"io/ioutil"
	"log"
	"reflect"
	"sync"
	"testing"
)

func TestCheckName(t *testing.T) {
	// Test a good name passes.
	goodName := "this-is-a-g00d-name"
	if !checkName(goodName) {
		t.Errorf("%s incorrectly labeled as a bad name", goodName)
	}

	// Test bad names fail.
	badNames := []string{
		"This-is-a-bad-name",
		"this-IS-a-bad-name",
		"this_is_a_bad_name",
		"-this-is-a-bad-name",
		"2this-is-a-bad-name",
		"this-is-@-bad-name",
	}
	for _, badName := range badNames {
		if checkName(badName) {
			t.Errorf("%s incorrectly labeled as a good name", badName)
		}
	}
}

func TestDiskExists(t *testing.T) {
	w := &Workflow{}
	// Try a disk that has not been added.
	if diskValid(w, "DNE") {
		t.Error("reported non-existent disk name, DNE, as found")
	}

	// Try a disk that is added.
	validatedDisks.add(w, "test-exists-1")
	if !diskValid(w, "test-exists-1") {
		t.Error("reported disk test-exists-1 does not exist")
	}

	// Try a disk that has been added, but also deleted.
	validatedDisks.add(w, "test-exists-2")
	validatedDiskDeletions.add(w, "test-exists-2")
	if diskValid(w, "test-exists-2") {
		t.Error("reported disk test-exists-2 exists when it is to be deleted")
	}
}

func TestImageExists(t *testing.T) {
	w := &Workflow{}
	// Try an image that has not been added.
	if imageValid(w, "DNE") {
		t.Error("reported non-existent image name, DNE, as found")
	}

	// Try an image that is added.
	validatedImages.add(w, "test-exists-1")
	if !imageValid(w, "test-exists-1") {
		t.Error("reported image test-exists-1 does not exist")
	}
}

func TestInstanceExists(t *testing.T) {
	w := &Workflow{}
	// Try an instance that has not been added.
	if instanceValid(w, "DNE") {
		t.Error("reported non-existent instance name, DNE, as found")
	}

	// Try an instance that is added.
	validatedInstances.add(w, "test-exists-1")
	if !instanceValid(w, "test-exists-1") {
		t.Error("reported instance test-exists-1 does not exist")
	}

	// Try an instance that has been added, but also deleted.
	validatedInstances.add(w, "test-exists-2")
	validatedInstanceDeletions.add(w, "test-exists-2")
	if instanceValid(w, "test-exists-2") {
		t.Error("reported instance test-exists-2 exists when it is to be deleted")
	}
}

func TestNameSet(t *testing.T) {
	n := nameSet{}
	expected := nameSet{}
	w := &Workflow{}
	w2 := &Workflow{}
	w3 := &Workflow{}

	// Check init value.
	if !reflect.DeepEqual(n, expected) {
		t.Error("nameSet did not init as empty")
	}

	// Check add(). nameSet state persists across test cases.
	addTests := []struct {
		desc      string
		wf        *Workflow
		s         string
		shouldErr bool
		wantSet   nameSet
	}{
		{"normal add", w, "hello", false, nameSet{w: {"hello"}}},
		{"add ordering", w, "world", false, nameSet{w: {"hello", "world"}}},
		{"add dupe", w, "world", true, nameSet{w: {"hello", "world"}}},
		{"add bad name", w, "b@dname", true, nameSet{w: {"hello", "world"}}},
		{"add to second workflow", w2, "hello", false, nameSet{w: {"hello", "world"}, w2: {"hello"}}},
	}

	for _, test := range addTests {
		err := n.add(test.wf, test.s)
		if test.shouldErr && err == nil {
			t.Errorf("%q should have erred", test.desc)
		} else if !test.shouldErr && err != nil {
			t.Errorf("%q had incorrect error: %s", test.desc, err)
		}
		if !reflect.DeepEqual(n, test.wantSet) {
			t.Errorf("bad state after %q, want: %v; got: %v", test.desc, test.wantSet, n)
		}
	}

	// w has "hello" and "world", w2 has "hello" and (now) "bob".
	// Check has().
	n.add(w2, "bob")
	hasTests := []struct {
		desc string
		wf   *Workflow
		s    string
		want bool
	}{
		{"w has hello", w, "hello", true},
		{"w2 has hello", w2, "hello", true},
		{"w has world", w, "world", true},
		{"w2 does not have world", w2, "world", false},
		{"w does not have bob", w, "bob", false},
		{"w2 has bob", w2, "bob", true},
		{"w3 should have nothing", w3, "empty", false},
	}
	for _, test := range hasTests {
		if got := n.has(test.wf, test.s); got != test.want {
			t.Errorf("%q failed, bad n.has() result, want: %t; got: %t", test.desc, test.want, got)
		}
	}
}

func TestValidateVarsSubbed(t *testing.T) {
	w := testWorkflow()

	if err := w.validateVarsSubbed(); err != nil {
		t.Errorf("unexpected error on good workflow: %s", err)
	}

	w.Name = "workflow-${unsubbed}"
	if err := w.validateVarsSubbed(); err == nil {
		t.Error("bad workflow with unsubbed var should have returned an error, but didn't")
	}
}

func TestValidateWorkflow(t *testing.T) {
	s := &Step{Timeout: "10s", testType: &mockStep{}}

	// Normal, good validation.
	w := testWorkflow()
	w.Steps = map[string]*Step{"s0": s}
	if err := w.populate(); err != nil {
		t.Fatal(err)
	}
	if err := w.validate(); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	ctx := context.Background()
	logger := log.New(ioutil.Discard, "", 0)
	// Bad test cases.
	tests := []struct {
		desc string
		wf   *Workflow
	}{
		{"no name", &Workflow{Project: "p", Zone: "z", GCSPath: "b", OAuthPath: "o", Steps: map[string]*Step{"s": s}, Ctx: ctx, logger: logger}},
		{"no project", &Workflow{Name: "n", Zone: "z", GCSPath: "b", OAuthPath: "o", Steps: map[string]*Step{"s": s}, Ctx: ctx, logger: logger}},
		{"no zone", &Workflow{Name: "n", Project: "p", GCSPath: "b", OAuthPath: "o", Steps: map[string]*Step{"s": s}, Ctx: ctx, logger: logger}},
		{"no bucket", &Workflow{Name: "n", Project: "p", Zone: "z", OAuthPath: "o", Steps: map[string]*Step{"s": s}, Ctx: ctx, logger: logger}},
		{"no steps", &Workflow{Name: "n", Project: "p", Zone: "z", GCSPath: "b", OAuthPath: "o", Ctx: ctx, logger: logger}},
		{"no step name", &Workflow{Name: "n", Project: "p", Zone: "z", GCSPath: "b", OAuthPath: "o", Steps: map[string]*Step{"": s}, Ctx: ctx, logger: logger}},
		{"no step type", &Workflow{Name: "n", Project: "p", Zone: "z", GCSPath: "b", OAuthPath: "o", Steps: map[string]*Step{"s": {Timeout: defaultTimeout}}, Ctx: ctx, logger: logger}},
	}

	for _, tt := range tests {
		if err := tt.wf.validate(); err == nil {
			t.Errorf("validation should have failed on %s because of %q", tt.wf, tt.desc)
		}
	}
}

func TestValidateDAG(t *testing.T) {
	calls := make([]int, 5)
	errs := make([]error, 5)
	var rw sync.Mutex
	mockValidate := func(i int) func(w *Workflow) error {
		return func(w *Workflow) error {
			rw.Lock()
			defer rw.Unlock()
			calls[i] = calls[i] + 1
			return errs[i]
		}
	}
	reset := func() {
		rw.Lock()
		defer rw.Unlock()
		calls = make([]int, 5)
		errs = make([]error, 5)
	}

	// s0---->s1---->s3
	//   \         /
	//    --->s2---
	// s4
	w := testWorkflow()
	w.Steps = map[string]*Step{
		"s0": {testType: &mockStep{validateImpl: mockValidate(0)}},
		"s1": {testType: &mockStep{validateImpl: mockValidate(1)}},
		"s2": {testType: &mockStep{validateImpl: mockValidate(2)}},
		"s3": {testType: &mockStep{validateImpl: mockValidate(3)}},
		"s4": {testType: &mockStep{validateImpl: mockValidate(4)}},
	}
	w.Dependencies = map[string][]string{
		"s1": {"s0"},
		"s2": {"s0", "s0"}, // Check that dupes are removed.
		"s3": {"s1", "s2"},
	}

	// Normal case -- no issues.
	if err := w.populate(); err != nil {
		t.Fatal(err)
	}
	if err := w.validateDAG(); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	for i, callCount := range calls {
		if callCount != 1 {
			t.Errorf("step %d did not get validated", i)
		}
	}
	if !reflect.DeepEqual(w.Dependencies["s2"], []string{"s0"}) {
		t.Error("duplicate dependency not removed")
	}

	// Reset.
	reset()

	// Failed step 2.
	errs[2] = errors.New("fail")
	if err := w.validateDAG(); err == nil {
		t.Error("step 2 should have failed validation")
	}

	// Reset.
	reset()

	// Fail, missing dep.
	w.Dependencies["s0"] = []string{"dne"}
	if err := w.validateDAG(); err == nil {
		t.Error("validation should have failed due to missing dependency")
	}

	// Fail, cyclical deps.
	w.Dependencies["s0"] = []string{"s3"}
	if err := w.validateDAG(); err == nil {
		t.Error("validation should have failed due to dependency cycle")
	}
}

func TestDiskValid(t *testing.T) {
	w := testWorkflow()
	tests := []struct {
		disk  string
		valid bool
	}{
		{
			"projects/project/zones/zone/disks/disk1",
			true,
		},
		{
			"disk2",
			false,
		},
		{
			"disk3",
			false,
		},
		{
			"disk4",
			true,
		},
	}

	validatedDisks = nameSet{w: {"disk3", "disk4"}}
	if err := validatedDiskDeletions.add(w, "disk3"); err != nil {
		t.Errorf("error scheduling disk for deletion: %s", err)
	}
	for _, tt := range tests {
		if valid := diskValid(w, tt.disk); valid != tt.valid {
			t.Errorf("unexpected return from diskValid() for disk %q, want: %v, got: %v", tt.disk, tt.valid, valid)
		}
	}
}

func TestInstanceValid(t *testing.T) {
	w := testWorkflow()
	tests := []struct {
		instance string
		valid    bool
	}{
		{
			"projects/project/zones/zone/instances/instance1",
			true,
		},
		{
			"instance2",
			false,
		},
		{
			"instance3",
			false,
		},
		{
			"instance4",
			true,
		},
	}

	validatedInstances = nameSet{w: {"instance3", "instance4"}}
	if err := validatedInstanceDeletions.add(w, "instance3"); err != nil {
		t.Errorf("error scheduling instance for deletion: %s", err)
	}
	for _, tt := range tests {
		if valid := instanceValid(w, tt.instance); valid != tt.valid {
			t.Errorf("unexpected return from instanceValid() for instance %q, want: %v, got: %v", tt.instance, tt.valid, valid)
		}
	}
}

func TestImageValid(t *testing.T) {
	w := testWorkflow()
	tests := []struct {
		image string
		valid bool
	}{
		{
			"projects/project/global/images/image1",
			true,
		},
		{
			"image2",
			false,
		},
		{
			"image3",
			false,
		},
		{
			"image4",
			true,
		},
	}

	validatedImages = nameSet{w: {"image3", "image4"}}
	if err := validatedImageDeletions.add(w, "image3"); err != nil {
		t.Errorf("error scheduling image for deletion: %s", err)
	}
	for _, tt := range tests {
		if valid := imageValid(w, tt.image); valid != tt.valid {
			t.Errorf("unexpected return from imageValid() for image %q, want: %v, got: %v", tt.image, tt.valid, valid)
		}
	}
}
