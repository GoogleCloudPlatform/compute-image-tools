//  Copyright 2020 Google Inc. All Rights Reserved.
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

func TestCreateSnapshotsRunSuccess(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	createCalled := false

	w.ComputeClient.(*daisyCompute.TestClient).CreateSnapshotFn = func(p, z, d string, ss *compute.Snapshot) error {
		ss.SelfLink = "insertedLink"
		createCalled = true
		return nil
	}
	w.disks.m = map[string]*Resource{"sd": {link: "dLink"}}

	ss0 := &Snapshot{Resource: Resource{daisyName: "ss0"}, Snapshot: compute.Snapshot{Name: "realSS0", SourceDisk: "sd"}}
	css := &CreateSnapshots{ss0}
	if err := css.run(ctx, s); err != nil {
		t.Errorf("unexpected error running CreateSnapshots.run(): %v", err)
	}
	if !createCalled {
		t.Errorf("CreateSnapshot not called")
	}
}

func TestCreateSnapshotsRunFailureOnComputeCreateError(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.disks.m = map[string]*Resource{}
	createErr := Errf("client error")
	w.ComputeClient.(*daisyCompute.TestClient).CreateSnapshotFn = func(p, z, d string, i *compute.Snapshot) error {
		i.SelfLink = "insertedLink"
		return createErr
	}

	css := &CreateSnapshots{
		{Resource: Resource{daisyName: "ss0"}, Snapshot: compute.Snapshot{Name: "realSS0", SourceDisk: "sd"}},
	}
	if err := css.run(ctx, s); err != createErr {
		t.Errorf("CreateSnapshot.run() should have return compute client error: %v != %v", err, createErr)
	}
}
