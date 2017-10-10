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
	"testing"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/kylelemons/godebug/pretty"
	compute "google.golang.org/api/compute/v1"
)

func TestCreateImagePopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.ComputeClient = nil
	w.StorageClient = nil
	w.Sources = map[string]string{"d": "d"}
	s, _ := w.NewStep("s")

	genFoo := w.genName("foo")
	gcsAPIPath, _ := getGCSAPIPath("gs://bucket/d")
	tests := []struct {
		desc        string
		input, want *CreateImage
		wantErr     bool
	}{
		{
			"defaults case",
			&CreateImage{Image: compute.Image{Name: "foo"}},
			&CreateImage{Image: compute.Image{Name: genFoo}, daisyName: "foo", Project: w.Project},
			false,
		},
		{
			"nondefaults case",
			&CreateImage{Image: compute.Image{Name: "foo"}, Project: "pfoo"},
			&CreateImage{Image: compute.Image{Name: genFoo}, daisyName: "foo", Project: "pfoo"},
			false,
		},
		{
			"ExactName case",
			&CreateImage{Image: compute.Image{Name: "foo"}, ExactName: true},
			&CreateImage{Image: compute.Image{Name: "foo"}, daisyName: "foo", Project: w.Project, ExactName: true, RealName: "foo"},
			false,
		},
		{
			"RealName case",
			&CreateImage{Image: compute.Image{Name: "foo"}, RealName: "foo-foo"},
			&CreateImage{Image: compute.Image{Name: "foo-foo"}, daisyName: "foo", Project: w.Project, RealName: "foo-foo"},
			false,
		},
		{
			"SourceDisk case",
			&CreateImage{Image: compute.Image{Name: "foo", SourceDisk: "d"}},
			&CreateImage{Image: compute.Image{Name: genFoo, SourceDisk: "d"}, daisyName: "foo", Project: w.Project},
			false,
		},
		{
			"SourceDisk URL case",
			&CreateImage{Image: compute.Image{Name: "foo", SourceDisk: "projects/p/zones/z/disks/d"}},
			&CreateImage{Image: compute.Image{Name: genFoo, SourceDisk: "projects/p/zones/z/disks/d"}, daisyName: "foo", Project: w.Project},
			false,
		},
		{
			"extend SourceDisk URL case",
			&CreateImage{Image: compute.Image{Name: "foo", SourceDisk: "zones/z/disks/d"}, Project: "p"},
			&CreateImage{Image: compute.Image{Name: genFoo, SourceDisk: "projects/p/zones/z/disks/d"}, daisyName: "foo", Project: "p"},
			false,
		},
		{
			"RawDisk.Source from Sources case",
			&CreateImage{Image: compute.Image{Name: "foo", RawDisk: &compute.ImageRawDisk{Source: "d"}}},
			&CreateImage{Image: compute.Image{Name: genFoo, RawDisk: &compute.ImageRawDisk{Source: w.getSourceGCSAPIPath("d")}}, daisyName: "foo", Project: w.Project},
			false,
		},
		{
			"RawDisk.Source GCS URL case",
			&CreateImage{Image: compute.Image{Name: "foo", RawDisk: &compute.ImageRawDisk{Source: "gs://bucket/d"}}},
			&CreateImage{Image: compute.Image{Name: genFoo, RawDisk: &compute.ImageRawDisk{Source: gcsAPIPath}}, daisyName: "foo", Project: w.Project},
			false,
		},
		{
			"Bad RawDisk.Source case",
			&CreateImage{Image: compute.Image{Name: "foo", RawDisk: &compute.ImageRawDisk{Source: "blah"}}},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		cis := &CreateImages{tt.input}
		err := cis.populate(ctx, s)
		// Short circuit the description field -- difficult to test, and unimportant.
		if tt.want != nil {
			tt.want.Description = tt.input.Description
		}
		if tt.wantErr {
			if err == nil {
				t.Errorf("%s: should have returned an error but didn't", tt.desc)
			}
		} else if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		} else if diff := pretty.Compare(tt.input, tt.want); diff != "" {
			t.Errorf("%s: populated CreateImage does not match expectation: (-got,+want)\n%s", tt.desc, diff)
		}
	}
}

func TestCreateImagesRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	p := "project"
	disks[w].m = map[string]*resource{"d": {real: w.genName("d"), link: "dLink"}}
	w.Sources = map[string]string{"file": "gs://some/path"}

	testClient := &daisyCompute.TestClient{}
	w.ComputeClient = testClient
	tests := []struct {
		desc      string
		ci        *CreateImage
		clientErr error
		shouldErr bool
	}{
		{"source disk case", &CreateImage{Image: compute.Image{SourceDisk: "d"}, Project: p}, nil, false},
		{"raw image case", &CreateImage{Image: compute.Image{RawDisk: &compute.ImageRawDisk{Source: "gs://bucket/object"}}, Project: p}, nil, false},
		{"client err case", &CreateImage{Image: compute.Image{SourceDisk: "d"}, Project: p}, errors.New("error"), true},
	}

	type call struct {
		p string
		i *compute.Image
	}
	calls := []call{}
	for _, tt := range tests {
		testClient.CreateImageFn = func(p string, i *compute.Image) error {
			calls = append(calls, call{p, i})
			return tt.clientErr
		}
		cis := &CreateImages{tt.ci}
		if err := cis.run(ctx, s); err == nil && tt.shouldErr {
			t.Errorf("%s: should have returned an error, but didn't", tt.desc)
		} else if err != nil && !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
	wantCalls := []call{
		{p, &compute.Image{SourceDisk: "dLink"}},
		{p, &compute.Image{RawDisk: &compute.ImageRawDisk{Source: "gs://bucket/object"}}},
		{p, &compute.Image{SourceDisk: "dLink"}},
	}
	if diff := pretty.Compare(calls, wantCalls); diff != "" {
		t.Errorf("client was not called as expected:  (-got +want)\n%s", diff)
	}
}

