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

package publish

import (
	"context"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/kylelemons/godebug/pretty"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/compute/v1"
)

func TestPublishImage(t *testing.T) {
	tests := []struct {
		desc    string
		p       *Publish
		img     *Image
		pubImgs []*computeAlpha.Image
		skipDup bool
		replace bool
		wantCI  *daisy.CreateImages
		wantDI  *daisy.DeprecateImages
		wantErr bool
	}{
		{
			"normal case",
			&Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			&Image{Prefix: "foo", Family: "foo-family", GuestOsFeatures: []string{"foo-feature", "bar-feature"}},
			[]*computeAlpha.Image{
				{Name: "bar-2", Family: "bar-family"},
				{Name: "foo-2", Family: "foo-family"},
				{Name: "foo-1", Family: "foo-family", Deprecated: &computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
				{Name: "bar-1", Family: "bar-family", Deprecated: &computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
			},
			false,
			false,
			&daisy.CreateImages{{GuestOsFeatures: []string{"foo-feature", "bar-feature"}, Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"}, Image: computeAlpha.Image{
				Name: "foo-3", Family: "foo-family", SourceImage: "projects/bar-project/global/images/foo-3"},
			}},
			&daisy.DeprecateImages{{Image: "foo-2", Project: "foo-project", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED", Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/foo-3"}}},
			false,
		},
		{
			"multiple images to deprecate",
			&Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			&Image{Prefix: "foo", Family: "foo-family"},
			[]*computeAlpha.Image{
				{Name: "bar-2", Family: "bar-family"},
				{Name: "foo-2", Family: "foo-family"},
				{Name: "foo-1", Family: "foo-family"},
				{Name: "bar-1", Family: "bar-family"},
			},
			false,
			false,
			&daisy.CreateImages{{Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"}, Image: computeAlpha.Image{Name: "foo-3", Family: "foo-family", SourceImage: "projects/bar-project/global/images/foo-3"}}},
			&daisy.DeprecateImages{
				{Image: "foo-2", Project: "foo-project", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED", Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/foo-3"}},
				{Image: "foo-1", Project: "foo-project", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED", Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/foo-3"}},
			},
			false,
		},
		{
			"GCSPath case",
			&Publish{SourceGCSPath: "gs://bar-project-path", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			&Image{Prefix: "foo", Family: "foo-family"},
			[]*computeAlpha.Image{},
			false,
			false,
			&daisy.CreateImages{
				{Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"}, Image: computeAlpha.Image{Name: "foo-3", Family: "foo-family", RawDisk: &computeAlpha.ImageRawDisk{Source: "gs://bar-project-path/foo-3/root.tar.gz"}}},
			},
			nil,
			false,
		},
		{
			"both SourceGCSPath and SourceProject set",
			&Publish{SourceGCSPath: "gs://bar-project-path", SourceProject: "bar-project"},
			&Image{},
			nil,
			false,
			false,
			nil,
			nil,
			true,
		},
		{
			"neither SourceGCSPath and SourceProject set",
			&Publish{},
			&Image{},
			nil,
			false,
			false,
			nil,
			nil,
			true,
		},
		{
			"image already exists",
			&Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			&Image{Prefix: "foo", Family: "foo-family", GuestOsFeatures: []string{"foo-feature"}},
			[]*computeAlpha.Image{{Name: "foo-3", Family: "foo-family"}},
			false,
			false,
			nil,
			nil,
			true,
		},
		{
			"image already exists, skipDup set",
			&Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			&Image{Prefix: "foo", Family: "foo-family", GuestOsFeatures: []string{"foo-feature"}},
			[]*computeAlpha.Image{
				{Name: "foo-3", Family: "bar-family"},
				{Name: "foo-2", Family: "foo-family"},
			},
			true,
			false,
			nil,
			&daisy.DeprecateImages{{Image: "foo-2", Project: "foo-project", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED", Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/foo-3"}}},
			false,
		},
		{
			"image already exists, replace set",
			&Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			&Image{Prefix: "foo", Family: "foo-family"},
			[]*computeAlpha.Image{
				{Name: "foo-3", Family: "bar-family"},
				{Name: "foo-2", Family: "foo-family"},
			},
			false,
			true,
			&daisy.CreateImages{{OverWrite: true, Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"}, Image: computeAlpha.Image{Name: "foo-3", Family: "foo-family", SourceImage: "projects/bar-project/global/images/foo-3"}}},
			&daisy.DeprecateImages{{Image: "foo-2", Project: "foo-project", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED", Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/foo-3"}}},
			false,
		},
		{
			"new image from src, without version",
			&Publish{SourceProject: "bar-project", PublishProject: "foo-project"},
			&Image{Prefix: "foo-x", Family: "foo-family", GuestOsFeatures: []string{"foo-feature", "bar-feature"}},
			[]*computeAlpha.Image{
				{Name: "bar-x", Family: "bar-family"},
			},
			false,
			false,
			&daisy.CreateImages{{GuestOsFeatures: []string{"foo-feature", "bar-feature"}, Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-x"}, Image: computeAlpha.Image{
				Name: "foo-x", Family: "foo-family", SourceImage: "projects/bar-project/global/images/foo-x"},
			}},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		dr, di, _, err := publishImage(tt.p, tt.img, tt.pubImgs, tt.skipDup, tt.replace)
		if tt.wantErr && err != nil {
			continue
		}
		if !tt.wantErr && err != nil {
			t.Errorf("%s: error from publishImage(): %v", tt.desc, err)
			continue
		} else if tt.wantErr && err == nil {
			t.Errorf("%s: did not get expected error from publishImage()", tt.desc)
		}

		if diff := pretty.Compare(dr, tt.wantCI); diff != "" {
			t.Errorf("%s: returned CreateImages does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
		if diff := pretty.Compare(di, tt.wantDI); diff != "" {
			t.Errorf("%s: returned DeprecateImages does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
	}

}

func TestRollbackImage(t *testing.T) {
	tests := []struct {
		desc    string
		p       *Publish
		img     *Image
		pubImgs []*computeAlpha.Image
		wantDR  *daisy.DeleteResources
		wantDI  *daisy.DeprecateImages
	}{
		{
			"normal case",
			&Publish{PublishProject: "foo-project", publishVersion: "3"},
			&Image{Prefix: "foo", Family: "foo-family"},
			[]*computeAlpha.Image{
				{Name: "bar-3", Family: "bar-family"},
				{Name: "foo-3", Family: "foo-family"},
				{Name: "bar-2", Family: "bar-family", Deprecated: &computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
				{Name: "foo-2", Family: "foo-family", Deprecated: &computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
				{Name: "foo-1", Family: "foo-family", Deprecated: &computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
				{Name: "bar-1", Family: "bar-family", Deprecated: &computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
			},
			&daisy.DeleteResources{Images: []string{"projects/foo-project/global/images/foo-3"}},
			&daisy.DeprecateImages{{Image: "foo-2", Project: "foo-project"}},
		},
		{
			"no image to undeprecate",
			&Publish{PublishProject: "foo-project", publishVersion: "3"},
			&Image{Prefix: "foo", Family: "foo-family"},
			[]*computeAlpha.Image{
				{Name: "bar-3", Family: "bar-family"},
				{Name: "foo-3", Family: "foo-family"},
				{Name: "bar-2", Family: "bar-family", Deprecated: &computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
				{Name: "bar-1", Family: "bar-family", Deprecated: &computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
			},
			&daisy.DeleteResources{Images: []string{"projects/foo-project/global/images/foo-3"}},
			&daisy.DeprecateImages{},
		},
		{
			"image DNE",
			&Publish{PublishProject: "foo-project", publishVersion: "1"},
			&Image{Prefix: "foo", Family: "foo-family"},
			[]*computeAlpha.Image{
				{Name: "bar-1", Family: "bar-family"},
			},
			nil,
			nil,
		},
	}
	for _, tt := range tests {
		dr, di := rollbackImage(tt.p, tt.img, tt.pubImgs)
		if diff := pretty.Compare(dr, tt.wantDR); diff != "" {
			t.Errorf("%s: returned DeleteResources does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
		if diff := pretty.Compare(di, tt.wantDI); diff != "" {
			t.Errorf("%s: returned DeprecateImages does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
	}

}

func TestPopulateSteps(t *testing.T) {
	// This scenario is a bit contrived as there's no way you will get
	// DeleteResources steps and CreateImages steps in the same workflow,
	// but this simplifies the test data.
	got := daisy.New()
	err := populateSteps(
		got,
		"foo",
		&daisy.CreateImages{{Image: computeAlpha.Image{Name: "create-image"}}},
		&daisy.DeprecateImages{{Image: "deprecate-image"}},
		&daisy.DeleteResources{Images: []string{"delete-image"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	got.Cancel = nil

	want := &daisy.Workflow{
		Steps: map[string]*daisy.Step{
			"delete-foo":    {DeleteResources: &daisy.DeleteResources{Images: []string{"delete-image"}}},
			"deprecate-foo": {DeprecateImages: &daisy.DeprecateImages{{Image: "deprecate-image"}}},
			"publish-foo":   {Timeout: "1h", CreateImages: &daisy.CreateImages{{Image: computeAlpha.Image{Name: "create-image"}}}},
		},
		Dependencies: map[string][]string{
			"delete-foo":    {"publish-foo", "deprecate-foo"},
			"deprecate-foo": {"publish-foo"},
		},
		DefaultTimeout: "10m",
	}

	if diff := (&pretty.Config{Diffable: true, Formatter: pretty.DefaultFormatter}).Compare(got, want); diff != "" {
		t.Errorf("-got +want\n%s", diff)
	}

}

func TestPopulateWorkflow(t *testing.T) {
	got := daisy.New()
	p := &Publish{
		SourceProject:  "foo-project",
		PublishProject: "foo-project",
		publishVersion: "pv",
		sourceVersion:  "sv",
		Images: []*Image{
			{
				Prefix: "test",
				Family: "test-family",
			},
		},
	}
	err := p.populateWorkflow(
		context.Background(),
		got,
		[]*computeAlpha.Image{
			{Name: "test-old", Family: "test-family"},
		},
		p.Images[0],
		false,
		false,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	got.Cancel = nil

	want := &daisy.Workflow{
		Steps: map[string]*daisy.Step{
			"publish-test": {Timeout: "1h", CreateImages: &daisy.CreateImages{
				{Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "test-pv"}, Image: computeAlpha.Image{Name: "test-pv", Family: "test-family", SourceImage: "projects/foo-project/global/images/test-sv"}}},
			},
			"deprecate-test": {DeprecateImages: &daisy.DeprecateImages{
				{Project: "foo-project", Image: "test-old", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED", Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/test-pv"}}},
			},
		},
		Dependencies: map[string][]string{
			"deprecate-test": {"publish-test"},
		},
		DefaultTimeout: "10m",
	}

	if diff := (&pretty.Config{Diffable: true, Formatter: pretty.DefaultFormatter}).Compare(got, want); diff != "" {
		t.Errorf("-got +want\n%s", diff)
	}

}

func TestCreatePrintOut(t *testing.T) {
	tests := []struct {
		name string
		args *daisy.CreateImages
		want []string
	}{
		{"empty", nil, nil},
		{"one image", &daisy.CreateImages{&daisy.Image{Image: computeAlpha.Image{Name: "foo", Description: "bar"}}}, []string{"foo: (bar)"}},
		{"two images", &daisy.CreateImages{
			&daisy.Image{Image: computeAlpha.Image{Name: "foo1", Description: "bar1"}},
			&daisy.Image{Image: computeAlpha.Image{Name: "foo2", Description: "bar2"}}},
			[]string{"foo1: (bar1)", "foo2: (bar2)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Publish{}
			p.createPrintOut(tt.args)
			if !reflect.DeepEqual(p.toCreate, tt.want) {
				t.Errorf("createPrintOut() got = %v, want %v", p.toCreate, tt.want)
			}
		})
	}
}

func TestDeletePrintOut(t *testing.T) {
	tests := []struct {
		name string
		args *daisy.DeleteResources
		want []string
	}{
		{"empty", nil, nil},
		{"not an image", &daisy.DeleteResources{Disks: []string{"foo"}}, nil},
		{"one image", &daisy.DeleteResources{Images: []string{"foo"}}, []string{"foo"}},
		{"two images", &daisy.DeleteResources{Images: []string{"foo", "bar"}}, []string{"foo", "bar"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Publish{}
			p.deletePrintOut(tt.args)
			if !reflect.DeepEqual(p.toDelete, tt.want) {
				t.Errorf("deletePrintOut() got = %v, want %v", p.toDelete, tt.want)
			}
		})
	}
}

func TestDeprecatePrintOut(t *testing.T) {
	tests := []struct {
		name          string
		args          *daisy.DeprecateImages
		toDeprecate   []string
		toObsolete    []string
		toUndeprecate []string
	}{
		{"empty", nil, nil, nil, nil},
		{"unknown state", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatus: compute.DeprecationStatus{State: "foo"}}}, nil, nil, nil},
		{"only DEPRECATED", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED"}}}, []string{"foo"}, nil, nil},
		{"only OBSOLETE", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatus: compute.DeprecationStatus{State: "OBSOLETE"}}}, nil, []string{"foo"}, nil},
		{"only un-deprecated", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatus: compute.DeprecationStatus{State: ""}}}, nil, nil, []string{"foo"}},
		{"all three", &daisy.DeprecateImages{
			&daisy.DeprecateImage{Image: "foo", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED"}},
			&daisy.DeprecateImage{Image: "bar", DeprecationStatus: compute.DeprecationStatus{State: "OBSOLETE"}},
			&daisy.DeprecateImage{Image: "baz", DeprecationStatus: compute.DeprecationStatus{State: ""}}},
			[]string{"foo"}, []string{"bar"}, []string{"baz"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Publish{}
			p.deprecatePrintOut(tt.args)
			if !reflect.DeepEqual(p.toDeprecate, tt.toDeprecate) {
				t.Errorf("deprecatePrintOut() got = %v, want %v", p.toDeprecate, tt.toDeprecate)
			}
			if !reflect.DeepEqual(p.toObsolete, tt.toObsolete) {
				t.Errorf("deprecatePrintOut() got1 = %v, want %v", p.toObsolete, tt.toObsolete)
			}
			if !reflect.DeepEqual(p.toUndeprecate, tt.toUndeprecate) {
				t.Errorf("deprecatePrintOut() got2 = %v, want %v", p.toUndeprecate, tt.toUndeprecate)
			}
		})
	}
}
