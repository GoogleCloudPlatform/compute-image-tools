//  Copyright 2021 Google Inc. All Rights Reserved.
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

func TestCreateTargetPoolsRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}

	e := Errf("error")

	wantTargetPool := compute.TargetPool{}
	wantTargetPool.Description = "TargetPool created by Daisy in workflow \"test-wf\" on behalf of ."
	wantTargetPool.Name = "test-wf-abcdef"

	tests := []struct {
		desc      string
		n, wantN  compute.TargetPool
		clientErr error
		wantErr   DError
	}{
		{"good case", compute.TargetPool{}, wantTargetPool, nil, nil},
		{"client error case", compute.TargetPool{}, wantTargetPool, e, e},
	}

	for _, tt := range tests {
		var gotN compute.TargetPool
		fake := func(_, _ string, n *compute.TargetPool) error { gotN = *n; return tt.clientErr }
		w.ComputeClient = &daisyCompute.TestClient{CreateTargetPoolFn: fake}
		cds := &CreateTargetPools{{TargetPool: tt.n}}
		cds.populate(ctx, s)
		if err := cds.run(ctx, s); err != tt.wantErr {
			t.Errorf("%s: unexpected error returned, got: %v, want: %v", tt.desc, err, tt.wantErr)
		}
		if diff := pretty.Compare(gotN, tt.wantN); diff != "" {
			t.Errorf("%s: client got incorrect TargetPool, diff: %s", tt.desc, diff)
		}
	}
}
