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
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/google/go-cmp/cmp"
	cmpopts "github.com/google/go-cmp/cmp/cmpopts"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
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
				Labels: map[string]string{"foo": "bar"},
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
							Labels: map[string]string{"foo": "bar"},
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

		if diff := cmp.Diff(tt.wantCI, dr, cmpopts.IgnoreUnexported(daisy.Resource{})); diff != "" {
			t.Errorf("%s: returned CreateImages does not match expectation: (-want +got)\n%s", tt.desc, diff)
		}
		if diff := cmp.Diff(tt.wantDI, di, cmpopts.IgnoreUnexported(daisy.Resource{})); diff != "" {
			t.Errorf("%s: returned DeprecateImages does not match expectation: (-want +got)\n%s", tt.desc, diff)
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
			&daisy.DeleteResources{},
			&daisy.DeprecateImages{
				{Image: "foo-3", Project: "foo-project", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
				{Image: "foo-2", Project: "foo-project", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "ACTIVE"}},
			},
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
			&daisy.DeleteResources{},
			&daisy.DeprecateImages{{Image: "foo-3", Project: "foo-project", DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "DEPRECATED"}}},
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
		dr, di := rollbackImage(tt.p, tt.img, tt.pubImgs, false)
		if diff := cmp.Diff(tt.wantDR, dr); diff != "" {
			t.Errorf("%s: returned DeleteResources does not match expectation: (-want +got)\n%s", tt.desc, diff)
		}
		if diff := cmp.Diff(tt.wantDI, di); diff != "" {
			t.Errorf("%s: returned DeprecateImages does not match expectation: (-want +got)\n%s", tt.desc, diff)
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
		// &daisy.CreateImages{{Image: computeAlpha.Image{Name: "create-image"}}},
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

	if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(daisy.Workflow{}, daisy.Resource{}, daisy.Step{}), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("-want +got\n%s", diff)
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
					{Name: "us-central1-c", Region: "https://www.googleapis.com/compute/v1/projects/projectname/regions/us-central1"},
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
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	got.Cancel = nil

	wantrp := computeAlpha.RolloutPolicy{DefaultRolloutTime: now.Add(time.Minute * 2).Format(time.RFC3339)}
	wantrp.LocationRolloutPolicies = make(map[string]string)
	wantrp.LocationRolloutPolicies["zones/us-central1-a"] = now.Format(time.RFC3339)
	wantrp.LocationRolloutPolicies["zones/us-central1-b"] = now.Add(time.Minute).Format(time.RFC3339)
	wantrp.LocationRolloutPolicies["zones/us-central1-c"] = now.Add(time.Minute * 2).Format(time.RFC3339)

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
	if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(daisy.Workflow{}, daisy.Resource{}, daisy.Step{}), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("-want +got\n%s", diff)
	}

}

