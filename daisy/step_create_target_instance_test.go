//  Copyright 2018 Google Inc. All Rights Reserved.
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
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/compute/v1"
)

func TestCreateTargetInstancesRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}

	e := Errf("error")

	wantTargetInstance := compute.TargetInstance{}
	wantTargetInstance.Description = "TargetInstance created by Daisy in workflow \"test-wf\" on behalf of ."
	wantTargetInstance.Name = "test-wf-abcdef"
	wantTargetInstance.Instance = "projects/test-project/zones/test-zone/instances/"
	wantTargetInstance.Zone = "test-zone"

	tests := []struct {
		desc      string
		n, wantN  compute.TargetInstance
		clientErr error
		wantErr   DError
	}{
		{"good case", compute.TargetInstance{}, wantTargetInstance, nil, nil},
		{"client error case", compute.TargetInstance{}, wantTargetInstance, e, e},
	}

	for _, tt := range tests {
		var gotN compute.TargetInstance
		fake := func(_, _ string, n *compute.TargetInstance) error { gotN = *n; return tt.clientErr }
		w.ComputeClient = &daisyCompute.TestClient{CreateTargetInstanceFn: fake}
		cds := &CreateTargetInstances{{TargetInstance: tt.n}}
		cds.populate(ctx, s)
		if err := cds.run(ctx, s); err != tt.wantErr {
			t.Errorf("%s: unexpected error returned, got: %v, want: %v", tt.desc, err, tt.wantErr)
		}
		if diff := pretty.Compare(gotN, tt.wantN); diff != "" {
			t.Errorf("%s: client got incorrect TargetInstance, diff: %s", tt.desc, diff)
		}
	}
}
