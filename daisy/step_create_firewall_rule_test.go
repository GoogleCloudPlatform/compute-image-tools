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
	"fmt"
	"testing"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/compute/v1"
)

func TestFirewallRulesValidate(t *testing.T) {
	w := testWorkflow()
	s, e1 := w.NewStep("s")
	if errs := addErrs(nil, e1); errs != nil {
		t.Fatalf("test set up error: %v", errs)
	}
	net := fmt.Sprintf("projects/%s/global/networks/default", w.Project)

	tests := []struct {
		desc      string
		fir       *FirewallRule
		shouldErr bool
	}{
		{
			"valid",
			&FirewallRule{Firewall: compute.Firewall{Name: "d1", Network: net}},
			false,
		},
		{
			"missing network",
			&FirewallRule{Firewall: compute.Firewall{Name: "d4"}},
			true,
		},
		{
			"missing name",
			&FirewallRule{Firewall: compute.Firewall{Network: net}},
			true,
		},
	}

	for _, tt := range tests {
		// Test sanitation -- clean/set irrelevant fields.
		tt.fir.daisyName = tt.fir.Name
		tt.fir.RealName = tt.fir.Name
		tt.fir.link = fmt.Sprintf("projects/%s/global/firewalls/%s", w.Project, tt.fir.Name)
		tt.fir.Project = w.Project

		s.CreateFirewallRules = &CreateFirewallRules{tt.fir}
		err := s.validate(context.Background())
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: did not return an error as expected", tt.desc)
			}
			if res, _ := w.firewallRules.get(tt.fir.Name); res != &tt.fir.Resource {
				t.Errorf("%s: %q not in FirewallRules registry as expected: got=%v want=%v", tt.desc, tt.fir.Name, &tt.fir.Resource, res)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestCreateFirewallRulesRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}

	e := Errf("error")

	wantFirewallRule := compute.Firewall{}
	wantFirewallRule.Description = "FirewallRule created by Daisy in workflow \"test-wf\" on behalf of ."
	wantFirewallRule.Name = "test-wf-abcdef"
	wantFirewallRule.Network = "projects/test-project/global/networks/bar"

	tests := []struct {
		desc      string
		n, wantN  compute.Firewall
		clientErr error
		wantErr   DError
	}{
		{"good case", compute.Firewall{Network: "global/networks/bar"}, wantFirewallRule, nil, nil},
		{"client error case", compute.Firewall{Network: "global/networks/bar"}, wantFirewallRule, e, e},
	}

	for _, tt := range tests {
		var gotN compute.Firewall
		fake := func(_ string, n *compute.Firewall) error { gotN = *n; return tt.clientErr }
		w.ComputeClient = &daisyCompute.TestClient{CreateFirewallRuleFn: fake}
		cds := &CreateFirewallRules{{Firewall: tt.n}}
		cds.populate(ctx, s)
		if err := cds.run(ctx, s); err != tt.wantErr {
			t.Errorf("%s: unexpected error returned, got: %v, want: %v", tt.desc, err, tt.wantErr)
		}
		if diff := pretty.Compare(gotN, tt.wantN); diff != "" {
			t.Errorf("%s: client got incorrect FirewallRule, diff: %s", tt.desc, diff)
		}
	}
}
