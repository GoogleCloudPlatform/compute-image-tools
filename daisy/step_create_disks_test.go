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
	"google.golang.org/api/compute/v1"
)

func TestCreateDisksRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.images.m = map[string]*Resource{"i1": {RealName: "i1", link: "i1link"}}

	e := Errf("error")
	quotaExceededErr := Errf("Some error\nCode: QUOTA_EXCEEDED\nMessage: some message.")
	tests := []struct {
		desc      string
		d         compute.Disk
		wantD     compute.Disk
		clientErr error
		wantErr   DError
	}{
		{"blank case", compute.Disk{}, compute.Disk{}, nil, nil},
		{"resolve source image case", compute.Disk{SourceImage: "i1"}, compute.Disk{SourceImage: "i1link"}, nil, nil},
		{"client error case", compute.Disk{}, compute.Disk{}, e, e},
		{"not fallback to pd-standard", compute.Disk{Type: "prefix/pd-ssd"}, compute.Disk{Type: "prefix/pd-ssd"}, e, e},
		{"not fallback to pd-standard without retry hook", compute.Disk{Type: "prefix/pd-ssd"}, compute.Disk{Type: "prefix/pd-ssd"}, quotaExceededErr, quotaExceededErr},
	}
	for _, tt := range tests {
		var gotD compute.Disk
		fake := func(_, _ string, d *compute.Disk) error {
			gotD = *d
			return tt.clientErr
		}
		w.ComputeClient = &daisyCompute.TestClient{CreateDiskFn: fake}
		cds := &CreateDisks{{Disk: tt.d}}
		if err := cds.run(ctx, s); err != tt.wantErr {
			t.Errorf("%s: unexpected error returned, got: %v, want: %v", tt.desc, err, tt.wantErr)
		}
		if diffRes := diff(gotD, tt.wantD, 0); diffRes != "" {
			t.Errorf("%s: client got incorrect disk, got: %v, want: %v", tt.desc, gotD, tt.wantD)
		}
	}
}
