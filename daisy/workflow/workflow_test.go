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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/option"
	"reflect"
)

func TestCleanup(t *testing.T) {
	w := testWorkflow()

	d1 := &resource{name: "d1", link: "link", noCleanup: false}
	d2 := &resource{name: "d2", link: "link", noCleanup: true}
	im1 := &resource{name: "im1", link: "link", noCleanup: false}
	im2 := &resource{name: "im2", link: "link", noCleanup: true}
	in1 := &resource{name: "in1", link: "link", noCleanup: false}
	in2 := &resource{name: "in2", link: "link", noCleanup: true}
	w.diskRefs.m = map[string]*resource{"d1": d1, "d2": d2}
	w.imageRefs.m = map[string]*resource{"im1": im1, "im2": im2}
	w.instanceRefs.m = map[string]*resource{"in1": in1, "in2": in2}

	w.cleanup()

	want := map[string]*resource{"d2": d2}
	if !reflect.DeepEqual(w.diskRefs.m, want) {
		t.Errorf("cleanup didn't clean disks properly, want: %v; got: %v", want, w.diskRefs.m)
	}
	want = map[string]*resource{"im2": im2}
	if !reflect.DeepEqual(w.imageRefs.m, want) {
		t.Errorf("cleanup didn't clean images properly, want: %v; got: %v", want, w.imageRefs.m)
	}
	want = map[string]*resource{"in2": in2}
	if !reflect.DeepEqual(w.instanceRefs.m, want) {
		t.Errorf("cleanup didn't clean instances properly, want: %v; got: %v", want, w.instanceRefs.m)
	}
}

func TestEphemeralName(t *testing.T) {
	tests := []struct{ name, wfName, wfID, want string }{
		{"name", "wfname", "123456789", "name-wfname-123456789"},
		{"super-long-name-really-long", "super-long-workflow-name-like-really-really-long", "1", "super-long-name-really-long-super-long-workflow-name-lik-1"},
		{"super-long-name-really-long", "super-long-workflow-name-like-really-really-long", "123456789", "super-long-name-really-long-super-long-workflow-name-lik-123456"},
	}
	w := &Workflow{}
	for _, tt := range tests {
		w.id = tt.wfID
		w.Name = tt.wfName
		result := w.ephemeralName(tt.name)
		if result != tt.want {
			t.Errorf("bad result, input: name=%s wfName=%s wfId=%s; got: %s; want: %s", tt.name, tt.wfName, tt.wfID, result, tt.want)
		}
		if len(result) > 64 {
			t.Errorf("result > 64 characters, input: name=%s wfName=%s wfId=%s; got: %s", tt.name, tt.wfName, tt.wfID, result)
		}
	}
}

