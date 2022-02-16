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
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var (
	machineImageURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/machineImages\/(?P<machineImage>%[2]s)$`, projectRgxStr, rfc1035))
)

// machineImageExists should only be used during validation for existing GCE
// machine images and should not be relied or populated for daisy created resources.
func (w *Workflow) machineImageExists(project, machineImage string) (bool, DError) {
	return w.machineImageCache.resourceExists(func(project string, opts ...daisyCompute.ListCallOption) (interface{}, error) {
		return w.ComputeClient.ListMachineImages(project)
	}, project, machineImage)
}

// MachineImage is used to create a GCE machine image.
type MachineImage struct {
	compute.MachineImage
	Resource

	// Should an existing machine image of the same name be deleted.
	// Defaults to false which will fail validation if a machine image with the
	// same name exists in the project.
	OverWrite bool `json:",omitempty"`
}

// MarshalJSON is a workaround to prevent MachineImage from using computeBeta.MachineImage's implementation.
func (mi *MachineImage) MarshalJSON() ([]byte, error) {
	return json.Marshal(*mi)
}

func (mi *MachineImage) populate(ctx context.Context, s *Step) DError {
	var errs DError

	mi.Name, errs = mi.Resource.populateWithGlobal(ctx, s, mi.Name)
	mi.Description = strOr(mi.Description, fmt.Sprintf("Machine Image created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username))
	mi.link = fmt.Sprintf("projects/%s/global/machineImages/%s", mi.Project, mi.Name)

	errs = addErrs(errs, mi.populateSourceInstance())
	return errs
}

func (mi *MachineImage) populateSourceInstance() DError {
	if instanceURLRgx.MatchString(mi.SourceInstance) {
		mi.SourceInstance = extendPartialURL(mi.SourceInstance, mi.Project)
	}
	return nil
}

func (mi *MachineImage) validate(ctx context.Context, s *Step) DError {
	pre := fmt.Sprintf("cannot create machine image %q", mi.daisyName)
	errs := mi.Resource.validate(ctx, s, pre)

	// Source instance checking.
	if mi.SourceInstance == "" {
		errs = addErrs(errs, Errf("%s: must provide SourceInstance", pre))
	}
	if _, err := s.w.instances.regUse(mi.SourceInstance, s); err != nil {
		errs = addErrs(errs, newErr("failed to get source instance", err))
	}

	// Register machine image creation.
	errs = addErrs(errs, s.w.machineImages.regCreate(mi.daisyName, &mi.Resource, s, mi.OverWrite))
	return errs
}

type machineImageRegistry struct {
	baseResourceRegistry
}

func newMachineImageRegistry(w *Workflow) *machineImageRegistry {
	ir := &machineImageRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "machineImage", urlRgx: machineImageURLRgx}}
	ir.baseResourceRegistry.deleteFn = ir.deleteFn
	ir.init()
	return ir
}

func (ir *machineImageRegistry) deleteFn(res *Resource) DError {
	m := NamedSubexp(machineImageURLRgx, res.link)
	err := ir.w.ComputeClient.DeleteMachineImage(m["project"], m["machineImage"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, "failed to delete machine image", err)
	}
	return newErr("failed to delete machine image", err)
}
