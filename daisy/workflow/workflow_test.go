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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/kylelemons/godebug/diff"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/option"
)

func TestCleanup(t *testing.T) {
	cleanedup1 := false
	cleanedup2 := false
	cleanup1 := func() error {
		cleanedup1 = true
		return nil
	}
	cleanup2 := func() error {
		cleanedup2 = true
		return nil
	}
	cleanupFail := func() error {
		return errors.New("failed cleanup")
	}

	w := testWorkflow()
	w.addCleanupHook(cleanup1)
	w.addCleanupHook(cleanupFail)
	w.addCleanupHook(cleanup2)
	w.cleanup()

	if !cleanedup1 {
		t.Error("cleanup1 was not run")
	}
	if !cleanedup2 {
		t.Error("cleanup2 was not run")
	}
}

func TestGenName(t *testing.T) {
	tests := []struct{ name, wfName, wfID, want string }{
		{"name", "wfname", "123456789", "name-wfname-123456789"},
		{"super-long-name-really-long", "super-long-workflow-name-like-really-really-long", "1", "super-long-name-really-long-super-long-workflow-name-lik-1"},
		{"super-long-name-really-long", "super-long-workflow-name-like-really-really-long", "123456789", "super-long-name-really-long-super-long-workflow-name-lik-123456"},
	}
	w := &Workflow{}
	for _, tt := range tests {
		w.id = tt.wfID
		w.Name = tt.wfName
		result := w.genName(tt.name)
		if result != tt.want {
			t.Errorf("bad result, input: name=%s wfName=%s wfId=%s; got: %s; want: %s", tt.name, tt.wfName, tt.wfID, result, tt.want)
		}
		if len(result) > 64 {
			t.Errorf("result > 64 characters, input: name=%s wfName=%s wfId=%s; got: %s", tt.name, tt.wfName, tt.wfID, result)
		}
	}
}

func TestNewFromFileError(t *testing.T) {
	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	tf := filepath.Join(td, "test.workflow")

	localDNEErr := "open %s/sub.workflow: no such file or directory"
	if runtime.GOOS == "windows" {
		localDNEErr = "open %s\\sub.workflow: The system cannot find the file specified."
	}
	tests := []struct{ data, error string }{
		{
			`{"test":["1", "2",]}`,
			tf + ": JSON syntax error in line 1: invalid character ']' looking for beginning of value \n{\"test\":[\"1\", \"2\",]}\n                  ^",
		},
		{
			`{"test":{"key1":"value1" "key2":"value2"}}`,
			tf + ": JSON syntax error in line 1: invalid character '\"' after object key:value pair \n{\"test\":{\"key1\":\"value1\" \"key2\":\"value2\"}}\n                         ^",
		},
		{
			`{"test": value}`,
			tf + ": JSON syntax error in line 1: invalid character 'v' looking for beginning of value \n{\"test\": value}\n         ^",
		},
		{
			`{"test": "value"`,
			tf + ": JSON syntax error in line 1: unexpected end of JSON input \n{\"test\": \"value\"\n               ^",
		},
		{
			"{\n\"test\":[\"1\", \"2\",],\n\"test2\":[\"1\", \"2\"]\n}",
			tf + ": JSON syntax error in line 2: invalid character ']' looking for beginning of value \n\"test\":[\"1\", \"2\",],\n                 ^",
		},
		{
			`{"steps": {"somename": {"subWorkflow": {"path": "sub.workflow"}}}}`,
			fmt.Sprintf(localDNEErr, td),
		},
	}

	for _, tt := range tests {
		if err := ioutil.WriteFile(tf, []byte(tt.data), 0600); err != nil {
			t.Fatalf("error creating json file: %v", err)
		}

		if _, err := NewFromFile(context.Background(), tf); err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != tt.error {
			t.Errorf("did not get expected error from NewFromFile():\ngot: %q\nwant: %q", err.Error(), tt.error)
		}
	}
}