func TestFromFile(t *testing.T) {
	got := New(context.Background())
	err := got.FromFile("./test.workflow")
	if err != nil {
		t.Fatal(err)
	}
	subGot := got.Steps["sub workflow"].SubWorkflow.workflow

	want := &Workflow{
		id:      got.id,
		Name:    "some-name",
		Project: "some-project",
		Zone:    "us-central1-a",
		Bucket:  "some-bucket/images",
		Vars: map[string]string{
			"bootstrap_instance_name": "bootstrap",
			"machine_type":            "n1-standard-1",
		},
		Steps: map[string]*Step{
			"create disks": {
				name: "create disks",
				CreateDisks: &CreateDisks{
					{
						Name:        "bootstrap",
						SourceImage: "projects/windows-cloud/global/images/family/windows-server-2016-core",
						SizeGB:      "50",
						SSD:         true,
					},
					{
						Name:        "image",
						SourceImage: "projects/windows-cloud/global/images/family/windows-server-2016-core",
						SizeGB:      "50",
						SSD:         true,
					},
				},
			},
			"${bootstrap_instance_name}": {
				name: "${bootstrap_instance_name}",
				CreateInstances: &CreateInstances{
					{
						Name:          "${bootstrap_instance_name}",
						AttachedDisks: []string{"bootstrap", "image"},
						MachineType:   "${machine_type}",
						StartupScript: "shutdown /h",
						Metadata:      map[string]string{"test_metadata": "this was a test"},
					},
				},
			},
			"${bootstrap_instance_name} stopped": {
				name:                    "${bootstrap_instance_name} stopped",
				Timeout:                 "1h",
				WaitForInstancesStopped: &WaitForInstancesStopped{"${bootstrap_instance_name}"},
			},
			"postinstall": {
				name: "postinstall",
				CreateInstances: &CreateInstances{
					{
						Name:          "postinstall",
						AttachedDisks: []string{"image", "bootstrap"},
						MachineType:   "${machine_type}",
						StartupScript: "shutdown /h",
					},
				},
			},
			"postinstall stopped": {
				name: "postinstall stopped",
				WaitForInstancesStopped: &WaitForInstancesStopped{"postinstall"},
			},
			"create image": {
				name:         "create image",
				CreateImages: &CreateImages{{Name: "image-from-disk", SourceDisk: "image"}},
			},
			"sub workflow": {
				name: "sub workflow",
				SubWorkflow: &SubWorkflow{
					Path: "./test_sub.workflow",
					workflow: &Workflow{
						id: subGot.id,
						Steps: map[string]*Step{
							"create disks": {
								name: "create disks",
								CreateDisks: &CreateDisks{
									{
										Name:        "bootstrap",
										SourceImage: "projects/windows-cloud/global/images/family/windows-server-2016-core",
										SizeGB:      "50",
										SSD:         true,
									},
								},
							},
							"bootstrap": {
								name: "bootstrap",
								CreateInstances: &CreateInstances{
									{
										Name:          "bootstrap",
										AttachedDisks: []string{"bootstrap"},
										MachineType:   "n1-standard-1",
										StartupScript: "shutdown /h",
										Metadata:      map[string]string{"test_metadata": "this was a test"},
									},
								},
							},
							"bootstrap stopped": {
								name:                    "bootstrap stopped",
								Timeout:                 "1h",
								WaitForInstancesStopped: &WaitForInstancesStopped{"bootstrap"},
							},
						},
						Dependencies: map[string][]string{
							"bootstrap":         {"create disks"},
							"bootstrap stopped": {"bootstrap"},
						},
					},
				},
			},
		},
		Dependencies: map[string][]string{
			"create disks":        {},
			"bootstrap":           {"create disks"},
			"bootstrap stopped":   {"bootstrap"},
			"postinstall":         {"bootstrap stopped"},
			"postinstall stopped": {"postinstall"},
			"create image":        {"postinstall stopped"},
			"sub workflow":        {"create image"},
		},
	}

	// Check that subworkflow has workflow as parent.
	if subGot.parent != got {
		t.Error("subworkflow does not point to parent workflow")
	}

	// Fix pretty.Compare recursion freak outs.
	got.Ctx = nil
	got.Cancel = nil
	subGot.Ctx = nil
	subGot.Cancel = nil
	subGot.parent = nil

	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("parsed workflow does not match expectation: (-got +want)\n%s", diff)
	}
}

