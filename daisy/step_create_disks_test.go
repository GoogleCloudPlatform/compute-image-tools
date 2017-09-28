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

func TestCreateDisksPopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.ComputeClient = nil
	w.StorageClient = nil
	s, _ := w.NewStep("s")

	genFoo := w.genName("foo")
	defType := fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-standard", w.Project, w.Zone)
	tests := []struct {
		desc        string
		input, want *CreateDisk
		wantErr     bool
	}{
		{
			"defaults case",
			&CreateDisk{Disk: compute.Disk{Name: "foo"}},
			&CreateDisk{Disk: compute.Disk{Name: genFoo, Type: defType}, daisyName: "foo", Project: w.Project, Zone: w.Zone},
			false,
		},
		{
			"nondefaults case",
			&CreateDisk{Disk: compute.Disk{Name: "foo", Type: "pd-ssd"}, SizeGb: "10", Project: "pfoo", Zone: "zfoo"},
			&CreateDisk{Disk: compute.Disk{Name: genFoo, Type: "projects/pfoo/zones/zfoo/diskTypes/pd-ssd", SizeGb: 10}, daisyName: "foo", SizeGb: "10", Project: "pfoo", Zone: "zfoo"},
			false,
		},
		{
			"ExactName case",
			&CreateDisk{Disk: compute.Disk{Name: "foo"}, ExactName: true},
			&CreateDisk{Disk: compute.Disk{Name: "foo", Type: defType}, daisyName: "foo", Project: w.Project, Zone: w.Zone, ExactName: true, RealName: "foo"},
			false,
		},
		{
			"RealName case",
			&CreateDisk{Disk: compute.Disk{Name: "foo"}, RealName: "foo-foo"},
			&CreateDisk{Disk: compute.Disk{Name: "foo-foo", Type: defType}, daisyName: "foo", Project: w.Project, Zone: w.Zone, RealName: "foo-foo"},
			false,
		},
		{
			"extend Type URL case",
			&CreateDisk{Disk: compute.Disk{Name: "foo", Type: "zones/zfoo/diskTypes/pd-ssd"}, Project: "pfoo"},
			&CreateDisk{Disk: compute.Disk{Name: genFoo, Type: "projects/pfoo/zones/zfoo/diskTypes/pd-ssd"}, daisyName: "foo", Project: "pfoo", Zone: w.Zone},
			false,
		},
		{
			"extend SourceImage URL case",
			&CreateDisk{Disk: compute.Disk{Name: "foo"}},
			&CreateDisk{Disk: compute.Disk{Name: genFoo, Type: defType}, daisyName: "foo", Project: w.Project, Zone: w.Zone},
			false,
		},
		{
			"SourceImage daisy name case",
			&CreateDisk{Disk: compute.Disk{Name: "foo", SourceImage: "ifoo"}},
			&CreateDisk{Disk: compute.Disk{Name: genFoo, SourceImage: "ifoo", Type: defType}, daisyName: "foo", Project: w.Project, Zone: w.Zone},
			false,
		},
		{
			"bad SizeGb case",
			&CreateDisk{Disk: compute.Disk{Name: "foo"}, SizeGb: "ten"},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		cds := &CreateDisks{tt.input}
		err := cds.populate(ctx, s)
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
			t.Errorf("%s: populated CreateDisk does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestCreateDisksRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	images[w].m = map[string]*resource{"i1": {real: "i1", link: "i1link"}}

	e := errors.New("error")
	tests := []struct {
		desc      string
		d         compute.Disk
		wantD     compute.Disk
		clientErr error
		wantErr   error
	}{
		{"blank case", compute.Disk{}, compute.Disk{}, nil, nil},
		{"resolve source image case", compute.Disk{SourceImage: "i1"}, compute.Disk{SourceImage: "i1link"}, nil, nil},
		{"client error case", compute.Disk{}, compute.Disk{}, e, e},
	}
	for _, tt := range tests {
		var gotD compute.Disk
		fake := func(_, _ string, d *compute.Disk) error { gotD = *d; return tt.clientErr }
		w.ComputeClient = &daisyCompute.TestClient{CreateDiskFn: fake}
		cds := &CreateDisks{{Disk: tt.d}}
		if err := cds.run(ctx, s); err != tt.wantErr {
			t.Errorf("%s: unexpected error returned, got: %v, want: %v", tt.desc, err, tt.wantErr)
		}
		if diff := pretty.Compare(gotD, tt.wantD); diff != "" {
			t.Errorf("%s: client got incorrect disk, got: %v, want: %v", tt.desc, gotD, tt.wantD)
		}
	}
}

func TestCreateDisksValidate(t *testing.T) {
	ctx := context.Background()
	// Set up.
	w := testWorkflow()

	iCreator := &Step{name: "iCreator", w: w}
	w.Steps["iCreator"] = iCreator
	images[w].m = map[string]*resource{"i1": {creator: iCreator}}

	expType := func(p, z, t string) string { return fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", p, z, t) }
	n := "n"
	ty := expType(testProject, testZone, "pd-standard")
	tests := []struct {
		desc      string
		cd        *CreateDisk
		shouldErr bool
	}{
		{
			"source image case",
			&CreateDisk{daisyName: "d1", Disk: compute.Disk{Name: n, SourceImage: "i1", Type: ty}, Project: testProject, Zone: testZone},
			false,
		},
		{
			"source image url case",
			&CreateDisk{daisyName: "d2", Disk: compute.Disk{Name: n, SourceImage: fmt.Sprintf("projects/%s/global/images/%s", testProject, testImage), Type: ty}, Project: testProject, Zone: testZone},
			false,
		},
		{
			"source image dne case",
			&CreateDisk{daisyName: "d3", Disk: compute.Disk{Name: n, SourceImage: "dne", Type: ty}, Project: testProject, Zone: testZone},
			true,
		},
		{
			"blank disk case",
			&CreateDisk{daisyName: "d3", Disk: compute.Disk{Name: n, SizeGb: 1, Type: ty}, Project: testProject, Zone: testZone},
			false,
		},
		{
			"dupe disk case",
			&CreateDisk{daisyName: "d1", Disk: compute.Disk{Name: n, SizeGb: 1, Type: ty}, Project: testProject, Zone: testZone},
			true,
		},
		{
			"no size/source case",
			&CreateDisk{daisyName: "d4", Disk: compute.Disk{Name: n, Type: ty}, Project: testProject, Zone: testZone},
			true,
		},
		{
			"bad name case",
			&CreateDisk{daisyName: "d4", Disk: compute.Disk{Name: "n!", SizeGb: 1, Type: ty}, Project: testProject, Zone: testZone},
			true,
		},
		{
			"bad project case",
			&CreateDisk{daisyName: "d4", Disk: compute.Disk{Name: n, SizeGb: 1, Type: ty}, Project: "p!", Zone: testZone},
			true,
		},
		{
			"bad zone case",
			&CreateDisk{daisyName: "d4", Disk: compute.Disk{Name: n, SizeGb: 1, Type: ty}, Project: testProject, Zone: "z!"},
			true,
		},
		{
			"bad type case",
			&CreateDisk{daisyName: "d4", Disk: compute.Disk{Name: n, SizeGb: 1, Type: "t!"}, Project: testProject, Zone: testZone},
			true,
		},
	}
	for _, tt := range tests {
		w.Steps[tt.desc] = &Step{name: tt.desc, w: w, CreateDisks: &CreateDisks{tt.cd}}
		w.Dependencies[tt.desc] = []string{"iCreator"}
	}

	// These track the expected state of the disk references and the image reference.
	// Each good case adds a disk reference and adds a user to the image.
	wantDisks := map[string]*resource{}
	wantImages := &baseResourceRegistry{w: w, m: map[string]*resource{"i1": {creator: iCreator}}, urlRgx: imageURLRgx}

	// These are compare helper functions. These clear up infinite recursion issues.
	preCompare := func() {
		for _, d := range disks[w].m {
			d.creator.w = nil
		}
		iCreator.w = nil
	}
	postCompare := func() {
		for _, d := range disks[w].m {
			d.creator.w = w
		}
		iCreator.w = w
	}

	for _, tt := range tests {
		s := w.Steps[tt.desc]
		err := s.CreateDisks.validate(ctx, s)
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: did not return an error as expected", tt.desc)
			}
			wantLink := fmt.Sprintf("projects/%s/zones/%s/disks/%s", tt.cd.Project, tt.cd.Zone, tt.cd.Name)
			wantDisks[tt.cd.daisyName] = &resource{real: tt.cd.Name, link: wantLink, noCleanup: tt.cd.NoCleanup, deleted: false, creator: s, deleter: nil}
			if tt.cd.SourceImage != "" {
				wantImages.registerUsage(tt.cd.SourceImage, s)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}

		preCompare()
		if diff := pretty.Compare(disks[w].m, wantDisks); diff != "" {
			t.Errorf("%s: disk resources don't meet expectation: (-got +want)\n%s", tt.desc, diff)
		}
		if diff := pretty.Compare(images[w].m, wantImages.m); diff != "" {
			t.Errorf("%q: images don't meet expectation: (-got +want)\n%s", tt.desc, diff)
		}
		postCompare()
	}
}
