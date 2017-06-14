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
	"github.com/kylelemons/godebug/pretty"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestResourceCleanup(t *testing.T) {
	w := testWorkflow()

	d1 := &resource{name: "d1", link: "link", noCleanup: false}
	d2 := &resource{name: "d2", link: "link", noCleanup: true}
	im1 := &resource{name: "im1", link: "link", noCleanup: false}
	im2 := &resource{name: "im2", link: "link", noCleanup: true}
	in1 := &resource{name: "in1", link: "link", noCleanup: false}
	in2 := &resource{name: "in2", link: "link", noCleanup: true}
	disks[w].m = map[string]*resource{"d1": d1, "d2": d2}
	images[w].m = map[string]*resource{"im1": im1, "im2": im2}
	instances[w].m = map[string]*resource{"in1": in1, "in2": in2}

	w.cleanup()

	for _, r := range []*resource{d1, d2, im1, im2, in1, in2} {
		if r.noCleanup && r.deleted {
			t.Errorf("cleanup deleted %q which was marked for noCleanup", r.name)
		} else if !r.noCleanup && !r.deleted {
			t.Errorf("cleanup didn't delete %q", r.name)
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
		{"normal add", "x", &resource{name: "x"}, map[string]*resource{"x": {name: "x"}}},
		{"dupe add", "x", &resource{name: "otherx"}, map[string]*resource{"x": {name: "otherx"}}},
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