func TestPublishDetermination(t *testing.T) {
	launched := true
	notLaunched := false
	eol := true
	loc, err := time.LoadLocation(pacificTimeZone)
	if err != nil {
		t.Fatalf("LoadLocation failed for %q: %v", pacificTimeZone, err)
	}
	secondTuesday := time.Date(2026, time.July, 14, 12, 0, 0, 0, loc)
	notSecondTuesday := time.Date(2026, time.July, 15, 12, 0, 0, 0, loc)
	tests := []struct {
		desc        string
		img         *Image
		environment string
		now         time.Time
		want        []string
	}{
		{
			desc:        "non-prod does not gate",
			img:         &Image{},
			environment: "test",
			now:         notSecondTuesday,
		},
		{
			desc:        "prod blocks unknown launched status",
			img:         &Image{},
			environment: "prod",
			now:         secondTuesday,
			want:        []string{"launched status is unknown"},
		},
		{
			desc:        "prod blocks not launched image",
			img:         &Image{Launched: &notLaunched},
			environment: "prod",
			now:         secondTuesday,
			want:        []string{"image is not launched"},
		},
		{
			desc:        "prod allows launched non-EOL image",
			img:         &Image{Launched: &launched},
			environment: "prod",
			now:         secondTuesday,
		},
		{
			desc:        "prod blocks EOL image",
			img:         &Image{Launched: &launched, EOL: &eol},
			environment: "prod",
			now:         secondTuesday,
			want:        []string{"image is EOL"},
		},
		{
			desc:        "empty cadence does not gate cadence date",
			img:         &Image{Launched: &launched},
			environment: "prod",
			now:         notSecondTuesday,
		},
		{
			desc:        "cadence allows second Tuesday",
			img:         &Image{Launched: &launched, Cadence: "monthly"},
			environment: "prod",
			now:         secondTuesday,
		},
		{
			desc:        "cadence blocks non-second Tuesday",
			img:         &Image{Launched: &launched, Cadence: "monthly"},
			environment: "prod",
			now:         notSecondTuesday,
			want:        []string{"publish cadence requires the second Tuesday of the month; current date is 2026-07-15"},
		},
		{
			desc:        "cadence blocks disabled image",
			img:         &Image{Launched: &launched, Cadence: "disabled"},
			environment: "prod",
			now:         secondTuesday,
			want:        []string{"image cadence is \"disabled\""},
		},
		{
			desc:        "prod collects multiple block reasons",
			img:         &Image{Launched: &notLaunched, EOL: &eol, Cadence: "monthly"},
			environment: "prod",
			now:         notSecondTuesday,
			want: []string{
				"image is not launched",
				"image is EOL",
				"publish cadence requires the second Tuesday of the month; current date is 2026-07-15",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := publishDetermination(tt.img, tt.environment, tt.now)
			if err != nil {
				t.Fatalf("%s: publishDetermination() failed: %v", tt.desc, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: publishDetermination() got = %v, want %v", tt.desc, got, tt.want)
			}
		})
	}
}

func TestIsSecondTuesday(t *testing.T) {
	tests := []struct {
		desc string
		date time.Time
		want bool
	}{
		{
			desc: "second Tuesday of the month",
			date: time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			desc: "first Tuesday of the month",
			date: time.Date(2026, time.July, 7, 12, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			desc: "third Tuesday of the month",
			date: time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			desc: "second Wednesday of the month",
			date: time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := isSecondTuesday(tt.date); got != tt.want {
				t.Errorf("isSecondTuesday(%s) got = %v, want %v", tt.date.Format("2006-01-02"), got, tt.want)
			}
		})
	}
}

func TestCadenceDetermination(t *testing.T) {
	secondTuesday := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	notSecondTuesday := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		desc             string
		img              *Image
		rolloutStartTime time.Time
		want             []string
	}{
		{
			desc:             "disabled cadence",
			img:              &Image{Cadence: "disabled"},
			rolloutStartTime: secondTuesday,
			want:             []string{"image cadence is \"disabled\""},
		},
		{
			desc:             "paused cadence",
			img:              &Image{Cadence: "paused"},
			rolloutStartTime: secondTuesday,
			want:             []string{"image cadence is \"paused\""},
		},
		{
			desc:             "monthly cadence on second Tuesday allowed",
			img:              &Image{Cadence: "monthly"},
			rolloutStartTime: secondTuesday,
			want:             nil,
		},
		{
			desc:             "monthly cadence on non-second Tuesday blocked",
			img:              &Image{Cadence: "monthly"},
			rolloutStartTime: notSecondTuesday,
			want:             []string{"publish cadence requires the second Tuesday of the month; current date is 2026-07-15"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := cadenceDetermination(tt.img, tt.rolloutStartTime)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: cadenceDetermination() got = %v, want %v", tt.desc, got, tt.want)
			}
		})
	}
}

