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
	"fmt"
	"testing"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/compute/v1"
)

func TestHealthChecksValidate(t *testing.T) {
	w := testWorkflow()
	s, e1 := w.NewStep("s")
	if errs := addErrs(nil, e1); errs != nil {
		t.Fatalf("test set up error: %v", errs)
	}

	tests := []struct {
		desc      string
		hc        *HealthCheck
		shouldErr bool
	}{
		{
			"valid",
			&HealthCheck{HealthCheck: compute.HealthCheck{Name: "hc1"}},
			false,
		},
		{
			"missing name",
			&HealthCheck{HealthCheck: compute.HealthCheck{}},
			true,
		},
	}

	for _, tt := range tests {
		// Test sanitation -- clean/set irrelevant fields.
		tt.hc.daisyName = tt.hc.Name
		tt.hc.RealName = tt.hc.Name
		tt.hc.link = fmt.Sprintf("projects/%s/global/healthChecks/%s", w.Project, tt.hc.Name)
		tt.hc.Project = w.Project

		s.CreateHealthChecks = &CreateHealthChecks{tt.hc}
		err := s.validate(context.Background())
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: did not return an error as expected", tt.desc)
			}
			if res, _ := w.healthChecks.get(tt.hc.Name); res != &tt.hc.Resource {
				t.Errorf("%s: %q not in HealthChecks registry as expected: got=%v want=%v", tt.desc, tt.hc.Name, &tt.hc.Resource, res)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestCreateHealthChecksRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}

	e := Errf("error")

	wantHealthCheck := compute.HealthCheck{}
	wantHealthCheck.Description = "HealthCheck created by Daisy in workflow \"test-wf\" on behalf of ."
	wantHealthCheck.Name = "test-wf-abcdef"

	tests := []struct {
		desc      string
		n, wantN  compute.HealthCheck
		clientErr error
		wantErr   DError
	}{
		{"good case", compute.HealthCheck{}, wantHealthCheck, nil, nil},
		{"client error case", compute.HealthCheck{}, wantHealthCheck, e, e},
	}

	for _, tt := range tests {
		var gotN compute.HealthCheck
		fake := func(_ string, n *compute.HealthCheck) error { gotN = *n; return tt.clientErr }
		w.ComputeClient = &daisyCompute.TestClient{CreateHealthCheckFn: fake}
		cds := &CreateHealthChecks{{HealthCheck: tt.n}}
		cds.populate(ctx, s)
		if err := cds.run(ctx, s); err != tt.wantErr {
			t.Errorf("%s: unexpected error returned, got: %v, want: %v", tt.desc, err, tt.wantErr)
		}
		if diff := pretty.Compare(gotN, tt.wantN); diff != "" {
			t.Errorf("%s: client got incorrect HealthCheck, diff: %s", tt.desc, diff)
		}
	}
}