func TestNewFromFile(t *testing.T) {
	got, err := NewFromFile(context.Background(), "./test.workflow")
	if err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	subGot := got.Steps["sub-workflow"].SubWorkflow.workflow

	wantOAuthPath := filepath.Join(wd, "somefile")
	want := &Workflow{
		id:          got.id,
		workflowDir: wd,
		Name:        "some-name",
		Project:     "some-project",
		Zone:        "us-central1-a",
		GCSPath:     "gs://some-bucket/images",
		OAuthPath:   wantOAuthPath,
		Vars: map[string]string{
			"bootstrap_instance_name": "bootstrap",
			"machine_type":            "n1-standard-1",
		},
		Steps: map[string]*Step{
			"create-disks": {
				name: "create-disks",
				CreateDisks: &CreateDisks{
					{
						Name:        "bootstrap",
						SourceImage: "projects/windows-cloud/global/images/family/windows-server-2016-core",
						SizeGB:      "50",
						Type:        "pd-ssd",
					},
					{
						Name:        "image",
						SourceImage: "projects/windows-cloud/global/images/family/windows-server-2016-core",
						SizeGB:      "50",
						Type:        "pd-standard",
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
			"${bootstrap_instance_name}-stopped": {
				name:                   "${bootstrap_instance_name}-stopped",
				Timeout:                "1h",
				WaitForInstancesSignal: &WaitForInstancesSignal{{Name: "${bootstrap_instance_name}", Stopped: true, Interval: "1s"}},
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
			"postinstall-stopped": {
				name: "postinstall-stopped",
				WaitForInstancesSignal: &WaitForInstancesSignal{{Name: "postinstall", Stopped: true}},
			},
			"create-image": {
				name:         "create-image",
				CreateImages: &CreateImages{{Name: "image-from-disk", SourceDisk: "image"}},
			},
			"sub-workflow": {
				name: "sub-workflow",
				SubWorkflow: &SubWorkflow{
					Path: "./test_sub.workflow",
					workflow: &Workflow{
						id:          subGot.id,
						workflowDir: wd,
						Steps: map[string]*Step{
							"create-disks": {
								name: "create-disks",
								CreateDisks: &CreateDisks{
									{
										Name:        "bootstrap",
										SourceImage: "projects/windows-cloud/global/images/family/windows-server-2016-core",
										SizeGB:      "50",
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
							"bootstrap-stopped": {
								name:    "bootstrap-stopped",
								Timeout: "1h",
								WaitForInstancesSignal: &WaitForInstancesSignal{
									{
										Name: "bootstrap",
										SerialOutput: &SerialOutput{
											Port: 1, SuccessMatch: "complete", FailureMatch: "fail",
										},
									},
								},
							},
						},
						Dependencies: map[string][]string{
							"bootstrap":         {"create-disks"},
							"bootstrap-stopped": {"bootstrap"},
						},
					},
				},
			},
		},
		Dependencies: map[string][]string{
			"create-disks":        {},
			"bootstrap":           {"create-disks"},
			"bootstrap-stopped":   {"bootstrap"},
			"postinstall":         {"bootstrap-stopped"},
			"postinstall-stopped": {"postinstall"},
			"create-image":        {"postinstall-stopped"},
			"sub-workflow":        {"create-image"},
		},
	}

	// Check that subworkflow has workflow as parent.
	if subGot.parent != got {
		t.Error("subworkflow does not point to parent workflow")
	}

	// Fix pretty.Compare recursion freak outs.
	got.Ctx = nil
	got.Cancel = nil
	for _, s := range got.Steps {
		s.w = nil
	}
	subGot.Ctx = nil
	subGot.Cancel = nil
	subGot.parent = nil
	for _, s := range subGot.Steps {
		s.w = nil
	}

	// Cleanup hooks are impossible to check right now.
	got.cleanupHooks = nil
	subGot.cleanupHooks = nil

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

	cu, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}

	got := &Workflow{
		Name:         "${wf-name}",
		GCSPath:      "gs://${bucket}/images",
		Zone:         "parent-zone",
		Project:      "parent-project",
		OAuthPath:    tf,
		RequiredVars: []string{"bucket"},
		Vars: map[string]string{
			"bucket":    "parent-bucket",
			"step_name": "parent-step1",
			"timeout":   "60m",
			"path":      "./test_sub.workflow",
			"wf-name":   "parent",
		},
		Steps: map[string]*Step{
			"${step_name}": {
				Timeout: "${timeout}",
				CreateImages: &CreateImages{
					{SourceFile: "${SOURCESPATH}/image_file"},
				},
			},
			"${NAME}-step2": {
				WaitForInstancesSignal: &WaitForInstancesSignal{
					{Name: "blah", Stopped: true},
				},
			},
			"${NAME}-step3": {
				SubWorkflow: &SubWorkflow{
					Path: "${path}",
					Vars: map[string]string{
						"overridden": "bar",
					},
					workflow: &Workflow{
						Name:      "${wf-name}",
						GCSPath:   "gs://sub-bucket/images",
						Project:   "sub-project",
						Zone:      "sub-zone",
						OAuthPath: "sub-oauth-path",
						Steps: map[string]*Step{
							"${step_name}": {
								Timeout: "${timeout}",
							},
						},
						Vars: map[string]string{
							"wf-name":    "sub",
							"step_name":  "sub-step1",
							"timeout":    "60m",
							"overridden": "foo", // This should be changed to "bar" by populate().
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
	got.logger = nil
	subGot.ComputeClient = nil
	subGot.StorageClient = nil
	subGot.logger = nil

	// For simplicity, here is the subworkflow scratch path.
	// The subworkflow scratch path is a subdir of the parent workflow scratch path.
	subScratch := subGot.scratchPath

	want := &Workflow{
		Name:         "parent",
		GCSPath:      "gs://parent-bucket/images",
		Zone:         "parent-zone",
		Project:      "parent-project",
		OAuthPath:    tf,
		id:           got.id,
		Ctx:          got.Ctx,
		Cancel:       got.Cancel,
		RequiredVars: []string{"bucket"},
		Vars: map[string]string{
			"bucket":    "parent-bucket",
			"step_name": "parent-step1",
			"timeout":   "60m",
			"path":      "./test_sub.workflow",
			"wf-name":   "parent",
		},
		bucket:      "parent-bucket",
		scratchPath: got.scratchPath,
		sourcesPath: fmt.Sprintf("%s/sources", got.scratchPath),
		logsPath:    fmt.Sprintf("%s/logs", got.scratchPath),
		outsPath:    fmt.Sprintf("%s/outs", got.scratchPath),
		username:    cu.Username,
		Steps: map[string]*Step{
			"parent-step1": {
				name:    "parent-step1",
				Timeout: "60m",
				timeout: time.Duration(60 * time.Minute),
				CreateImages: &CreateImages{
					{SourceFile: fmt.Sprintf("gs://parent-bucket/%s/sources/image_file", got.scratchPath)},
				},
			},
			"parent-step2": {
				name:    "parent-step2",
				Timeout: "10m",
				timeout: time.Duration(10 * time.Minute),
				WaitForInstancesSignal: &WaitForInstancesSignal{
					{Name: "blah", Stopped: true, interval: 5 * time.Second},
				},
			},
			"parent-step3": {
				name:    "parent-step3",
				Timeout: "10m",
				timeout: time.Duration(10 * time.Minute),
				SubWorkflow: &SubWorkflow{
					Path: "./test_sub.workflow",
					Vars: map[string]string{
						"overridden": "bar",
					},
					workflow: &Workflow{
						Name:      "parent-step3",
						GCSPath:   fmt.Sprintf("gs://%s/%s", got.bucket, got.scratchPath),
						Zone:      "parent-zone",
						Project:   "parent-project",
						OAuthPath: tf,
						id:        subGot.id,
						Steps: map[string]*Step{
							"sub-step1": {
								name:    "sub-step1",
								Timeout: "60m",
								timeout: time.Duration(60 * time.Minute),
							},
						},
						Vars: map[string]string{
							"wf-name":    "sub",
							"step_name":  "sub-step1",
							"timeout":    "60m",
							"overridden": "bar", // Check that this changed from "foo" to "bar".
						},
						bucket:      "parent-bucket",
						scratchPath: subScratch,
						sourcesPath: fmt.Sprintf("%s/sources", subScratch),
						logsPath:    fmt.Sprintf("%s/logs", subScratch),
						outsPath:    fmt.Sprintf("%s/outs", subScratch),
						username:    cu.Username,
					},
				},
			},
		},
	}

	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("parsed workflow does not match expectation: (-got +want)\n%s", diff)
	}

	got.RequiredVars = []string{"required-var"}
	got.Vars = map[string]string{"required-var": ""}
	got.GCSPath = "${required-var}"
	wantErr := `required var "required-var" cannot be blank`
	if err := got.populate(); err.Error() != wantErr {
		t.Errorf("workflow with unsubbed required var bad error, want: %q got: %q", wantErr, err.Error())
	}
}

func testTraverseWorkflow(mockRun func(i int) func(*Step) error) *Workflow {
	// s0---->s1---->s3
	//   \         /
	//    --->s2---
	// s4
	w := testWorkflow()
	w.Steps = map[string]*Step{
		"s0": {name: "s0", testType: &mockStep{runImpl: mockRun(0)}, w: w},
		"s1": {name: "s1", testType: &mockStep{runImpl: mockRun(1)}, w: w},
		"s2": {name: "s2", testType: &mockStep{runImpl: mockRun(2)}, w: w},
		"s3": {name: "s3", testType: &mockStep{runImpl: mockRun(3)}, w: w},
		"s4": {name: "s4", testType: &mockStep{runImpl: mockRun(4)}, w: w},
	}
	w.Dependencies = map[string][]string{
		"s1": {"s0"},
		"s2": {"s0"},
		"s3": {"s1", "s2"},
	}
	return w
}

func TestTraverseDAG(t *testing.T) {
	var callOrder []int
	errs := make([]error, 5)
	var rw sync.Mutex
	mockRun := func(i int) func(*Step) error {
		return func(_ *Step) error {
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
	w := testTraverseWorkflow(mockRun)
	if err := w.Run(); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if err := checkCallOrder(); err != nil {
		t.Errorf("call order error: %s", err)
	}

	callOrder = []int{}
	errs = make([]error, 5)

	// s2 failure.
	w = testTraverseWorkflow(mockRun)
	errs[2] = errors.New("failure")
	want := w.Steps["s2"].wrapRunError(errs[2])
	if err := w.Run(); err.Error() != want.Error() {
		t.Errorf("unexpected error: %s != %s", err, want)
	}
	if err := checkCallOrder(); err != nil {
		t.Errorf("call order error: %s", err)
	}
}

func TestPrint(t *testing.T) {
	data := []byte(`{
"name": "some-name",
"project": "some-project",
"zone": "us-central1-a",
"gcsPath": "gs://some-bucket/images",
"vars": {
  "instance_name": "step1",
  "machine_type": "n1-standard-1"
},
"steps": {
  "${instance_name}Run": {
    "createInstances": [
      {
        "name": "${instance_name}",
        "attachedDisks": ["disk"],
        "machineType": "${machine_type}"
      }
    ]
  }
}
}`)

	want := `{
  "Name": "some-name",
  "Project": "some-project",
  "Zone": "us-central1-a",
  "GCSPath": "gs://some-bucket/images",
  "Vars": {
    "instance_name": "step1",
    "machine_type": "n1-standard-1"
  },
  "Steps": {
    "step1Run": {
      "Timeout": "10m",
      "CreateInstances": [
        {
          "Name": "step1",
          "AttachedDisks": [
            "disk"
          ],
          "MachineType": "n1-standard-1",
          "NoCleanup": false,
          "ExactName": false
        }
      ]
    }
  },
  "Dependencies": null
}
`

	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	tf := filepath.Join(td, "test.workflow")
	ioutil.WriteFile(tf, data, 0600)

	got, err := NewFromFile(context.Background(), tf)
	if err != nil {
		t.Fatal(err)
	}

	got.ComputeClient = testGCEClient
	got.StorageClient = testGCSClient

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	got.Print()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	if diff := diff.Diff(buf.String(), want); diff != "" {
		t.Errorf("printed workflow does not match expectation: (-got +want)\n%s", diff)
	}
}

func testValidateErrors(w *Workflow, want string) error {
	if err := w.Validate(); err == nil {
		return errors.New("expected error, got nil")
	} else if err.Error() != want {
		return fmt.Errorf("did not get expected error from Validate():\ngot: %q\nwant: %q", err.Error(), want)
	}
	select {
	case <-w.Cancel:
		return nil
	default:
		return errors.New("expected cancel to be closed after error")
	}
}

func TestValidateErrors(t *testing.T) {
	// Error from validateRequiredFields().
	w := testWorkflow()
	w.Name = "1"
	want := "error validating workflow: workflow field 'Name' must start with a letter and only contain letters, numbers, and hyphens"
	if err := testValidateErrors(w, want); err != nil {
		t.Error(err)
	}

	// Error from populate().
	w = testWorkflow()
	w.Steps = map[string]*Step{"s0": {Timeout: "10", testType: &mockStep{}}}
	want = "error populating workflow: time: missing unit in duration 10"
	if err := testValidateErrors(w, want); err != nil {
		t.Error(err)
	}

	// Error from validate().
	w = testWorkflow()
	w.Steps = map[string]*Step{"s0": {testType: &mockStep{}}}
	w.Project = "${var}"
	want = "Unresolved var \"${var}\" found in \"${var}\""
	if err := testValidateErrors(w, want); err != nil {
		t.Error(err)
	}
}

func TestWrite(t *testing.T) {
	var buf bytes.Buffer
	testBucket := "bucket"
	testObject := "object"
	var gotObj string
	var gotBkt string
	nameRgx := regexp.MustCompile(`"name":"([^"].*)"`)
	uploadRgx := regexp.MustCompile(`/b/([^/]+)/o?.*uploadType=multipart.*`)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.String()
		m := r.Method
		if match := uploadRgx.FindStringSubmatch(u); m == "POST" && match != nil {
			body, _ := ioutil.ReadAll(r.Body)
			buf.Write(body)
			gotObj = nameRgx.FindStringSubmatch(string(body))[1]
			gotBkt = match[1]
			fmt.Fprintf(w, `{"kind":"storage#object","bucket":"%s","name":"%s"}`, gotBkt, gotObj)
		}

	}))

	gcsClient, err := storage.NewClient(context.Background(), option.WithEndpoint(ts.URL), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		t.Fatal(err)
	}
	l := gcsLogger{
		client: gcsClient,
		bucket: testBucket,
		object: testObject,
		ctx:    context.Background(),
	}

	tests := []struct {
		test, want string
	}{
		{"test log 1\n", "test log 1\n"},
		{"test log 2\n", "test log 1\ntest log 2\n"},
	}

	for _, tt := range tests {
		l.Write([]byte(tt.test))
		if gotObj != testObject {
			t.Errorf("object does not match, want: %q, got: %q", testObject, gotObj)
		}
		if gotBkt != testBucket {
			t.Errorf("bucket does not match, want: %q, got: %q", testBucket, gotBkt)
		}
		if !strings.Contains(buf.String(), tt.want) {
			t.Errorf("expected text did not get sent to GCS, want: %q, got: %q", tt.want, buf.String())
		}
		if l.buf.String() != tt.want {
			t.Errorf("buffer does mot match expectation, want: %q, got: %q", tt.want, l.buf.String())
		}
	}
}
