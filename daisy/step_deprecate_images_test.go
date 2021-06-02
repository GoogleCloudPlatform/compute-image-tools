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

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/compute/v1"
)

func TestDeprecateImagesPopulate(t *testing.T) {
	w := testWorkflow()
	s, _ := w.NewStep("s")
	s.DeprecateImages = &DeprecateImages{
		&DeprecateImage{
			Image: testImage,
		},
		&DeprecateImage{
			Image: "test-image",
			DeprecationStatus: compute.DeprecationStatus{
				State: "DEPRECATED",
			},
			Project: "foo",
		},
	}

	if err := (s.DeprecateImages).populate(context.Background(), s); err != nil {
		t.Error("err should be nil")
	}

	want := &DeprecateImages{
		&DeprecateImage{
			Image:   testImage,
			Project: testProject,
		},
		&DeprecateImage{
			Image: "test-image",
			DeprecationStatus: compute.DeprecationStatus{
				State: "DEPRECATED",
			},
			Project: "foo",
		},
	}
	if diffRes := diff(s.DeprecateImages, want, 0); diffRes != "" {
		t.Errorf("DeprecateImages not populated as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestDeprecateImagesValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()

	iCreator := &Step{name: "iCreator", w: w}
	w.Steps["iCreator"] = iCreator
	w.images.m = map[string]*Resource{"i1": {creator: iCreator}}

	tests := []struct {
		desc      string
		di        *DeprecateImage
		shouldErr bool
	}{
		{
			"DEPRECATED case",
			&DeprecateImage{Image: "i1", Project: testProject, DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED"}},
			false,
		},
		{
			"DEPRECATED case not in workflow",
			&DeprecateImage{Image: testImage, Project: testProject, DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED"}},
			false,
		},
		{
			"unDEPRECATED case",
			&DeprecateImage{Image: "i1", Project: testProject, DeprecationStatus: compute.DeprecationStatus{State: "", ForceSendFields: []string{"State"}}},
			false,
		},
		{
			"bad state case",
			&DeprecateImage{Image: "i1", Project: testProject, DeprecationStatus: compute.DeprecationStatus{State: "BAD"}},
			true,
		},
		{
			"bad image case",
			&DeprecateImage{Image: "bad", Project: testProject},
			true,
		},
		{
			"bad project case",
			&DeprecateImage{Image: "i1", Project: "bad"},
			true,
		},
		{
			"alpha DEPRECATED case",
			&DeprecateImage{Image: "i1", Project: testProject, DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "DEPRECATED"}},
			false,
		},
		{
			"alpha unDEPRECATED case",
			&DeprecateImage{Image: "i1", Project: testProject, DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "ACTIVE"}},
			false,
		},
		{
			"alpha bad case",
			&DeprecateImage{Image: "i1", Project: testProject, DeprecationStatusAlpha: computeAlpha.DeprecationStatus{State: "BAD"}},
			true,
		},
	}
	for _, tt := range tests {
		w.Steps[tt.desc] = &Step{name: tt.desc, w: w, DeprecateImages: &DeprecateImages{tt.di}}
		w.Dependencies[tt.desc] = []string{"iCreator"}
		s := w.Steps[tt.desc]
		err := s.DeprecateImages.validate(ctx, s)
		if err != nil {
			if tt.shouldErr {
				continue
			}
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
		if err == nil && tt.shouldErr {
			t.Errorf("%s: did not return an error as expected", tt.desc)
		}
	}
}

func TestDeprecateImagesRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.images.m = map[string]*Resource{"i1": {RealName: "i1", link: "i1link"}}

	e := Errf("error")
	tests := []struct {
		desc      string
		di        *DeprecateImage
		clientErr error
		wantErr   DError
	}{
		{"DEPRECATED case", &DeprecateImage{Image: "i1", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED"}}, nil, nil},
		{"client error case", &DeprecateImage{Image: "i1", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED"}}, e, e},
	}
	for _, tt := range tests {
		fake := func(_, _ string, ds *compute.DeprecationStatus) error { return tt.clientErr }
		w.ComputeClient = &daisyCompute.TestClient{DeprecateImageFn: fake}

		dis := &DeprecateImages{tt.di}
		if err := dis.run(ctx, s); err != tt.wantErr {
			t.Errorf("%s: unexpected error returned, got: %v, want: %v", tt.desc, err, tt.wantErr)
		}
	}
}
