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
	"testing"
	"time"

	computeAlpha "google.golang.org/api/compute/v0.alpha"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

func TestCreateImagesValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.disks.m = map[string]*Resource{testDisk: {RealName: w.genName(testDisk), link: testDisk}}
	lic := fmt.Sprintf("projects/%s/global/licenses/license", p403)

	tests := []struct {
		desc      string
		ci        *Image
		shouldErr bool
	}{
		{desc: "403 listing licenses",
			ci:        &Image{ImageBase: ImageBase{Resource: Resource{Project: testProject}}, Image: compute.Image{Name: "test1", SourceDisk: testDisk, Licenses: []string{lic}}},
			shouldErr: true,
		},
		{desc: "403 listing licenses, IgnoreLicenseValidationIfForbidden set",
			ci:        &Image{ImageBase: ImageBase{Resource: Resource{Project: testProject}, IgnoreLicenseValidationIfForbidden: true}, Image: compute.Image{Name: "test2", SourceDisk: testDisk, Licenses: []string{lic}}},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		cis := &CreateImages{Images: []*Image{tt.ci}}
		if err := cis.populate(ctx, s); err != nil {
			t.Errorf("%s: populate error: %v", tt.desc, err)
		}
		if err := cis.validate(ctx, s); err == nil && tt.shouldErr {
			t.Errorf("%s: should have returned an error, but didn't", tt.desc)
		} else if err != nil && !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestCreateImagesRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.disks.m = map[string]*Resource{testDisk: {RealName: w.genName(testDisk), link: testDisk}}
	w.Sources = map[string]string{"file": "gs://some/path"}

	tests := []struct {
		desc      string
		ci        *Image
		cia       *ImageAlpha
		cib       *ImageBeta
		shouldErr bool
	}{
		{desc: "source disk with overwrite case", ci: &Image{ImageBase: ImageBase{Resource: Resource{Project: testProject}, OverWrite: true}, Image: compute.Image{Name: testImage, SourceDisk: testDisk}}, shouldErr: false},
		{desc: "raw image case", ci: &Image{ImageBase: ImageBase{Resource: Resource{Project: testProject}}, Image: compute.Image{Name: testImage, RawDisk: &compute.ImageRawDisk{Source: "gs://bucket/object"}}}, shouldErr: false},
		{desc: "bad disk case", ci: &Image{ImageBase: ImageBase{Resource: Resource{Project: testProject}}, Image: compute.Image{Name: testImage, SourceDisk: "bad"}}, shouldErr: true},
		{desc: "bad overwrite case", ci: &Image{ImageBase: ImageBase{Resource: Resource{Project: testProject}, OverWrite: true}, Image: compute.Image{Name: "bad", SourceDisk: testDisk}}, shouldErr: true},
		{
			desc: "image rolloutOverride using alpha API",
			cia: &ImageAlpha{
				ImageBase: ImageBase{Resource: Resource{Project: testProject}, OverWrite: true},
				Image: computeAlpha.Image{
					Name:       "alpha",
					SourceDisk: testDisk,
					RolloutOverride: &computeAlpha.RolloutPolicy{
						DefaultRolloutTime: "2021-04-02T23:23:59.851301Z",
					},
				},
			},
			shouldErr: false,
		},
		{
			desc:      "image location using beta API",
			cib:       &ImageBeta{ImageBase: ImageBase{Resource: Resource{Project: testProject}, OverWrite: true}, Image: computeBeta.Image{Name: "beta", SourceDisk: testDisk, StorageLocations: []string{"eu"}}},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		var cis *CreateImages
		if tt.cia != nil {
			cis = &CreateImages{ImagesAlpha: []*ImageAlpha{tt.cia}}
		} else if tt.cib != nil {
			cis = &CreateImages{ImagesBeta: []*ImageBeta{tt.cib}}
		} else {
			cis = &CreateImages{Images: []*Image{tt.ci}}
		}

		if err := cis.run(ctx, s); err == nil && tt.shouldErr {
			t.Errorf("%s: should have returned an error, but didn't", tt.desc)
		} else if err != nil && !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestImageUsesAlphaFeaturesTrue(t *testing.T) {
	tests := []struct {
		desc      string
		cia       []*ImageAlpha
		wantResult bool
	}{
		{
			desc: "Use AlphaAPI due to RolloutOverride.",
			cia: []*ImageAlpha{
				{
					Image: computeAlpha.Image{
						RolloutOverride: &computeAlpha.RolloutPolicy{
							DefaultRolloutTime: time.Now().Format(time.RFC3339),
							LocationRolloutPolicies: map[string]string{
								"zones/one": time.Now().Add(time.Hour).Format(time.RFC3339),
							},
						},
					},
				},
			},
			wantResult: true,
		},
		{
			desc: "Use AlphaAPI due to Deprecated.StateOverride.",
			cia: []*ImageAlpha{
				{
					Image: computeAlpha.Image{
						Deprecated: &computeAlpha.DeprecationStatus{
							State: "OBSOLETE",
							StateOverride: &computeAlpha.RolloutPolicy{
								DefaultRolloutTime: time.Now().Format(time.RFC3339),
								LocationRolloutPolicies: map[string]string{
									"zones/one": time.Now().Add(time.Hour).Format(time.RFC3339),
								},
							},
						},
					},
				},
			},
			wantResult: true,
		},
		{
			desc: "Do not use AlphaAPI, empty",
			cia: []*ImageAlpha{{Image: computeAlpha.Image{}}},
			wantResult: false,
		},
		{
			desc: "Do not use AlphaAPI, deprecated with no StateOverride.",
			cia: []*ImageAlpha{{Image: computeAlpha.Image{
				Deprecated: &computeAlpha.DeprecationStatus{
					State: "OBSOLETE",
				},
			}}},
			wantResult: false,
		},
		{
			desc: "Do not use AlphaAPI, deprecated with StateOverride = nil.",
			cia: []*ImageAlpha{{Image: computeAlpha.Image{
				Deprecated: &computeAlpha.DeprecationStatus{
					State: "OBSOLETE",
					StateOverride: nil,
				},
			}}},
			wantResult: false,
		},
		{
			desc: "Do not use AlphaAPI, deprecated and StateOverride present with no value.",
			cia: []*ImageAlpha{{Image: computeAlpha.Image{
				Deprecated: &computeAlpha.DeprecationStatus{
					State: "OBSOLETE",
					StateOverride: &computeAlpha.RolloutPolicy{},
				},
			}}},
			wantResult: false,
		},
		{
			desc: "Do not use AlphaAPI, Deprecated is nil.",
			cia: []*ImageAlpha{{Image: computeAlpha.Image{
				Deprecated: nil,
			}}},
			wantResult: false,
		},
		{
			desc: "Do not use AlphaAPI, RolloutOverride present with all default values.",
			cia: []*ImageAlpha{{Image: computeAlpha.Image{
				RolloutOverride: &computeAlpha.RolloutPolicy{},
			}}},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		if result := imageUsesAlphaFeatures(tt.cia); result != tt.wantResult {
			t.Errorf("%s: imageUsesAlphaFeatures() got %t, want %t", tt.desc, result, tt.wantResult)
		}
	}
}