func TestImageIsEOL(t *testing.T) {
	launchedTime := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	pastTime := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
	futureTime := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	eolTrue := true
	eolFalse := false

	tests := []struct {
		desc             string
		img              *Image
		rolloutStartTime time.Time
		want             bool
	}{
		{
			desc:             "EOL flag is true",
			img:              &Image{EOL: &eolTrue},
			rolloutStartTime: launchedTime,
			want:             true,
		},
		{
			desc:             "EOL flag is false, dates not EOL",
			img:              &Image{EOL: &eolFalse},
			rolloutStartTime: launchedTime,
			want:             false,
		},
		{
			desc:             "EOLDate in the past",
			img:              &Image{EOLDate: &pastTime},
			rolloutStartTime: launchedTime,
			want:             true,
		},
		{
			desc:             "EOLDate in the future",
			img:              &Image{EOLDate: &futureTime},
			rolloutStartTime: launchedTime,
			want:             false,
		},
		{
			desc:             "ObsoleteDate in the past",
			img:              &Image{ObsoleteDate: &pastTime},
			rolloutStartTime: launchedTime,
			want:             true,
		},
		{
			desc:             "ObsoleteDate in the future",
			img:              &Image{ObsoleteDate: &futureTime},
			rolloutStartTime: launchedTime,
			want:             false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := imageIsEOL(tt.img, tt.rolloutStartTime); got != tt.want {
				t.Errorf("%s: imageIsEOL() got = %v, want %v", tt.desc, got, tt.want)
			}
		})
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
					Name:   "us-central2-c",
					Region: "https://www.googleapis.com/compute/v1/projects/projectname/regions/us-central2",
				},
				{
					Name:   "us-central2-b",
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

func TestCreatePublishWithFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"no valid path", "", true},
		{"pass with valid path", "test_data/debian_13.publish.json", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreatePublish("", "", "", "", "", "", "", tt.path, map[string]string{}, map[string][]*computeAlpha.Image{})
			if err != nil && !tt.wantErr {
				t.Errorf("CreatePublish() called with path %s: got error %v", tt.path, err)
			}
			if tt.wantErr && err == nil {
				t.Errorf("CreatePublish() called with path %s: did not get expected error", tt.path)
			}
		})
	}
}

func TestCreatePublishWithTemplate(t *testing.T) {
	tests := []struct {
		name           string
		sourceVersion  string
		publishVersion string
		workProject    string
		publishProject string
		sourceGCS      string
		sourceProject  string
		ce             string
		template       string
		varMap         map[string]string
		imagesCache    map[string][]*computeAlpha.Image
		wantErr        bool
	}{
		{
			name:     "pass template",
			template: `{"WorkProject": "blah"}`,
			wantErr:  false,
		},
		{
			name:     "pass with invalid template",
			template: "{",
			wantErr:  true,
		},
		{
			name:           "pass all parameters invalid expire",
			sourceVersion:  "sv",
			publishVersion: "pv",
			workProject:    "wp",
			publishProject: "pp",
			sourceGCS:      "gcs",
			sourceProject:  "sp",
			ce:             "ce",
			template:       `{"Name": "test-publish", "DeleteAfter": "invalid-expire"}`,
			wantErr:        true,
		},
		{
			name:           "pass all parameters success",
			sourceVersion:  "sv",
			publishVersion: "pv",
			workProject:    "wp",
			publishProject: "pp",
			sourceGCS:      "gcs",
			sourceProject:  "sp",
			ce:             "ce",
			template:       `{"Name": "test-publish", "DeleteAfter": "24h"}`,
			varMap:         map[string]string{"foo": "bar"},
			imagesCache:    map[string][]*computeAlpha.Image{},
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			varMap := tt.varMap
			if varMap == nil {
				varMap = map[string]string{}
			}
			_, err := CreatePublishWithTemplate(
				tt.sourceVersion,
				tt.publishVersion,
				tt.workProject,
				tt.publishProject,
				tt.sourceGCS,
				tt.sourceProject,
				tt.ce,
				tt.template,
				varMap,
				tt.imagesCache,
			)
			if err != nil && !tt.wantErr {
				t.Errorf("CreatePublishWithTemplate() got unexpected error: %v", err)
			}
			if tt.wantErr && err == nil {
				t.Errorf("CreatePublishWithTemplate() template %s: expected error, got nil", tt.template)
			}
		})
	}
}

