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
	// Try a disk that has not been added.
	if diskExists("DNE") {
		t.Errorf("reported non-existent disk name, DNE, as found")
	}

	// Try a disk that is added.
	diskNames.add("test-exists-1")
	if !diskExists("test-exists-1") {
		t.Errorf("reported disk test-exists-1 does not exist")
	}

	// Try a disk that has been added, but also deleted.
	diskNames.add("test-exists-2")
	diskNamesToDelete.add("test-exists-2")
	if diskExists("test-exists-2") {
		t.Errorf("reported disk test-exists-2 exists when it is to be deleted")
	}
}

func TestImageExists(t *testing.T) {
	// Try an image that has not been added.
	if imageExists("DNE") {
		t.Errorf("reported non-existent image name, DNE, as found")
	}

	// Try an image that is added.
	imageNames.add("test-exists-1")
	if !imageExists("test-exists-1") {
		t.Errorf("reported image test-exists-1 does not exist")
	}
}

func TestInstanceExists(t *testing.T) {
	// Try an instance that has not been added.
	if instanceExists("DNE") {
		t.Errorf("reported non-existent instance name, DNE, as found")
	}

	// Try an instance that is added.
	instanceNames.add("test-exists-1")
	if !instanceExists("test-exists-1") {
		t.Errorf("reported instance test-exists-1 does not exist")
	}

	// Try an instance that has been added, but also deleted.
	instanceNames.add("test-exists-2")
	instanceNamesToDelete.add("test-exists-2")
	if instanceExists("test-exists-2") {
		t.Errorf("reported instance test-exists-2 exists when it is to be deleted")
	}
}

func TestNameSet(t *testing.T) {
	var s nameSet
	var expected []string

	// Check init value.
	if !reflect.DeepEqual([]string(s), expected) {
		t.Error("nameSet did not init as empty string array")
	}

	// Simple add check.
	if s.add("hello") != nil {
		t.Error("nameSet.add returned an incorrect error")
	}
	expected = append(expected, "hello")
	if !reflect.DeepEqual([]string(s), expected) {
		t.Errorf("nameSet.add didn't add %s != %s", s, expected)
	}

	// Check adds are ordered.
	if s.add("world") != nil {
		t.Error("nameSet.add returned an incorrect error")
	}
	expected = append(expected, "world")
	if !reflect.DeepEqual([]string(s), expected) {
		t.Errorf("nameSet.add didn't add %s != %s", s, expected)
	}

	// Check that dupe add of "world" fails.
	if s.add("world") == nil {
		t.Error("nameSet.add didn't err when adding dupe name")
	}
	if !reflect.DeepEqual([]string(s), expected) {
		t.Errorf("nameSet.add shouldn't have modified set: %s != %s", s, expected)
	}

	// Check adding a bad name.
	if s.add("b@dname") == nil {
		t.Error("nameSet.add didn't err when adding bad name")
	}
	if !reflect.DeepEqual([]string(s), expected) {
		t.Errorf("nameSet.add shouldn't have modified set: %s != %s", s, expected)
	}

	// Check has on a name that DNE.
	if s.has("DNE") {
		t.Error("nameSet.has reporting a non-existent name exists")
	}

	// Check has on a name that exists.
	if !s.has("world") {
		t.Error("nameSet.has reporting a existent name doesn't exist")
	}
}

func TestValidateWorkflow(t *testing.T) {
	s := &Step{Timeout: "my-timeout", testType: &mockStep{}}

	// Normal, good validation.
	w := testWorkflow()
	w.Steps = map[string]*Step{"s0": s}
	if err := w.validate(); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	ctx := context.Background()
	// Bad test cases.
	tests := []struct {
		desc string
		wf   *Workflow
	}{
		{"no name", &Workflow{Project: "p", Zone: "z", Bucket: "b", OAuthPath: "o", Steps: map[string]*Step{"s": s}, Ctx: ctx}},
		{"no project", &Workflow{Name: "n", Zone: "z", Bucket: "b", OAuthPath: "o", Steps: map[string]*Step{"s": s}, Ctx: ctx}},
		{"no zone", &Workflow{Name: "n", Project: "p", Bucket: "b", OAuthPath: "o", Steps: map[string]*Step{"s": s}, Ctx: ctx}},
		{"no bucket", &Workflow{Name: "n", Project: "p", Zone: "z", OAuthPath: "o", Steps: map[string]*Step{"s": s}, Ctx: ctx}},
		{"no steps", &Workflow{Name: "n", Project: "p", Zone: "z", Bucket: "b", OAuthPath: "o", Ctx: ctx}},
		{"no step name", &Workflow{Name: "n", Project: "p", Zone: "z", Bucket: "b", OAuthPath: "o", Steps: map[string]*Step{"": s}, Ctx: ctx}},
		{"no step timeout", &Workflow{Name: "n", Project: "p", Zone: "z", Bucket: "b", OAuthPath: "o", Steps: map[string]*Step{"s": {testType: &mockStep{}}}, Ctx: ctx}},
		{"no step type", &Workflow{Name: "n", Project: "p", Zone: "z", Bucket: "b", OAuthPath: "o", Steps: map[string]*Step{"s": {Timeout: defaultTimeout}}, Ctx: ctx}},
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
	mockValidate := func(i int) func() error {
		return func() error {
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