func TestCreateImagesValidate(t *testing.T) {
	ctx := context.Background()

	w := testWorkflow()

	d1Creator := &Step{name: "d1Creator", w: w}
	w.Steps["d1Creator"] = d1Creator
	d2Creator := &Step{name: "d2Creator", w: w}
	w.Steps["d2Creator"] = d2Creator
	d2Deleter := &Step{name: "d2Deleter", w: w}
	w.Steps["d2Deleter"] = d2Deleter
	w.Dependencies["d2Deleter"] = []string{"d2Creator"}
	d3Creator := &Step{name: "d3Creator", w: w}
	w.Steps["d3Creator"] = d3Creator
	if err := disks[w].registerCreation("d1", &resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d1", testProject, testZone)}, d1Creator, false); err != nil {
		t.Fatal(err)
	}
	if err := disks[w].registerCreation("d2", &resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d2", testProject, testZone)}, d2Creator, false); err != nil {
		t.Fatal(err)
	}
	if err := disks[w].registerDeletion("d2", d2Deleter); err != nil {
		t.Fatal(err)
	}
	if err := disks[w].registerCreation("d3", &resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d3", testProject, testZone)}, d3Creator, false); err != nil {
		t.Fatal(err)
	}
	w.Sources = map[string]string{"source": "gs://some/file"}

	n := "n"
	tests := []struct {
		desc      string
		ci        *CreateImage
		shouldErr bool
	}{
		{"good disk case", &CreateImage{daisyName: "i1", Project: testProject, Image: compute.Image{Name: n, SourceDisk: "d1", Licenses: []string{fmt.Sprintf("projects/%s/global/licenses/%s", testProject, testLicense)}}}, false},
		{"good raw disk case", &CreateImage{daisyName: "i2", Project: testProject, Image: compute.Image{Name: n, RawDisk: &compute.ImageRawDisk{Source: "source"}}}, false},
		{"good raw disk case 2", &CreateImage{daisyName: "i3", Project: testProject, Image: compute.Image{Name: n, RawDisk: &compute.ImageRawDisk{Source: "gs://some/path"}}}, false},
		{"good disk url case ", &CreateImage{daisyName: "i5", Project: testProject, Image: compute.Image{Name: n, SourceDisk: fmt.Sprintf("projects/%s/zones/%s/disks/%s", testProject, testZone, testDisk)}}, false},
		{"bad license case", &CreateImage{daisyName: "i6", Project: testProject, Image: compute.Image{Name: n, SourceDisk: "d1", Licenses: []string{fmt.Sprintf("projects/%s/global/licenses/bad", testProject)}}}, true},
		{"bad name case", &CreateImage{Project: testProject, Image: compute.Image{Name: "bad!", SourceDisk: "d1"}}, true},
		{"bad project case", &CreateImage{Project: "bad!", Image: compute.Image{Name: "i6", SourceDisk: "d1"}}, true},
		{"bad dupe name case", &CreateImage{daisyName: "i1", Project: testProject, Image: compute.Image{Name: n, SourceDisk: "d1"}}, true},
		{"bad dupe image name case", &CreateImage{Project: testProject, Image: compute.Image{Name: testImage, SourceDisk: "d1"}}, true},
		{"bad missing dep on disk creator case", &CreateImage{Project: testProject, Image: compute.Image{Name: "i6", SourceDisk: "d3"}}, true},
		{"bad disk deleted case", &CreateImage{Project: testProject, Image: compute.Image{Name: "i6", SourceDisk: "d2"}}, true},
		{"bad using disk and raw disk case", &CreateImage{Project: testProject, Image: compute.Image{Name: "i6", SourceDisk: "d1", RawDisk: &compute.ImageRawDisk{Source: "gs://some/path"}}}, true},
	}

	for _, tt := range tests {
		s := &Step{name: tt.desc, w: w, CreateImages: &CreateImages{tt.ci}}
		w.Steps[tt.desc] = s
		w.Dependencies[tt.desc] = []string{"d1Creator", "d2Deleter"}
		if err := s.CreateImages.validate(ctx, s); err == nil {
			if tt.shouldErr {
				t.Errorf("%s: should have returned an error but didn't", tt.desc)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
		s.w = nil // prepare for pretty.Compare below
	}
}
