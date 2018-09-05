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
	"errors"
	"fmt"
	"time"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/genproto/googleapis/type/timeofday"
)

type patchWindow struct {
	start, end time.Time
}

func (w *patchWindow) in() bool {
	now := time.Now()
	return now.After(w.start) && now.Before(w.end)
}

// nextWindow will return the next applicable time window start relative to now.
func nextWindow(now time.Time, window osconfigpb.PatchWindow, offset int) (*patchWindow, error) {
	var start time.Time
	var err error
	switch {
	case window.GetDaily() != nil:
		start, err = dailyWindowStart(now, window.GetStartTime(), offset)
	case window.GetWeekly() != nil:
		start, err = weeklyWindowStart(now, window.GetStartTime(), window.GetWeekly(), offset)
	case window.GetMonthly() != nil:
		start, err = monthlyWindowStart(now, window.GetStartTime(), window.GetMonthly(), offset)
	default:
		return nil, errors.New("no window set in PatchWindow")
	}
	if err != nil {
		return nil, err
	}

	length := time.Duration(window.GetDuration().GetSeconds()) * time.Second
	end := start.Add(length)
	if now.After(end) {
		return nextWindow(now, window, offset+1)
	}
	return &patchWindow{start: start, end: end}, nil
}

func monthlyWindowStart(now time.Time, start *timeofday.TimeOfDay, window *osconfigpb.PatchWindow_Monthly, offset int) (time.Time, error) {
	var dom int
	month := time.Month(int(now.Month()) + offset)
	firstOfMonth := time.Date(now.Year(), month, 1, 0, 0, 0, 0, now.Location())
	if window.GetOccurrenceOfDay() != nil {
		day := window.GetOccurrenceOfDay().GetDay()
		if day == dayofweek.DayOfWeek_DAY_OF_WEEK_UNSPECIFIED {
			return now, fmt.Errorf("%q not a valid day", day)
		}
		dom = (7+int(day)-int(firstOfMonth.Weekday()))%7 + 1 + ((int(window.GetOccurrenceOfDay().GetOccurrence()) - 1) * 7)
	} else {
		dom = int(window.GetDayOfMonth())
		if dom == -1 {
			dom = firstOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond).Day()
		}
	}

	// TODO: This will rollover into the following/preceding month, add a check for that.
	return time.Date(now.Year(), month, dom, int(start.Hours), int(start.Minutes), 0, int(start.Nanos), now.Location()), nil
}

func weeklyWindowStart(now time.Time, start *timeofday.TimeOfDay, window *osconfigpb.PatchWindow_Weekly, offset int) (time.Time, error) {
	day := window.GetDay()
	if day == dayofweek.DayOfWeek_DAY_OF_WEEK_UNSPECIFIED {
		return now, fmt.Errorf("%q not a valid day", day)
	}
	t := now.AddDate(0, 0, -int(now.Weekday())).AddDate(0, 0, int(day)+(offset*7))
	return time.Date(t.Year(), t.Month(), t.Day(), int(start.Hours), int(start.Minutes), 0, int(start.Nanos), now.Location()), nil
}

func dailyWindowStart(now time.Time, start *timeofday.TimeOfDay, offset int) (time.Time, error) {
	return time.Date(now.Year(), now.Month(), now.Day()+offset, int(start.Hours), int(start.Minutes), 0, int(start.Nanos), now.Location()), nil
}