func TestPopulate(t *testing.T) {
	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	tf := filepath.Join(td, "test.cred")
	if err := ioutil.WriteFile(tf, []byte(`{ "type": "service_account" }`), 0600); err != nil {
		t.Fatalf("error creating temp file: %v", err)
	}

	got := &Workflow{
		Name:      "${name}",
		Bucket:    "parent-bucket/images",
		Zone:      "parent-zone",
		Project:   "parent-project",
		OAuthPath: tf,
		Vars: map[string]string{
			"step_name": "parent-step1",
			"timeout":   "60m",
			"path":      "./test_sub.workflow",
			"name":      "parent-name",
		},
		Steps: map[string]*Step{
			"${step_name}": {
				Timeout: "${timeout}",
			},
			"parent-step2": {},
			"parent-step3": {
				SubWorkflow: &SubWorkflow{
					Path: "${path}",
					workflow: &Workflow{
						Name:      "${name}",
						Bucket:    "sub-bucket/images",
						Project:   "sub-project",
						Zone:      "sub-zone",
						OAuthPath: "sub-oauth-path",
						Steps: map[string]*Step{
							"${step_name}": {
								Timeout: "${timeout}",
							},
						},
						Vars: map[string]string{
							"name":      "sub-name",
							"step_name": "sub-step1",
							"timeout":   "60m",
						},
					},
				},
			},
		},
	}

	if err := got.populate(); err != nil {
		t.Fatal(err)
	}

	subGot := got.Steps["parent-step3"].SubWorkflow.workflow

	// Set the clients to nil as pretty.Diff will cause a stack overflow otherwise.
	got.ComputeClient = nil
	got.StorageClient = nil
	subGot.ComputeClient = nil
	subGot.StorageClient = nil

	want := &Workflow{
		Name:         "parent-name",
		Bucket:       "parent-bucket/images",
		Zone:         "parent-zone",
		Project:      "parent-project",
		OAuthPath:    tf,
		id:           got.id,
		Ctx:          got.Ctx,
		Cancel:       got.Cancel,
		diskRefs:     &refMap{},
		imageRefs:    &refMap{},
		instanceRefs: &refMap{},
		Vars: map[string]string{
			"step_name": "parent-step1",
			"timeout":   "60m",
			"path":      "./test_sub.workflow",
			"name":      "parent-name",
		},
		scratchPath: "gs://parent-bucket/images/daisy-parent-name-" + got.id,
		sourcesPath: fmt.Sprintf("gs://parent-bucket/images/daisy-parent-name-%s/sources", got.id),
		logsPath:    fmt.Sprintf("gs://parent-bucket/images/daisy-parent-name-%s/logs", got.id),
		outsPath:    fmt.Sprintf("gs://parent-bucket/images/daisy-parent-name-%s/outs", got.id),
		Steps: map[string]*Step{
			"parent-step1": {
				name:    "parent-step1",
				Timeout: "60m",
				timeout: time.Duration(60 * time.Minute),
			},
			"parent-step2": {
				name:    "parent-step2",
				Timeout: "10m",
				timeout: time.Duration(10 * time.Minute),
			},
			"parent-step3": {
				name:    "parent-step3",
				Timeout: "10m",
				timeout: time.Duration(10 * time.Minute),
				SubWorkflow: &SubWorkflow{
					Path: "./test_sub.workflow",
					workflow: &Workflow{
						// This subworkflow should not have been modified by the parent's populate().
						Name:         "sub-name",
						Bucket:       "parent-bucket/images",
						Zone:         "parent-zone",
						Project:      "parent-project",
						OAuthPath:    tf,
						id:           subGot.id,
						diskRefs:     &refMap{},
						imageRefs:    &refMap{},
						instanceRefs: &refMap{},
						Steps: map[string]*Step{
							"sub-step1": {
								name:    "sub-step1",
								Timeout: "60m",
								timeout: time.Duration(60 * time.Minute),
							},
						},
						Vars: map[string]string{
							"name":      "sub-name",
							"step_name": "sub-step1",
							"timeout":   "60m",
						},
						scratchPath: "gs://parent-bucket/images/daisy-sub-name-" + subGot.id,
						sourcesPath: fmt.Sprintf("gs://parent-bucket/images/daisy-sub-name-%s/sources", subGot.id),
						logsPath:    fmt.Sprintf("gs://parent-bucket/images/daisy-sub-name-%s/logs", subGot.id),
						outsPath:    fmt.Sprintf("gs://parent-bucket/images/daisy-sub-name-%s/outs", subGot.id),
					},
				},
			},
		},
	}

	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("parsed workflow does not match expectation: (-got +want)\n%s", diff)
	}
}

