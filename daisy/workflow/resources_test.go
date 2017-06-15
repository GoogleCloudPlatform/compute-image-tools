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
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
)

func TestResourceCleanup(t *testing.T) {
	w := testWorkflow()

	d1 := &resource{real: "d1", link: "link", noCleanup: false}
	d2 := &resource{real: "d2", link: "link", noCleanup: true}
	im1 := &resource{real: "im1", link: "link", noCleanup: false}
	im2 := &resource{real: "im2", link: "link", noCleanup: true}
	in1 := &resource{real: "in1", link: "link", noCleanup: false}
	in2 := &resource{real: "in2", link: "link", noCleanup: true}
	disks[w].m = map[string]*resource{"d1": d1, "d2": d2}
	images[w].m = map[string]*resource{"im1": im1, "im2": im2}
	instances[w].m = map[string]*resource{"in1": in1, "in2": in2}

	w.cleanup()

	for _, r := range []*resource{d1, d2, im1, im2, in1, in2} {
		if r.noCleanup && r.deleted {
			t.Errorf("cleanup deleted %q which was marked for noCleanup", r.real)
		} else if !r.noCleanup && !r.deleted {
			t.Errorf("cleanup didn't delete %q", r.real)
		}
	}
}

func TestResourceMapAdd(t *testing.T) {
	rm := resourceMap{}

	tests := []struct {
		desc, ref string
		res       *resource
		want      map[string]*resource
	}{
		{"normal add", "x", &resource{real: "x"}, map[string]*resource{"x": {real: "x"}}},
		{"dupe add", "x", &resource{real: "otherx"}, map[string]*resource{"x": {real: "otherx"}}},
	}

	for _, tt := range tests {
		rm.add(tt.ref, tt.res)
		if diff := pretty.Compare(rm.m, tt.want); diff != "" {
			t.Errorf("%q case failed, refmap does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestResourceMapConcurrency(t *testing.T) {
	rm := resourceMap{}

	tests := []struct {
		desc string
		f    func()
	}{
		{"add", func() { rm.add("foo", nil) }},
		{"del", func() { rm.del("foo") }},
		{"get", func() { rm.get("foo") }},
	}

	for _, tt := range tests {
		order := []string{}
		releaseStr := "lock released"
		returnedStr := "func returned"
		want := []string{releaseStr, returnedStr}
		gunshot := sync.Mutex{}
		gunshot.Lock() // Wait for the goroutine to say we can go ahead.
		go func() {
			rm.mx.Lock()
			defer rm.mx.Unlock()
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

func TestResourceMapDel(t *testing.T) {
	xRes := &resource{}
	yRes := &resource{}
	rm := resourceMap{m: map[string]*resource{"x": xRes, "y": yRes}}

	tests := []struct {
		desc, input string
		want        map[string]*resource
	}{
		{"normal del", "y", map[string]*resource{"x": xRes}},
		{"del dne", "foo", map[string]*resource{"x": xRes}},
	}

	for _, tt := range tests {
		rm.del(tt.input)
		if !reflect.DeepEqual(rm.m, tt.want) {
			t.Errorf("%q case failed, refmap does not match expectation, want: %v; got: %v", tt.desc, tt.want, rm.m)
		}
	}
}

func TestResourceMapGet(t *testing.T) {
	xRes := &resource{}
	yRes := &resource{}
	rm := resourceMap{m: map[string]*resource{"x": xRes, "y": yRes}}

	tests := []struct {
		desc, input string
		wantR       *resource
		wantOk      bool
	}{
		{"normal get", "y", yRes, true},
		{"get dne", "dne", nil, false},
	}

	for _, tt := range tests {
		if gotR, gotOk := rm.get(tt.input); !(gotOk == tt.wantOk && gotR == tt.wantR) {
			t.Errorf("%q case failed, want: (%v, %t); got: (%v, %t)", tt.desc, tt.wantR, tt.wantOk, gotR, gotOk)
		}
	}
}

func TestResourceMapRegisterCreation(t *testing.T) {
	rm := &resourceMap{}
	r := &resource{}
	s := &Step{}

	// Normal create.
	if err := rm.registerCreation("foo", r, s); err != nil {
		t.Error("unexpected error registering creation of foo")
	}
	if r.creator != s {
		t.Error("foo does not have the correct creator")
	}
	if diff := pretty.Compare(rm.m, map[string]*resource{"foo": r}); diff != "" {
		t.Errorf("resource map does not match expectation: (-got +want)\n%s", diff)
	}

	// Test duplication create.
	if err := rm.registerCreation("foo", nil, nil); err == nil {
		t.Error("should have returned an error, but didn't")
	}
}

func TestResourceMapRegisterDeletion(t *testing.T) {
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
	r := &resource{creator: creator, users: []*Step{user}}
	rm := &resourceMap{m: map[string]*resource{"r": r}}

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
		err := rm.registerDeletion(tt.name, tt.step)
		if tt.wantErr {
			if err == nil {
				t.Errorf("%s: did not return an error as expected", tt.desc)
			}
		} else if err != nil {
			t.Errorf("%s: unexepected error: %v", tt.desc, err)
		}
	}
}

func TestResourceMapRegisterUsage(t *testing.T) {
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
	r1 := &resource{creator: creator}
	r2 := &resource{creator: creator, deleter: deleter}
	hook := func(name string, s *Step) error {
		if s.name == "badUser3" {
			return errors.New("fail")
		}
		return nil
	}
	rm := &resourceMap{m: map[string]*resource{"r1": r1, "r2": r2}, usageRegistrationHook: hook}

	tests := []struct {
		desc    string
		name    string
		step    *Step
		wantErr bool
	}{
		{"normal case", "r1", user, false},
		{"missing dependency on creator case", "r1", badUser, true},
		{"use deleted case", "r2", badUser2, true},
		{"hook error case", "r1", badUser3, true},
	}

	for _, tt := range tests {
		err := rm.registerUsage(tt.name, tt.step)
		if tt.wantErr {
			if err == nil {
				t.Errorf("%s: did not return an error as expected", tt.desc)
			}
		} else if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}

	// Clear s.w -- unimportant and causes pretty.Compare stack overflow otherwise.
	ss := []*Step{creator, deleter, user, badUser, badUser2, badUser3}
	for _, s := range ss {
		s.w = nil
	}
	if diff := pretty.Compare(r1.users, []*Step{user}); diff != "" {
		t.Errorf("r1 users list does not match expectation: (-got +want)\n%s", diff)
	}
	if diff := pretty.Compare(r2.users, []*Step{}); diff != "" {
		t.Errorf("r2 users list does not match expectation: (-got +want)\n%s", diff)
	}

}
