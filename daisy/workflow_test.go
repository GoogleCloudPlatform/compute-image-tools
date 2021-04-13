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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

func TestAddDependency(t *testing.T) {
	w := &Workflow{}
	a, _ := w.NewStep("a")
	b, _ := w.NewStep("b")

	otherW := &Workflow{}
	c, _ := otherW.NewStep("c")

	tests := []struct {
		desc      string
		in1, in2  *Step
		shouldErr bool
	}{
		{"good case", a, b, false},
		{"idempotent good case", a, b, false},
		{"bad case 1", a, c, true},
		{"bad case 2", c, b, true},
	}

	for _, tt := range tests {
		if err := w.AddDependency(tt.in1, tt.in2); err == nil && tt.shouldErr {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if err != nil && !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}

	wantDeps := map[string][]string{"a": {"b"}}
	if diffRes := diff(w.Dependencies, wantDeps, 0); diffRes != "" {
		t.Errorf("incorrect dependencies: (-got,+want)\n%s", diffRes)
	}
}

func TestDaisyBkt(t *testing.T) {
	client, err := newTestGCSClient()
	if err != nil {
		t.Fatal(err)
	}
	project := "foo-project"
	got, err := daisyBkt(context.Background(), client, project)
	if err != nil {
		t.Fatal(err)
	}
	want := project + "-daisy-bkt"
	if got != project+"-daisy-bkt" {
		t.Errorf("bucket does not match, got: %q, want: %q", got, want)
	}

	project = "bar-project"
	got, err = daisyBkt(context.Background(), client, project)
	if err != nil {
		t.Fatal(err)
	}
	want = project + "-daisy-bkt"
	if got != project+"-daisy-bkt" {
		t.Errorf("bucket does not match, got: %q, want: %q", got, want)
	}
}

func TestCleanup(t *testing.T) {
	cleanedup1 := false
	cleanedup2 := false
	cleanup1 := func() DError {
		cleanedup1 = true
		return nil
	}
	cleanup2 := func() DError {
		cleanedup2 = true
		return nil
	}
	cleanupFail := func() DError {
		return Errf("failed cleanup")
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
			t.Errorf("bad result, i: name=%s wfName=%s wfId=%s; got: %s; want: %s", tt.name, tt.wfName, tt.wfID, result, tt.want)
		}
		if len(result) > 64 {
			t.Errorf("result > 64 characters, i: name=%s wfName=%s wfId=%s; got: %s", tt.name, tt.wfName, tt.wfID, result)
		}
	}
}

func TestGetSourceGCSAPIPath(t *testing.T) {
	w := testWorkflow()
	w.sourcesPath = "my/sources"
	got := w.getSourceGCSAPIPath("foo")
	want := "https://storage.cloud.google.com/my/sources/foo"
	if got != want {
		t.Errorf("unexpected result: got: %q, want %q", got, want)
	}
}

func TestCancelWorkflow_IsIdempotent(t *testing.T) {
	w := testWorkflow()
	if w.isCanceled {
		t.Error("Didn't expect workflow to be canceled.")
	}
	w.CancelWorkflow()
	w.CancelWorkflow()
	if !w.isCanceled {
		t.Error("Expect workflow to be canceled.")
	}
}

func TestCancelWithReason_IsCallableMultipleTimes_AndKeepsFirstCancelReason(t *testing.T) {
	w := testWorkflow()
	reason1 := "reason1"
	reason2 := "reason2"
	w.CancelWithReason(reason1)
	w.CancelWithReason(reason2)
	if !w.isCanceled {
		t.Error("Expect workflow to be canceled.")
	}
	if w.getCancelReason() != reason1 {
		t.Errorf("Expected reason1 mismatch. got=%q, want=%q", w.getCancelReason(), reason1)
	}
}

func TestCancelWorkflow_RecoversFromManuallyClosedChannel(t *testing.T) {
	w := testWorkflow()
	if w.isCanceled {
		t.Error("Didn't expect workflow to be canceled.")
	}
	close(w.Cancel)
	w.CancelWorkflow()
	if !w.isCanceled {
		t.Error("Expect workflow to be canceled.")
	}
}

