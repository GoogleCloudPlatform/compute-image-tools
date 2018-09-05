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
	"testing"
	"time"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/golang/protobuf/ptypes/duration"
	"google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/genproto/googleapis/type/timeofday"
)

func TestNextWindow(t *testing.T) {
	now := time.Date(2018, 7, 1, 5, 0, 0, 0, time.UTC) // July 1st 2018 is a Sunday
	var tests = []struct {
		desc string
		pw   osconfigpb.PatchWindow
		now  time.Time
		want *patchWindow
	}{
		{
			"daily (today before patch window)",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Daily at 5
			now.Add(-2 * time.Hour),                                    // We should be before the patch window
			&patchWindow{start: now, end: now.Add(3600 * time.Second)}, // Todays patch window
		},
		{
			"daily (today inside patch window)",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Daily at 5
			now, // We should be inside the patch window
			&patchWindow{start: now, end: now.Add(3600 * time.Second)}, // Todays patch window
		},
		{
			"daily (today after patch window)",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Daily at 5
			now.Add(2 * time.Hour), // Now is after todays patch window
			&patchWindow{start: now.AddDate(0, 0, 1), end: now.Add(3600*time.Second).AddDate(0, 0, 1)}, // Tomorrows patch window
		},
		{
			"weekly (before this weeks patch window)",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Weekly_{
					Weekly: &osconfigpb.PatchWindow_Weekly{Day: dayofweek.DayOfWeek_FRIDAY},
				}, StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration: &duration.Duration{Seconds: 3600},
			}, // Weekly on Friday at 5
			now, // We should be before the patch window
			&patchWindow{start: time.Date(2018, 7, 6, 5, 0, 0, 0, time.UTC), end: time.Date(2018, 7, 6, 5, 0, 3600, 0, time.UTC)}, // This week, 6th July
		},
		{
			"weekly (during this weeks patch window)",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Weekly_{
					Weekly: &osconfigpb.PatchWindow_Weekly{Day: dayofweek.DayOfWeek_FRIDAY},
				}, StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration: &duration.Duration{Seconds: 3600},
			}, // Weekly on Friday at 5
			now.AddDate(0, 0, 5), // Sunday + 5 = Friday
			&patchWindow{start: time.Date(2018, 7, 6, 5, 0, 0, 0, time.UTC), end: time.Date(2018, 7, 6, 5, 0, 3600, 0, time.UTC)}, // This week, 6th July
		},
		{
			"weekly (after this weeks patch window)",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Weekly_{
					Weekly: &osconfigpb.PatchWindow_Weekly{Day: dayofweek.DayOfWeek_FRIDAY},
				}, StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration: &duration.Duration{Seconds: 3600},
			}, // Weekly on Friday at 5.
			now.AddDate(0, 0, 6), // Sunday + 6 = Saturday
			&patchWindow{start: time.Date(2018, 7, 13, 5, 0, 0, 0, time.UTC), end: time.Date(2018, 7, 13, 5, 0, 3600, 0, time.UTC)}, // Next week, 13th July
		},
		{
			"monthly 5th day of the month (before this months patch window)",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{Day: &osconfigpb.PatchWindow_Monthly_DayOfMonth{DayOfMonth: 5}},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the 5th at 5.
			now, // We should be before the patch window
			&patchWindow{start: time.Date(2018, 7, 5, 5, 0, 0, 0, time.UTC), end: time.Date(2018, 7, 5, 5, 0, 3600, 0, time.UTC)}, // This month, 5th July
		},
		{
			"monthly 5th day of the month (during this months patch window)",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{Day: &osconfigpb.PatchWindow_Monthly_DayOfMonth{DayOfMonth: 5}},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the 5th at 5.
			now.AddDate(0, 0, 4), // 1st + 4 = 5th
			&patchWindow{start: time.Date(2018, 7, 5, 5, 0, 0, 0, time.UTC), end: time.Date(2018, 7, 5, 5, 0, 3600, 0, time.UTC)}, // This month, 5th July
		},
		{
			"monthly 5th day of the month (after this months patch window)",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{Day: &osconfigpb.PatchWindow_Monthly_DayOfMonth{DayOfMonth: 5}},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the 5th at 5.
			now.AddDate(0, 0, 6), // 1st + 6 = 7th
			&patchWindow{start: time.Date(2018, 8, 5, 5, 0, 0, 0, time.UTC), end: time.Date(2018, 8, 5, 5, 0, 3600, 0, time.UTC)}, // Next month, 5th Aug
		},
		{
			"monthly last day of the month (before this months patch window)",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{Day: &osconfigpb.PatchWindow_Monthly_DayOfMonth{DayOfMonth: -1}},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the last day at 5.
			now, // We should be before the patch window
			&patchWindow{start: time.Date(2018, 7, 31, 5, 0, 0, 0, time.UTC), end: time.Date(2018, 7, 31, 5, 0, 3600, 0, time.UTC)}, // This month, 31st of July
		},
		{
			"monthly last day of the month (after this months patch window)",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Monthly_{
					Monthly: &osconfigpb.PatchWindow_Monthly{Day: &osconfigpb.PatchWindow_Monthly_DayOfMonth{DayOfMonth: -1}},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Monthly on the last day at 5.
			now.Add(5*time.Hour).AddDate(0, 0, 30),
			&patchWindow{start: time.Date(2018, 8, 31, 5, 0, 0, 0, time.UTC), end: time.Date(2018, 8, 31, 5, 0, 3600, 0, time.UTC)}, // Next month, 31st of Aug
		},
		{
			"monthly on the second Tuesday (before this weeks patch window)",
			osconfigpb.PatchWindow{
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
			&patchWindow{start: time.Date(2018, 7, 10, 5, 0, 0, 0, time.UTC), end: time.Date(2018, 7, 10, 5, 0, 3600, 0, time.UTC)}, // This month, 10th of July
		},
		{
			"monthly on the second Tuesday (after this weeks patch window)",
			osconfigpb.PatchWindow{
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
			&patchWindow{start: time.Date(2018, 8, 14, 5, 0, 0, 0, time.UTC), end: time.Date(2018, 8, 14, 5, 0, 3600, 0, time.UTC)}, // Next month, 14th of Aug
		},
	}

	for _, tt := range tests {
		got, err := nextWindow(tt.now, tt.pw, 0)
		if err != nil {
			t.Errorf("%s: %v", tt.desc, err)
			continue
		}

		if tt.want.start != got.start {
			t.Errorf("%s start: want(%q) != got(%q)", tt.desc, tt.want.start, got.start)
		}
		if tt.want.end != got.end {
			t.Errorf("%s end: want(%q) != got(%q)", tt.desc, tt.want.end, got.end)
		}
	}
}

func TestNextWindowErrors(t *testing.T) {
	var tests = []struct {
		desc string
		pw   osconfigpb.PatchWindow
	}{
		{
			"no window",
			osconfigpb.PatchWindow{
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			},
		},
		{
			"bad duration",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Daily_{},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			},
		},
		{
			"weekly invalid day",
			osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_Weekly_{
					Weekly: &osconfigpb.PatchWindow_Weekly{Day: dayofweek.DayOfWeek_DAY_OF_WEEK_UNSPECIFIED},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			},
		},
		{
			"monthly invalid day",
			osconfigpb.PatchWindow{
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
		if _, err := nextWindow(time.Now(), tt.pw, 0); err == nil {
			t.Errorf("%s: expected error", tt.desc)
			continue
		}
	}
}
