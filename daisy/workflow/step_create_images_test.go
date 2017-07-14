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
	"fmt"
	"testing"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/kylelemons/godebug/pretty"
	compute "google.golang.org/api/compute/v1"
)

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
	p := "p"
	w.ComputeClient = nil
	w.StorageClient = nil
	d1Creator := &Step{name: "d1Creator", w: w}
	w.Steps["d1Creator"] = d1Creator
	d2Creator := &Step{name: "d2Creator", w: w}
	w.Steps["d2Creator"] = d2Creator
	d2Deleter := &Step{name: "d2Deleter", w: w}
	w.Steps["d2Deleter"] = d2Deleter
	w.Dependencies["d2Deleter"] = []string{"d2Creator"}
	d3Creator := &Step{name: "d3Creator", w: w}
	w.Steps["d3Creator"] = d3Creator
	disks[w].registerCreation("d1", &resource{}, d1Creator)
	disks[w].registerCreation("d2", &resource{}, d2Creator)
	disks[w].registerDeletion("d2", d2Deleter)
	disks[w].registerCreation("d3", &resource{}, d3Creator)
	w.Sources = map[string]string{"source": "gs://some/file"}

	tests := []struct {
		desc      string
		ci        *CreateImage
		shouldErr bool
	}{
		{"good disk case", &CreateImage{Project: p, Image: compute.Image{Name: "i1", SourceDisk: "d1"}}, false},
		{"good raw disk case", &CreateImage{Project: p, Image: compute.Image{Name: "i2", RawDisk: &compute.ImageRawDisk{Source: "source"}}}, false},
		{"good raw disk case 2", &CreateImage{Project: p, Image: compute.Image{Name: "i3", RawDisk: &compute.ImageRawDisk{Source: "gs://some/path"}}}, false},
		{"good disk url case", &CreateImage{Project: p, Image: compute.Image{Name: "i4", SourceDisk: "zones/z/disks/d"}}, false},
		{"good disk url case 2", &CreateImage{Project: p, Image: compute.Image{Name: "i5", SourceDisk: fmt.Sprintf("projects/%s/zones/z/disks/d", p)}}, false},
		{"bad name case", &CreateImage{Project: p, Image: compute.Image{Name: "bad!", SourceDisk: "d1"}}, true},
		{"bad dupe name case", &CreateImage{Project: p, Image: compute.Image{Name: "i1", SourceDisk: "d1"}}, true},
		{"bad missing dep on disk creator case", &CreateImage{Project: p, Image: compute.Image{Name: "i6", SourceDisk: "d3"}}, true},
		{"bad disk deleted case", &CreateImage{Project: p, Image: compute.Image{Name: "i6", SourceDisk: "d2"}}, true},
		{"bad using disk and raw disk case", &CreateImage{Project: p, Image: compute.Image{Name: "i6", SourceDisk: "d1", RawDisk: &compute.ImageRawDisk{Source: "gs://some/path"}}}, true},
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
