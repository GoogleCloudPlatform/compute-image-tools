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

package daisy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"regexp"

	computeAlpha "google.golang.org/api/compute/v0.alpha"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

var (
	imageURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/images\/((family/(?P<family>%[2]s))?|(?P<image>%[2]s))$`, projectRgxStr, rfc1035))
)

// imageExists should only be used during validation for existing GCE images
// and should not be relied or populated for daisy created resources.
func (w *Workflow) imageExists(project, family, image string) (bool, DError) {
	if family != "" {
		w.imageFamilyCache.mu.Lock()
		defer w.imageFamilyCache.mu.Unlock()
		if w.imageFamilyCache.exists == nil {
			w.imageFamilyCache.exists = map[string]map[string]interface{}{}
		}
		if _, ok := w.imageFamilyCache.exists[project]; !ok {
			w.imageFamilyCache.exists[project] = map[string]interface{}{}
		}
		if nameInResourceMap(image, w.imageFamilyCache.exists[project]) {
			return true, nil
		}

		img, err := w.ComputeClient.GetImageFromFamily(project, family)
		if err != nil {
			if apiErr, ok := err.(*googleapi.Error); ok && apiErr.Code == http.StatusNotFound {
				return false, nil
			}
			return false, typedErr(apiError, "failed to get image from family", err)
		}
		if img.Deprecated != nil {
			if img.Deprecated.State == "OBSOLETE" || img.Deprecated.State == "DELETED" {
				return true, typedErrf(imageObsoleteDeletedError, "image %q in state %q", img.Name, img.Deprecated.State)
			}
		}
		w.imageFamilyCache.exists[project][img.Name] = img
		return true, nil
	}

	if image == "" {
		return false, Errf("must provide either family or name")
	}
	w.imageCache.mu.Lock()
	defer w.imageCache.mu.Unlock()
	err := w.imageCache.loadCache(func(project string, opts ...daisyCompute.ListCallOption) (interface{}, error) {
		return w.ComputeClient.ListImages(project)
	}, project, image)
	if err != nil {
		ic, err := w.ComputeClient.GetImage(project, image)
		if err != nil {
			return false, typedErr(apiError, "error getting resource for project", err)
		}
		return true, errIfDeprecatedOrDeleted(ic, image)
	}

	for _, i := range w.imageCache.exists[project] {
		if ic, ok := i.(*compute.Image); ok && image == ic.Name {
			return true, errIfDeprecatedOrDeleted(ic, image)
		}
	}

	return false, nil
}

func errIfDeprecatedOrDeleted(ic *compute.Image, image string) DError {
	if ic.Deprecated != nil && (ic.Deprecated.State == "OBSOLETE" || ic.Deprecated.State == "DELETED") {
		return typedErrf(imageObsoleteDeletedError, "image %q in state %q", image, ic.Deprecated.State)
	}
	return nil
}

//ImageInterface represent abstract Image across different API stages (Alpha, Beta, API)
type ImageInterface interface {
	getName() string
	setName(name string)
	getDescription() string
	setDescription(description string)
	getSourceDisk() string
	setSourceDisk(sourceDisk string)
	getSourceImage() string
	setSourceImage(sourceImage string)
	hasRawDisk() bool
	getRawDiskSource() string
	setRawDiskSource(rawDiskSource string)
	create(cc daisyCompute.Client) error
	markCreatedInWorkflow()
	delete(cc daisyCompute.Client) error
	populateGuestOSFeatures()
}

//ImageBase is a base struct for GA/Beta/Alpha images. It holds the shared properties between the two.
type ImageBase struct {
	Resource

	// Should an existing image of the same name be deleted, defaults to false
	// which will fail validation.
	OverWrite bool `json:",omitempty"`

	//Ignores license validation if 403/forbidden returned
	IgnoreLicenseValidationIfForbidden bool `json:",omitempty"`
}

// Image is used to create a GCE image using GA API.
// Supported sources are a GCE disk or a RAW image listed in Workflow.Sources.
type Image struct {
	ImageBase
	compute.Image

	// GuestOsFeatures to set for the image.
	GuestOsFeatures guestOsFeatures `json:"guestOsFeatures,omitempty"`
}

func (i *Image) getName() string {
	return i.Name
}

func (i *Image) setName(name string) {
	i.Name = name
}

func (i *Image) getDescription() string {
	return i.Description
}

func (i *Image) setDescription(description string) {
	i.Description = description
}

func (i *Image) getSourceDisk() string {
	return i.SourceDisk
}

func (i *Image) setSourceDisk(sourceDisk string) {
	i.SourceDisk = sourceDisk
}

func (i *Image) getSourceImage() string {
	return i.SourceImage
}
func (i *Image) setSourceImage(sourceImage string) {
	i.SourceImage = sourceImage
}

func (i *Image) hasRawDisk() bool {
	return i.RawDisk != nil
}

func (i *Image) getRawDiskSource() string {
	return i.RawDisk.Source
}

func (i *Image) setRawDiskSource(rawDiskSource string) {
	i.RawDisk.Source = rawDiskSource
}

func (i *Image) create(cc daisyCompute.Client) error {
	return cc.CreateImage(i.Project, &i.Image)
}

func (i *Image) markCreatedInWorkflow() {
	i.createdInWorkflow = true
}

func (i *Image) delete(cc daisyCompute.Client) error {
	return cc.DeleteImage(i.Project, i.Name)
}

func (i *Image) populateGuestOSFeatures() {
	if i.GuestOsFeatures == nil {
		return
	}
	for _, f := range i.GuestOsFeatures {
		i.Image.GuestOsFeatures = append(i.Image.GuestOsFeatures, &compute.GuestOsFeature{Type: f})
	}
}

// ImageBeta is used to create a GCE image using Beta API.
// Supported sources are a GCE disk or a RAW image listed in Workflow.Sources.
type ImageBeta struct {
	ImageBase
	computeBeta.Image

	// GuestOsFeatures to set for the image.
	GuestOsFeatures guestOsFeatures `json:"guestOsFeatures,omitempty"`
}

func (i *ImageBeta) getName() string {
	return i.Name
}

func (i *ImageBeta) setName(name string) {
	i.Name = name
}

func (i *ImageBeta) getDescription() string {
	return i.Description
}

func (i *ImageBeta) setDescription(description string) {
	i.Description = description
}

func (i *ImageBeta) getSourceDisk() string {
	return i.SourceDisk
}

func (i *ImageBeta) setSourceDisk(sourceDisk string) {
	i.SourceDisk = sourceDisk
}

func (i *ImageBeta) getSourceImage() string {
	return i.SourceImage
}

func (i *ImageBeta) setSourceImage(sourceImage string) {
	i.SourceImage = sourceImage
}

func (i *ImageBeta) hasRawDisk() bool {
	return i.RawDisk != nil
}

func (i *ImageBeta) getRawDiskSource() string {
	return i.RawDisk.Source
}

func (i *ImageBeta) setRawDiskSource(rawDiskSource string) {
	i.RawDisk.Source = rawDiskSource
}

func (i *ImageBeta) create(cc daisyCompute.Client) error {
	return cc.CreateImageBeta(i.Project, &i.Image)
}

func (i *ImageBeta) markCreatedInWorkflow() {
	i.createdInWorkflow = true
}

func (i *ImageBeta) delete(cc daisyCompute.Client) error {
	return cc.DeleteImage(i.Project, i.Name)
}

func (i *ImageBeta) populateGuestOSFeatures() {
	if i.GuestOsFeatures == nil {
		return
	}
	for _, f := range i.GuestOsFeatures {
		i.Image.GuestOsFeatures = append(i.Image.GuestOsFeatures, &computeBeta.GuestOsFeature{Type: f})
	}
}

// ImageAlpha is used to create a GCE image using Alpha API.
// Supported sources are a GCE disk or a RAW image listed in Workflow.Sources.
type ImageAlpha struct {
	ImageBase
	computeAlpha.Image

	// GuestOsFeatures to set for the image.
	GuestOsFeatures guestOsFeatures `json:"guestOsFeatures,omitempty"`
}

func (i *ImageAlpha) getName() string {
	return i.Name
}

func (i *ImageAlpha) setName(name string) {
	i.Name = name
}

func (i *ImageAlpha) getDescription() string {
	return i.Description
}

func (i *ImageAlpha) setDescription(description string) {
	i.Description = description
}

func (i *ImageAlpha) getSourceDisk() string {
	return i.SourceDisk
}

func (i *ImageAlpha) setSourceDisk(sourceDisk string) {
	i.SourceDisk = sourceDisk
}

func (i *ImageAlpha) getSourceImage() string {
	return i.SourceImage
}

func (i *ImageAlpha) setSourceImage(sourceImage string) {
	i.SourceImage = sourceImage
}

func (i *ImageAlpha) hasRawDisk() bool {
	return i.RawDisk != nil
}

func (i *ImageAlpha) getRawDiskSource() string {
	return i.RawDisk.Source
}

func (i *ImageAlpha) setRawDiskSource(rawDiskSource string) {
	i.RawDisk.Source = rawDiskSource
}

func (i *ImageAlpha) create(cc daisyCompute.Client) error {
	return cc.CreateImageAlpha(i.Project, &i.Image)
}

func (i *ImageAlpha) markCreatedInWorkflow() {
	i.createdInWorkflow = true
}

func (i *ImageAlpha) delete(cc daisyCompute.Client) error {
	return cc.DeleteImage(i.Project, i.Name)
}

func (i *ImageAlpha) populateGuestOSFeatures() {
	if i.GuestOsFeatures == nil {
		return
	}
	for _, f := range i.GuestOsFeatures {
		i.Image.GuestOsFeatures = append(i.Image.GuestOsFeatures, &computeAlpha.GuestOsFeature{Type: f})
	}
}


// MarshalJSON is a hacky workaround to prevent Image from using compute.Image's implementation.
func (i *Image) MarshalJSON() ([]byte, error) {
	return json.Marshal(*i)
}

type guestOsFeatures []string

// UnmarshalJSON unmarshals GuestOsFeatures.
func (g *guestOsFeatures) UnmarshalJSON(b []byte) error {
	// Support GCE API struct.
	var cg []compute.GuestOsFeature
	if err := json.Unmarshal(b, &cg); err == nil {
		for _, f := range cg {
			*g = append(*g, f.Type)
		}
		return nil
	}

	type dg guestOsFeatures
	return json.Unmarshal(b, (*dg)(g))
}

func (ib *ImageBase) populate(ctx context.Context, ii ImageInterface, s *Step) DError {
	name, errs := ib.Resource.populateWithGlobal(ctx, s, ii.getName())
	ii.setName(name)

	ii.setDescription(strOr(ii.getDescription(), fmt.Sprintf("Image created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username)))

	if diskURLRgx.MatchString(ii.getSourceDisk()) {
		ii.setSourceDisk(extendPartialURL(ii.getSourceDisk(), ib.Project))
	}

	if imageURLRgx.MatchString(ii.getSourceImage()) {
		ii.setSourceImage(extendPartialURL(ii.getSourceImage(), ib.Project))
	}

	if ii.hasRawDisk() {
		if s.w.sourceExists(ii.getRawDiskSource()) {
			ii.setRawDiskSource(s.w.getSourceGCSAPIPath(ii.getRawDiskSource()))
		} else if p, err := getGCSAPIPath(ii.getRawDiskSource()); err == nil {
			ii.setRawDiskSource(p)
		} else {
			errs = addErrs(errs, Errf("bad value for RawDisk.Source: %q", ii.getRawDiskSource()))
		}
	}
	ib.link = fmt.Sprintf("projects/%s/global/images/%s", ib.Project, ii.getName())
	ii.populateGuestOSFeatures()
	return errs
}

func (ib *ImageBase) validate(ctx context.Context, ii ImageInterface, licenses []string, s *Step) DError {
	pre := fmt.Sprintf("cannot create image %q", ib.daisyName)
	errs := ib.Resource.validate(ctx, s, pre)

	if !xor(!xor(ii.getSourceDisk() == "", ii.getSourceImage() == ""), !ii.hasRawDisk()) {
		errs = addErrs(errs, Errf("%s: must provide either SourceImage, SourceDisk or RawDisk, exclusively", pre))
	}

	// Source disk checking.
	if ii.getSourceDisk() != "" {
		if _, err := s.w.disks.regUse(ii.getSourceDisk(), s); err != nil {
			errs = addErrs(errs, newErr("failed to get source disk", err))
		}
	}

	// Source image checking.
	if ii.getSourceImage() != "" {
		_, err := s.w.images.regUse(ii.getSourceImage(), s)
		errs = addErrs(errs, err)
	}

	// RawDisk.Source checking.
	if ii.hasRawDisk() {
		sBkt, sObj, err := splitGCSPath(ii.getRawDiskSource())
		errs = addErrs(errs, err)

		// Check if this image object is created by this workflow, otherwise check if object exists.
		if !strIn(path.Join(sBkt, sObj), s.w.objects.created) {
			if _, err := s.w.StorageClient.Bucket(sBkt).Object(sObj).Attrs(ctx); err != nil {
				errs = addErrs(errs, Errf("error reading object %s/%s: %v", sBkt, sObj, err))
			}
		}
	}

	// License checking.
	for _, l := range licenses {
		result := NamedSubexp(licenseURLRegex, l)
		if exists, err := s.w.licenseExists(result["project"], result["license"]); err != nil {
			if !(isGoogleAPIForbiddenError(err) && ib.IgnoreLicenseValidationIfForbidden) {
				errs = addErrs(errs, Errf("%s: bad license lookup: %q, error: %v", pre, l, err))
			}
		} else if !exists {
			errs = addErrs(errs, Errf("%s: license does not exist: %q", pre, l))
		}
	}

	// Register image creation.
	errs = addErrs(errs, s.w.images.regCreate(ib.daisyName, &ib.Resource, s, ib.OverWrite))
	return errs
}

func isGoogleAPIForbiddenError(err DError) bool {
	dErrConcrete, isDErrConcrete := err.(*dErrImpl)
	if isDErrConcrete && len(dErrConcrete.errs) > 0 {
		gAPIErr, isGAPIErr := dErrConcrete.errs[0].(*googleapi.Error)
		if isGAPIErr && gAPIErr.Code == 403 {
			return true
		}
	}
	return false
}

type imageRegistry struct {
	baseResourceRegistry
}

func newImageRegistry(w *Workflow) *imageRegistry {
	ir := &imageRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "image", urlRgx: imageURLRgx}}
	ir.baseResourceRegistry.deleteFn = ir.deleteFn
	ir.init()
	return ir
}

func (ir *imageRegistry) deleteFn(res *Resource) DError {
	m := NamedSubexp(imageURLRgx, res.link)
	err := ir.w.ComputeClient.DeleteImage(m["project"], m["image"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, "failed to delete image", err)
	}
	return newErr("failed to delete image", err)
}
