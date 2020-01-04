//  Copyright 2019 Google Inc. All Rights Reserved.
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

package performance

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// PrintPerfProfile prints performance profile for the workflow on console
func PrintPerfProfile(workflow *daisy.Workflow) {
	timeRecords := workflow.GetStepTimeRecords()
	if len(timeRecords) == 0 {
		return
	}

	wfStartTime := time.Now()
	wfEndTime := time.Time{}
	fmt.Println("\nPerf Profile:")
	for _, r := range timeRecords {
		if wfStartTime.After(r.StartTime) {
			wfStartTime = r.StartTime
		}
		if wfEndTime.Before(r.EndTime) {
			wfEndTime = r.EndTime
		}
		fmt.Printf("- %v: [%v ~ %v] %v\n", r.Name, r.StartTime, r.EndTime, formatDuration(r.EndTime.Sub(r.StartTime)))
	}
	fmt.Printf("Total time: [%v ~ %v] %v\n\n", wfStartTime, wfEndTime, formatDuration(wfEndTime.Sub(wfStartTime)))
}

func formatDuration(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("[hh:mm:ss] %v:%v:%v", s/3600, s/60%60, s%60)
}