func TestPrerun(t *testing.T) {
	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	tf := filepath.Join(td, "test.cred")
	if err := ioutil.WriteFile(tf, []byte(`{ "type": "service_account" }`), 0600); err != nil {
		t.Fatalf("error creating temp file: %v", err)
	}

	// Test that populating and validation occurs recursively on subworkflows.
	// Subworkflows validate and populate themselves, with the exception that
	// parent fields for GCS and GCE locations get passed to children subworkflow,
	// e.g. Bucket, Project, Zone, OAuth stuff.
	subGot := &Workflow{
		Name:      "${name}",
		Bucket:    "sub-bucket/images",
		Project:   "sub-project",
		Zone:      "sub-zone",
		OAuthPath: "sub-oauth-path",
		Steps: map[string]*Step{
			"${step_name}": {
				name:    "${step_name}",
				Timeout: "${timeout}",
				CreateImages: &CreateImages{
					{
						Name:       "image-name",
						Project:    "image-project",
						SourceFile: "/local/path",
					},
				},
			},
		},
		Vars: map[string]string{
			"name":      "sub-name",
			"step_name": "sub-step1",
			"timeout":   "60m",
		},
		Ctx: context.Background(),
	}

	subWant := &Workflow{
		Name:      "sub-name",
		Bucket:    "parent-bucket/images",
		Project:   "parent-project",
		Zone:      "parent-zone",
		OAuthPath: tf,
		Steps: map[string]*Step{
			"sub-step1": {
				name:    "sub-step1",
				Timeout: "60m",
				timeout: time.Duration(60 * time.Minute),
				CreateImages: &CreateImages{
					{
						Name:       "image-name",
						Project:    "image-project",
						SourceFile: "/local/path",
					},
				},
			},
		},
		Vars: map[string]string{
			"name":      "sub-name",
			"step_name": "sub-step1",
			"timeout":   "60m",
		},
		Ctx:          subGot.Ctx,
		diskRefs:     &refMap{},
		imageRefs:    &refMap{},
		instanceRefs: &refMap{},
	}

	got := &Workflow{
		Name:      "${name}",
		Bucket:    "parent-bucket/images",
		Zone:      "parent-zone",
		Project:   "parent-project",
		OAuthPath: tf,
		Steps: map[string]*Step{
			"${step_name}": {
				name:    "${step_name}",
				Timeout: "${timeout}",
				CreateDisks: &CreateDisks{
					{
						Name:        "d",
						SourceImage: "${source_image}",
						SizeGB:      "50",
						SSD:         true,
					},
				},
			},
			"parent-step2": {
				name:            "parent-step2",
				Timeout:         "15m",
				DeleteResources: &DeleteResources{Disks: []string{"d"}},
			},
			"parent-step3": {
				name: "parent-step3",
				SubWorkflow: &SubWorkflow{
					Path:     "${path}",
					workflow: subGot,
				},
			},
		},
		Dependencies: map[string][]string{
			"parent-step2": []string{"${step_name}"},
		},
		Vars: map[string]string{
			"source_image": "projects/windows-cloud/global/images/family/windows-server-2016-core",
			"step_name":    "parent-step1",
			"timeout":      "60m",
			"path":         "./test_sub.workflow",
			"name":         "parent-name",
		},
		Ctx: context.Background(),
	}

	want := &Workflow{
		Name:      "parent-name",
		Bucket:    "parent-bucket/images",
		Project:   "parent-project",
		Zone:      "parent-zone",
		OAuthPath: tf,
		Steps: map[string]*Step{
			"parent-step1": {
				name:    "parent-step1",
				Timeout: "60m",
				timeout: time.Duration(60 * time.Minute),
				CreateDisks: &CreateDisks{
					{
						Name:        "d",
						SourceImage: "projects/windows-cloud/global/images/family/windows-server-2016-core",
						SizeGB:      "50",
						SSD:         true,
					},
				},
			},
			"parent-step2": {
				name:            "parent-step2",
				Timeout:         "15m",
				timeout:         time.Duration(15 * time.Minute),
				DeleteResources: &DeleteResources{Disks: []string{"d"}},
			},
			"parent-step3": {
				name:    "parent-step3",
				Timeout: defaultTimeout,
				timeout: defaultTimeoutParsed,
				SubWorkflow: &SubWorkflow{
					Path:     "./test_sub.workflow",
					workflow: subWant,
				},
			},
		},
		Dependencies: map[string][]string{
			"parent-step2": []string{"parent-step1"},
		},
		Vars: map[string]string{
			"source_image": "projects/windows-cloud/global/images/family/windows-server-2016-core",
			"step_name":    "parent-step1",
			"timeout":      "60m",
			"path":         "./test_sub.workflow",
			"name":         "parent-name",
		},
		Ctx:          got.Ctx,
		diskRefs:     &refMap{},
		imageRefs:    &refMap{},
		instanceRefs: &refMap{},
	}

	if err := got.prerun(); err != nil {
		t.Fatal(err)
	}

	// These are generated at populate time, so we have to cheat.
	want.id = got.id
	subWant.id = subGot.id
	want.scratchPath = fmt.Sprintf("gs://parent-bucket/images/daisy-parent-name-%s", want.id)
	want.sourcesPath = want.scratchPath + "/sources"
	want.logsPath = want.scratchPath + "/logs"
	want.outsPath = want.scratchPath + "/outs"
	subWant.scratchPath = fmt.Sprintf("gs://parent-bucket/images/daisy-sub-name-%s", subWant.id)
	subWant.sourcesPath = subWant.scratchPath + "/sources"
	subWant.logsPath = subWant.scratchPath + "/logs"
	subWant.outsPath = subWant.scratchPath + "/outs"

	// Fix pretty.Compare recursion freak outs.
	got.ComputeClient = nil
	got.StorageClient = nil
	subGot.ComputeClient = nil
	subGot.StorageClient = nil
	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("parsed workflow does not match expectation: (-got +want)\n%s", diff)
	}
}

