package main

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func printPerfProfile(workflow *daisy.Workflow) {
        if !*printPerf {
                return
        }

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
                fmt.Printf("- %v: %v\n", r.Name, formatDuration(r.EndTime.Sub(r.StartTime)))
        }
        fmt.Printf("Total time: %v\n\n", formatDuration(wfEndTime.Sub(wfStartTime)))
}

func formatDuration(d time.Duration) string {
        s := int(d.Seconds())
        return fmt.Sprintf("%v:%v:%v", s/3600, s/60%60, s%60)
}

