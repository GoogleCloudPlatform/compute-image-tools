//  Copyright 2020 Google Inc. All Rights Reserved.
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

package daisy

import (
	"context"
	"fmt"
	"testing"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

func TestCreateMachineImagesRunSuccess(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	createCalled := false

	w.ComputeClient.(*daisyCompute.TestClient).CreateMachineImageFn = func(p string, i *compute.MachineImage) error {
		i.SelfLink = "insertedLink"
		createCalled = true
		return nil
	}
	w.instances.m = map[string]*Resource{"si": {link: "iLink"}}

	mi0 := &MachineImage{Resource: Resource{daisyName: "mi0"}, MachineImage: compute.MachineImage{Name: "realMI0", SourceInstance: "si"}}
	cmi := &CreateMachineImages{mi0}
	if err := cmi.run(ctx, s); err != nil {
		t.Errorf("unexpected error running CreateMachineImages.run(): %v", err)
	}
	if !createCalled {
		t.Errorf("CreateMachineImage not called")
	}
}

func TestCreateMachineImagesRunSuccessOnOverwrite(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}

	createCalled := false
	deleteCalled := false

	w.instances.m = map[string]*Resource{}
	w.ComputeClient.(*daisyCompute.TestClient).CreateMachineImageFn = func(p string, i *compute.MachineImage) error {
		i.SelfLink = "insertedLink"
		createCalled = true
		return nil
	}
	w.ComputeClient.(*daisyCompute.TestClient).DeleteMachineImageFn = func(p, mi string) error {
		deleteCalled = true
		return nil
	}
	cmi := &CreateMachineImages{
		{OverWrite: true, Resource: Resource{daisyName: "mi0"}, MachineImage: compute.MachineImage{Name: "realMI0", SourceInstance: "si"}},
	}
	if err := cmi.run(ctx, s); err != nil {
		t.Errorf("unexpected error running CreateMachineImages.run(): %v", err)
	}
	if !createCalled {
		t.Errorf("CreateMachineImage not called")
	}
	if !deleteCalled {
		t.Errorf("DeleteMachineImage not called")
	}
}

func TestCreateMachineImagesRunFailureOnComputeCreateError(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.instances.m = map[string]*Resource{}
	createErr := Errf("client error")
	w.ComputeClient.(*daisyCompute.TestClient).CreateMachineImageFn = func(p string, i *compute.MachineImage) error {
		i.SelfLink = "insertedLink"
		return createErr
	}

	cmi := &CreateMachineImages{
		{Resource: Resource{daisyName: "mi0"}, MachineImage: compute.MachineImage{Name: "realMI0", SourceInstance: "si"}},
	}
	if err := cmi.run(ctx, s); err != createErr {
		t.Errorf("CreateInstances.run() should have return compute client error: %v != %v", err, createErr)
	}
}

func TestCreateMachineImagesRunFailureOnComputeDeleteOnOverwriteError(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}

	w.instances.m = map[string]*Resource{}
	deleteErr := Errf("client error")
	w.ComputeClient.(*daisyCompute.TestClient).CreateMachineImageFn = func(p string, i *compute.MachineImage) error {
		i.SelfLink = "insertedLink"
		return nil
	}
	w.ComputeClient.(*daisyCompute.TestClient).DeleteMachineImageFn = func(p, mi string) error {
		return deleteErr
	}
	cmi := &CreateMachineImages{
		{OverWrite: true, Resource: Resource{daisyName: "mi0"}, MachineImage: compute.MachineImage{Name: "realMI0", SourceInstance: "si"}},
	}
	expectedErrorMessage := fmt.Sprintf("error deleting existing machine image: %v", deleteErr)
	if err := cmi.run(ctx, s); fmt.Sprintf("%v", err) != expectedErrorMessage {
		t.Errorf("CreateInstances.run() should have return compute client error: %v != %v", err, expectedErrorMessage)
	}
}
