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

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

func TestResizeDisksPopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}

	tests := []struct {
		rds *ResizeDisks
	}{
		{&ResizeDisks{{Name: "disk1", DisksResizeRequest: compute.DisksResizeRequest{SizeGb: 100}}}},
		{&ResizeDisks{{Name: fmt.Sprintf("zones/%s/disks/%s", testZone, testDisk), DisksResizeRequest: compute.DisksResizeRequest{SizeGb: 100}}}},
	}
	for _, tt := range tests {
		if err := tt.rds.populate(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResizeDisksValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	sCreateDisk, _ := w.NewStep("step-create-disk")
	w.disks.m = map[string]*Resource{"disk1": {RealName: "disk1", link: "disk1link", creator: sCreateDisk}}
	cd := &CreateDisks{{Disk: compute.Disk{Name: "disk1"}}}
	cd.validate(ctx, sCreateDisk)
	cd.run(ctx, sCreateDisk)

	s, _ := w.NewStep("test")
	w.AddDependency(s, sCreateDisk)

	tests := []struct {
		desc    string
		rds     *ResizeDisks
		wantErr bool
	}{
		{"resize existing disk", &ResizeDisks{{Name: "disk1", DisksResizeRequest: compute.DisksResizeRequest{SizeGb: 100}}}, false},
		{"resize inexisting disk", &ResizeDisks{{Name: "foo", DisksResizeRequest: compute.DisksResizeRequest{SizeGb: 100}}}, true},
		{"resize invalid size", &ResizeDisks{{Name: "disk1", DisksResizeRequest: compute.DisksResizeRequest{SizeGb: -1}}}, true},
		{"resize no size", &ResizeDisks{{Name: "disk1"}}, true},
	}
	for _, tt := range tests {
		err := tt.rds.validate(ctx, s)
		if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
		if tt.wantErr && err == nil {
			t.Errorf("%s: expected error, got none", tt.desc)
		}
	}
}

func TestResizeDisksRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	sCreateDisk, _ := w.NewStep("step-create-disk")
	w.disks.m = map[string]*Resource{"disk1": {RealName: "disk1", link: "disk1link", creator: sCreateDisk}}
	cd := &CreateDisks{{Disk: compute.Disk{Name: "disk1", SizeGb: 10}}}
	cd.validate(ctx, sCreateDisk)
	cd.run(ctx, sCreateDisk)

	s, _ := w.NewStep("test")
	w.AddDependency(s, sCreateDisk)

	e := Errf("error")
	tests := []struct {
		desc      string
		rd        *ResizeDisks
		wantDrr   *compute.DisksResizeRequest
		clientErr error
		wantErr   DError
	}{
		{"blank case", &ResizeDisks{}, &compute.DisksResizeRequest{}, nil, nil},
		{"existing disk", &ResizeDisks{{Name: "disk1", DisksResizeRequest: compute.DisksResizeRequest{SizeGb: 10}}}, &compute.DisksResizeRequest{SizeGb: 10}, nil, nil},
		{"inexisting disk", &ResizeDisks{{Name: "foo", DisksResizeRequest: compute.DisksResizeRequest{SizeGb: 8}}}, &compute.DisksResizeRequest{SizeGb: 8}, e, e},
	}
	for _, tt := range tests {
		var gotDrr compute.DisksResizeRequest
		fake := func(_, _, _ string, drr *compute.DisksResizeRequest) error { gotDrr = *drr; return tt.clientErr }
		w.ComputeClient = &daisyCompute.TestClient{ResizeDiskFn: fake}
		if err := tt.rd.run(ctx, s); err != tt.wantErr {
			t.Errorf("%s: unexpected error returned, got: %v, want: %v", tt.desc, err, tt.wantErr)
		}
		if tt.wantDrr != nil {
			if diffRes := diff(gotDrr, *tt.wantDrr, 0); diffRes != "" {
				t.Errorf("%s: client got incorrect disk, got: %v, want: %v", tt.desc, gotDrr, *tt.wantDrr)
			}
		}
	}
}
