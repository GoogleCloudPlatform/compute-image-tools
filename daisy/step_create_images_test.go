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

	computeAlpha "google.golang.org/api/compute/v0.alpha"
)

func TestCreateImagesRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.disks.m = map[string]*Resource{testDisk: {RealName: w.genName(testDisk), link: testDisk}}
	w.Sources = map[string]string{"file": "gs://some/path"}

	tests := []struct {
		desc      string
		ci        *Image
		shouldErr bool
	}{
		{"source disk with overwrite case", &Image{Resource: Resource{Project: testProject}, Image: computeAlpha.Image{Name: testImage, SourceDisk: testDisk}, OverWrite: true}, false},
		{"raw image case", &Image{Resource: Resource{Project: testProject}, Image: computeAlpha.Image{Name: testImage, RawDisk: &computeAlpha.ImageRawDisk{Source: "gs://bucket/object"}}}, false},
		{"bad disk case", &Image{Resource: Resource{Project: testProject}, Image: computeAlpha.Image{Name: testImage, SourceDisk: "bad"}}, true},
		{"bad overwrite case", &Image{Resource: Resource{Project: testProject}, Image: computeAlpha.Image{Name: "bad", SourceDisk: testDisk}, OverWrite: true}, true},
	}

	for _, tt := range tests {
		cis := &CreateImages{tt.ci}
		if err := cis.run(ctx, s); err == nil && tt.shouldErr {
			t.Errorf("%s: should have returned an error, but didn't", tt.desc)
		} else if err != nil && !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}
