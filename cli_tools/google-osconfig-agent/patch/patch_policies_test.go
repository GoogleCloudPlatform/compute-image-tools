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
	"testing"
	"time"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/kylelemons/godebug/pretty"
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
				ScheduledStart: time.Now().Add(-10 * time.Second),
				ScheduledEnd:   time.Now().Add(10 * time.Second),
			},
			true,
		},
		{
			"in window, canceled",
			&patchWindow{
				ScheduledStart: time.Now().Add(-10 * time.Second),
				ScheduledEnd:   time.Now().Add(10 * time.Second),
				cancel:         c,
			},
			false,
		},
		{
			"start time in the future",
			&patchWindow{
				ScheduledStart: time.Now().Add(5 * time.Second),
				ScheduledEnd:   time.Now().Add(10 * time.Second),
				cancel:         c,
			},
			false,
		},
		{
			"end time passed",
			&patchWindow{
				ScheduledStart: time.Now().Add(-10 * time.Second),
				ScheduledEnd:   time.Now().Add(-5 * time.Second),
				cancel:         c,
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

func TestSetPatchPoliciesRunAndCancel(t *testing.T) {
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

	// This patch should queue imediately, and just hang as no one is listening on the channel.
	SetPatchPolicies([]*osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{foo})

	window, ok := aw.get("foo")
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
	go SetPatchPolicies([]*osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{foo})

	select {
	case <-window.cancel:
	case <-time.Tick(1 * time.Second):
		t.Fatal("window was not canceled")
	}
}

func TestSetPatchPolicies(t *testing.T) {
	aw.mx.Lock()
	aw.windows = map[string]*patchWindow{}
	aw.mx.Unlock()
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

	SetPatchPolicies([]*osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{foo, bar, baz})

	want := map[string]*patchWindow{
		"foo": &patchWindow{
			Name:           foo.GetFullName(),
			Policy:         &patchPolicy{foo},
			ScheduledStart: now.Add(5 * time.Hour),
			ScheduledEnd:   now.Add(6 * time.Hour),
		},
		"bar": &patchWindow{
			Name:           bar.GetFullName(),
			Policy:         &patchPolicy{bar},
			ScheduledStart: now.Add(5 * time.Hour),
			ScheduledEnd:   now.Add(6 * time.Hour),
		},
		"baz": &patchWindow{
			Name:           baz.GetFullName(),
			Policy:         &patchPolicy{baz},
			ScheduledStart: now.Add(5 * time.Hour),
			ScheduledEnd:   now.Add(6 * time.Hour),
		},
	}

	if diff := compare.Compare(want, aw.windows); diff != "" {
		t.Errorf("active windows do not match expectation: (-got +want)\n%s", diff)
	}

	// Modify "foo" policy, delete "bar", and add "boo".
	newFoo := &osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{
		FullName: "foo",
		PatchWindow: &osconfigpb.PatchWindow{
			Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
			StartTime: &timeofday.TimeOfDay{Hours: int32(now.Add(5 * time.Hour).Hour()), Minutes: int32(now.Minute()), Seconds: int32(now.Second()), Nanos: int32(now.Nanosecond())},
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
	newBaz := &osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{
		FullName: "baz",
		PatchWindow: &osconfigpb.PatchWindow{
			Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
			StartTime: &timeofday.TimeOfDay{Hours: int32(now.Add(5 * time.Hour).Hour()), Minutes: int32(now.Minute()), Seconds: int32(now.Second()), Nanos: int32(now.Nanosecond())},
			Duration:  &duration.Duration{Seconds: 3600},
		},
	}

	SetPatchPolicies([]*osconfigpb.LookupConfigsResponse_EffectivePatchPolicy{newFoo, boo, newBaz})

	newWant := map[string]*patchWindow{
		"foo": &patchWindow{
			Name:           newFoo.GetFullName(),
			Policy:         &patchPolicy{newFoo},
			ScheduledStart: now.Add(5 * time.Hour),
			ScheduledEnd:   now.Add(6 * time.Hour),
		},
		"boo": &patchWindow{
			Name:           boo.GetFullName(),
			Policy:         &patchPolicy{boo},
			ScheduledStart: now.Add(5 * time.Hour),
			ScheduledEnd:   now.Add(6 * time.Hour),
		},
		"baz": &patchWindow{
			Name:           newBaz.GetFullName(),
			Policy:         &patchPolicy{newBaz},
			ScheduledStart: now.Add(5 * time.Hour),
			ScheduledEnd:   now.Add(6 * time.Hour),
		},
	}

	if diff := compare.Compare(newWant, aw.windows); diff != "" {
		t.Errorf("active windows do not match expectation after update: (-got +want)\n%s", diff)
	}
}