func TestNewFromFileError(t *testing.T) {
	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	tf := filepath.Join(td, "test.wf.json")

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
	}

	for i, tt := range tests {
		if err := ioutil.WriteFile(tf, []byte(tt.data), 0600); err != nil {
			t.Fatalf("error creating json file: %v", err)
		}

		if _, err := NewFromFile(tf); err == nil {
			t.Errorf("expected error, got nil for test %d", i+1)
		} else if err.Error() != tt.error {
			t.Errorf("did not get expected error from NewFromFile():\ngot: %q\nwant: %q", err.Error(), tt.error)
		}
	}
}

func TestNewFromFile(t *testing.T) {
	got, derr := NewFromFile("./test_data/test.wf.json")
	if derr != nil {
		t.Fatal(derr)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	want := New()
	// These are difficult to validate and irrelevant, so we cheat.
	want.id = got.ID()
	want.Cancel = got.Cancel
	want.cleanupHooks = got.cleanupHooks
	want.disks = newDiskRegistry(want)
	want.images = newImageRegistry(want)
	want.machineImages = newMachineImageRegistry(want)
	want.instances = newInstanceRegistry(want)
	want.networks = newNetworkRegistry(want)

	want.workflowDir = filepath.Join(wd, "test_data")
	want.Name = "some-name"
	want.Project = "some-project"
	want.Zone = "us-central1-a"
	want.GCSPath = "gs://some-bucket/images"
	want.OAuthPath = filepath.Join(wd, "test_data", "somefile")
	want.Sources = map[string]string{}
	want.autovars = map[string]string{}
	want.Vars = map[string]Var{
		"bootstrap_instance_name": {Value: "bootstrap-${NAME}", Required: true},
		"machine_type":            {Value: "n1-standard-1"},
		"key1":                    {Value: "var1"},
		"key2":                    {Value: "var2"},
	}
	want.Steps = map[string]*Step{
		"create-disks": {
			name: "create-disks",
			CreateDisks: &CreateDisks{
				{
					Disk: compute.Disk{
						Name:        "bootstrap",
						SourceImage: "projects/windows-cloud/global/images/family/windows-server-2016-core",
						Type:        "pd-ssd",
					},
					SizeGb: "50",
				},
				{
					Disk: compute.Disk{
						Name:        "image",
						SourceImage: "projects/windows-cloud/global/images/family/windows-server-2016-core",
						Type:        "pd-standard",
					},
					SizeGb: "50",
				},
			},
		},
		"${bootstrap_instance_name}": {
			name: "${bootstrap_instance_name}",
			CreateInstances: &CreateInstances{
				Instances: []*Instance{
					{
						Instance: compute.Instance{
							Name:        "${bootstrap_instance_name}",
							Disks:       []*compute.AttachedDisk{{Source: "bootstrap"}, {Source: "image"}},
							MachineType: "${machine_type}",
						},
						InstanceBase: InstanceBase{
							StartupScript: "shutdown /h",
							Scopes:        []string{"scope1", "scope2"},
						},
						Metadata: map[string]string{"test_metadata": "this was a test"},
					},
				},
				InstancesAlpha: []*InstanceAlpha{
					{
						Instance: computeAlpha.Instance{
							Name:        "${bootstrap_instance_name}",
							Disks:       []*computeAlpha.AttachedDisk{{Source: "bootstrap"}, {Source: "image"}},
							MachineType: "${machine_type}",
						},
						InstanceBase: InstanceBase{
							StartupScript: "shutdown /h",
							Scopes:        []string{"scope1", "scope2"},
						},
						Metadata: map[string]string{"test_metadata": "this was a test"},
					},
				},
				InstancesBeta: []*InstanceBeta{
					{
						Instance: computeBeta.Instance{
							Name:        "${bootstrap_instance_name}",
							Disks:       []*computeBeta.AttachedDisk{{Source: "bootstrap"}, {Source: "image"}},
							MachineType: "${machine_type}",
						},
						InstanceBase: InstanceBase{
							StartupScript: "shutdown /h",
							Scopes:        []string{"scope1", "scope2"},
						},
						Metadata: map[string]string{"test_metadata": "this was a test"},
					},
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
				Instances: []*Instance{
					{
						Instance: compute.Instance{
							Name:        "postinstall",
							Disks:       []*compute.AttachedDisk{{Source: "image"}, {Source: "bootstrap"}},
							MachineType: "${machine_type}",
						},
						InstanceBase: InstanceBase{
							StartupScript: "shutdown /h",
							Scopes:        []string{"scope3", "scope4"},
						},
					},
					{
						Instance: compute.Instance{
							Name:        "postinstallBeta",
							MachineType: "${machine_type}",
						},
					},
				},
				InstancesAlpha: []*InstanceAlpha{
					{
						Instance: computeAlpha.Instance{
							Name:        "postinstall",
							Disks:       []*computeAlpha.AttachedDisk{{Source: "image"}, {Source: "bootstrap"}},
							MachineType: "${machine_type}",
						},
						InstanceBase: InstanceBase{
							StartupScript: "shutdown /h",
							Scopes:        []string{"scope3", "scope4"},
						},
					},
					{
						Instance: computeAlpha.Instance{
							Name:               "postinstallBeta",
							MachineType:        "${machine_type}",
							SourceMachineImage: "source-machine-image",
						},
					},
				},
				InstancesBeta: []*InstanceBeta{
					{
						Instance: computeBeta.Instance{
							Name:        "postinstall",
							Disks:       []*computeBeta.AttachedDisk{{Source: "image"}, {Source: "bootstrap"}},
							MachineType: "${machine_type}",
						},
						InstanceBase: InstanceBase{
							StartupScript: "shutdown /h",
							Scopes:        []string{"scope3", "scope4"},
						},
					},
					{
						Instance: computeBeta.Instance{
							Name:               "postinstallBeta",
							MachineType:        "${machine_type}",
							SourceMachineImage: "source-machine-image",
						},
					},
				},
			},
		},
		"postinstall-stopped": {
			name:                   "postinstall-stopped",
			WaitForInstancesSignal: &WaitForInstancesSignal{{Name: "postinstall", Stopped: true}},
		},
		"create-image-locality": {
			name: "create-image-locality",
			CreateImages: &CreateImages{
				Images: []*Image{{
					Image: compute.Image{Name: "image-from-local-disk", SourceDisk: "local-image", StorageLocations: []string{"europe-west1"}, Description: "Some Ubuntu", Family: "ubuntu-1404"},
					ImageBase: ImageBase{OverWrite: false,
						Resource: Resource{Project: "a_project", NoCleanup: true, ExactName: false},
					},
					GuestOsFeatures: []string{"VIRTIO_SCSI_MULTIQUEUE", "UBUNTU", "MULTI_IP_SUBNET"},
				}},
				ImagesAlpha: []*ImageAlpha{{
					Image: computeAlpha.Image{Name: "image-from-local-disk", SourceDisk: "local-image", StorageLocations: []string{"europe-west1"}, Description: "Some Ubuntu", Family: "ubuntu-1404"},
					ImageBase: ImageBase{OverWrite: false,
						Resource: Resource{Project: "a_project", NoCleanup: true, ExactName: false},
					},
					GuestOsFeatures: []string{"VIRTIO_SCSI_MULTIQUEUE", "UBUNTU", "MULTI_IP_SUBNET"},
				}},
				ImagesBeta: []*ImageBeta{{
					Image: computeBeta.Image{Name: "image-from-local-disk", SourceDisk: "local-image", StorageLocations: []string{"europe-west1"}, Description: "Some Ubuntu", Family: "ubuntu-1404"},
					ImageBase: ImageBase{OverWrite: false,
						Resource: Resource{Project: "a_project", NoCleanup: true, ExactName: false},
					},
					GuestOsFeatures: []string{"VIRTIO_SCSI_MULTIQUEUE", "UBUNTU", "MULTI_IP_SUBNET"},
				}},
			},
		},
		"create-image": {
			name: "create-image",
			CreateImages: &CreateImages{
				Images: []*Image{{
					Image: compute.Image{Name: "image-from-disk", SourceDisk: "image", Description: "Microsoft, SQL Server 2016 Web, on Windows Server 2019", Family: "sql-web-2016-win-2019"},
					ImageBase: ImageBase{OverWrite: true,
						Resource: Resource{Project: "a_project", NoCleanup: true, ExactName: true},
					},
					GuestOsFeatures: []string{"VIRTIO_SCSI_MULTIQUEUE", "WINDOWS", "MULTI_IP_SUBNET"},
				}},
				ImagesAlpha: []*ImageAlpha{{
					Image: computeAlpha.Image{Name: "image-from-disk", SourceDisk: "image", Description: "Microsoft, SQL Server 2016 Web, on Windows Server 2019", Family: "sql-web-2016-win-2019"},
					ImageBase: ImageBase{OverWrite: true,
						Resource: Resource{Project: "a_project", NoCleanup: true, ExactName: true},
					},
					GuestOsFeatures: []string{"VIRTIO_SCSI_MULTIQUEUE", "WINDOWS", "MULTI_IP_SUBNET"},
				}},
				ImagesBeta: []*ImageBeta{{
					Image: computeBeta.Image{Name: "image-from-disk", SourceDisk: "image", Description: "Microsoft, SQL Server 2016 Web, on Windows Server 2019", Family: "sql-web-2016-win-2019"},
					ImageBase: ImageBase{OverWrite: true,
						Resource: Resource{Project: "a_project", NoCleanup: true, ExactName: true},
					},
					GuestOsFeatures: []string{"VIRTIO_SCSI_MULTIQUEUE", "WINDOWS", "MULTI_IP_SUBNET"},
				}},
			},
		},
		"create-image-guest-os-features-compute-api": {
			name: "create-image-guest-os-features-compute-api",
			CreateImages: &CreateImages{
				Images: []*Image{{
					Image: compute.Image{Name: "image-from-disk", SourceDisk: "image", Description: "GuestOS Features Compute API", Family: "guest-os"},
					ImageBase: ImageBase{OverWrite: true,
						Resource: Resource{Project: "a_project", NoCleanup: true, ExactName: true},
					},
					GuestOsFeatures: []string{"VIRTIO_SCSI_MULTIQUEUE", "WINDOWS", "MULTI_IP_SUBNET"},
				}},
				ImagesAlpha: []*ImageAlpha{{
					Image: computeAlpha.Image{Name: "image-from-disk", SourceDisk: "image", Description: "GuestOS Features Compute API", Family: "guest-os"},
					ImageBase: ImageBase{OverWrite: true,
						Resource: Resource{Project: "a_project", NoCleanup: true, ExactName: true},
					},
					GuestOsFeatures: []string{"VIRTIO_SCSI_MULTIQUEUE", "WINDOWS", "MULTI_IP_SUBNET"},
				}},
				ImagesBeta: []*ImageBeta{{
					Image: computeBeta.Image{Name: "image-from-disk", SourceDisk: "image", Description: "GuestOS Features Compute API", Family: "guest-os"},
					ImageBase: ImageBase{OverWrite: true,
						Resource: Resource{Project: "a_project", NoCleanup: true, ExactName: true},
					},
					GuestOsFeatures: []string{"VIRTIO_SCSI_MULTIQUEUE", "WINDOWS", "MULTI_IP_SUBNET"},
				}},
			},
		},
		"create-machine-image": {
			name: "create-machine-image",
			CreateMachineImages: &CreateMachineImages{
				{MachineImage: computeBeta.MachineImage{
					Name:             "machine-image-from-instance",
					SourceInstance:   "source-instance",
					StorageLocations: []string{"eu", "us-west2"},
				}},
			},
		},
		"include-workflow": {
			name: "include-workflow",
			IncludeWorkflow: &IncludeWorkflow{
				Vars: map[string]string{
					"key": "value",
				},
				Path: "./test_sub.wf.json",
			},
		},
		"sub-workflow": {
			name: "sub-workflow",
			SubWorkflow: &SubWorkflow{
				Vars: map[string]string{
					"key": "value",
				},
				Path: "./test_sub.wf.json",
			},
		},
	}
	want.Dependencies = map[string][]string{
		"create-disks":          {},
		"bootstrap":             {"create-disks"},
		"bootstrap-stopped":     {"bootstrap"},
		"postinstall":           {"bootstrap-stopped"},
		"postinstall-stopped":   {"postinstall"},
		"create-image-locality": {"postinstall-stopped"},
		"create-image":          {"create-image-locality"},
		"create-machine-image":  {"create-image"},
		"include-workflow":      {"create-image"},
		"sub-workflow":          {"create-image"},
	}

	for _, s := range want.Steps {
		s.w = want
	}

	if diffRes := diff(got, want, 0); diffRes != "" {
		t.Errorf("parsed workflow does not match expectation: (-got +want)\n%s", diffRes)
	}
}

func TestNewStep(t *testing.T) {
	w := &Workflow{}

	s, err := w.NewStep("s")
	wantS := &Step{name: "s", w: w}
	if s == nil || s.name != "s" || s.w != w {
		t.Errorf("step does not meet expectation: got: %v, want: %v", s, wantS)
	}
	if err != nil {
		t.Error("unexpected error when creating new step")
	}

	s, err = w.NewStep("s")
	if s != nil {
		t.Errorf("step should not have been created: %v", s)
	}
	if err == nil {
		t.Error("should have erred, but didn't")
	}
}

func TestPopulate(t *testing.T) {
	ctx := context.Background()
	client, err := newTestGCSClient()
	if err != nil {
		t.Fatal(err)
	}
	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	tf := filepath.Join(td, "test.cred")
	if err := ioutil.WriteFile(tf, []byte(`{ "type": "service_account" }`), 0600); err != nil {
		t.Fatalf("error creating temp file: %v", err)
	}

	called := false
	var stepPopErr DError
	stepPop := func(ctx context.Context, s *Step) DError {
		called = true
		return stepPopErr
	}

	got := New()
	got.Name = "${wf_name}"
	got.Zone = "wf-zone"
	got.Project = "bar-project"
	got.OAuthPath = tf
	got.Logger = &MockLogger{}
	got.Vars = map[string]Var{
		"bucket":    {Value: "wf-bucket", Required: true},
		"step_name": {Value: "step1"},
		"timeout":   {Value: "60m"},
		"path":      {Value: "./test_sub.wf.json"},
		"wf_name":   {Value: "wf-name"},
		"test-var":  {Value: "${ZONE}-this-should-populate-${NAME}"},
	}
	got.Steps = map[string]*Step{
		"${NAME}-${step_name}": {
			w:       got,
			Timeout: "${timeout}",
			testType: &mockStep{
				populateImpl: stepPop,
			},
		},
	}
	got.StorageClient = client
	got.externalLogging = true

	if err := got.populate(ctx); err != nil {
		t.Fatalf("error populating workflow: %v", err)
	}

	want := New()
	// These are difficult to validate and irrelevant, so we cheat.
	want.id = got.id
	want.Cancel = got.Cancel
	want.cleanupHooks = got.cleanupHooks
	want.StorageClient = got.StorageClient
	want.cloudLoggingClient = got.cloudLoggingClient
	want.Logger = got.Logger
	want.disks = newDiskRegistry(want)
	want.images = newImageRegistry(want)
	want.machineImages = newMachineImageRegistry(want)
	want.instances = newInstanceRegistry(want)
	want.networks = newNetworkRegistry(want)

	want.Name = "wf-name"
	want.GCSPath = "gs://bar-project-daisy-bkt"
	want.Zone = "wf-zone"
	want.Project = "bar-project"
	want.OAuthPath = tf
	want.externalLogging = true
	want.Sources = map[string]string{}
	want.DefaultTimeout = defaultTimeout
	want.defaultTimeout = 10 * time.Minute
	want.Vars = map[string]Var{
		"bucket":    {Value: "wf-bucket", Required: true},
		"step_name": {Value: "step1"},
		"timeout":   {Value: "60m"},
		"path":      {Value: "./test_sub.wf.json"},
		"wf_name":   {Value: "wf-name"},
		"test-var":  {Value: "wf-zone-this-should-populate-wf-name"},
	}
	want.autovars = got.autovars
	want.bucket = "bar-project-daisy-bkt"
	want.scratchPath = got.scratchPath
	want.sourcesPath = fmt.Sprintf("%s/sources", got.scratchPath)
	want.logsPath = fmt.Sprintf("%s/logs", got.scratchPath)
	want.outsPath = fmt.Sprintf("%s/outs", got.scratchPath)
	want.username = got.username
	want.Steps = map[string]*Step{
		"wf-name-step1": {
			name:    "wf-name-step1",
			Timeout: "60m",
			timeout: time.Duration(60 * time.Minute),
			testType: &mockStep{
				populateImpl: stepPop,
			},
		},
	}
	want.Dependencies = map[string][]string{}

	for _, s := range want.Steps {
		s.w = want
	}

	if diffRes := diff(got, want, 0); diffRes != "" {
		t.Errorf("parsed workflow does not match expectation: (-got +want)\n%s", diffRes)
	}

	if !called {
		t.Error("did not call step's populate")
	}

	stepPopErr = Errf("error")
	wantErr := Errf("error populating step \"wf-name-step1\": %v", stepPopErr)
	if err := got.populate(ctx); err.Error() != wantErr.Error() {
		t.Errorf("did not get proper step populate error: %v != %v", err, wantErr)
	}
}

func TestRequiredVars(t *testing.T) {
	w := testWorkflow()

	tests := []struct {
		desc      string
		vars      map[string]Var
		shouldErr bool
	}{
		{"normal case", map[string]Var{"foo": {Value: "foo", Required: true, Description: "foo"}}, false},
		{"missing req case", map[string]Var{"foo": {Value: "", Required: true, Description: "foo"}}, true},
	}

	for _, tt := range tests {
		w.Vars = tt.vars
		err := w.populate(context.Background())
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have erred, but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func testTraverseWorkflow(mockRun func(i int) func(context.Context, *Step) DError) *Workflow {
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
	ctx := context.Background()
	var callOrder []int
	errs := make([]DError, 5)
	var rw sync.Mutex
	mockRun := func(i int) func(context.Context, *Step) DError {
		return func(_ context.Context, _ *Step) DError {
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
	if err := w.Run(ctx); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if err := checkCallOrder(); err != nil {
		t.Errorf("call order error: %s", err)
	}

	callOrder = []int{}
	errs = make([]DError, 5)

	// s2 failure.
	w = testTraverseWorkflow(mockRun)
	errs[2] = Errf("failure")
	want := w.Steps["s2"].wrapRunError(errs[2])
	if err := w.Run(ctx); err.Error() != want.Error() {
		t.Errorf("unexpected error: %s != %s", err, want)
	}
	if err := checkCallOrder(); err != nil {
		t.Errorf("call order error: %s", err)
	}
}

func TestForceCleanupSetOnRunError(t *testing.T) {
	doTestForceCleanup(t, true, true, true)
}

func TestForceCleanupNotSetOnRunErrorWhenForceCleanupFalse(t *testing.T) {
	doTestForceCleanup(t, true, false, false)
}

func TestForceCleanupNotSetOnNoErrorWhenForceCleanupTrue(t *testing.T) {
	doTestForceCleanup(t, false, true, false)
}

func TestForceCleanupNotSetOnNoErrorWhenForceCleanupFalse(t *testing.T) {
	doTestForceCleanup(t, false, false, false)
}

func doTestForceCleanup(t *testing.T, runErrorFromStep bool, forceCleanupOnError bool, forceCleanup bool) {
	mockRun := func(i int) func(context.Context, *Step) DError {
		return func(_ context.Context, _ *Step) DError {
			if runErrorFromStep {
				return Errf("failure")
			}
			return nil
		}
	}
	ctx := context.Background()
	w := testWorkflow()
	w.ForceCleanupOnError = forceCleanupOnError
	w.Steps = map[string]*Step{
		"s0": {name: "s0", testType: &mockStep{runImpl: mockRun(0)}, w: w},
	}

	if err := w.Run(ctx); (err != nil) != runErrorFromStep {
		if runErrorFromStep {
			t.Errorf("expected error from w.Run but nil received")
		} else {
			t.Errorf("expected no error from w.Run but %v received", err)
		}

	}
	if w.forceCleanup != forceCleanup {
		t.Errorf("w.forceCleanup should be set to %v but is %v", forceCleanup, w.forceCleanup)
	}
}

func TestPrint(t *testing.T) {
	data := []byte(`{
"Name": "some-name",
"Project": "some-project",
"Zone": "some-zone",
"GCSPath": "gs://some-bucket/images",
"Vars": {
  "instance_name": "i1",
  "machine_type": {"Value": "n1-standard-1", "Required": true}
},
"Steps": {
  "${instance_name}Delete": {
    "DeleteResources": {
      "Instances": ["${instance_name}"]
    }
  }
}
}`)

	want := `{
  "Name": "some-name",
  "Project": "some-project",
  "Zone": "some-zone",
  "GCSPath": "gs://some-bucket/images",
  "Vars": {
    "instance_name": {
      "Value": "i1"
    },
    "machine_type": {
      "Value": "n1-standard-1",
      "Required": true
    }
  },
  "Steps": {
    "i1Delete": {
      "Timeout": "10m",
      "DeleteResources": {
        "Instances": [
          "i1"
        ]
      }
    }
  },
  "DefaultTimeout": "10m",
  "ForceCleanupOnError": false
}
`

	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	tf := filepath.Join(td, "test.wf.json")
	ioutil.WriteFile(tf, data, 0600)

	got, err := NewFromFile(tf)
	if err != nil {
		t.Fatal(err)
	}

	got.ComputeClient, _ = newTestGCEClient()
	got.StorageClient, _ = newTestGCSClient()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	got.Print(context.Background())
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	if diffRes := diff(buf.String(), want, 0); diffRes != "" {
		t.Errorf("printed workflow does not match expectation: (-got +want)\n%s", diffRes)
	}
}

func testValidateErrors(w *Workflow, want string) error {
	if err := w.Validate(context.Background()); err == nil {
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
	want = "error populating workflow: error populating step \"s0\": time: missing unit in duration \"10\""
	if err := testValidateErrors(w, want); err != nil {
		t.Error(err)
	}

	// Error from validate().
	w = testWorkflow()
	w.Steps = map[string]*Step{"s0": {testType: &mockStep{}}}
	w.Project = "foo"
	want = "error validating workflow: bad project lookup: \"foo\", error: APIError: bad project"
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
	l := GCSLogger{
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

func TestRunStepTimeout(t *testing.T) {
	w := testWorkflow()
	s, _ := w.NewStep("test")
	s.timeout = 1 * time.Nanosecond
	s.testType = &mockStep{runImpl: func(ctx context.Context, s *Step) DError {
		time.Sleep(1 * time.Second)
		return nil
	}}
	want := `step "test" did not complete within the specified timeout of 1ns`
	if err := w.runStep(context.Background(), s); err == nil || err.Error() != want {
		t.Errorf("did not get expected error, got: %q, want: %q", err.Error(), want)
	}
}

func TestPopulateClients(t *testing.T) {
	w := testWorkflow()

	initialComputeClient := w.ComputeClient
	tryPopulateClients(t, w)
	if w.ComputeClient != initialComputeClient {
		t.Errorf("Should not repopulate compute client.")
	}

	w.ComputeClient = nil
	tryPopulateClients(t, w)
	if w.ComputeClient == nil {
		t.Errorf("Did not populate compute client.")
	}

	initialStorageClient := w.StorageClient
	tryPopulateClients(t, w)
	if w.StorageClient != initialStorageClient {
		t.Errorf("Should not repopulate storage client.")
	}

	w.StorageClient = nil
	tryPopulateClients(t, w)
	if w.StorageClient == nil {
		t.Errorf("Did not populate storage client.")
	}

	initialCloudLoggingClient := w.cloudLoggingClient
	tryPopulateClients(t, w)
	if w.cloudLoggingClient != initialCloudLoggingClient {
		t.Errorf("Should not repopulate logging client.")
	}

	w.cloudLoggingClient = nil
	w.externalLogging = false
	tryPopulateClients(t, w)
	if w.cloudLoggingClient != nil {
		t.Errorf("Should not populate Cloud Logging client.")
	}

	w.cloudLoggingClient = nil
	w.externalLogging = true
	tryPopulateClients(t, w)
	if w.cloudLoggingClient == nil {
		t.Errorf("Did not populate Cloud Logging client.")
	}
}

func tryPopulateClients(t *testing.T, w *Workflow) {
	if err := w.PopulateClients(context.Background()); err != nil {
		t.Errorf("Failed to populate clients for workflow: %v", err)
	}
}

func TestCancelReasonEmptySingleWorkflow(t *testing.T) {
	w1 := testWorkflow()
	assertWorkflowCancelReason(t, w1, "")
}

func TestCancelReasonProvidedSingleWorkflow(t *testing.T) {
	w1 := testWorkflow()
	w1.cancelReason = "w1 cr"
	assertWorkflowCancelReason(t, w1, "w1 cr")
}

func TestCancelReasonChild(t *testing.T) {
	w1 := testWorkflow()
	w2 := testWorkflow()

	w2.parent = w1
	w1.cancelReason = "w1 cr"
	w2.cancelReason = "w2 cr"
	assertWorkflowCancelReason(t, w1, "w1 cr")
	assertWorkflowCancelReason(t, w2, "w2 cr")
}

func TestCancelReasonInheritedFromParent(t *testing.T) {
	w1 := testWorkflow()
	w2 := testWorkflow()

	w2.parent = w1
	w1.cancelReason = "w1 cr"
	assertWorkflowCancelReason(t, w1, "w1 cr")
	assertWorkflowCancelReason(t, w2, "w1 cr")
}

func TestCancelReasonInheritedFromGrandParent(t *testing.T) {
	w1 := testWorkflow()
	w2 := testWorkflow()
	w3 := testWorkflow()

	w2.parent = w1
	w3.parent = w2
	w1.cancelReason = "w1 cr"

	assertWorkflowCancelReason(t, w1, "w1 cr")
	assertWorkflowCancelReason(t, w2, "w1 cr")
	assertWorkflowCancelReason(t, w3, "w1 cr")
}

func TestCancelReasonInheritedFromParentWhenGrandchild(t *testing.T) {
	w1 := testWorkflow()
	w2 := testWorkflow()
	w3 := testWorkflow()

	w2.parent = w1
	w3.parent = w2
	w2.cancelReason = "w2 cr"

	assertWorkflowCancelReason(t, w1, "")
	assertWorkflowCancelReason(t, w2, "w2 cr")
	assertWorkflowCancelReason(t, w3, "w2 cr")
}

func assertWorkflowCancelReason(t *testing.T, w *Workflow, expected string) {
	if cr := w.getCancelReason(); cr != expected {
		t.Errorf("Expected cancel reason `%v` but got `%v` ", expected, cr)
	}
}

func TestOnStepCancelDefaultCancelReason(t *testing.T) {
	w := testWorkflow()
	s := &Step{name: "s", w: w}
	err := w.onStepCancel(s, "Dummy")
	expectedErrorMessage := "Step \"s\" (Dummy) is canceled."
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message `%v` but got `%v` ", expectedErrorMessage, err.Error())
	}
}

func TestOnStepCancelCustomCancelReason(t *testing.T) {
	w := testWorkflow()
	w.cancelReason = "failed horribly"
	s := &Step{name: "s", w: w}
	err := w.onStepCancel(s, "Dummy")
	expectedErrorMessage := "Step \"s\" (Dummy) failed horribly."
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message `%v` but got `%v` ", expectedErrorMessage, err.Error())
	}
}
