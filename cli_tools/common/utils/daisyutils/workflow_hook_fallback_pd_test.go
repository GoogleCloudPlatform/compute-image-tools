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
	"errors"
	"testing"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
)

var errQuota = errors.New("insufficient quota: SSD_TOTAL_GB")
var errNotQuota = errors.New("failed to start workflow")

func Test_FallbackToPDStandard_PostRunHook_RequestsRetryIfQuotaError(t *testing.T) {
	hook := FallbackToPDStandard{
		logger: logging.NewToolLogger("test"),
	}
	wantRetry, wrapped := hook.PostRunHook(errQuota)
	assert.Equal(t, errQuota, wrapped)
	assert.True(t, wantRetry)
	assert.True(t, hook.shouldFallback)
}

func Test_FallbackToPDStandard_PostRunHook_DoesntRequestsRetryIfNotQuotaError(t *testing.T) {
	hook := FallbackToPDStandard{
		shouldFallback: true,
		logger:         logging.NewToolLogger("test"),
	}
	wantRetry, wrapped := hook.PostRunHook(errNotQuota)
	assert.Equal(t, errNotQuota, wrapped)
	assert.False(t, wantRetry)
}

func Test_FallbackToPDStandard_PostRunHook_OnlyFallsBackOnce(t *testing.T) {
	hook := FallbackToPDStandard{
		shouldFallback: true,
		logger:         logging.NewToolLogger("test"),
	}
	wantRetry, wrapped := hook.PostRunHook(errQuota)
	assert.Equal(t, errQuota, wrapped)
	assert.False(t, wantRetry)
}

func Test_FallbackToPDStandard_PreRunHook_DoesntModifyWorkflowOnFirstRun(t *testing.T) {
	hook := FallbackToPDStandard{
		logger: logging.NewToolLogger("test"),
	}
	// If the hook attempts to modify the workflow,
	// then the test will fail with a panic.
	var wf *daisy.Workflow
	assert.NoError(t, hook.PreRunHook(wf))
}

func Test_FallbackToPDStandard_PreRunHook_RemovesSSDIfFallbackRequired(t *testing.T) {
	hook := FallbackToPDStandard{
		shouldFallback: true,
		logger:         logging.NewToolLogger("test"),
	}
	w := createSSDWorkflow()

	assert.NoError(t, hook.PreRunHook(w))

	assert.Equal(t, 1, len(*w.Steps["cd"].CreateDisks))
	assert.Equal(t, "pd-standard", (*w.Steps["cd"].CreateDisks)[0].Type)

	assert.Equal(t, 1, len((*w.Steps["ci"].CreateInstances).Instances))
	assert.Equal(t, 1, len(((*w.Steps["ci"].CreateInstances).Instances)[0].Instance.Disks))
	assert.Equal(t, "pd-standard", (w.Steps["ci"].CreateInstances.Instances)[0].Instance.Disks[0].InitializeParams.DiskType)

	assert.Equal(t, 1, len(*w.Steps["iw"].IncludeWorkflow.Workflow.Steps["iw-cd"].CreateDisks))
	assert.Equal(t, "pd-standard", (*w.Steps["iw"].IncludeWorkflow.Workflow.Steps["iw-cd"].CreateDisks)[0].Type)
}

func createSSDWorkflow() *daisy.Workflow {
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"cd": {
			CreateDisks: &daisy.CreateDisks{
				{
					Disk: compute.Disk{
						Type: "pd-ssd",
					},
				},
			},
		},
		"ci": {
			CreateInstances: &daisy.CreateInstances{
				Instances: []*daisy.Instance{
					{
						Instance: compute.Instance{
							Disks: []*compute.AttachedDisk{{
								InitializeParams: &compute.AttachedDiskInitializeParams{
									DiskType: "pd-ssd",
								},
							}},
						},
					},
				},
			},
		},
		"iw": {
			IncludeWorkflow: &daisy.IncludeWorkflow{
				Workflow: &daisy.Workflow{
					Steps: map[string]*daisy.Step{
						"iw-cd": {
							CreateDisks: &daisy.CreateDisks{
								{
									Disk: compute.Disk{
										Type: "pd-ssd",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return w
}