func TestResolveLink(t *testing.T) {
	rm := &refMap{}
	rm.m = map[string]*resource{"x": {"x", "realX", "link", false}}

	tests := []struct{ desc, input, want string }{
		{"", "x", "link"},
		{"", "projects/foo/bar/baz", "projects/foo/bar/baz"},
		{"unresolved link", "blah", ""},
	}

	for _, tt := range tests {
		if got := resolveLink(tt.input, rm); got != tt.want {
			t.Errorf("%q case failed, input: %q; want: %q; got: %q", tt.desc, tt.input, tt.want, got)
		}
	}
}

func TestStepDepends(t *testing.T) {
	w := Workflow{
		Steps: map[string]*Step{
			"s1": {testType: &mockStep{}},
			"s2": {testType: &mockStep{}},
			"s3": {testType: &mockStep{}},
			"s4": {testType: &mockStep{}},
			"s5": {testType: &mockStep{}},
		},
		Dependencies: map[string][]string{},
	}
	// Check proper false.
	if w.stepDepends("s1", "s2") {
		t.Error("s1 shouldn't depend on s2")
	}

	// Check proper true.
	w.Dependencies["s1"] = []string{"s2"}
	if !w.stepDepends("s1", "s2") {
		t.Error("s1 should depend on s2")
	}

	// Check transitive dependency returns true.
	w.Dependencies["s2"] = []string{"s3"}
	if !w.stepDepends("s1", "s3") {
		t.Error("s1 should transitively depend on s3")
	}

	// Check cyclical graph terminates.
	w.Dependencies["s2"] = append(w.Dependencies["s2"], "s4")
	w.Dependencies["s4"] = []string{"s2"}
	// s1 doesn't have any relation to s5, but we need to check
	// if this can terminate on graphs with cycles.
	if w.stepDepends("s1", "s5") {
		t.Error("s1 shouldn't depend on s5")
	}

	// Check self depends on self -- false case.
	if w.stepDepends("s1", "s1") {
		t.Error("s1 shouldn't depend on s1")
	}

	// Check self depends on self true.
	w.Dependencies["s5"] = []string{"s5"}
	if !w.stepDepends("s5", "s5") {
		t.Error("s5 should depend on s5")
	}
}

