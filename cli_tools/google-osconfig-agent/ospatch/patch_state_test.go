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
	"testing"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/kylelemons/godebug/pretty"
)

var (
	testPatchRunJSON = "{\"Job\":{\"ReportPatchJobInstanceDetailsResponse\":{\"patchJob\":\"flipyflappy\",\"patchConfig\":{\"rebootConfig\":\"ALWAYS\"}}},\"StartedAt\":\"0001-01-01T00:00:00Z\",\"EndedAt\":\"0001-01-01T00:00:00Z\",\"Complete\":false,\"PatchStep\":\"\"}"
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
	if _, err := loadState(testState); err != nil {
		t.Errorf("no state file: unexpected error: %v", err)
	}

	var tests = []struct {
		desc    string
		state   []byte
		wantErr bool
		want    *patchRun
	}{
		{
			"blank state",
			[]byte("{}"),
			false,
			&patchRun{},
		},
		{
			"bad state",
			[]byte("foo"),
			true,
			&patchRun{},
		},
		{
			"test patchRun",
			[]byte(testPatchRunJSON),
			false,
			testPatchRun,
		},
	}
	for _, tt := range tests {
		if err := ioutil.WriteFile(testState, tt.state, 0600); err != nil {
			t.Errorf("%s: error writing state: %v", tt.desc, err)
			continue
		}

		got, err := loadState(testState)
		if err != nil && !tt.wantErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
			continue
		}
		if err == nil && tt.wantErr {
			t.Errorf("%s: expected error", tt.desc)
			continue
		}
		if diff := pretty.Compare(tt.want, got); diff != "" {
			t.Errorf("%s: patchWindow does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestSaveState(t *testing.T) {
	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	testState := filepath.Join(td, "testState")

	var tests = []struct {
		desc  string
		state *patchRun
		want  string
	}{
		{
			"blank state",
			nil,
			"{}",
		},
		{
			"test patchWindow",
			testPatchRun,
			testPatchRunJSON,
		},
	}
	for _, tt := range tests {
		err := saveState(testState, tt.state)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
			continue
		}

		got, err := ioutil.ReadFile(testState)
		if err != nil {
			t.Errorf("%s: error reading state: %v", tt.desc, err)
			continue
		}

		if string(got) != tt.want {
			t.Errorf("%s: got:\n%q\nwant:\n %q", tt.desc, got, tt.want)
		}
	}
}
