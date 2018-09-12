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

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/genproto/googleapis/type/timeofday"
)

func TestPatchWindowIn(t *testing.T) {
	c := make(chan struct{})
	close(c)

	var tests = []struct {
		desc string
		pw   *patchWindow
		want bool
	}{
		{
			"in window",
			&patchWindow{
				Start: time.Now().Add(-10 * time.Second),
				End:   time.Now().Add(10 * time.Second),
			},
			true,
		},
		{
			"in window, canceled",
			&patchWindow{
				Start:  time.Now().Add(-10 * time.Second),
				End:    time.Now().Add(10 * time.Second),
				cancel: c,
			},
			false,
		},
		{
			"start time in the future",
			&patchWindow{
				Start:  time.Now().Add(5 * time.Second),
				End:    time.Now().Add(10 * time.Second),
				cancel: c,
			},
			false,
		},
		{
			"end time passed",
			&patchWindow{
				Start:  time.Now().Add(-10 * time.Second),
				End:    time.Now().Add(-5 * time.Second),
				cancel: c,
			},
			false,
		},
	}

	for _, tt := range tests {
		got := tt.pw.in()
		if got != tt.want {
			t.Errorf("%s: want(%t) != got(%t)", tt.desc, tt.want, got)
		}
	}

}

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
			[]byte(`{"Name":"foo","Policy":{"LookupConfigsResponse_EffectivePatchPolicy":{"fullName":"flipyflappy","patchWindow":{"startTime":{"hours":1,"minutes":2,"seconds":3},"duration":"60m","daily":{}}}},"Start":"2018-09-11T01:00:00Z","End":"2018-09-11T02:00:00Z"}`),
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
				Start: time.Date(2018, 9, 11, 1, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 9, 11, 2, 0, 0, 0, time.UTC),
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
				Start: time.Date(2018, 9, 11, 1, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 9, 11, 2, 0, 0, 0, time.UTC),
			},
			"{\"Name\":\"foo\",\"Policy\":{\"LookupConfigsResponse_EffectivePatchPolicy\":{\"fullName\":\"flipyflappy\",\"patchWindow\":{\"startTime\":{\"hours\":1,\"minutes\":2,\"seconds\":3},\"duration\":\"3600s\",\"daily\":{}}}},\"Start\":\"2018-09-11T01:00:00Z\",\"End\":\"2018-09-11T02:00:00Z\"}",
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

func TestPatchManagerRunAndCancel(t *testing.T) {
	aw.windows = map[string]*patchWindow{}
	rc = make(chan *patchWindow)

	now := time.Now().UTC()
	foo := &osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{
		FullName: "foo",
		PatchWindow: &osconfigpb.PatchWindow{
			Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
			StartTime: &timeofday.TimeOfDay{Hours: int32(now.Hour())},
			Duration:  &duration.Duration{Seconds: 7200},
		},
	}

	// Run should start imediately, and just hang as no one is listening on the channel.
	patchManager([]*osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{foo})

	window, ok := aw.windows["foo"]
	if !ok {
		t.Fatal("new window not registered in activeWindows")
	}

	select {
	case <-rc:
	case <-time.Tick(1 * time.Second):
		t.Fatal("window did not run")
	}

	// Should cancel current run.
	foo.PatchWindow.StartTime = &timeofday.TimeOfDay{Hours: int32(now.Add(1 * time.Hour).Hour())}
	go patchManager([]*osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{foo})

	select {
	case <-window.cancel:
	case <-time.Tick(1 * time.Second):
		t.Fatal("window was not canceled")
	}
}

