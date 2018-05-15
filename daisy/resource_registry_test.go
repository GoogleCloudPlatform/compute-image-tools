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
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestResourceRegistryCleanup(t *testing.T) {
	w := testWorkflow()

	d1 := &Resource{RealName: "d1", link: "link", NoCleanup: false}
	d2 := &Resource{RealName: "d2", link: "link", NoCleanup: true}
	im1 := &Resource{RealName: "im1", link: "link", NoCleanup: false}
	im2 := &Resource{RealName: "im2", link: "link", NoCleanup: true}
	in1 := &Resource{RealName: "in1", link: "link", NoCleanup: false}
	in2 := &Resource{RealName: "in2", link: "link", NoCleanup: true}
	w.disks.m = map[string]*Resource{"d1": d1, "d2": d2}
	w.images.m = map[string]*Resource{"im1": im1, "im2": im2}
	w.instances.m = map[string]*Resource{"in1": in1, "in2": in2}

	w.cleanup()

	for _, r := range []*Resource{d1, d2, im1, im2, in1, in2} {
		if r.NoCleanup && r.deleted {
			t.Errorf("cleanup deleted %q which was marked for NoCleanup", r.RealName)
		} else if !r.NoCleanup && !r.deleted {
			t.Errorf("cleanup didn't delete %q", r.RealName)
		}
	}
}

func TestResourceRegistryConcurrency(t *testing.T) {
	rr := baseResourceRegistry{w: testWorkflow()}
	rr.init()

	tests := []struct {
		desc string
		f    func()
	}{
		{"regCreate", func() { rr.regCreate("foo", &Resource{}, nil, false) }},
		{"regDelete", func() { rr.regDelete("foo", nil) }},
		{"regUse", func() { rr.regUse("foo", nil) }},
		{"get", func() { rr.get("foo") }},
		{"delete", func() { rr.get("foo") }},
	}

	for _, tt := range tests {
		order := []string{}
		releaseStr := "lock released"
		returnedStr := "func returned"
		want := []string{releaseStr, returnedStr}
		gunshot := sync.Mutex{}
		gunshot.Lock() // Wait for the goroutine to say we can go ahead.
		go func() {
			rr.mx.Lock()
			defer rr.mx.Unlock()
			gunshot.Unlock()
			time.Sleep(1 * time.Millisecond)
			order = append(order, releaseStr)
		}()
		gunshot.Lock() // Wait for the go ahead.
		tt.f()
		order = append(order, returnedStr)
		if !reflect.DeepEqual(order, want) {
			t.Errorf("%q case failed, unexpected concurrency order, want: %v; got: %v", tt.desc, want, order)
		}
	}
}

