//  Copyright 2017 Google Inc. All Rights Reserved.
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

package workflow

import (
	"reflect"
	"testing"
)

func TestDeleteResourcesRun(t *testing.T) {
	wf := testWorkflow()
	wf.createdInstances = []string{
		namer("instance1", testWf, testSuffix),
		namer("instance2", testWf, testSuffix),
		namer("instance3", testWf, testSuffix),
		namer("instance4", testWf, testSuffix)}
	wf.createdImages = map[string]string{
		"image1": "link",
		"image2": "link",
		"image3": "link",
		"image4": "link"}
	wf.createdDisks = map[string]string{
		namer("disk1", testWf, testSuffix): "link",
		namer("disk2", testWf, testSuffix): "link",
		namer("disk3", testWf, testSuffix): "link",
		namer("disk4", testWf, testSuffix): "link"}

	dr := &DeleteResources{
		Instances: []string{"instance1", "instance2", "instance3"},
		Images:    []string{"image1", "image2", "image3"},
		Disks:     []string{"disk1", "disk2", "disk3"}}
	if err := dr.run(wf); err != nil {
		t.Fatalf("error running DeleteResources.run(): %v", err)
	}

	instWant := []string{namer("instance4", testWf, testSuffix)}
	if !reflect.DeepEqual(wf.createdInstances, instWant) {
		t.Errorf("Workflow.createdInstances does not match expectations, got: %q, want: %q", wf.createdInstances, instWant)
	}

	imgWant := map[string]string{"image4": "link"}
	if !reflect.DeepEqual(wf.createdImages, imgWant) {
		t.Errorf("Workflow.createdImages does not match expectations, got: %+v, want: %+v", wf.createdImages, imgWant)
	}

	diskWant := map[string]string{namer("disk4", testWf, testSuffix): "link"}
	if !reflect.DeepEqual(wf.createdDisks, diskWant) {
		t.Errorf("Workflow.createdDisks does not match expectations, got: %+v, want: %+v", wf.createdDisks, diskWant)
	}
}

func TestDeleteResourcesValidate(t *testing.T) {}