func TestPatchManager(t *testing.T) {
	aw.windows = map[string]*patchWindow{}
	compare := &pretty.Config{
		Diffable:  true,
		Formatter: pretty.DefaultFormatter,
	}

	// Start with some basic policies.
	now := time.Now().UTC()
	foo := &osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{
		FullName: "foo",
		PatchWindow: &osconfigpb.PatchWindow{
			Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
			StartTime: &timeofday.TimeOfDay{Hours: int32(now.Add(5 * time.Hour).Hour()), Minutes: int32(now.Minute()), Seconds: int32(now.Second()), Nanos: int32(now.Nanosecond())},
			Duration:  &duration.Duration{Seconds: 3600},
		},
	}
	bar := &osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{
		FullName: "bar",
		PatchWindow: &osconfigpb.PatchWindow{
			Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
			StartTime: &timeofday.TimeOfDay{Hours: int32(now.Add(5 * time.Hour).Hour()), Minutes: int32(now.Minute()), Seconds: int32(now.Second()), Nanos: int32(now.Nanosecond())},
			Duration:  &duration.Duration{Seconds: 3600},
		},
	}
	baz := &osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{
		FullName: "baz",
		PatchWindow: &osconfigpb.PatchWindow{
			Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
			StartTime: &timeofday.TimeOfDay{Hours: int32(now.Add(5 * time.Hour).Hour()), Minutes: int32(now.Minute()), Seconds: int32(now.Second()), Nanos: int32(now.Nanosecond())},
			Duration:  &duration.Duration{Seconds: 3600},
		},
	}

	patchManager([]*osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{foo, bar, baz})

	want := map[string]*patchWindow{
		"foo": &patchWindow{
			Name:   foo.GetFullName(),
			Policy: &patchPolicy{foo},
			Start:  now.Add(5 * time.Hour),
			End:    now.Add(6 * time.Hour),
		},
		"bar": &patchWindow{
			Name:   bar.GetFullName(),
			Policy: &patchPolicy{bar},
			Start:  now.Add(5 * time.Hour),
			End:    now.Add(6 * time.Hour),
		},
		"baz": &patchWindow{
			Name:   baz.GetFullName(),
			Policy: &patchPolicy{baz},
			Start:  now.Add(5 * time.Hour),
			End:    now.Add(6 * time.Hour),
		},
	}

	if diff := compare.Compare(want, aw.windows); diff != "" {
		t.Errorf("active windows do not match expectation: (-got +want)\n%s", diff)
	}

	// Modify "foo" policy, delete "bar", and add "boo".
	// Start updated
	foo = &osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{
		FullName: "foo",
		PatchWindow: &osconfigpb.PatchWindow{
			Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
			StartTime: &timeofday.TimeOfDay{Hours: int32(now.Add(6 * time.Hour).Hour()), Minutes: int32(now.Minute()), Seconds: int32(now.Second()), Nanos: int32(now.Nanosecond())},
			Duration:  &duration.Duration{Seconds: 3600},
		},
	}
	// Replaces bar
	boo := &osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{
		FullName: "boo",
		PatchWindow: &osconfigpb.PatchWindow{
			Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
			StartTime: &timeofday.TimeOfDay{Hours: int32(now.Add(5 * time.Hour).Hour()), Minutes: int32(now.Minute()), Seconds: int32(now.Second()), Nanos: int32(now.Nanosecond())},
			Duration:  &duration.Duration{Seconds: 3600},
		},
	}
	// Unchanged
	baz = &osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{
		FullName: "baz",
		PatchWindow: &osconfigpb.PatchWindow{
			Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
			StartTime: &timeofday.TimeOfDay{Hours: int32(now.Add(5 * time.Hour).Hour()), Minutes: int32(now.Minute()), Seconds: int32(now.Second()), Nanos: int32(now.Nanosecond())},
			Duration:  &duration.Duration{Seconds: 3600},
		},
	}

	patchManager([]*osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{foo, boo, baz})

	want = map[string]*patchWindow{
		"foo": &patchWindow{
			Name:   foo.GetFullName(),
			Policy: &patchPolicy{foo},
			Start:  now.Add(6 * time.Hour),
			End:    now.Add(7 * time.Hour),
		},
		"boo": &patchWindow{
			Name:   boo.GetFullName(),
			Policy: &patchPolicy{boo},
			Start:  now.Add(5 * time.Hour),
			End:    now.Add(6 * time.Hour),
		},
		"baz": &patchWindow{
			Name:   baz.GetFullName(),
			Policy: &patchPolicy{baz},
			Start:  now.Add(5 * time.Hour),
			End:    now.Add(6 * time.Hour),
		},
	}

	if diff := compare.Compare(want, aw.windows); diff != "" {
		t.Errorf("active windows do not match expectation: (-got +want)\n%s", diff)
	}
}

