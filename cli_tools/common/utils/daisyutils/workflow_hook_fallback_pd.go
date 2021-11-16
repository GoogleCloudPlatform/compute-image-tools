//  Copyright 2021 Google Inc. All Rights Reserved.
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

package daisyutils

import (
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// FallbackToPDStandard detects if a workflow fails due to insufficient SSD
// quota. If so, it re-runs the workflow and modifies the disks to use PD Standard.
type FallbackToPDStandard struct {
	logger         logging.Logger
	shouldFallback bool
}

// PreRunHook modifies the workflow to use standard disks if `shouldFallback` is true.
func (f *FallbackToPDStandard) PreRunHook(wf *daisy.Workflow) error {
	if f.shouldFallback {
		useStandardDisks(wf)
	}
	return nil
}

// PostRunHook inspects the workflow error to see if it's related to insufficient SSD quota.
// If so, it requests a retry that will re-run the workflow using standard disks.
func (f *FallbackToPDStandard) PostRunHook(err error) (wantRetry bool, wrapped error) {
	if f.shouldFallback {
		// A fallback has already occurred; don't request retry.
		return false, err
	} else if err != nil && strings.Contains(err.Error(), "SSD_TOTAL_GB") {
		f.logger.Debug("Workflow failed with insufficient SSD quota. Requesting retry. error=" + err.Error())
		f.shouldFallback = true
	}
	return f.shouldFallback, err
}

func useStandardDisks(workflow *daisy.Workflow) {
	workflow.IterateWorkflowSteps(func(step *daisy.Step) {
		if step.CreateDisks != nil {
			for _, disk := range *step.CreateDisks {
				disk.Disk.Type = "pd-standard"
			}
		}
	})
}
