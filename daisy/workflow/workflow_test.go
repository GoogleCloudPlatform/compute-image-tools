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
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/option"
)

var (
	testGCEClient *compute.Client
	testGCSClient *storage.Client
	testGCSDNEVal = "dne"
	testSuffix    = randString(5)
	testWf        = "test-wf"
	testProject   = "test-project"
	testZone      = "test-zone"
	testBucket    = "test-bucket"
)

func init() {
	var err error
	testGCEClient, err = newTestGCEClient()
	if err != nil {
		panic(err)
	}
	testGCSClient, err = newTestGCSClient()
	if err != nil {
		panic(err)
	}
}

func testWorkflow() *Workflow {
	return &Workflow{Name: testWf, Bucket: testBucket, Project: testProject, Zone: testZone, ComputeClient: testGCEClient, StorageClient: testGCSClient, id: testSuffix, Ctx: context.Background()}
}

func newTestGCEClient() (*compute.Client, error) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.String(), fmt.Sprintf("/%s/zones/%s/instances/", testProject, testZone)) {
			fmt.Fprintln(w, `{"Status":"TERMINATED"}`)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), fmt.Sprintf("/%s/zones/%s/machineTypes", testProject, testZone)) {
			fmt.Fprintln(w, `{"Items":[{"Name": "foo-type"}]}`)
		} else {
			fmt.Fprintln(w, `{"Status":"DONE","SelfLink":"link"}`)
		}
	}))

	return compute.NewClient(context.Background(), option.WithEndpoint(ts.URL), option.WithHTTPClient(http.DefaultClient))
}

func newTestGCSClient() (*storage.Client, error) {
	nameRgx := regexp.MustCompile(`"name":"([^"].*)"`)
	rewriteRgx := regexp.MustCompile("/b/([^/]+)/o/([^/]+)/rewriteTo/b/([^/]+)/o/([^?]+)")
	uploadRgx := regexp.MustCompile("/b/([^/]+)/o?.*uploadType=multipart.*")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.String()
		m := r.Method

		if match := uploadRgx.FindStringSubmatch(u); m == "POST" && match != nil {
			body, _ := ioutil.ReadAll(r.Body)
			n := nameRgx.FindStringSubmatch(string(body))[1]
			fmt.Fprintf(w, `{"kind":"storage#object","bucket":"%s","name":"%s"}\n`, match[1], n)
		} else if match := rewriteRgx.FindStringSubmatch(u); m == "POST" && match != nil {
			if strings.Contains(match[1], testGCSDNEVal) || strings.Contains(match[2], testGCSDNEVal) {
				w.WriteHeader(http.StatusNotFound)
			}
			o := fmt.Sprintf(`{"bucket":"%s","name":"%s"}`, match[3], match[4])
			fmt.Fprintf(w, `{"kind": "storage#rewriteResponse", "done": true, "objectSize": "1", "totalBytesRewritten": "1", "resource": %s}\n`, o)
		} else {
			fmt.Println("got something else")
		}
	}))

	return storage.NewClient(context.Background(), option.WithEndpoint(ts.URL), option.WithHTTPClient(http.DefaultClient))
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
					Workflow: &Workflow{
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

	subGot := got.Steps["parent-step3"].SubWorkflow.Workflow

	// Set the clients to nil as pretty.Diff will cause a stack overflow otherwise.
	got.ComputeClient = nil
	got.StorageClient = nil
	subGot.ComputeClient = nil
	subGot.StorageClient = nil

	want := &Workflow{
		Name:      "parent-name",
		Bucket:    "parent-bucket/images",
		Zone:      "parent-zone",
		Project:   "parent-project",
		OAuthPath: tf,
		id:        got.id,
		Ctx:       got.Ctx,
		Cancel:    got.Cancel,
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
					Workflow: &Workflow{
						// This subworkflow should not have been modified by the parent's populate().
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

	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("parsed workflow does not match expectation: (-got +want)\n%s", diff)
	}
}

func TestFromFile(t *testing.T) {
	got := New(context.Background())
	err := got.FromFile("./test.workflow")
	if err != nil {
		t.Fatal(err)
	}
	subGot := got.Steps["sub workflow"].SubWorkflow.Workflow
	// pretty.Compare freaks out (nil pointer dereference somewhere) over the Cancel functions.
	got.Ctx = nil
	got.Cancel = nil
	subGot.Ctx = nil
	subGot.Cancel = nil

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
					Workflow: &Workflow{
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
								name: "bootstrap stopped",
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

	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("parsed workflow does not match expectation: (-got +want)\n%s", diff)
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

func testTraverseWorkflow(mockRun func(i int) func(*Workflow) error) (*Workflow, error) {
	// s0---->s1---->s3
	//   \         /
	//    --->s2---
	// s4
	w := testWorkflow()
	w.Steps = map[string]*Step{
		"s0": {testType: &mockStep{runImpl: mockRun(0)}},
		"s1": {testType: &mockStep{runImpl: mockRun(1)}},
		"s2": {testType: &mockStep{runImpl: mockRun(2)}},
		"s3": {testType: &mockStep{runImpl: mockRun(3)}},
		"s4": {testType: &mockStep{runImpl: mockRun(4)}},
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
	errs[2] = errors.New("s2 failure")
	if err := w.Run(); err != errs[2] {
		t.Errorf("error %s != %s", err, errs[2])
	}
	if err := checkCallOrder(); err != nil {
		t.Errorf("call order error: %s", err)
	}
}
