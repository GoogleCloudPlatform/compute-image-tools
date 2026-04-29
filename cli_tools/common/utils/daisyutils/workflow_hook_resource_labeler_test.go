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

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/stretchr/testify/assert"
	computeBeta "google.golang.org/api/compute/v0.beta"
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
				Instances: []*daisy.Instance{
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
		},
		"cibeta": {
			CreateInstances: &daisy.CreateInstances{
				InstancesBeta: []*daisy.InstanceBeta{
					{
						Instance: computeBeta.Instance{
							Disks:  []*computeBeta.AttachedDisk{{Source: "key1"}},
							Labels: map[string]string{"labelKey": "labelValue"},
						},
					},
					{
						Instance: computeBeta.Instance{
							Disks: []*computeBeta.AttachedDisk{{Source: "key2"}},
						},
					},
				},
			},
		},
	}

	rl := createTestResourceLabeler(buildID, userLabels)
	rl.LabelResources(w)
	validateLabels(t, &(*w.Steps["ci"].CreateInstances).Instances[0].Instance.Labels,
		"gce-image-import-tmp", buildID, &existingLabels)
	validateLabels(t, &(*w.Steps["ci"].CreateInstances).Instances[1].Instance.Labels,
		"gce-image-import-tmp", buildID)

	validateLabels(t, &(*w.Steps["cibeta"].CreateInstances).InstancesBeta[0].Instance.Labels,
		"gce-image-import-tmp", buildID, &existingLabels)
	validateLabels(t, &(*w.Steps["cibeta"].CreateInstances).InstancesBeta[1].Instance.Labels,
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
				Images: []*daisy.Image{
					{
						Image: compute.Image{
							Name:   "final-image-1",
							Labels: map[string]string{"labelKey": "labelValue"},
						},
					},
					{
						Image: compute.Image{
							Name: "final-image-2",
						},
					},
					{
						Image: compute.Image{
							Name:   "untranslated-image-1",
							Labels: map[string]string{"labelKey": "labelValue"},
						},
					},
					{
						Image: compute.Image{
							Name: "untranslated-image-2",
						},
					},
				},
				ImagesBeta: []*daisy.ImageBeta{
					{
						Image: computeBeta.Image{
							Name:   "final-image-1",
							Labels: map[string]string{"labelKey": "labelValue"},
						},
					},
					{
						Image: computeBeta.Image{
							Name: "final-image-2",
						},
					},
					{
						Image: computeBeta.Image{
							Name:   "untranslated-image-1",
							Labels: map[string]string{"labelKey": "labelValue"},
						},
					},
					{
						Image: computeBeta.Image{
							Name: "untranslated-image-2",
						},
					},
				},
			},
		},
	}

	rl := createTestResourceLabeler(buildID, userLabels)
	rl.LabelResources(w)

	validateLabels(t, &(*w.Steps["cimg"].CreateImages).Images[0].Image.Labels, "gce-image-import",
		buildID, &existingLabels)
	validateLabels(t, &(*w.Steps["cimg"].CreateImages).Images[1].Image.Labels, "gce-image-import",
		buildID)
	validateLabels(t, &(*w.Steps["cimg"].CreateImages).Images[2].Image.Labels,
		"gce-image-import-tmp", buildID, &existingLabels)
	validateLabels(t, &(*w.Steps["cimg"].CreateImages).Images[3].Image.Labels,
		"gce-image-import-tmp", buildID)
}

func TestUpdateWorkflowImageStorageLocationSet(t *testing.T) {
	buildID := "abc"

	w := daisy.New()
	existingLabels := map[string]string{"labelKey": "labelValue"}
	w.Steps = map[string]*daisy.Step{
		"cimg": {
			CreateImages: &daisy.CreateImages{
				Images: []*daisy.Image{
					{
						Image: compute.Image{
							Name:   "final-image-1",
							Labels: map[string]string{"labelKey": "labelValue"},
						},
					},
				},
			},
		},
	}

	rl := createTestResourceLabeler(buildID, userLabels)
	rl.ImageLocation = "europe-west5"

	rl.LabelResources(w)

	validateLabels(t, &(*w.Steps["cimg"].CreateImages).Images[0].Image.Labels, "gce-image-import",
		buildID, &existingLabels)

	assert.Equal(t, 1, len((*w.Steps["cimg"].CreateImages).Images[0].Image.StorageLocations))
	assert.Equal(t, "europe-west5", (*w.Steps["cimg"].CreateImages).Images[0].Image.StorageLocations[0])
}

func createTestResourceLabeler(buildID string, userLabels map[string]string) *ResourceLabeler {
	return &ResourceLabeler{
		BuildID: buildID, UserLabels: userLabels, BuildIDLabelKey: "gce-image-import-build-id",
		InstanceLabelKeyRetriever: func(instanceName string) string {
			return "gce-image-import-tmp"
		},
		DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
			return "gce-image-import-tmp"
		},
		ImageLabelKeyRetriever: func(imageName string) string {
			imageTypeLabel := "gce-image-import"
			if strings.Contains(imageName, "untranslated") {
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
