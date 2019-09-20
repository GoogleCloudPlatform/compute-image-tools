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
	"reflect"
	"testing"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

func TestCreateDisksRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.images.m = map[string]*Resource{"i1": {RealName: "i1", link: "i1link"}}

	e := Errf("error")
	tests := []struct {
		desc      string
		inputDisk Disk
		wantDisk  compute.Disk
		clientErr error
		wantErr   DError
	}{
		{
			desc:      "generic disks shouldn't be modified",
			inputDisk: Disk{Disk: compute.Disk{}},
			wantDisk:  compute.Disk{},
		},
		{
			desc:      "replace source image's name with its link",
			inputDisk: Disk{Disk: compute.Disk{SourceImage: "i1"}},
			wantDisk:  compute.Disk{SourceImage: "i1link"},
		},
		{
			desc: "if disk is marked as Windows, add WINDOWS guest os feature",
			inputDisk: Disk{
				Disk: compute.Disk{
					SourceImage:     "i1",
					GuestOsFeatures: featuresOf("UEFI_COMPATIBLE"),
				},
				IsWindows: "true"},
			wantDisk: compute.Disk{
				SourceImage:     "i1link",
				GuestOsFeatures: featuresOf("UEFI_COMPATIBLE", "WINDOWS"),
			},
		},
		{
			desc:      "propagate errors unchanged",
			clientErr: e,
			wantErr:   e,
		},
	}
	for _, tt := range tests {
		var gotD compute.Disk
		fake := func(_, _ string, d *compute.Disk) error { gotD = *d; return tt.clientErr }
		w.ComputeClient = &daisyCompute.TestClient{CreateDiskFn: fake}
		cds := &CreateDisks{&tt.inputDisk}
		if err := cds.run(ctx, s); err != tt.wantErr {
			t.Errorf("%s: unexpected error returned, got: %v, want: %v", tt.desc, err, tt.wantErr)
		}
		if diffRes := diff(gotD, tt.wantDisk, 0); diffRes != "" {
			t.Errorf("%s: client got incorrect disk, got: %v, want: %v", tt.desc, gotD, tt.wantDisk)
		}
	}
}

func TestCombineGuestOSFeatures(t *testing.T) {

	tests := []struct {
		currentFeatures    []*compute.GuestOsFeature
		additionalFeatures []string
		want               []*compute.GuestOsFeature
	}{
		{
			currentFeatures:    featuresOf(),
			additionalFeatures: []string{},
			want:               featuresOf(),
		},
		{
			currentFeatures:    featuresOf("WINDOWS"),
			additionalFeatures: []string{},
			want:               featuresOf("WINDOWS"),
		},
		{
			currentFeatures:    featuresOf(),
			additionalFeatures: []string{"WINDOWS"},
			want:               featuresOf("WINDOWS"),
		},
		{
			currentFeatures:    featuresOf("WINDOWS"),
			additionalFeatures: []string{"WINDOWS"},
			want:               featuresOf("WINDOWS"),
		},
		{
			currentFeatures:    featuresOf("SECURE_BOOT"),
			additionalFeatures: []string{"WINDOWS"},
			want:               featuresOf("SECURE_BOOT", "WINDOWS"),
		},
		{
			currentFeatures:    featuresOf("SECURE_BOOT", "UEFI_COMPATIBLE"),
			additionalFeatures: []string{"WINDOWS", "UEFI_COMPATIBLE"},
			want:               featuresOf("SECURE_BOOT", "UEFI_COMPATIBLE", "WINDOWS"),
		},
	}

	for _, test := range tests {
		got := combineGuestOSFeatures(test.currentFeatures, test.additionalFeatures...)

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("combineGuestOSFeatures(%v, %v) = %v, want %v",
				test.currentFeatures, test.additionalFeatures, got, test.want)
		}
	}
}

func featuresOf(features ...string) []*compute.GuestOsFeature {
	ret := make([]*compute.GuestOsFeature, 0)
	for _, feature := range features {
		ret = append(ret, &compute.GuestOsFeature{
			Type: feature,
		})
	}
	return ret
}
