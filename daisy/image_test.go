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
	"fmt"
	"reflect"
	"strconv"
	"testing"

	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  guestOsFeatures
	}{
		{"[]", nil},
		{`["foo","bar"]`, guestOsFeatures{"foo", "bar"}},
		{`[{"Type":"foo"},{"Type":"bar"}]`, guestOsFeatures{"foo", "bar"}},
	}

	for _, tt := range tests {
		var got guestOsFeatures
		if err := got.UnmarshalJSON([]byte(tt.input)); err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("want: %q, got: %q", tt.want, got)
		}
	}
}

func TestImagePopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.Sources = map[string]string{"d": "d"}
	s, _ := w.NewStep("s")

	gcsAPIPath, _ := getGCSAPIPath("gs://bucket/d")
	tests := []struct {
		desc        string
		input, want *Image
		wantErr     bool
	}{
		{
			"SourceDisk case",
			&Image{Image: compute.Image{SourceDisk: "d"}},
			&Image{Image: compute.Image{SourceDisk: "d"}},
			false,
		},
		{
			"SourceDisk URL case",
			&Image{Image: compute.Image{SourceDisk: "projects/p/zones/z/disks/d"}},
			&Image{Image: compute.Image{SourceDisk: "projects/p/zones/z/disks/d"}},
			false,
		},
		{
			"extend SourceDisk URL case",
			&Image{ImageBase: ImageBase{Resource: Resource{Project: "p"}}, Image: compute.Image{SourceDisk: "zones/z/disks/d"}},
			&Image{Image: compute.Image{SourceDisk: "projects/p/zones/z/disks/d"}},
			false,
		},
		{
			"SourceImage case",
			&Image{Image: compute.Image{SourceImage: "i"}},
			&Image{Image: compute.Image{SourceImage: "i"}},
			false,
		},
		{
			"SourceImage URL case",
			&Image{Image: compute.Image{SourceImage: "projects/p/global/images/i"}},
			&Image{Image: compute.Image{SourceImage: "projects/p/global/images/i"}},
			false,
		},
		{
			"extend SourceImage URL case",
			&Image{ImageBase: ImageBase{Resource: Resource{Project: "p"}}, Image: compute.Image{SourceImage: "global/images/i"}},
			&Image{Image: compute.Image{SourceImage: "projects/p/global/images/i"}},
			false,
		},
		{
			"RawDisk.Source from Sources case",
			&Image{Image: compute.Image{RawDisk: &compute.ImageRawDisk{Source: "d"}}},
			&Image{Image: compute.Image{RawDisk: &compute.ImageRawDisk{Source: w.getSourceGCSAPIPath("d")}}},
			false,
		},
		{
			"RawDisk.Source GCS URL case",
			&Image{Image: compute.Image{RawDisk: &compute.ImageRawDisk{Source: "gs://bucket/d"}}},
			&Image{Image: compute.Image{RawDisk: &compute.ImageRawDisk{Source: gcsAPIPath}}},
			false,
		},
		{
			"GuestOsFeatures",
			&Image{Image: compute.Image{SourceImage: "i"}, ImageBase: ImageBase{}, GuestOsFeatures: guestOsFeatures{"foo", "bar"}},
			&Image{Image: compute.Image{SourceImage: "i", GuestOsFeatures: []*compute.GuestOsFeature{{Type: "foo"}, {Type: "bar"}}}, ImageBase: ImageBase{}, GuestOsFeatures: guestOsFeatures{"foo", "bar"}},
			false,
		},
		{
			"Bad RawDisk.Source case",
			&Image{ImageBase: ImageBase{Resource: Resource{}}, Image: compute.Image{RawDisk: &compute.ImageRawDisk{Source: "blah"}}},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		err := (&tt.input.ImageBase).populate(ctx, tt.input, s)

		// Test sanitation -- clean/set irrelevant fields.
		if tt.want != nil {
			tt.want.Name = tt.input.RealName
			tt.want.Description = tt.input.Description
		}
		tt.input.Resource = Resource{} // These fields are tested in resource_test.

		if tt.wantErr {
			if err == nil {
				t.Errorf("%s: should have returned an error but didn't", tt.desc)
			}
		} else if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		} else if diffRes := diff(tt.input, tt.want, 0); diffRes != "" {
			t.Errorf("%s: populated Image does not match expectation: (-got,+want)\n%s", tt.desc, diffRes)
		}
	}
}

func TestImageBetaPopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.Sources = map[string]string{"d": "d"}
	s, _ := w.NewStep("s")

	gcsAPIPath, _ := getGCSAPIPath("gs://bucket/d")
	tests := []struct {
		desc        string
		input, want *ImageBeta
		wantErr     bool
	}{
		{
			"SourceDisk case",
			&ImageBeta{Image: computeBeta.Image{SourceDisk: "d"}},
			&ImageBeta{Image: computeBeta.Image{SourceDisk: "d"}},
			false,
		},
		{
			"SourceDisk URL case",
			&ImageBeta{Image: computeBeta.Image{SourceDisk: "projects/p/zones/z/disks/d"}},
			&ImageBeta{Image: computeBeta.Image{SourceDisk: "projects/p/zones/z/disks/d"}},
			false,
		},
		{
			"extend SourceDisk URL case",
			&ImageBeta{ImageBase: ImageBase{Resource: Resource{Project: "p"}}, Image: computeBeta.Image{SourceDisk: "zones/z/disks/d"}},
			&ImageBeta{Image: computeBeta.Image{SourceDisk: "projects/p/zones/z/disks/d"}},
			false,
		},
		{
			"SourceImage case",
			&ImageBeta{Image: computeBeta.Image{SourceImage: "i"}},
			&ImageBeta{Image: computeBeta.Image{SourceImage: "i"}},
			false,
		},
		{
			"SourceImage URL case",
			&ImageBeta{Image: computeBeta.Image{SourceImage: "projects/p/global/images/i"}},
			&ImageBeta{Image: computeBeta.Image{SourceImage: "projects/p/global/images/i"}},
			false,
		},
		{
			"extend SourceImage URL case",
			&ImageBeta{ImageBase: ImageBase{Resource: Resource{Project: "p"}}, Image: computeBeta.Image{SourceImage: "global/images/i"}},
			&ImageBeta{Image: computeBeta.Image{SourceImage: "projects/p/global/images/i"}},
			false,
		},
		{
			"RawDisk.Source from Sources case",
			&ImageBeta{Image: computeBeta.Image{RawDisk: &computeBeta.ImageRawDisk{Source: "d"}}},
			&ImageBeta{Image: computeBeta.Image{RawDisk: &computeBeta.ImageRawDisk{Source: w.getSourceGCSAPIPath("d")}}},
			false,
		},
		{
			"RawDisk.Source GCS URL case",
			&ImageBeta{Image: computeBeta.Image{RawDisk: &computeBeta.ImageRawDisk{Source: "gs://bucket/d"}}},
			&ImageBeta{Image: computeBeta.Image{RawDisk: &computeBeta.ImageRawDisk{Source: gcsAPIPath}}},
			false,
		},
		{
			"GuestOsFeatures",
			&ImageBeta{Image: computeBeta.Image{SourceImage: "i"}, ImageBase: ImageBase{}, GuestOsFeatures: guestOsFeatures{"foo", "bar"}},
			&ImageBeta{Image: computeBeta.Image{SourceImage: "i", GuestOsFeatures: []*computeBeta.GuestOsFeature{{Type: "foo"}, {Type: "bar"}}}, ImageBase: ImageBase{}, GuestOsFeatures: guestOsFeatures{"foo", "bar"}},
			false,
		},
		{
			"Bad RawDisk.Source case",
			&ImageBeta{ImageBase: ImageBase{Resource: Resource{}}, Image: computeBeta.Image{RawDisk: &computeBeta.ImageRawDisk{Source: "blah"}}},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		err := (&tt.input.ImageBase).populate(ctx, tt.input, s)

		// Test sanitation -- clean/set irrelevant fields.
		if tt.want != nil {
			tt.want.Name = tt.input.RealName
			tt.want.Description = tt.input.Description
		}
		tt.input.Resource = Resource{} // These fields are tested in resource_test.

		if tt.wantErr {
			if err == nil {
				t.Errorf("%s: should have returned an error but didn't", tt.desc)
			}
		} else if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		} else if diffRes := diff(tt.input, tt.want, 0); diffRes != "" {
			t.Errorf("%s: populated Image does not match expectation: (-got,+want)\n%s", tt.desc, diffRes)
		}
	}
}
func TestImageValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.Sources = map[string]string{"source": "gs://some/file"}
	d1Creator, e1 := w.NewStep("d1Creator")
	d2Creator, e2 := w.NewStep("d2Creator")
	d2Deleter, e3 := w.NewStep("d2Deleter")
	d3Creator, e4 := w.NewStep("d3Creator")
	si1Creator, e5 := w.NewStep("si1Creator")
	e6 := w.AddDependency(d2Deleter, d2Creator)

	// Set up some test resources
	e7 := w.disks.regCreate("d1", &Resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d1", w.Project, w.Zone)}, d1Creator, false)
	e8 := w.disks.regCreate("d2", &Resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d2", w.Project, w.Zone)}, d2Creator, false)
	e9 := w.disks.regDelete("d2", d2Deleter)
	e10 := w.disks.regCreate("d3", &Resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d3", w.Project, w.Zone)}, d3Creator, false)
	si1 := &Resource{link: fmt.Sprintf("projects/%s/global/images/si1", w.Project)}
	e11 := w.images.regCreate("si1", si1, si1Creator, false)
	if errs := addErrs(nil, e1, e2, e3, e4, e5, e6, e7, e8, e9, e10, e11); errs != nil {
		t.Fatalf("test set up error: %v", errs)
	}

	tests := []struct {
		desc      string
		i         *Image
		shouldErr bool
	}{
		{"good disk case", &Image{Image: compute.Image{Name: "i1", SourceDisk: "d1"}}, false},
		{"good licenses case", &Image{Image: compute.Image{Name: "i2", SourceDisk: "d1", Licenses: []string{fmt.Sprintf("projects/%s/global/licenses/%s", w.Project, testLicense)}}}, false},
		{"good image case", &Image{Image: compute.Image{Name: "i3", SourceImage: "si1"}}, false},
		{"good raw disk case", &Image{Image: compute.Image{Name: "i4", RawDisk: &compute.ImageRawDisk{Source: "https://storage.cloud.google.com/bucket/object"}}}, false},
		{"good disk url case ", &Image{Image: compute.Image{Name: "i5", SourceDisk: fmt.Sprintf("projects/%s/zones/%s/disks/%s", testProject, testZone, testDisk)}}, false},
		{"bad license case", &Image{Image: compute.Image{Name: "i6", SourceDisk: "d1", Licenses: []string{fmt.Sprintf("projects/%s/global/licenses/bad", testProject)}}}, true},
		{"bad dupe name case", &Image{Image: compute.Image{Name: "i1", SourceDisk: "d1"}}, true},
		{"bad missing dep on disk creator case", &Image{Image: compute.Image{Name: "i5", SourceDisk: "d3"}}, true},
		{"bad disk deleted case", &Image{Image: compute.Image{Name: "i6", SourceDisk: "d2"}}, true},
		{"bad image case", &Image{Image: compute.Image{Name: "i6", SourceImage: "si2"}}, true},
		{"bad raw disk URL dne case", &Image{Image: compute.Image{Name: "i6", RawDisk: &compute.ImageRawDisk{Source: "https://storage.cloud.google.com/bucket/dne"}}}, true},
		{"bad raw disk case", &Image{Image: compute.Image{Name: "i6", RawDisk: &compute.ImageRawDisk{Source: "not/a/gcs/url"}}}, true},
		{"bad using disk and raw disk case", &Image{Image: compute.Image{Name: "i6", SourceDisk: "d1", RawDisk: &compute.ImageRawDisk{Source: "https://storage.cloud.google.com/bucket/object"}}}, true},
		{"bad using disk and raw disk and image case", &Image{Image: compute.Image{Name: "i6", SourceDisk: "d1", RawDisk: &compute.ImageRawDisk{Source: "https://storage.cloud.google.com/bucket/object"}}}, true},
	}

	for testNum, tt := range tests {
		s, _ := w.NewStep("s" + strconv.Itoa(testNum))
		s.CreateImages = &CreateImages{Images: []*Image{tt.i}}
		w.AddDependency(s, d1Creator, d2Deleter, si1Creator)

		// Test sanitation -- clean/set irrelevant fields.
		tt.i.daisyName = tt.i.Name
		tt.i.RealName = tt.i.Name
		tt.i.link = fmt.Sprintf("projects/%s/global/images/%s", w.Project, tt.i.Name)
		tt.i.Project = w.Project // Resource{} fields are tested in resource_test.

		if err := s.CreateImages.validate(ctx, s); err == nil {
			if tt.shouldErr {
				t.Errorf("%s: should have returned an error but didn't", tt.desc)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}
