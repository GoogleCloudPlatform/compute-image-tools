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
	"testing"
)

func TestExtendPartialURL(t *testing.T) {
	want := "projects/foo/zones/bar/disks/baz"
	if s := extendPartialURL("zones/bar/disks/baz", "foo"); s != want {
		t.Errorf("got: %q, want: %q", s, want)
	}

	if s := extendPartialURL("projects/foo/zones/bar/disks/baz", "gaz"); s != want {
		t.Errorf("got: %q, want %q", s, want)
	}
}

func TestResourcePopulate(t *testing.T) {
	w := testWorkflow()
	s, _ := w.NewStep("foo")

	name := "name"
	genName := w.genName(name)
	tests := []struct {
		desc                     string
		r, wantR                 Resource
		zone, wantName, wantZone string
		wantErr                  bool
	}{
		{"defaults case", Resource{}, Resource{daisyName: name, RealName: genName, Project: w.Project}, "", genName, w.Zone, false},
		{"nondefaults case", Resource{Project: "pfoo"}, Resource{daisyName: name, RealName: genName, Project: "pfoo"}, "zfoo", genName, "zfoo", false},
		{"ExactName case", Resource{ExactName: true}, Resource{daisyName: name, RealName: name, Project: w.Project, ExactName: true}, "", name, w.Zone, false},
		{"RealName case", Resource{RealName: "foo"}, Resource{daisyName: name, RealName: "foo", Project: w.Project}, "", "foo", w.Zone, false},
		{"RealName and ExactName error case", Resource{RealName: "foo", ExactName: true}, Resource{}, "", "", "", true},
	}

	for _, tt := range tests {
		gotName, gotZone, err := tt.r.populateWithZone(context.Background(), s, name, tt.zone)
		if tt.wantErr && err == nil {
			t.Errorf("%s: should have returned an error but didn't", tt.desc)
		} else if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		} else if err == nil {
			if diffRes := diff(tt.r, tt.wantR, 0); diffRes != "" {
				t.Errorf("%s: populated Resource does not match expectation: (-got +want)\n%s", tt.desc, diffRes)
			}
			if gotName != tt.wantName {
				t.Errorf("%s: name population wrong; got: %q, want: %q", tt.desc, gotName, tt.wantName)
			}
			if gotZone != tt.wantZone {
				t.Errorf("%s: zone population wrong; got: %q, want: %q", tt.desc, gotZone, tt.wantZone)
			}
		}
	}
}

func TestResourceNameHelper(t *testing.T) {
	w := testWorkflow()
	want := w.genName("foo")
	got := resourceNameHelper("foo", w, false)
	if got != want {
		t.Errorf("%q != %q", got, want)
	}

	want = "foo"
	got = resourceNameHelper("foo", w, true)
	if got != want {
		t.Errorf("%q != %q", got, want)
	}
}

func TestResourceValidate(t *testing.T) {
	w := testWorkflow()
	s, _ := w.NewStep("foo")

	tests := []struct {
		desc    string
		r       Resource
		wantErr bool
	}{
		{"good case", Resource{RealName: "good", Project: testProject}, false},
		{"bad name case", Resource{RealName: "bad!", Project: testProject}, true},
		{"bad project case", Resource{RealName: "good", Project: "bad!"}, true},
		{"project DNE case", Resource{RealName: "good", Project: DNE}, true},
	}

	for _, tt := range tests {
		err := tt.r.validate(context.Background(), s, "prefix")
		if tt.wantErr && err == nil {
			t.Errorf("%s: should have returned an error but didn't", tt.desc)
		} else if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestResourceValidateWithZone(t *testing.T) {
	w := testWorkflow()
	s, _ := w.NewStep("foo")

	tests := []struct {
		desc, zone string
		wantErr    bool
	}{
		{"good case", testZone, false},
		{"bad zone case", "bad!", true},
		{"zone DNE case", DNE, true},
	}

	for _, tt := range tests {
		r := Resource{RealName: "goodname", Project: w.Project}
		err := r.validateWithZone(context.Background(), s, tt.zone, "prefix")
		if tt.wantErr && err == nil {
			t.Errorf("%s: should have returned an error but didn't", tt.desc)
		} else if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}
