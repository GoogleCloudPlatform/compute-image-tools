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

package patch

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/genproto/googleapis/type/timeofday"
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
		want    *patchWindow
	}{
		{
			"blank state",
			[]byte("{}"),
			false,
			&patchWindow{},
		},
		{
			"bad state",
			[]byte("foo"),
			true,
			&patchWindow{},
		},
		{
			"test patchWindow",
			[]byte(`{"Name":"foo","Policy":{"LookupConfigsResponse_EffectivePatchPolicy":{"fullName":"flipyflappy","patchWindow":{"startTime":{"hours":1,"minutes":2,"seconds":3},"duration":"60m","daily":{}}}},"ScheduledStart":"2018-09-11T01:00:00Z","ScheduledEnd":"2018-09-11T02:00:00Z"}`),
			false,
			&patchWindow{
				Name: "foo",
				Policy: &patchPolicy{
					LookupConfigsResponse_EffectivePatchPolicy: &osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{
						FullName: "flipyflappy",
						PatchWindow: &osconfigpb.PatchWindow{
							StartTime: &timeofday.TimeOfDay{
								Hours:   1,
								Minutes: 2,
								Seconds: 3,
							},
							Duration:  &duration.Duration{Seconds: 3600},
							Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
						},
					},
				},
				ScheduledStart: time.Date(2018, 9, 11, 1, 0, 0, 0, time.UTC),
				ScheduledEnd:   time.Date(2018, 9, 11, 2, 0, 0, 0, time.UTC),
			},
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
		state *patchWindow
		want  string
	}{
		{
			"blank state",
			nil,
			"{}",
		},
		{
			"test patchWindow",
			&patchWindow{
				Name: "foo",
				Policy: &patchPolicy{
					LookupConfigsResponse_EffectivePatchPolicy: &osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{
						FullName: "flipyflappy",
						PatchWindow: &osconfigpb.PatchWindow{
							StartTime: &timeofday.TimeOfDay{
								Hours:   1,
								Minutes: 2,
								Seconds: 3,
							},
							Duration:  &duration.Duration{Seconds: 3600},
							Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
						},
					},
				},
				ScheduledStart: time.Date(2018, 9, 11, 1, 0, 0, 0, time.UTC),
				ScheduledEnd:   time.Date(2018, 9, 11, 2, 0, 0, 0, time.UTC),
			},
			"{\"Name\":\"foo\",\"Policy\":{\"LookupConfigsResponse_EffectivePatchPolicy\":{\"fullName\":\"flipyflappy\",\"patchWindow\":{\"startTime\":{\"hours\":1,\"minutes\":2,\"seconds\":3},\"duration\":\"3600s\",\"daily\":{}}}},\"ScheduledStart\":\"2018-09-11T01:00:00Z\",\"ScheduledEnd\":\"2018-09-11T02:00:00Z\",\"StartedAt\":\"0001-01-01T00:00:00Z\",\"EndedAt\":\"0001-01-01T00:00:00Z\",\"Complete\":false}",
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
			t.Errorf("%s: %q != %q", tt.desc, got, tt.want)
		}
	}
}
