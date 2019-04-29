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

package daisyutils

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/compute/v1"
)

var (
	userLabels = map[string]string{"userkey1": "uservalue1", "userkey2": "uservalue2"}
)

func TestUpdateWorkflowInstancesLabelled(t *testing.T) {
	buildID := "abc"

	existingLabels := map[string]string{"labelKey": "labelValue"}
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"ci": {
			CreateInstances: &daisy.CreateInstances{
				{
					Instance: compute.Instance{
						Disks:  []*compute.AttachedDisk{{Source: "key1"}},
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Instance: compute.Instance{
						Disks: []*compute.AttachedDisk{{Source: "key2"}},
					},
				},
			},
		},
	}

	rl := createTestResourceLabeler(buildID, userLabels)
	rl.LabelResources(w)
	validateLabels(t, &(*w.Steps["ci"].CreateInstances)[0].Instance.Labels,
		"gce-image-import-tmp", buildID, &existingLabels)
	validateLabels(t, &(*w.Steps["ci"].CreateInstances)[1].Instance.Labels,
		"gce-image-import-tmp", buildID)
}

func TestUpdateWorkflowDisksLabelled(t *testing.T) {
	buildID := "abc"

	existingLabels := map[string]string{"labelKey": "labelValue"}
	w := createWorkflowWithCreateDisksStep()

	rl := createTestResourceLabeler(buildID, userLabels)
	rl.LabelResources(w)
	validateLabels(t, &(*w.Steps["cd"].CreateDisks)[0].Disk.Labels, "gce-image-import-tmp",
		buildID, &existingLabels)
	validateLabels(t, &(*w.Steps["cd"].CreateDisks)[1].Disk.Labels, "gce-image-import-tmp",
		buildID)
}

func TestUpdateWorkflowIncludedWorkflow(t *testing.T) {
	buildID := "abc"

	childWorkflow := createWorkflowWithCreateDisksStep()
	existingLabels := map[string]string{"labelKey": "labelValue"}

	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"cd": {
			IncludeWorkflow: &daisy.IncludeWorkflow{
				Workflow: childWorkflow,
			},
		},
	}

	rl := createTestResourceLabeler(buildID, userLabels)
	rl.LabelResources(w)
	validateLabels(t, &(*childWorkflow.Steps["cd"].CreateDisks)[0].Disk.Labels,
		"gce-image-import-tmp", buildID, &existingLabels)
	validateLabels(t, &(*childWorkflow.Steps["cd"].CreateDisks)[1].Disk.Labels,
		"gce-image-import-tmp", buildID)
}

func TestUpdateWorkflowImagesLabelled(t *testing.T) {
	buildID := "abc"

	w := daisy.New()
	existingLabels := map[string]string{"labelKey": "labelValue"}
	w.Steps = map[string]*daisy.Step{
		"cimg": {
			CreateImages: &daisy.CreateImages{
				{
					Image: computeAlpha.Image{
						Name:   "final-image-1",
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Image: computeAlpha.Image{
						Name: "final-image-2",
					},
				},
				{
					Image: computeAlpha.Image{
						Name:   "untranslated-image-1",
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Image: computeAlpha.Image{
						Name: "untranslated-image-2",
					},
				},
			},
		},
	}

	rl := createTestResourceLabeler(buildID, userLabels)
	rl.LabelResources(w)

	validateLabels(t, &(*w.Steps["cimg"].CreateImages)[0].Image.Labels, "gce-image-import",
		buildID, &existingLabels)
	validateLabels(t, &(*w.Steps["cimg"].CreateImages)[1].Image.Labels, "gce-image-import",
		buildID)

	validateLabels(t, &(*w.Steps["cimg"].CreateImages)[2].Image.Labels,
		"gce-image-import-tmp", buildID, &existingLabels)
	validateLabels(t, &(*w.Steps["cimg"].CreateImages)[3].Image.Labels,
		"gce-image-import-tmp", buildID)
}

func createTestResourceLabeler(buildID string, userLabels map[string]string) *ResourceLabeler {
	return &ResourceLabeler{
		BuildID: buildID, UserLabels: userLabels, BuildIDLabelKey: "gce-image-import-build-id",
		InstanceLabelKeyRetriever: func(instance *daisy.Instance) string {
			return "gce-image-import-tmp"
		},
		DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
			return "gce-image-import-tmp"
		},
		ImageLabelKeyRetriever: func(image *daisy.Image) string {
			imageTypeLabel := "gce-image-import"
			if strings.Contains(image.Image.Name, "untranslated") {
				imageTypeLabel = "gce-image-import-tmp"
			}
			return imageTypeLabel
		}}
}

func createWorkflowWithCreateDisksStep() *daisy.Workflow {
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"cd": {
			CreateDisks: &daisy.CreateDisks{
				{
					Disk: compute.Disk{
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Disk: compute.Disk{},
				},
			},
		},
	}
	return w
}

func validateLabels(t *testing.T, labels *map[string]string, typeLabel string, buildID string,
	extraLabelsArr ...*map[string]string) {
	var extraLabels *map[string]string
	if len(extraLabelsArr) > 0 {
		extraLabels = extraLabelsArr[0]
	}
	extraLabelCount := 0
	if extraLabels != nil {
		extraLabelCount = len(*extraLabels)
	}
	expectedLabelCount := extraLabelCount + 4
	if len(*labels) != expectedLabelCount {
		t.Errorf("Labels %v should have only %v elements", labels, expectedLabelCount)
	}
	assert.Equal(t, buildID, (*labels)["gce-image-import-build-id"])
	assert.Equal(t, "true", (*labels)[typeLabel])
	assert.Equal(t, "uservalue1", (*labels)["userkey1"])
	assert.Equal(t, "uservalue2", (*labels)["userkey2"])

	if extraLabels != nil {
		for extraKey, extraValue := range *extraLabels {
			if value, ok := (*labels)[extraKey]; !ok || value != extraValue {
				t.Errorf("Key %v from labels missing or value %v not matching", extraKey, value)
			}
		}
	}
}