func TestCreateDirectPublish(t *testing.T) {
	got, err := CreateDirectPublish("sv", "pv", &Publish{
		Name:            "image-prefix",
		WorkProject:     "work-project",
		SourceProject:   "source-project",
		PublishProject:  "publish-project",
		ComputeEndpoint: "https://compute.example.com/",
		DeleteAfter:     "24h",
		Images: []*Image{
			{
				Prefix:                             "image-prefix",
				Family:                             "image-family",
				Description:                        "image description",
				Architecture:                       "X86_64",
				Licenses:                           []string{"license-a", "license-b"},
				GuestOsFeatures:                    []string{"UEFI_COMPATIBLE", "GVNIC"},
				Labels:                             map[string]string{"public-image": "true", "os": "test"},
				IgnoreLicenseValidationIfForbidden: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateDirectPublish() got error %v", err)
	}
	want := &Publish{
		Name:            "image-prefix",
		WorkProject:     "work-project",
		SourceProject:   "source-project",
		PublishProject:  "publish-project",
		ComputeEndpoint: "https://compute.example.com/",
		DeleteAfter:     "24h",
		sourceVersion:   "sv",
		publishVersion:  "pv",
		imagesCache:     make(map[string][]*computeAlpha.Image),
		Images: []*Image{
			{
				Prefix:                             "image-prefix",
				Family:                             "image-family",
				Description:                        "image description",
				Architecture:                       "X86_64",
				Licenses:                           []string{"license-a", "license-b"},
				GuestOsFeatures:                    []string{"UEFI_COMPATIBLE", "GVNIC"},
				Labels:                             map[string]string{"public-image": "true", "os": "test"},
				IgnoreLicenseValidationIfForbidden: true,
			},
		},
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(Publish{}), cmpopts.IgnoreFields(Publish{}, "expiryDate")); diff != "" {
		t.Errorf("CreateDirectPublish() mismatch (-want +got):\n%s", diff)
	}
}

func TestCreateDirectPublishDefaultsPublishVersion(t *testing.T) {
	got, err := CreateDirectPublish("sv", "", &Publish{
		WorkProject:    "work-project",
		SourceProject:  "source-project",
		PublishProject: "publish-project",
		Images: []*Image{
			{
				Prefix: "image-prefix",
				Family: "image-family",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateDirectPublish() got error %v", err)
	}
	if got.publishVersion != "sv" {
		t.Errorf("CreateDirectPublish() publishVersion = %q, want sv", got.publishVersion)
	}
}

func TestCreateDirectPublishErrors(t *testing.T) {
	tests := []struct {
		name    string
		publish *Publish
	}{
		{"nil publish", nil},
		{"empty publish project", &Publish{SourceProject: "source-project", Images: []*Image{{Prefix: "prefix", Family: "family"}}}},
		{"missing source project and source GCS path", &Publish{PublishProject: "pub-project", Images: []*Image{{Prefix: "prefix", Family: "family"}}}},
		{"both source project and source GCS path set", &Publish{PublishProject: "pub-project", SourceProject: "source-project", SourceGCSPath: "gs://bucket/path", Images: []*Image{{Prefix: "prefix", Family: "family"}}}},
		{"no images", &Publish{PublishProject: "pub-project", SourceProject: "source-project", Images: []*Image{}}},
		{"multiple images", &Publish{PublishProject: "pub-project", SourceProject: "source-project", Images: []*Image{{Prefix: "prefix", Family: "family"}, {Prefix: "prefix", Family: "family"}}}},
		{"empty image prefix", &Publish{PublishProject: "pub-project", SourceProject: "source-project", Images: []*Image{{Family: "family"}}}},
		{"empty image family", &Publish{PublishProject: "pub-project", SourceProject: "source-project", Images: []*Image{{Prefix: "prefix"}}}},
		{"empty work project", &Publish{PublishProject: "pub-project", SourceProject: "source-project", Images: []*Image{{Prefix: "prefix", Family: "family"}}}},
		{"invalid delete after", &Publish{PublishProject: "pub-project", SourceProject: "source-project", WorkProject: "work-project", DeleteAfter: "invalid-duration", Images: []*Image{{Prefix: "prefix", Family: "family"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateDirectPublish("sv", "pv", tt.publish)
			if err == nil {
				t.Errorf("CreateDirectPublish() with %s: expected error, got nil", tt.name)
			}
		})
	}
}

func TestCreateDirectPublishSetsEmptyNameFromImagePrefix(t *testing.T) {
	p := directPublishForNameTest("")
	wantName := p.Images[0].Prefix
	got, err := CreateDirectPublish("sv", "pv", p)
	if err != nil {
		t.Fatalf("CreateDirectPublish() got unexpected error: %v", err)
	}
	if got.Name != wantName {
		t.Errorf("CreateDirectPublish().Name got = %q, want %q", got.Name, wantName)
	}
}

func TestCreateDirectPublishPreservesExistingName(t *testing.T) {
	const wantName = "existing-name"
	p := directPublishForNameTest(wantName)
	got, err := CreateDirectPublish("sv", "pv", p)
	if err != nil {
		t.Fatalf("CreateDirectPublish() got unexpected error: %v", err)
	}
	if got.Name != wantName {
		t.Errorf("CreateDirectPublish().Name got = %q, want %q", got.Name, wantName)
	}
}

func directPublishForNameTest(name string) *Publish {
	return &Publish{
		Name:           name,
		WorkProject:    "work-project",
		SourceProject:  "source-project",
		PublishProject: "publish-project",
		Images: []*Image{
			{
				Prefix: "test-prefix",
				Family: "test-family",
			},
		},
	}
}

func TestPublishCreateWorkflows(t *testing.T) {
	ctx := context.Background()

	var mockImages []byte

	// Set up the local HTTP test server to mock the GCE API responses.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/zones") {
			project := "project"
			parts := strings.Split(r.URL.Path, "/")
			for i, part := range parts {
				if part == "projects" && i+1 < len(parts) {
					project = parts[i+1]
					break
				}
			}
			w.Write([]byte(fmt.Sprintf(`{
				"items": [
					{"name": "us-central1-a", "region": "https://www.googleapis.com/compute/v1/projects/%[1]s/regions/us-central1"},
					{"name": "us-central1-b", "region": "https://www.googleapis.com/compute/v1/projects/%[1]s/regions/us-central1"}
				]
			}`, project)))
			return
		}
		if strings.Contains(r.URL.Path, "/images") {
			w.Write(mockImages)
			return
		}
		w.Write([]byte(`{"status": "DONE"}`))
	}))
	defer ts.Close()

	clientOpts := []option.ClientOption{
		option.WithoutAuthentication(),
		option.WithEndpoint(ts.URL),
		option.WithHTTPClient(ts.Client()),
	}

	launchedTrue := true
	launchedFalse := false

	tests := []struct {
		name               string
		publish            *Publish
		sourceVersion      string
		publishVersion     string
		varMap             map[string]string
		filterRegex        *regexp.Regexp
		rollback           bool
		skipDup            bool
		replace            bool
		noRoot             bool
		rbObsolete         bool
		force              bool
		oauth              string
		setup              func(*Publish)
		wantErr            bool
		errContains        string
		wantWorkflows      int
		wantName           string
		wantProject        string
		mockImagesJSON     string
		rolloutStartTime   time.Time
		rolloutRate        int
		useCreateWorkflows bool
	}{
		{
			name:           "normal publish",
			sourceVersion:  "new",
			publishVersion: "new",
			varMap:         map[string]string{"k": "v"},
			oauth:          "oauth-path",
			wantWorkflows:  1,
			mockImagesJSON: `{"items": [
				{"name": "test-old", "family": "test-family", "creationTimestamp": "2026-07-10T00:00:00Z"},
				{"name": "test-older", "family": "test-family", "creationTimestamp": "2026-07-09T00:00:00Z", "deprecated": {"state": "DEPRECATED"}}
			]}`,
		},
		{
			name:           "rollback case",
			sourceVersion:  "old",
			publishVersion: "old",
			varMap:         map[string]string{"k": "v"},
			rollback:       true,
			oauth:          "oauth-path",
			wantWorkflows:  1,
			mockImagesJSON: `{"items": [
				{"name": "test-old", "family": "test-family", "creationTimestamp": "2026-07-10T00:00:00Z"},
				{"name": "test-older", "family": "test-family", "creationTimestamp": "2026-07-09T00:00:00Z", "deprecated": {"state": "DEPRECATED"}}
			]}`,
		},
		{
			name:           "regex doesn't match",
			sourceVersion:  "old",
			publishVersion: "old",
			filterRegex:    regexp.MustCompile("^not-matching$"),
			wantWorkflows:  0,
			mockImagesJSON: `{"items": [
				{"name": "test-old", "family": "test-family", "creationTimestamp": "2026-07-10T00:00:00Z"},
				{"name": "test-older", "family": "test-family", "creationTimestamp": "2026-07-09T00:00:00Z", "deprecated": {"state": "DEPRECATED"}}
			]}`,
		},
		{
			name:           "normal publish with duplicate image (fails since skip/replace are false)",
			sourceVersion:  "old",
			publishVersion: "old",
			wantErr:        true,
			mockImagesJSON: `{"items": [
				{"name": "test-old", "family": "test-family", "creationTimestamp": "2026-07-10T00:00:00Z"},
				{"name": "test-older", "family": "test-family", "creationTimestamp": "2026-07-09T00:00:00Z", "deprecated": {"state": "DEPRECATED"}}
			]}`,
		},
		{
			name:           "normal publish with duplicate image & skipDuplicates=true (succeeds, returns 0 steps but valid workflow)",
			sourceVersion:  "old",
			publishVersion: "old",
			skipDup:        true,
			wantWorkflows:  0,
			mockImagesJSON: `{"items": [
				{"name": "test-old", "family": "test-family", "creationTimestamp": "2026-07-10T00:00:00Z"},
				{"name": "test-older", "family": "test-family", "creationTimestamp": "2026-07-09T00:00:00Z", "deprecated": {"state": "DEPRECATED"}}
			]}`,
		},
		{
			name:           "normal publish with duplicate image & replace=true (succeeds)",
			sourceVersion:  "old",
			publishVersion: "old",
			replace:        true,
			wantWorkflows:  1,
			mockImagesJSON: `{"items": [
				{"name": "test-old", "family": "test-family", "creationTimestamp": "2026-07-10T00:00:00Z"},
				{"name": "test-older", "family": "test-family", "creationTimestamp": "2026-07-09T00:00:00Z", "deprecated": {"state": "DEPRECATED"}}
			]}`,
		},
		{
			name:           "duplicate image & replace=true & skipDuplicates=true conflict (fails)",
			sourceVersion:  "old",
			publishVersion: "old",
			skipDup:        true,
			replace:        true,
			wantErr:        true,
			mockImagesJSON: `{"items": [
				{"name": "test-old", "family": "test-family", "creationTimestamp": "2026-07-10T00:00:00Z"},
				{"name": "test-older", "family": "test-family", "creationTimestamp": "2026-07-09T00:00:00Z", "deprecated": {"state": "DEPRECATED"}}
			]}`,
		},
		{
			name:           "rollback obscuring (rbObsolete = true)",
			sourceVersion:  "old",
			publishVersion: "old",
			rollback:       true,
			rbObsolete:     true,
			wantWorkflows:  1,
			mockImagesJSON: `{"items": [
				{"name": "test-old", "family": "test-family", "creationTimestamp": "2026-07-10T00:00:00Z"},
				{"name": "test-older", "family": "test-family", "creationTimestamp": "2026-07-09T00:00:00Z", "deprecated": {"state": "DEPRECATED"}}
			]}`,
		},
		{
			name:               "CreateWorkflows (wrapper function) and print list coverages",
			sourceVersion:      "new",
			publishVersion:     "new",
			useCreateWorkflows: true,
			setup: func(p *Publish) {
				p.toCreate = []string{"dummy-create"}
				p.toDeprecate = []string{"dummy-deprecate"}
				p.toObsolete = []string{"dummy-obsolete"}
				p.toUndeprecate = []string{"dummy-undeprecate"}
				p.toDelete = []string{"dummy-delete"}
			},
			wantWorkflows: 1,
			mockImagesJSON: `{"items": [
				{"name": "test-old", "family": "test-family", "creationTimestamp": "2026-07-10T00:00:00Z"},
				{"name": "test-older", "family": "test-family", "creationTimestamp": "2026-07-09T00:00:00Z", "deprecated": {"state": "DEPRECATED"}}
			]}`,
		},
		{
			name: "skip workflow creation when regex does not match image prefix",
			publish: &Publish{
				Name:           "test-publish",
				WorkProject:    "work-project",
				SourceProject:  "source-project",
				PublishProject: "pub-project",
				Images: []*Image{
					{Prefix: "ubuntu-2204", Family: "ubuntu"},
				},
			},
			filterRegex:    regexp.MustCompile("^debian"),
			wantWorkflows:  0,
			mockImagesJSON: `{"items": [{"name": "debian-13-sv", "family": "debian-13"}]}`,
			rolloutRate:    60,
		},
		{
			name: "publish determination blocks when env=prod, not launched and force=false",
			publish: &Publish{
				Name:           "test-publish",
				WorkProject:    "work-project",
				SourceProject:  "source-project",
				PublishProject: "pub-project",
				sourceVersion:  "sv",
				publishVersion: "pv",
				Images: []*Image{
					{
						Prefix:   "debian-13",
						Family:   "debian-13",
						Launched: &launchedFalse,
					},
				},
			},
			varMap: map[string]string{
				"environment":                  "prod",
				"enable_publish_determination": "true",
			},
			force:          false,
			wantErr:        true,
			errContains:    "publish determination blocked",
			mockImagesJSON: `{"items": [{"name": "debian-13-sv", "family": "debian-13"}]}`,
			rolloutRate:    60,
		},
		{
			name: "publish determination bypassed when env=prod, not launched but force=true",
			publish: &Publish{
				Name:           "test-publish",
				WorkProject:    "work-project",
				SourceProject:  "source-project",
				PublishProject: "pub-project",
				sourceVersion:  "sv",
				publishVersion: "pv",
				imagesCache:    make(map[string][]*computeAlpha.Image),
				Images: []*Image{
					{
						Prefix:   "debian-13",
						Family:   "debian-13",
						Launched: &launchedFalse,
					},
				},
			},
			varMap: map[string]string{
				"environment":                  "prod",
				"enable_publish_determination": "true",
			},
			force:          true,
			wantWorkflows:  1,
			wantName:       "debian-13",
			wantProject:    "work-project",
			mockImagesJSON: `{"items": [{"name": "debian-13-sv", "family": "debian-13"}]}`,
			rolloutRate:    60,
		},
		{
			name: "publish determination skipped when rollback is true",
			publish: &Publish{
				Name:           "test-publish",
				WorkProject:    "work-project",
				SourceProject:  "source-project",
				PublishProject: "pub-project",
				sourceVersion:  "sv",
				publishVersion: "pv",
				imagesCache:    make(map[string][]*computeAlpha.Image),
				Images: []*Image{
					{
						Prefix:   "debian-13",
						Family:   "debian-13",
						Launched: nil,
					},
				},
			},
			varMap: map[string]string{
				"environment":                  "prod",
				"enable_publish_determination": "true",
			},
			rollback:      true,
			force:         false,
			wantWorkflows: 1,
			wantName:      "debian-13",
			wantProject:   "work-project",
			mockImagesJSON: `{
				"items": [
					{"name": "debian-13-pv", "family": "debian-13"},
					{"name": "debian-13-sv", "family": "debian-13", "deprecated": {"state": "DEPRECATED"}}
				]
			}`,
			rolloutRate: 60,
		},
		{
			name: "normal case success",
			publish: &Publish{
				Name:           "test-publish",
				WorkProject:    "work-project",
				SourceProject:  "source-project",
				PublishProject: "pub-project",
				sourceVersion:  "sv",
				publishVersion: "pv",
				imagesCache:    make(map[string][]*computeAlpha.Image),
				Images: []*Image{
					{
						Prefix:   "debian-13",
						Family:   "debian-13",
						Launched: &launchedTrue,
					},
				},
			},
			varMap: map[string]string{
				"environment":                  "prod",
				"enable_publish_determination": "true",
			},
			force:          false,
			wantWorkflows:  1,
			wantName:       "debian-13",
			wantProject:    "work-project",
			mockImagesJSON: `{"items": [{"name": "debian-13-sv", "family": "debian-13"}]}`,
			rolloutRate:    60,
		},
		{
			name: "publish determination blocks when cadence second Tuesday is not met in Pacific time",
			publish: &Publish{
				Name:           "test-publish",
				WorkProject:    "work-project",
				SourceProject:  "source-project",
				PublishProject: "pub-project",
				sourceVersion:  "sv",
				publishVersion: "pv",
				imagesCache:    make(map[string][]*computeAlpha.Image),
				Images: []*Image{
					{
						Prefix:   "debian-13",
						Family:   "debian-13",
						Launched: &launchedTrue,
						Cadence:  "monthly",
					},
				},
			},
			varMap: map[string]string{
				"environment":                  "prod",
				"enable_publish_determination": "true",
			},
			rolloutStartTime: time.Date(2026, time.July, 14, 2, 0, 0, 0, time.UTC),
			wantErr:          true,
			errContains:      "publish determination blocked",
			mockImagesJSON:   `{"items": [{"name": "debian-13-sv", "family": "debian-13"}]}`,
			rolloutRate:      60,
		},
		{
			name: "publish determination succeeds when cadence second Tuesday is met in Pacific time",
			publish: &Publish{
				Name:           "test-publish",
				WorkProject:    "work-project",
				SourceProject:  "source-project",
				PublishProject: "pub-project",
				sourceVersion:  "sv",
				publishVersion: "pv",
				imagesCache:    make(map[string][]*computeAlpha.Image),
				Images: []*Image{
					{
						Prefix:   "debian-13",
						Family:   "debian-13",
						Launched: &launchedTrue,
						Cadence:  "monthly",
					},
				},
			},
			varMap: map[string]string{
				"environment":                  "prod",
				"enable_publish_determination": "true",
			},
			rolloutStartTime: time.Date(2026, time.July, 15, 2, 0, 0, 0, time.UTC),
			wantWorkflows:    1,
			wantName:         "debian-13",
			wantProject:      "work-project",
			mockImagesJSON:   `{"items": [{"name": "debian-13-sv", "family": "debian-13"}]}`,
			rolloutRate:      60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockImagesJSON != "" {
				mockImages = []byte(tt.mockImagesJSON)
			} else {
				mockImages = []byte(`{}`)
			}

			p := tt.publish
			if p == nil {
				p = &Publish{
					Name:            "test-publish",
					WorkProject:     "work-project",
					PublishProject:  "pub-project",
					SourceProject:   "src-project",
					DeleteAfter:     "1h",
					ComputeEndpoint: ts.URL,
					Images: []*Image{
						{
							Prefix: "test",
							Family: "test-family",
							Labels: map[string]string{"foo": "bar"},
						},
					},
					imagesCache: make(map[string][]*computeAlpha.Image),
				}
				p.SetVersions(tt.sourceVersion, tt.publishVersion)
			} else {
				p.ComputeEndpoint = ts.URL
				if p.imagesCache == nil {
					p.imagesCache = make(map[string][]*computeAlpha.Image)
				}
			}

			if tt.setup != nil {
				tt.setup(p)
			}

			testTime := time.Now()
			if !tt.rolloutStartTime.IsZero() {
				testTime = tt.rolloutStartTime
			}

			rate := 1
			if tt.rolloutRate > 0 {
				rate = tt.rolloutRate
			}

			var ws []*daisy.Workflow
			var err error
			if tt.useCreateWorkflows {
				ws, err = p.CreateWorkflows(ctx, tt.varMap, tt.filterRegex, tt.rollback, tt.skipDup, tt.replace, tt.noRoot, tt.oauth, testTime, rate, clientOpts...)
			} else {
				ws, err = p.PublishCreateWorkflows(ctx, tt.varMap, tt.filterRegex, tt.rollback, tt.skipDup, tt.replace, tt.noRoot, tt.rbObsolete, tt.oauth, testTime, rate, tt.force, clientOpts...)
			}

			if (err != nil) != tt.wantErr {
				t.Fatalf("got err: %v, wantErr: %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}
			if len(ws) != tt.wantWorkflows {
				t.Errorf("Expected %d workflow, got %d", tt.wantWorkflows, len(ws))
			}
			if tt.wantWorkflows > 0 && len(ws) > 0 {
				w := ws[0]
				if tt.wantName != "" && w.Name != tt.wantName {
					t.Errorf("workflow Name = %q, want %q", w.Name, tt.wantName)
				}
				if tt.wantProject != "" && w.Project != tt.wantProject {
					t.Errorf("workflow Project = %q, want %q", w.Project, tt.wantProject)
				}
			}
		})
	}
}