func TestRealStep(t *testing.T) {
	// Good. Try normal, working case.
	s := Step{
		RunTests: &RunTests{},
	}
	if _, err := s.realStep(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	// Bad. Try empty step.
	s = Step{}
	if _, err := s.realStep(); err == nil {
		t.Fatal("empty step should have thrown an error")
	}
	// Bad. Try step with multiple real steps.
	s = Step{
		AttachDisks: &AttachDisks{},
		RunTests:    &RunTests{},
	}
	if _, err := s.realStep(); err == nil {
		t.Fatal("malformed step should have thrown an error")
	}
}

func TestRefMapAdd(t *testing.T) {
	rm := refMap{}

	tests := []struct {
		desc, ref string
		res       *resource
		want      map[string]*resource
	}{
		{"normal add", "x", &resource{name: "x"}, map[string]*resource{"x": {name: "x"}}},
		{"dupe add", "x", &resource{name: "otherx"}, map[string]*resource{"x": {name: "otherx"}}},
	}

	for _, tt := range tests {
		rm.add(tt.ref, tt.res)
		if diff := pretty.Compare(rm.m, tt.want); diff != "" {
			t.Errorf("%q case failed, refmap does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestRefMapConcurrency(t *testing.T) {
	rm := refMap{}

	tests := []struct {
		desc string
		f    func()
	}{
		{"add", func() { rm.add("foo", nil) }},
		{"del", func() { rm.del("foo") }},
		{"get", func() { rm.get("foo") }},
	}

	for _, tt := range tests {
		order := []string{}
		releaseStr := "lock released"
		returnedStr := "func returned"
		want := []string{releaseStr, returnedStr}
		gunshot := sync.Mutex{}
		gunshot.Lock() // Wait for the goroutine to say we can go ahead.
		go func() {
			rm.mx.Lock()
			defer rm.mx.Unlock()
			gunshot.Unlock()
			time.Sleep(1 * time.Second)
			order = append(order, releaseStr)
		}()
		gunshot.Lock() // Wait for the go ahead.
		tt.f()
		order = append(order, returnedStr)
		if !reflect.DeepEqual(order, want) {
			t.Errorf("%q case failed, unexpected concurrency order, want: %v; got: %v", tt.desc, want, order)
		}
	}
}

func TestRefMapDel(t *testing.T) {
	xRes := &resource{}
	yRes := &resource{}
	rm := refMap{m: map[string]*resource{"x": xRes, "y": yRes}}

	tests := []struct {
		desc, input string
		want        map[string]*resource
	}{
		{"normal del", "y", map[string]*resource{"x": xRes}},
		{"del dne", "foo", map[string]*resource{"x": xRes}},
	}

	for _, tt := range tests {
		rm.del(tt.input)
		if !reflect.DeepEqual(rm.m, tt.want) {
			t.Errorf("%q case failed, refmap does not match expectation, want: %v; got: %v", tt.desc, tt.want, rm.m)
		}
	}
}

func TestRefMapGet(t *testing.T) {
	xRes := &resource{}
	yRes := &resource{}
	rm := refMap{m: map[string]*resource{"x": xRes, "y": yRes}}

	tests := []struct {
		desc, input string
		wantR       *resource
		wantOk      bool
	}{
		{"normal get", "y", yRes, true},
		{"get dne", "dne", nil, false},
	}

	for _, tt := range tests {
		if gotR, gotOk := rm.get(tt.input); !(gotOk == tt.wantOk && gotR == tt.wantR) {
			t.Errorf("%q case failed, want: (%v, %t); got: (%v, %t)", tt.desc, tt.wantR, tt.wantOk, gotR, gotOk)
		}
	}
}

func testTraverseWorkflow(mockRun func(i int) func(*Workflow) error) (*Workflow, error) {
	// s0---->s1---->s3
	//   \         /
	//    --->s2---
	// s4
	w := testWorkflow()
	w.Steps = map[string]*Step{
		"s0": {name: "s0", testType: &mockStep{runImpl: mockRun(0)}},
		"s1": {name: "s1", testType: &mockStep{runImpl: mockRun(1)}},
		"s2": {name: "s2", testType: &mockStep{runImpl: mockRun(2)}},
		"s3": {name: "s3", testType: &mockStep{runImpl: mockRun(3)}},
		"s4": {name: "s4", testType: &mockStep{runImpl: mockRun(4)}},
	}
	w.Dependencies = map[string][]string{
		"s1": {"s0"},
		"s2": {"s0"},
		"s3": {"s1", "s2"},
	}

	var err error
	ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	w.ComputeClient, err = compute.NewClient(context.Background(), option.WithEndpoint(ts.URL), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		return nil, fmt.Errorf("error creating test client: %s", err)
	}
	w.StorageClient, err = storage.NewClient(context.Background(), option.WithEndpoint(ts.URL), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		return nil, fmt.Errorf("error creating test client: %s", err)
	}
	return w, nil
}

func TestTraverseDAG(t *testing.T) {
	var callOrder []int
	errs := make([]error, 5)
	var rw sync.Mutex
	mockRun := func(i int) func(*Workflow) error {
		return func(_ *Workflow) error {
			rw.Lock()
			defer rw.Unlock()
			callOrder = append(callOrder, i)
			return errs[i]
		}
	}

	// Check call order: s1 and s2 must be after s0, s3 must be after s1 and s2.
	checkCallOrder := func() error {
		rw.Lock()
		defer rw.Unlock()
		stepOrderNum := []int{-1, -1, -1, -1, -1}
		for i, stepNum := range callOrder {
			stepOrderNum[stepNum] = i
		}
		// If s1 was called, check it was called after s0.
		if stepOrderNum[1] != -1 && stepOrderNum[1] < stepOrderNum[0] {
			return errors.New("s1 was called before s0")
		}
		// If s2 was called, check it was called after s0.
		if stepOrderNum[2] != -1 && stepOrderNum[2] < stepOrderNum[0] {
			return errors.New("s2 was called before s0")
		}
		// If s3 was called, check it was called after s1 and s2.
		if stepOrderNum[3] != -1 {
			if stepOrderNum[3] < stepOrderNum[1] {
				return errors.New("s3 was called before s1")
			}
			if stepOrderNum[3] < stepOrderNum[2] {
				return errors.New("s3 was called before s2")
			}
		}
		return nil
	}

	// Normal, good run.
	w, err := testTraverseWorkflow(mockRun)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Run(); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if err := checkCallOrder(); err != nil {
		t.Errorf("call order error: %s", err)
	}

	callOrder = []int{}
	errs = make([]error, 5)

	// s2 failure.
	w, err = testTraverseWorkflow(mockRun)
	if err != nil {
		t.Fatal(err)
	}
	errs[2] = errors.New("failure")
	want := w.Steps["s2"].wrapRunError(errs[2])
	if err := w.Run(); err.Error() != want.Error() {
		t.Errorf("unexpected error: %s != %s", err, want)
	}
	if err := checkCallOrder(); err != nil {
		t.Errorf("call order error: %s", err)
	}
}
