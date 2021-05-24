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
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/kylelemons/godebug/pretty"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/compute/v1"
)

func TestPublishImage(t *testing.T) {
	now := time.Now()
	fakeInitialState := &computeAlpha.InitialStateConfig{
		Dbs: []*computeAlpha.FileContentBuffer{
			{
				Content:  "abc",
				FileType: "BIN",
			},
		},
		Dbxs: []*computeAlpha.FileContentBuffer{
			{
				Content:  "abc",
				FileType: "X509",
			},
		},
		NullFields: []string{"Keks", "Pk"},
	}

	tests := []struct {
		desc    string
		p       *Publish
		img     *Image
		pubImgs []*computeAlpha.Image
		skipDup bool
		replace bool
		noRoot  bool
		wantCI  *daisy.CreateImages
		wantDI  *daisy.DeprecateImages
		wantErr bool
	}{
		{
			desc: "normal case",
			p:    &Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			img: &Image{
				Prefix:          "foo",
				Family:          "foo-family",
				GuestOsFeatures: []string{"foo-feature", "bar-feature"},
				ObsoleteDate:    &now,
				RolloutPolicy: &computeAlpha.RolloutPolicy{
					DefaultRolloutTime: now.Format(time.RFC3339),
				},
			},
			pubImgs: []*computeAlpha.Image{
				{Name: "bar-2", Family: "bar-family"},
				{Name: "foo-2", Family: "foo-family"},
				{
					Name:   "foo-1",
					Family: "foo-family",
					Deprecated: &computeAlpha.DeprecationStatus{
						State: "DEPRECATED",
						StateOverride: &computeAlpha.RolloutPolicy{
							DefaultRolloutTime: now.Format(time.RFC3339),
						},
					},
				},
				{
					Name:   "bar-1",
					Family: "bar-family",
					Deprecated: &computeAlpha.DeprecationStatus{
						State: "DEPRECATED",
						StateOverride: &computeAlpha.RolloutPolicy{
							DefaultRolloutTime: now.Format(time.RFC3339),
						},
					},
				},
			},
			wantCI: &daisy.CreateImages{
				ImagesAlpha: []*daisy.ImageAlpha{
					{
						ImageBase: daisy.ImageBase{Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"}},
						Image: computeAlpha.Image{
							Name: "foo-3", Family: "foo-family",
							SourceImage: "projects/bar-project/global/images/foo-3",
							RolloutOverride: &computeAlpha.RolloutPolicy{
								DefaultRolloutTime: now.Format(time.RFC3339),
							},
							Deprecated: &computeAlpha.DeprecationStatus{
								State:    "ACTIVE",
								Obsolete: now.Format(time.RFC3339),
							},
						},
						GuestOsFeatures: []string{"foo-feature", "bar-feature"},
					},
				},
			},
			wantDI: &daisy.DeprecateImages{
				{
					Image:   "foo-2",
					Project: "foo-project",
					DeprecationStatusAlpha: computeAlpha.DeprecationStatus{
						State:       "DEPRECATED",
						Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/foo-3",
						StateOverride: &computeAlpha.RolloutPolicy{
							DefaultRolloutTime: now.Format(time.RFC3339),
						},
					},
				},
			},
		},
		{
			desc: "multiple images to deprecate",
			p:    &Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			img: &Image{
				Prefix: "foo",
				Family: "foo-family",
				RolloutPolicy: &computeAlpha.RolloutPolicy{
					DefaultRolloutTime: now.Format(time.RFC3339),
				},
			},
			pubImgs: []*computeAlpha.Image{
				{Name: "bar-2", Family: "bar-family"},
				{Name: "foo-2", Family: "foo-family"},
				{Name: "foo-1", Family: "foo-family"},
				{Name: "bar-1", Family: "bar-family"},
			},
			wantCI: &daisy.CreateImages{
				ImagesAlpha: []*daisy.ImageAlpha{
					{
						ImageBase: daisy.ImageBase{Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"}},
						Image: computeAlpha.Image{
							Name:        "foo-3",
							Family:      "foo-family",
							SourceImage: "projects/bar-project/global/images/foo-3",
							RolloutOverride: &computeAlpha.RolloutPolicy{
								DefaultRolloutTime: now.Format(time.RFC3339),
							},
						},
					},
				},
			},
			wantDI: &daisy.DeprecateImages{
				{
					Image:   "foo-2",
					Project: "foo-project",
					DeprecationStatusAlpha: computeAlpha.DeprecationStatus{
						State:       "DEPRECATED",
						Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/foo-3",
						StateOverride: &computeAlpha.RolloutPolicy{
							DefaultRolloutTime: now.Format(time.RFC3339),
						},
					},
				},
				{
					Image:   "foo-1",
					Project: "foo-project",
					DeprecationStatusAlpha: computeAlpha.DeprecationStatus{
						State:       "DEPRECATED",
						Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/foo-3",
						StateOverride: &computeAlpha.RolloutPolicy{
							DefaultRolloutTime: now.Format(time.RFC3339),
						},
					},
				},
			},
		},
		{
			desc:    "GCSPath case",
			p:       &Publish{SourceGCSPath: "gs://bar-project-path", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			img:     &Image{Prefix: "foo", Family: "foo-family"},
			pubImgs: []*computeAlpha.Image{},
			wantCI: &daisy.CreateImages{
				ImagesAlpha: []*daisy.ImageAlpha{
					{
						ImageBase: daisy.ImageBase{Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"}},
						Image: computeAlpha.Image{
							Name:    "foo-3",
							Family:  "foo-family",
							RawDisk: &computeAlpha.ImageRawDisk{Source: "gs://bar-project-path/foo-3/root.tar.gz"},
						},
					},
				},
			},
		},
		{
			desc:    "GCSPath with noRoot case",
			p:       &Publish{SourceGCSPath: "gs://bar-project-path", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			img:     &Image{Prefix: "foo", Family: "foo-family"},
			pubImgs: []*computeAlpha.Image{},
			noRoot:  true,
			wantCI: &daisy.CreateImages{
				ImagesAlpha: []*daisy.ImageAlpha{
					{
						ImageBase: daisy.ImageBase{Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"}},
						Image: computeAlpha.Image{
							Name:    "foo-3",
							Family:  "foo-family",
							RawDisk: &computeAlpha.ImageRawDisk{Source: "gs://bar-project-path/foo-3.tar.gz"},
						},
					},
				},
			},
		},
		{
			desc:    "both SourceGCSPath and SourceProject set",
			p:       &Publish{SourceGCSPath: "gs://bar-project-path", SourceProject: "bar-project"},
			img:     &Image{},
			wantErr: true,
		},
		{
			desc:    "neither SourceGCSPath and SourceProject set",
			p:       &Publish{},
			img:     &Image{},
			wantErr: true,
		},
		{
			desc:    "image already exists",
			p:       &Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			img:     &Image{Prefix: "foo", Family: "foo-family", GuestOsFeatures: []string{"foo-feature"}},
			pubImgs: []*computeAlpha.Image{{Name: "foo-3", Family: "foo-family"}},
			wantErr: true,
		},
		{
			desc: "image already exists, skipDup set",
			p:    &Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			img: &Image{
				Prefix:          "foo",
				Family:          "foo-family",
				GuestOsFeatures: []string{"foo-feature"},
			},
			pubImgs: []*computeAlpha.Image{
				{Name: "foo-3", Family: "bar-family"},
				{Name: "foo-2", Family: "foo-family"},
			},
			skipDup: true,
			wantDI: &daisy.DeprecateImages{
				{
					Image: "foo-2", Project: "foo-project",
					DeprecationStatusAlpha: computeAlpha.DeprecationStatus{
						State:       "DEPRECATED",
						Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/foo-3",
					},
				},
			},
		},
		{
			desc: "image already exists, replace set",
			p:    &Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			img: &Image{
				Prefix: "foo",
				Family: "foo-family",
				RolloutPolicy: &computeAlpha.RolloutPolicy{
					DefaultRolloutTime: now.Format(time.RFC3339),
				},
			},
			pubImgs: []*computeAlpha.Image{
				{Name: "foo-3", Family: "bar-family"},
				{Name: "foo-2", Family: "foo-family"},
			},
			replace: true,
			wantCI: &daisy.CreateImages{
				ImagesAlpha: []*daisy.ImageAlpha{
					{
						ImageBase: daisy.ImageBase{
							OverWrite: true,
							Resource:  daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"},
						},
						Image: computeAlpha.Image{
							Name:        "foo-3",
							Family:      "foo-family",
							SourceImage: "projects/bar-project/global/images/foo-3",
							RolloutOverride: &computeAlpha.RolloutPolicy{
								DefaultRolloutTime: now.Format(time.RFC3339),
							},
						},
					},
				},
			},
			wantDI: &daisy.DeprecateImages{
				{
					Image:   "foo-2",
					Project: "foo-project",
					DeprecationStatusAlpha: computeAlpha.DeprecationStatus{
						State:       "DEPRECATED",
						Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/foo-3",
						StateOverride: &computeAlpha.RolloutPolicy{
							DefaultRolloutTime: now.Format(time.RFC3339),
						},
					},
				},
			},
		},
		{
			desc: "new image from src, without version",
			p:    &Publish{SourceProject: "bar-project", PublishProject: "foo-project"},
			img:  &Image{Prefix: "foo-x", Family: "foo-family", GuestOsFeatures: []string{"foo-feature", "bar-feature"}},
			pubImgs: []*computeAlpha.Image{
				{Name: "bar-x", Family: "bar-family"},
			},
			wantCI: &daisy.CreateImages{
				ImagesAlpha: []*daisy.ImageAlpha{
					{
						ImageBase: daisy.ImageBase{Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-x"}},
						Image: computeAlpha.Image{
							Name:        "foo-x",
							Family:      "foo-family",
							SourceImage: "projects/bar-project/global/images/foo-x",
						},
						GuestOsFeatures: []string{"foo-feature", "bar-feature"}},
				},
			},
		},
		{
			desc: "no image family, don't deprecate",
			p:    &Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			img:  &Image{Prefix: "foo", Family: "foo-family"},
			pubImgs: []*computeAlpha.Image{
				{Name: "foo-2", Family: ""},
				{Name: "foo-1", Family: "", Deprecated: &computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
			},
			wantCI: &daisy.CreateImages{
				ImagesAlpha: []*daisy.ImageAlpha{
					{
						ImageBase: daisy.ImageBase{Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"}},
						Image: computeAlpha.Image{
							Name:        "foo-3",
							Family:      "foo-family",
							SourceImage: "projects/bar-project/global/images/foo-3",
						},
					},
				},
			},
		},
		{
			desc:    "ignore license validation if forbidden",
			p:       &Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			img:     &Image{Prefix: "foo", Family: "foo-family", GuestOsFeatures: []string{"foo-feature"}, IgnoreLicenseValidationIfForbidden: true},
			pubImgs: []*computeAlpha.Image{},
			wantCI: &daisy.CreateImages{
				ImagesAlpha: []*daisy.ImageAlpha{
					{
						ImageBase: daisy.ImageBase{
							Resource:                           daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"},
							IgnoreLicenseValidationIfForbidden: true,
						},
						Image: computeAlpha.Image{
							Name:        "foo-3",
							Family:      "foo-family",
							SourceImage: "projects/bar-project/global/images/foo-3",
						},
						GuestOsFeatures: []string{"foo-feature"},
					},
				},
			},
		},
		{
			desc:    "don't ignore license validation if forbidden",
			p:       &Publish{SourceProject: "bar-project", PublishProject: "foo-project", sourceVersion: "3", publishVersion: "3"},
			img:     &Image{Prefix: "foo", Family: "foo-family", GuestOsFeatures: []string{"foo-feature"}, IgnoreLicenseValidationIfForbidden: false},
			pubImgs: []*computeAlpha.Image{},
			wantCI: &daisy.CreateImages{
				ImagesAlpha: []*daisy.ImageAlpha{
					{
						ImageBase: daisy.ImageBase{
							Resource:                           daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-3"},
							IgnoreLicenseValidationIfForbidden: false,
						},
						Image: computeAlpha.Image{
							Name:        "foo-3",
							Family:      "foo-family",
							SourceImage: "projects/bar-project/global/images/foo-3",
						},
						GuestOsFeatures: []string{"foo-feature"},
					},
				},
			},
		},
		{
			desc:    "new image from src, with ShieldedInstanceInitialState",
			p:       &Publish{SourceProject: "bar-project", PublishProject: "foo-project"},
			img:     &Image{Prefix: "foo-x", Family: "foo-family", ShieldedInstanceInitialState: fakeInitialState},
			pubImgs: []*computeAlpha.Image{},
			wantCI: &daisy.CreateImages{
				ImagesAlpha: []*daisy.ImageAlpha{
					{
						ImageBase: daisy.ImageBase{Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "foo-x"}},
						Image: computeAlpha.Image{
							Name:                         "foo-x",
							Family:                       "foo-family",
							SourceImage:                  "projects/bar-project/global/images/foo-x",
							ShieldedInstanceInitialState: fakeInitialState,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		dr, di, _, err := publishImage(tt.p, tt.img, tt.pubImgs, tt.skipDup, tt.replace, tt.noRoot)
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
			&daisy.DeprecateImages{{Image: "foo-2", Project: "foo-project", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "ACTIVE"}}},
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
		//&daisy.CreateImages{{Image: computeAlpha.Image{Name: "create-image"}}},
		&daisy.CreateImages{ImagesAlpha: []*daisy.ImageAlpha{{Image: computeAlpha.Image{Name: "create-image"}}}},

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
			"publish-foo":   {Timeout: "1h", CreateImages: &daisy.CreateImages{ImagesAlpha: []*daisy.ImageAlpha{{Image: computeAlpha.Image{Name: "create-image"}}}}},
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
	now := time.Now()
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
				RolloutPolicy: createRollOut([]*compute.Zone{
					{Name: "us-central1-a", Region: "https://www.googleapis.com/compute/v1/projects/projectname/regions/us-central1"},
					{Name: "us-central1-b", Region: "https://www.googleapis.com/compute/v1/projects/projectname/regions/us-central1"},
				}, now, 1),
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
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	got.Cancel = nil

	wantrp := computeAlpha.RolloutPolicy{DefaultRolloutTime: now.Format(time.RFC3339)}
	wantrp.LocationRolloutPolicies = make(map[string]string)
	wantrp.LocationRolloutPolicies["zones/us-central1-a"] = now.Format(time.RFC3339)
	wantrp.LocationRolloutPolicies["zones/us-central1-b"] = now.Add(time.Minute).Format(time.RFC3339)

	want := &daisy.Workflow{
		Steps: map[string]*daisy.Step{
			"publish-test": {Timeout: "1h", CreateImages: &daisy.CreateImages{
				ImagesAlpha: []*daisy.ImageAlpha{
					{
						ImageBase: daisy.ImageBase{Resource: daisy.Resource{Project: "foo-project", NoCleanup: true, RealName: "test-pv"}},
						Image: computeAlpha.Image{
							Name:            "test-pv",
							Family:          "test-family",
							SourceImage:     "projects/foo-project/global/images/test-sv",
							RolloutOverride: &wantrp,
						},
					},
				},
			}},
			"deprecate-test": {DeprecateImages: &daisy.DeprecateImages{
				{Project: "foo-project", Image: "test-old", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "DEPRECATED", Replacement: "https://www.googleapis.com/compute/v1/projects/foo-project/global/images/test-pv", StateOverride: &wantrp}}},
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
		{
			"one image",
			&daisy.CreateImages{ImagesAlpha: []*daisy.ImageAlpha{{Image: computeAlpha.Image{Name: "foo", Description: "bar"}}}},
			[]string{"foo: (bar)"},
		},
		{"two images", &daisy.CreateImages{ImagesAlpha: []*daisy.ImageAlpha{
			{Image: computeAlpha.Image{Name: "foo1", Description: "bar1"}},
			{Image: computeAlpha.Image{Name: "foo2", Description: "bar2"}},
		},
		},
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
		{"unknown state", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "foo"}}}, nil, nil, nil},
		{"only DEPRECATED", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "DEPRECATED", StateOverride: &computeAlpha.RolloutPolicy{DefaultRolloutTime: time.Now().Format(time.RFC3339)}}}}, []string{"foo"}, nil, nil},
		{"only OBSOLETE", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "OBSOLETE"}}}, nil, []string{"foo"}, nil},
		{"only un-deprecated", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "ACTIVE"}}}, nil, nil, []string{"foo"}},
		{"all three", &daisy.DeprecateImages{
			&daisy.DeprecateImage{Image: "foo", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
			&daisy.DeprecateImage{Image: "bar", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "OBSOLETE"}},
			&daisy.DeprecateImage{Image: "baz", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "ACTIVE"}}},
			[]string{"foo"}, []string{"bar"}, []string{"baz"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Publish{}
			p.deprecatePrintOut(tt.args)
			if !reflect.DeepEqual(p.toDeprecate, tt.toDeprecate) {
				t.Errorf("deprecatePrintOut() toDeprecate got = %v, want %v", p.toDeprecate, tt.toDeprecate)
			}
			if !reflect.DeepEqual(p.toObsolete, tt.toObsolete) {
				t.Errorf("deprecatePrintOut() toObsolete got = %v, want %v", p.toObsolete, tt.toObsolete)
			}
			if !reflect.DeepEqual(p.toUndeprecate, tt.toUndeprecate) {
				t.Errorf("deprecatePrintOut() toUndeprecate got = %v, want %v", p.toUndeprecate, tt.toUndeprecate)
			}
		})
	}
}

func TestCreateRollOut(t *testing.T) {
	startTime := time.Now().Round(time.Second)
	tests := []struct {
		desc             string
		zones            []*compute.Zone
		rolloutStartTime time.Time
		rolloutRate      int
		wantRollout      computeAlpha.RolloutPolicy
	}{
		{
			desc: "3 regions, each region has a different number of zones.",
			zones: []*compute.Zone{
				{
					Name:   "us-central1-a",
					Region: "https://www.googleapis.com/compute/v1/projects/projectname/regions/us-central1",
				},
				{
					Name:   "us-central1-b",
					Region: "https://www.googleapis.com/compute/v1/projects/projectname/regions/us-central1",
				},
				{
					Name:   "us-central2-a",
					Region: "https://www.googleapis.com/compute/v1/projects/projectname/regions/us-central2",
				},
				{
					Name:   "us-central2-b",
					Region: "https://www.googleapis.com/compute/v1/projects/projectname/regions/us-central2",
				},
				{
					Name:   "us-central2-c",
					Region: "https://www.googleapis.com/compute/v1/projects/projectname/regions/us-central2",
				},
				{
					Name:   "us-central3-a",
					Region: "https://www.googleapis.com/compute/v1/projects/projectname/regions/us-central2",
				},
			},
			rolloutStartTime: startTime,
			rolloutRate:      5,
			wantRollout: computeAlpha.RolloutPolicy{
				DefaultRolloutTime: startTime.Format(time.RFC3339),
				LocationRolloutPolicies: map[string]string{
					"us-central1-a": startTime.Format(time.RFC3339),
					"us-central2-a": startTime.Add(5 * time.Minute).Format(time.RFC3339),
					"us-central3-a": startTime.Add(10 * time.Minute).Format(time.RFC3339),
					"us-central1-b": startTime.Add(15 * time.Minute).Format(time.RFC3339),
					"us-central2-b": startTime.Add(20 * time.Minute).Format(time.RFC3339),
					"us-central2-c": startTime.Add(25 * time.Minute).Format(time.RFC3339),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			rollout := createRollOut(tt.zones, tt.rolloutStartTime, tt.rolloutRate)

			if reflect.DeepEqual(rollout, tt.wantRollout) {
				t.Errorf("unexpected rollout got = %s, want = %s", rollout, tt.wantRollout)
			}
		})
	}
}