func TestResourceRegistryDelete(t *testing.T) {
	var deleteFnErr dErr
	r := &baseResourceRegistry{m: map[string]*Resource{}}
	r.deleteFn = func(r *Resource) dErr {
		return deleteFnErr
	}

	r.m["foo"] = &Resource{}
	r.m["baz"] = &Resource{}

	tests := []struct {
		desc, input string
		deleteFnErr dErr
		shouldErr   bool
	}{
		{"good case", "foo", nil, false},
		{"bad redelete case", "foo", nil, true},
		{"bad resource dne case", "bar", nil, true},
		{"bad deleteFn error case", "baz", errf("error"), true},
	}

	for _, tt := range tests {
		deleteFnErr = tt.deleteFnErr
		err := r.delete(tt.input)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}

	wantM := map[string]*Resource{
		"foo": {deleted: true, deleteMx: r.m["foo"].deleteMx},
		"baz": {deleted: false, deleteMx: r.m["baz"].deleteMx},
	}
	if diffRes := diff(r.m, wantM, 0); diffRes != "" {
		t.Errorf("resourceMap not modified as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestResourceRegistryStop(t *testing.T) {
	var stopFnErr dErr
	r := &baseResourceRegistry{m: map[string]*Resource{}}
	r.stopFn = func(r *Resource) dErr {
		return stopFnErr
	}

	r.m["foo"] = &Resource{}
	r.m["baz"] = &Resource{}

	tests := []struct {
		desc, input string
		stopFnErr dErr
		shouldErr   bool
	}{
		{"good case", "foo", nil, false},
		{"bad restop case", "foo", nil, true},
		{"bad resource dne case", "bar", nil, true},
		{"bad stopFn error case", "baz", errf("error"), true},
	}

	for _, tt := range tests {
		stopFnErr = tt.stopFnErr
		err := r.stop(tt.input)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}

	wantM := map[string]*Resource{
		"foo": {stopped: true},
		"baz": {stopped: false},
	}
	if diffRes := diff(r.m, wantM, 0); diffRes != "" {
		t.Errorf("resourceMap not modified as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestResourceRegistryGet(t *testing.T) {
	xRes := &Resource{}
	yRes := &Resource{}
	rr := baseResourceRegistry{m: map[string]*Resource{"x": xRes, "y": yRes}}

	tests := []struct {
		desc, input string
		wantR       *Resource
		wantOk      bool
	}{
		{"normal get", "y", yRes, true},
		{"get dne", "dne", nil, false},
	}

	for _, tt := range tests {
		if gotR, gotOk := rr.get(tt.input); !(gotOk == tt.wantOk && gotR == tt.wantR) {
			t.Errorf("%q case failed, want: (%v, %t); got: (%v, %t)", tt.desc, tt.wantR, tt.wantOk, gotR, gotOk)
		}
	}
}

func TestResourceRegistryRegCreate(t *testing.T) {
	rr := &baseResourceRegistry{w: testWorkflow()}
	rr.init()
	r := &Resource{link: "projects/foo/global/images/bar"}
	s := &Step{}

	// Normal create.
	if err := rr.regCreate("foo", r, s, false); err != nil {
		t.Fatalf("unexpected error registering creation of foo: %v", err)
	}
	if r.creator != s {
		t.Error("foo does not have the correct creator")
	}
	if diffRes := diff(rr.m, map[string]*Resource{"foo": r}, 0); diffRes != "" {
		t.Errorf("resource registry does not match expectation: (-got +want)\n%s", diffRes)
	}

	// Test duplication create.
	if err := rr.regCreate("foo", r, nil, false); err == nil {
		t.Error("should have returned an error, but didn't")
	}

	// Test overwrite create should not error on dupe.
	if err := rr.regCreate("foo", r, nil, true); err == nil {
		t.Fatalf("unexpected error registering creation of foo: %v", err)
	}
}

func TestResourceRegistryRegDelete(t *testing.T) {
	w := testWorkflow()
	creator := &Step{name: "creator", w: w}
	user := &Step{name: "user", w: w}
	deleter := &Step{name: "deleter", w: w}
	badDeleter := &Step{name: "badDeleter", w: w}
	badDeleter2 := &Step{name: "badDeleter2", w: w}
	dupeDeleter := &Step{name: "dupeDeleter", w: w}
	w.Steps = map[string]*Step{"creator": creator, "user": user, "deleter": deleter, "badDeleter": badDeleter, "badDeleter2": badDeleter2, "dupeDeleter": dupeDeleter}
	w.Dependencies = map[string][]string{
		"user":        {"creator"},
		"deleter":     {"user"},
		"dupeDeleter": {"user"},
		"badDeleter2": {"creator"},
	}
	r := &Resource{creator: creator, users: []*Step{user}}
	rr := &baseResourceRegistry{m: map[string]*Resource{"r": r}}

	tests := []struct {
		desc    string
		name    string
		step    *Step
		wantErr bool
	}{
		{"missing dependency on creator case", "r", badDeleter, true},
		{"missing reference case", "bar", badDeleter, true},
		{"missing dependency on user case", "r", badDeleter2, true},
		{"normal case", "r", deleter, false},
		{"dupe delete case", "r", dupeDeleter, true},
	}

	for _, tt := range tests {
		err := rr.regDelete(tt.name, tt.step)
		if tt.wantErr {
			if err == nil {
				t.Errorf("%s: did not return an error as expected", tt.desc)
			}
		} else if err != nil {
			t.Errorf("%s: unexepected error: %v", tt.desc, err)
		}
	}
}

func TestResourceRegistryRegURL(t *testing.T) {
	rr := &baseResourceRegistry{w: testWorkflow()}
	rr.init()

	defURL := fmt.Sprintf("projects/%s/zones/%s/disks/%s", testProject, testZone, testDisk)
	tests := []struct {
		desc, url string
		wantR     *Resource
		shouldErr bool
	}{
		{"normal case", defURL, &Resource{RealName: testDisk, link: defURL, NoCleanup: true}, false},
		{"dupe case", defURL, &Resource{RealName: testDisk, link: defURL, NoCleanup: true}, false},
		{"incomplete partial URL case", "zones/z/disks/bad", nil, true},
		{"already exists", fmt.Sprintf("projects/%s/global/images/my-image", testProject), nil, true},
	}

	for _, tt := range tests {
		r, err := rr.regURL(tt.url)
		if !tt.shouldErr {
			if err != nil {
				t.Errorf("%s: unexpected error: %v", tt.desc, err)
			} else if diffRes := diff(r, tt.wantR, 0); diffRes != "" {
				t.Errorf("%s: generated resource doesn't match expectation (-got +want)\n%s", tt.desc, diffRes)
			} else if rr.m[tt.url] != r {
				t.Errorf("%s: resource was not added to the resource map", tt.desc)
			}
		} else if err == nil {
			t.Errorf("%s: should have returned an error, but didn't", tt.desc)
		}
	}

	if diffRes := diff(rr.m, map[string]*Resource{defURL: {RealName: testDisk, link: defURL, NoCleanup: true}}, 0); diffRes != "" {
		t.Errorf("resource registry doesn't match expectation (-got +want)\n%s", diffRes)
	}
}

func TestResourceRegistryRegUse(t *testing.T) {
	w := testWorkflow()
	creator := &Step{name: "creator", w: w}
	deleter := &Step{name: "deleter", w: w}
	user := &Step{name: "user", w: w}
	badUser := &Step{name: "badUser", w: w}   // Doesn't depend on creator.
	badUser2 := &Step{name: "badUser2", w: w} // Tries to use a resource marked for deletion.
	badUser3 := &Step{name: "badUser3", w: w} // Fails on the resource type's use hook.
	w.Steps = map[string]*Step{"creator": creator, "user": user, "badUser": badUser, "badUser2": badUser2, "badUser3": badUser3, "deleter": deleter}
	w.Dependencies = map[string][]string{
		"user":     {"creator"},
		"badUser2": {"creator"},
		"badUser3": {"creator"},
		"deleter":  {"user", "badUser3"},
	}
	r1 := &Resource{creator: creator}
	r2 := &Resource{creator: creator, deleter: deleter}
	rr := &baseResourceRegistry{m: map[string]*Resource{"r1": r1, "r2": r2}}

	tests := []struct {
		desc    string
		name    string
		step    *Step
		wantErr bool
		wantRes *Resource
	}{
		{"normal case", "r1", user, false, r1},
		{"missing dependency on creator case", "r1", badUser, true, nil},
		{"use deleted case", "r2", badUser2, true, nil},
	}

	for _, tt := range tests {
		r, err := rr.regUse(tt.name, tt.step)
		if tt.wantErr {
			if err == nil {
				t.Errorf("%s: did not return an error as expected", tt.desc)
			}
		} else if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		} else if r != tt.wantRes {
			t.Errorf("%s: unexpected resource returned: want: %v, got: %v", tt.desc, tt.wantRes, r)
		}
	}

	// Clear s.Workflow -- unimportant and causes pretty.Compare stack overflow otherwise.
	ss := []*Step{creator, deleter, user, badUser, badUser2, badUser3}
	for _, s := range ss {
		s.w = nil
	}
	if diffRes := diff(r1.users, []*Step{user}, 0); diffRes != "" {
		t.Errorf("r1 users list does not match expectation: (-got +want)\n%s", diffRes)
	}
	if diffRes := diff(r2.users, []*Step(nil), 0); diffRes != "" {
		t.Errorf("r2 users list does not match expectation: (-got +want)\n%s", diffRes)
	}

}
