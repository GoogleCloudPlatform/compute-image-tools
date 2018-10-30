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
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/genproto/googleapis/type/timeofday"
)

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
		{
			"run once in the future",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_RunOnce_{
					RunOnce: &osconfigpb.PatchWindow_RunOnce{
						Date: &date.Date{
							Year:  2018,
							Month: 10,
							Day:   10,
						},
					},
				},
				StartTime: &timeofday.TimeOfDay{Hours: 5},
				Duration:  &duration.Duration{Seconds: 3600},
			}, // Run once on 2018-10-10
			now,
			time.Date(2018, 10, 10, 5, 0, 0, 0, time.UTC),
			time.Date(2018, 10, 10, 5, 0, 3600, 0, time.UTC),
		},
		{
			"run once already should have started",
			&osconfigpb.PatchWindow{
				Frequency: &osconfigpb.PatchWindow_RunOnce_{
					RunOnce: &osconfigpb.PatchWindow_RunOnce{
						Date: &date.Date{
							Year:  int32(now.Year()),
							Month: int32(now.Month()),
							Day:   int32(now.Day()),
						},
					},
				},
				StartTime: &timeofday.TimeOfDay{Hours: int32(now.Hour() - 1)},
				Duration:  &duration.Duration{Seconds: 7200},
			}, // Run once window started an hour ago
			now, // We are already in the patch window
			now.Add(-3600 * time.Second),
			now.Add(3600 * time.Second),
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