func TestNextWindow(t *testing.T) {
	now := time.Date(2018, 7, 1, 5, 0, 0, 0, time.UTC) // July 1st 2018 is a Sunday
	var tests = []struct {
		desc      string
		pw        *osconfigpb.PatchWindow
		now       time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			"daily (today before patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Daily at 5
			now.Add(-2 * time.Hour), // We should be before the patch window
			now,
			now.Add(3600 * time.Second), // Todays patch window
		},
		{
			"daily (today inside patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Daily at 5
			now, // We should be inside the patch window
			now,
			now.Add(3600 * time.Second), // Todays patch window
		},
		{
			"daily (today after patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Daily at 5
			now.Add(2 * time.Hour), // Now is after todays patch window
			now.AddDate(0, 0, 1),
			now.Add(3600*time.Second).AddDate(0, 0, 1), // Tomorrows patch window
		},
		{
			"weekly (before this weeks patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Weekly_{
					Weekly: &osconfigpb.PatchWindow_Weekly{Day: dayofweek.DayOfWeek_FRIDAY},
				}, StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration: &duration.Duration{Seconds: 3600},
			}, // Weekly on Friday at 5
			now, // We should be before the patch window
			time.Date(2018, 7, 6, 5, 0, 0, 0, time.UTC),
			time.Date(2018, 7, 6, 5, 0, 3600, 0, time.UTC), // This week, 6th July
		},
		{
			"weekly (during this weeks patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Weekly_{
					Weekly: &osconfigpb.PatchWindow_Weekly{Day: dayofweek.DayOfWeek_FRIDAY},
				}, StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration: &duration.Duration{Seconds: 3600},
			}, // Weekly on Friday at 5
			now.AddDate(0, 0, 5), // Sunday + 5 = Friday
			time.Date(2018, 7, 6, 5, 0, 0, 0, time.UTC),
			time.Date(2018, 7, 6, 5, 0, 3600, 0, time.UTC), // This week, 6th July
		},
		{
			"weekly (after this weeks patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Weekly_{
					Weekly: &osconfigpb.PatchWindow_Weekly{Day: dayofweek.DayOfWeek_FRIDAY},
				}, StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration: &duration.Duration{Seconds: 3600},
			}, // Weekly on Friday at 5.
			now.AddDate(0, 0, 6), // Sunday + 6 = Saturday
			time.Date(2018, 7, 13, 5, 0, 0, 0, time.UTC),
			time.Date(2018, 7, 13, 5, 0, 3600, 0, time.UTC), // Next week, 13th July
		},
		{
			"monthly 5th day of the month (before this months patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{Day: &osconfigpb.PatchWindow_Monthly_DayOfMonth{DayOfMonth: 5}},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the 5th at 5.
			now, // We should be before the patch window
			time.Date(2018, 7, 5, 5, 0, 0, 0, time.UTC),
			time.Date(2018, 7, 5, 5, 0, 3600, 0, time.UTC), // This month, 5th July
		},
		{
			"monthly 5th day of the month (during this months patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{Day: &osconfigpb.PatchWindow_Monthly_DayOfMonth{DayOfMonth: 5}},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the 5th at 5.
			now.AddDate(0, 0, 4), // 1st + 4 = 5th
			time.Date(2018, 7, 5, 5, 0, 0, 0, time.UTC),
			time.Date(2018, 7, 5, 5, 0, 3600, 0, time.UTC), // This month, 5th July
		},
		{
			"monthly 5th day of the month (after this months patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{Day: &osconfigpb.PatchWindow_Monthly_DayOfMonth{DayOfMonth: 5}},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the 5th at 5.
			now.AddDate(0, 0, 6), // 1st + 6 = 7th
			time.Date(2018, 8, 5, 5, 0, 0, 0, time.UTC),
			time.Date(2018, 8, 5, 5, 0, 3600, 0, time.UTC), // Next month, 5th Aug
		},
		{
			"monthly last day of the month (before this months patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{Day: &osconfigpb.PatchWindow_Monthly_DayOfMonth{DayOfMonth: -1}},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the last day at 5.
			now, // We should be before the patch window
			time.Date(2018, 7, 31, 5, 0, 0, 0, time.UTC),
			time.Date(2018, 7, 31, 5, 0, 3600, 0, time.UTC), // This month, 31st of July
		},
		{
			"monthly last day of the month (after this months patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{Day: &osconfigpb.PatchWindow_Monthly_DayOfMonth{DayOfMonth: -1}},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the last day at 5.
			now.Add(5*time.Hour).AddDate(0, 0, 30),
			time.Date(2018, 8, 31, 5, 0, 0, 0, time.UTC),
			time.Date(2018, 8, 31, 5, 0, 3600, 0, time.UTC), // Next month, 31st of Aug
		},
		{
			"monthly on the second Tuesday (before this weeks patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{
						Day: &osconfigpb.PatchWindow_Monthly_OccurrenceOfDay_{
							OccurrenceOfDay: &osconfigpb.PatchWindow_Monthly_OccurrenceOfDay{
								Day:        dayofweek.DayOfWeek_TUESDAY,
								Occurrence: 2,
							},
						},
					},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the second Tuesday at 5
			now, // We should be before the patch window
			time.Date(2018, 7, 10, 5, 0, 0, 0, time.UTC),
			time.Date(2018, 7, 10, 5, 0, 3600, 0, time.UTC), // This month, 10th of July
		},
		{
			"monthly on the second Tuesday (after this weeks patch window)",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{
						Day: &osconfigpb.PatchWindow_Monthly_OccurrenceOfDay_{
							OccurrenceOfDay: &osconfigpb.PatchWindow_Monthly_OccurrenceOfDay{
								Day:        dayofweek.DayOfWeek_TUESDAY,
								Occurrence: 2,
							},
						},
					},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the second Tuesday at 5
			now.AddDate(0, 0, 15), // 15 days is after this months window
			time.Date(2018, 8, 14, 5, 0, 0, 0, time.UTC),
			time.Date(2018, 8, 14, 5, 0, 3600, 0, time.UTC), // Next month, 14th of Aug
		},
	}

	for _, tt := range tests {
		gotStart, gotEnd, err := nextWindow(tt.now, tt.pw, 0)
		if err != nil {
			t.Errorf("%s: %v", tt.desc, err)
			continue
		}

		if tt.wantStart != gotStart {
			t.Errorf("%s start: want(%q) != got(%q)", tt.desc, tt.wantStart, gotStart)
		}
		if tt.wantEnd != gotEnd {
			t.Errorf("%s end: want(%q) != got(%q)", tt.desc, tt.wantEnd, gotEnd)
		}
	}
}

func TestNextWindowErrors(t *testing.T) {
	var tests = []struct {
		desc string
		pw   *osconfigpb.PatchWindow
	}{
		{
			"no window",
			&osconfigpb.PatchWindow{
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			},
		},
		{
			"bad duration",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Daily_{},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			},
		},
		{
			"weekly invalid day",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Weekly_{
					Weekly: &osconfigpb.PatchWindow_Weekly{Day: dayofweek.DayOfWeek_DAY_OF_WEEK_UNSPECIFIED},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			},
		},
		{
			"monthly invalid day",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{
						Day: &osconfigpb.PatchWindow_Monthly_OccurrenceOfDay_{
							OccurrenceOfDay: &osconfigpb.PatchWindow_Monthly_OccurrenceOfDay{
								Day:        dayofweek.DayOfWeek_DAY_OF_WEEK_UNSPECIFIED,
								Occurrence: 2,
							},
						},
					},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			},
		},
	}

	for _, tt := range tests {
		if _, _, err := nextWindow(time.Now(), tt.pw, 0); err == nil {
			t.Errorf("%s: expected error", tt.desc)
			continue
		}
	}
}
