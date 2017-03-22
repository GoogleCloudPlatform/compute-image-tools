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

package workflow

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestDeleteResourcesRun(t *testing.T) {
	wf := testWorkflow()
	wf.instanceRefs.m = map[string]*resource{
		"in1": {"in1", wf.ephemeralName("in1"), "link", false},
		"in2": {"in2", wf.ephemeralName("in2"), "link", false},
		"in3": {"in3", wf.ephemeralName("in3"), "link", false},
		"in4": {"in4", wf.ephemeralName("in4"), "link", false}}
	wf.imageRefs.m = map[string]*resource{
		"im1": {"im1", wf.ephemeralName("im1"), "link", false},
		"im2": {"im2", wf.ephemeralName("im2"), "link", false},
		"im3": {"im3", wf.ephemeralName("im3"), "link", false},
		"im4": {"im4", wf.ephemeralName("im4"), "link", false}}
	wf.diskRefs.m = map[string]*resource{
		"d1": {"d1", wf.ephemeralName("d1"), "link", false},
		"d2": {"d2", wf.ephemeralName("d2"), "link", false},
		"d3": {"d3", wf.ephemeralName("d3"), "link", false},
		"d4": {"d4", wf.ephemeralName("d4"), "link", false}}

	dr := &DeleteResources{
		Instances: []string{"in1", "in2", "in3"},
		Images:    []string{"im1", "im2", "im3"},
		Disks:     []string{"d1", "d2", "d3"}}
	if err := dr.run(wf); err != nil {
		t.Fatalf("error running DeleteResources.run(): %v", err)
	}

	want := map[string]*resource{"in4": {"in4", wf.ephemeralName("in4"), "link", false}}
	if diff := pretty.Compare(wf.instanceRefs.m, want); diff != "" {
		t.Errorf("instanceRefs do not match expectation: (-got +want)\n%s", diff)
	}

	want = map[string]*resource{"im4": {"im4", wf.ephemeralName("im4"), "link", false}}
	if diff := pretty.Compare(wf.imageRefs.m, want); diff != "" {
		t.Errorf("imageRefs do not match expectation: (-got +want)\n%s", diff)
	}

	want = map[string]*resource{"d4": {"d4", wf.ephemeralName("d4"), "link", false}}
	if diff := pretty.Compare(wf.diskRefs.m, want); diff != "" {
		t.Errorf("diskRefs do not match expectation: (-got +want)\n%s", diff)
	}
}

func TestDeleteResourcesValidate(t *testing.T) {}
