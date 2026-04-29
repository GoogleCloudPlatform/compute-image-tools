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
	daisy "github.com/GoogleCloudPlatform/compute-daisy"
)

// ResourceLabeler is responsible for labelling GCE resources (instances, disks and images) with
// labels used to track resource creation by import processes.
type ResourceLabeler struct {
	BuildID                   string
	UserLabels                map[string]string
	BuildIDLabelKey           string
	ImageLocation             string
	InstanceLabelKeyRetriever InstanceLabelKeyRetrieverFunc
	DiskLabelKeyRetriever     DiskLabelKeyRetrieverFunc
	ImageLabelKeyRetriever    ImageLabelKeyRetrieverFunc
}

// NewResourceLabeler creates a ResourceLabeler that labels based on the tool's name.
func NewResourceLabeler(tool string, buildID string, labels map[string]string, location string) *ResourceLabeler {
	return &ResourceLabeler{
		BuildID:         buildID,
		UserLabels:      labels,
		BuildIDLabelKey: tool + "-build-id",
		ImageLocation:   location,
		InstanceLabelKeyRetriever: func(instanceName string) string {
			return tool + "-tmp"
		},
		DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
			return tool + "-tmp"
		},
		ImageLabelKeyRetriever: func(imageName string) string {
			return tool + "-tmp"
		}}
}

// InstanceLabelKeyRetrieverFunc returns GCE label key to be added to given instance
type InstanceLabelKeyRetrieverFunc func(instanceName string) string

// DiskLabelKeyRetrieverFunc returns GCE label key to be added to given disk
type DiskLabelKeyRetrieverFunc func(disk *daisy.Disk) string

// ImageLabelKeyRetrieverFunc returns GCE label key to be added to given image
type ImageLabelKeyRetrieverFunc func(imageName string) string

// PreRunHook is a facade over LabelResources to implement the WorkflowHook interface; no errors will be returned.
func (rl *ResourceLabeler) PreRunHook(wf *daisy.Workflow) error {
	rl.LabelResources(wf)
	return nil
}

// LabelResources labels workflow resources temporary and permanent resources with appropriate
// labels
func (rl *ResourceLabeler) LabelResources(workflow *daisy.Workflow) {
	workflow.IterateWorkflowSteps(rl.labelResourcesInStep)
}

func (rl *ResourceLabeler) labelResourcesInStep(step *daisy.Step) {
	if step.CreateInstances != nil {
		for _, instance := range step.CreateInstances.Instances {
			instance.Instance.Labels =
				rl.updateResourceLabels(instance.Instance.Labels, rl.InstanceLabelKeyRetriever(instance.Name))
		}
		for _, instance := range step.CreateInstances.InstancesBeta {
			instance.Instance.Labels =
				rl.updateResourceLabels(instance.Instance.Labels, rl.InstanceLabelKeyRetriever(instance.Name))
		}
	}
	if step.CreateDisks != nil {
		for _, disk := range *step.CreateDisks {
			disk.Disk.Labels =
				rl.updateResourceLabels(disk.Disk.Labels, rl.DiskLabelKeyRetriever(disk))
		}
	}
	if step.CreateImages != nil {
		for _, image := range step.CreateImages.Images {
			if rl.ImageLocation != "" {
				image.Image.StorageLocations = []string{rl.ImageLocation}
			}

			image.Image.Labels =
				rl.updateResourceLabels(image.Image.Labels, rl.ImageLabelKeyRetriever(image.Name))
		}
	}
}

func (rl *ResourceLabeler) updateResourceLabels(labels map[string]string, imageTypeLabel string) map[string]string {
	labels = rl.extendWithImageImportLabels(labels, imageTypeLabel)
	labels = rl.extendWithUserLabels(labels)
	return labels
}

// Extend labels with image import related labels
func (rl *ResourceLabeler) extendWithImageImportLabels(labels map[string]string, imageTypeLabel string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[imageTypeLabel] = "true"
	labels[rl.BuildIDLabelKey] = rl.BuildID

	return labels
}

func (rl *ResourceLabeler) extendWithUserLabels(labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}

	if rl.UserLabels == nil || len(rl.UserLabels) == 0 {
		return labels
	}

	for key, value := range rl.UserLabels {
		labels[key] = value
	}
	return labels
}
