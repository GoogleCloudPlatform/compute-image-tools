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

package ospatch

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/kylelemons/godebug/pretty"
)

var (
	testPatchRunJSON = "{\"PatchRuns\":[{\"Job\":{\"ReportPatchJobInstanceDetailsResponse\":{\"patchJob\":\"flipyflappy\",\"patchConfig\":{\"rebootConfig\":\"ALWAYS\"}}},\"StartedAt\":\"0001-01-01T00:00:00Z\",\"EndedAt\":\"0001-01-01T00:00:00Z\",\"Complete\":false,\"PatchStep\":\"\",\"RebootCount\":0}],\"PastJobs\":null}"
	testPatchRun     = &patchRun{
		Job: &patchJob{
			&osconfigpb.ReportPatchJobInstanceDetailsResponse{
				PatchJob: "flipyflappy",
				PatchConfig: &osconfigpb.PatchConfig{
					RebootConfig: osconfigpb.PatchConfig_ALWAYS,
				},
			},
		},
	}
)

func TestLoadState(t *testing.T) {
	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	testState := filepath.Join(td, "testState")

	// test no state file
	if err := loadState(testState); err != nil {
		t.Errorf("no state file: unexpected error: %v", err)
	}

	var tests = []struct {
		desc    string
		state   []byte
		wantErr bool
		want    *state
	}{
		{
			"blank state",
			[]byte("{}"),
			false,
			&state{},
		},
		{
			"bad state",
			[]byte("foo"),
			true,
			&state{},
		},
		{
			"test patchRun",
			[]byte(testPatchRunJSON),
			false,
			&state{PatchRuns: []*patchRun{testPatchRun}},
		},
	}
	for _, tt := range tests {
		if err := ioutil.WriteFile(testState, tt.state, 0600); err != nil {
			t.Errorf("%s: error writing state: %v", tt.desc, err)
			continue
		}

		err := loadState(testState)
		if err != nil && !tt.wantErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
			continue
		}
		if err == nil && tt.wantErr {
			t.Errorf("%s: expected error", tt.desc)
			continue
		}
		if diff := pretty.Compare(tt.want, &liveState); diff != "" {
			t.Errorf("%s: patchWindow does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestStateSave(t *testing.T) {
	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	testState := filepath.Join(td, "testState")

	var tests = []struct {
		desc  string
		state *state
		want  string
	}{
		{
			"blank state",
			&state{},
			"{\"PatchRuns\":null,\"PastJobs\":null}",
		},
		{
			"test patchWindow",
			&state{PatchRuns: []*patchRun{testPatchRun}},
			testPatchRunJSON,
		},
	}
	for _, tt := range tests {
		err := tt.state.save(testState)
		if err != nil {
			t.Errorf("%s: unexpected save error: %v", tt.desc, err)
			continue
		}

		got, err := ioutil.ReadFile(testState)
		if err != nil {
			t.Errorf("%s: error reading state: %v", tt.desc, err)
			continue
		}

		if string(got) != tt.want {
			t.Errorf("%s:\ngot:\n%q\nwant:\n%q", tt.desc, got, tt.want)
		}
	}
}

func TestStateAdd(t *testing.T) {
	pr1 := &patchRun{
		Job: &patchJob{
			&osconfigpb.ReportPatchJobInstanceDetailsResponse{
				PatchJob: "pr1",
			},
		},
	}
	pr2 := &patchRun{
		Job: &patchJob{
			&osconfigpb.ReportPatchJobInstanceDetailsResponse{
				PatchJob: "prw",
			},
		},
	}

	want := &state{
		PatchRuns: []*patchRun{pr1, pr2},
	}

	st := &state{
		PatchRuns: []*patchRun{pr1},
	}

	st.addPatchRun(pr2)

	if !reflect.DeepEqual(want, st) {
		t.Errorf("state does not match expectations afer add:\ngot:\n%+v\nwant:\n%+v", st, want)
	}
}

func TestStateRemove(t *testing.T) {
	pr1 := &patchRun{
		Job: &patchJob{
			&osconfigpb.ReportPatchJobInstanceDetailsResponse{
				PatchJob: "pr1",
			},
		},
	}
	pr2 := &patchRun{
		Job: &patchJob{
			&osconfigpb.ReportPatchJobInstanceDetailsResponse{
				PatchJob: "prw",
			},
		},
	}

	want := &state{
		PatchRuns: []*patchRun{pr1},
	}

	st := &state{
		PatchRuns: []*patchRun{pr1, pr2},
	}

	st.removePatchRun(pr2)

	if !reflect.DeepEqual(want, st) {
		t.Errorf("state does not match expectations after add:\ngot:\n%+v\nwant:\n%+v", st, want)
	}
}

func TestAckedJob(t *testing.T) {
	job := "ackedJob"
	pr := &patchRun{
		Job: &patchJob{
			&osconfigpb.ReportPatchJobInstanceDetailsResponse{
				PatchJob: "pr1",
			},
		},
	}
	var tests = []struct {
		desc  string
		state *state
		want  bool
	}{
		{
			"not acked, not complete",
			&state{
				PatchRuns: []*patchRun{pr},
				PastJobs:  []string{"job1"},
			},
			false,
		},
		{
			"already acked",
			&state{
				PatchRuns: []*patchRun{
					pr,
					&patchRun{
						Job: &patchJob{
							&osconfigpb.ReportPatchJobInstanceDetailsResponse{
								PatchJob: job,
							},
						},
					},
				},
				PastJobs: []string{"job1"},
			},
			true,
		},
		{
			"job complete",
			&state{
				PatchRuns: []*patchRun{pr},
				PastJobs:  []string{"job1", job},
			},
			true,
		},
	}
	for _, tt := range tests {
		got := tt.state.alreadyAckedJob(job)
		if got != tt.want {
			t.Errorf("%s: want(%v) got(%v)", tt.desc, tt.want, got)
		}
	}
}

func TestJobComplete(t *testing.T) {
	newjob := "newJob"
	var tests = []struct {
		desc  string
		state *state
		want  []string
	}{
		{
			"add 1 job",
			&state{PastJobs: []string{"job1"}},
			[]string{"job1", newjob},
		},
		{
			"add 1 job, remove 2 first jobs",
			&state{PastJobs: []string{"job1", "job2", "job3", "job4", "job5", "job6", "job7", "job8", "job9", "job10", "job11"}},
			[]string{"job3", "job4", "job5", "job6", "job7", "job8", "job9", "job10", "job11", newjob},
		},
	}
	for _, tt := range tests {
		tt.state.jobComplete(newjob)
		if !reflect.DeepEqual(tt.state.PastJobs, tt.want) {
			t.Errorf("PastJobs do not match expectations after jobComplete: got: %q, want: %q", tt.state.PastJobs, tt.want)
		}
	}
}
