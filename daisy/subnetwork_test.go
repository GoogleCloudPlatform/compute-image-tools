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

	"google.golang.org/api/compute/v1"
)

func TestSubnetworkPopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s, _ := w.NewStep("s")

	desc := defaultDescription("Subnetwork", w.Name, w.username)
	name := "name"
	tests := []struct {
		desc     string
		sn, want *Subnetwork
	}{
		{"defaults case", &Subnetwork{}, &Subnetwork{Subnetwork: compute.Subnetwork{Description: desc}, Resource: Resource{link: fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", w.Project, getRegionFromZone(w.Zone), name)}}},
	}

	for _, tt := range tests {
		// Test sanitation -- clean/set irrelevant fields.
		tt.sn.Name = name
		tt.sn.ExactName = true
		tt.want.Name = name
		tt.want.Project = w.Project // Tested in resource_test.
		tt.want.ExactName = true    // Tested in resource_test.
		tt.want.RealName = name     // Tested in resource_test.
		tt.want.daisyName = name    // Tested in resource_test.

		if err := tt.sn.populate(ctx, s); err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		} else if diffRes := diff(tt.sn, tt.want, 0); diffRes != "" {
			t.Errorf("%s: populated Subnetwork does not match expectation: (-got +want)\n%s", tt.desc, diffRes)
		}
	}
}

func TestSubnetworkValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s, _ := w.NewStep("s")

	def := &Subnetwork{Resource: Resource{
		Project:  w.Project,
		RealName: "goodname",
		link:     fmt.Sprintf("projects/%s/regions/%s/subnetworks/goodname", w.Project, getRegionFromZone(w.Zone)),
	}}
	tests := []struct {
		desc      string
		sn        *Subnetwork
		shouldErr bool
	}{
		{"good case", &Subnetwork{Subnetwork: compute.Subnetwork{Name: "foo", Network: "bar", IpCidrRange: "192.168.1.0/32"}}, false},
		{"bad case", &Subnetwork{Subnetwork: compute.Subnetwork{Name: "foo", Network: "bar", IpCidrRange: "192.168.1.0/33"}}, true},
	}

	for _, tt := range tests {
		// Test sanitation -- clean/set irrelevant fields.
		tt.sn.RealName = def.RealName
		tt.sn.Project = def.Project
		tt.sn.link = def.link

		err := tt.sn.validate(ctx, s)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestSubnetworkRegConnect(t *testing.T) {
	// Test:
	// - good: normal connect
	// - good: connect after disconnect
	// - bad: already connected
	// - bad: connector doesn't depend on disconnector
	w := testWorkflow()
	s, _ := w.NewStep("s")
	conn1, _ := w.NewStep("conn1")
	conn2, _ := w.NewStep("conn2")
	dconn1, _ := w.NewStep("dconn1")
	dconn2, _ := w.NewStep("dconn2")
	w.AddDependency(s, dconn1)

	reset := func() {
		w.subnetworks.connections = map[string]map[string]*subnetworkConnection{
			"n1": {},
			"n2": {"i": {connector: conn1, disconnector: dconn1}},
			"n3": {"i": {connector: conn2}},
			"n4": {"i": {connector: conn1, disconnector: dconn2}},
		}
	}

	tests := []struct {
		desc, nName string
		shouldErr   bool
	}{
		{"normal case", "n1", false},
		{"connect after disconnect case", "n2", false},
		{"already connected case", "n3", true},
		{"connector doesn't depend on disconnector case", "n4", true},
	}

	for _, tt := range tests {
		reset()
		if err := w.subnetworks.regConnect(tt.nName, "i", s); err != nil {
			if !tt.shouldErr {
				t.Errorf("%s: unexpected error: %v", tt.desc, err)
			}
		} else if tt.shouldErr {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if nc := w.subnetworks.connections[tt.nName]["i"]; nc.connector != s || nc.disconnector != nil {
			t.Errorf("%s: connection not created as expected: %v != %v", tt.desc, nc, &subnetworkConnection{s, nil})
		}
	}

}

func TestSubnetworkDisconnectHelper(t *testing.T) {
	// Test:
	// - normal disconnect
	// - disconnector doesn't depend on connector
	// - already disconnected
	// - not attached
	// - subnetwork DNE
	w := testWorkflow()
	s, _ := w.NewStep("s")
	conn1, _ := w.NewStep("conn1")
	conn2, _ := w.NewStep("conn2")
	dconn, _ := w.NewStep("dconn")
	w.AddDependency(s, conn1)

	reset := func() {
		w.subnetworks.connections = map[string]map[string]*subnetworkConnection{
			"n1": {"i": {connector: conn1}},
			"n2": {"i": {connector: conn2}},
			"n3": {"i": {disconnector: dconn}},
			"n4": {},
		}
	}

	tests := []struct {
		desc, nName string
		shouldErr   bool
	}{
		{"normal case", "n1", false},
		{"not dependent on connector case", "n2", true},
		{"already disconnected case", "n3", true},
		{"not connected case", "n4", true},
		{"subnetwork DNE case", "n5", true},
	}

	for _, tt := range tests {
		reset()
		if err := w.subnetworks.disconnectHelper(tt.nName, "i", s); err != nil {
			if !tt.shouldErr {
				t.Errorf("%s: unexpected error: %v", tt.desc, err)
			}
		} else if tt.shouldErr {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if w.subnetworks.connections[tt.nName]["i"].disconnector != s {
			t.Errorf("%s: step s should have been registered as disconnector", tt.desc)
		}
	}
}

func TestSubnetworkRegDisconnect(t *testing.T) {
	// Test:
	// - no error from helper
	// - error from helper
	w := testWorkflow()

	var helperErr *DError
	w.subnetworks.testDisconnectHelper = func(_, _ string, _ *Step) DError {
		return *helperErr
	}

	tests := []struct {
		desc      string
		helperErr DError
		shouldErr bool
	}{
		{"normal case", nil, false},
		{"disconnect helper error case", Errf("error!"), true},
	}

	for _, tt := range tests {
		helperErr = &tt.helperErr
		if err := w.subnetworks.regDisconnect("", "", nil); tt.shouldErr && err == nil {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestSubnetworkRegDisconnectAll(t *testing.T) {
	// Test:
	// - no error from helper
	// - error from helper
	// - skip already disconnected
	w := testWorkflow()
	w.cloudLoggingClient = nil
	s, _ := w.NewStep("s")
	otherDisconnector, _ := w.NewStep("other-disconnector")

	var callsArgs [][]interface{}
	var helperErr *DError
	w.subnetworks.testDisconnectHelper = func(nName, iName string, s *Step) DError {
		callsArgs = append(callsArgs, []interface{}{nName, iName, s})
		return *helperErr
	}

	reset := func() {
		callsArgs = nil
		w.subnetworks.connections = map[string]map[string]*subnetworkConnection{
			"n1": {"i": {}},
			"n2": {},
			"n3": {"i": {disconnector: otherDisconnector}},
		}
	}

	tests := []struct {
		desc      string
		helperErr DError
		shouldErr bool
	}{
		{"normal case", nil, false},
		{"disconnect helper error case", Errf("error!"), true},
	}

	for _, tt := range tests {
		reset()
		helperErr = &tt.helperErr
		wantCallsArgs := [][]interface{}{{"n1", "i", s}}
		if err := w.subnetworks.regDisconnectAll("i", s); err != nil {
			if !tt.shouldErr {
				t.Errorf("%s: unexpected error: %v", tt.desc, err)
			}
		} else if tt.shouldErr {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if diffRes := diff(callsArgs, wantCallsArgs, 0); diffRes != "" {
			t.Errorf("%s: disconnectHelper not called as expected: (-got,+want)\n%s", tt.desc, diffRes)
		}
	}
}
