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
	"sync"

	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

var (
	imageCache struct {
		exists map[string][]*compute.Image
		mu     sync.Mutex
	}
	imageFamilyCache struct {
		exists map[string][]string
		mu     sync.Mutex
	}
	imageURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/images\/((family/(?P<family>%[2]s))?|(?P<image>%[2]s))$`, projectRgxStr, rfc1035))
)

// imageExists should only be used during validation for existing GCE images
// and should not be relied or populated for daisy created resources.
func imageExists(client daisyCompute.Client, project, family, name string) (bool, DError) {
	if family != "" {
		imageFamilyCache.mu.Lock()
		defer imageFamilyCache.mu.Unlock()
		if imageFamilyCache.exists == nil {
			imageFamilyCache.exists = map[string][]string{}
		}
		if _, ok := imageFamilyCache.exists[project]; !ok {
			imageFamilyCache.exists[project] = []string{}
		}
		if strIn(name, imageFamilyCache.exists[project]) {
			return true, nil
		}

		img, err := client.GetImageFromFamily(project, family)
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
		imageFamilyCache.exists[project] = append(imageFamilyCache.exists[project], name)
		return true, nil
	}

	if name == "" {
		return false, Errf("must provide either family or name")
	}
	imageCache.mu.Lock()
	defer imageCache.mu.Unlock()
	if imageCache.exists == nil {
		imageCache.exists = map[string][]*compute.Image{}
	}
	if _, ok := imageCache.exists[project]; !ok {
		il, err := client.ListImages(project)
		if err != nil {
			return false, Errf("error listing images for project %q: %v", project, err)
		}
		imageCache.exists[project] = il
	}

	for _, i := range imageCache.exists[project] {
		if name == i.Name {
			if i.Deprecated != nil && (i.Deprecated.State == "OBSOLETE" || i.Deprecated.State == "DELETED") {
				return true, typedErrf(imageObsoleteDeletedError, "image %q in state %q", name, i.Deprecated.State)
			}
			return true, nil
		}
	}

	return false, nil
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
	appendGuestOsFeatures(featureType string)
	getGuestOsFeatures() guestOsFeatures
}

//ImageBase is a base struct for GA/Beta images. It holds the shared properties between the two.
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

func (i *Image) appendGuestOsFeatures(featureType string) {
	i.Image.GuestOsFeatures = append(i.Image.GuestOsFeatures, &compute.GuestOsFeature{Type: featureType})
}

func (i *Image) getGuestOsFeatures() guestOsFeatures {
	return i.GuestOsFeatures
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

func (i *ImageBeta) appendGuestOsFeatures(featureType string) {
	i.Image.GuestOsFeatures = append(i.Image.GuestOsFeatures, &computeBeta.GuestOsFeature{Type: featureType})
}

func (i *ImageBeta) getGuestOsFeatures() guestOsFeatures {
	return i.GuestOsFeatures
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

func populate(ctx context.Context, ii ImageInterface, ib *ImageBase, s *Step) DError {
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
	populateGuestOSFeatures(ii, ib, s.w)
	return errs
}

func populateGuestOSFeatures(ii ImageInterface, ib *ImageBase, w *Workflow) {
	if ii.getGuestOsFeatures() == nil {
		return
	}
	for _, f := range ii.getGuestOsFeatures() {
		ii.appendGuestOsFeatures(f)
	}
	return
}

func validate(ctx context.Context, ii ImageInterface, ib *ImageBase, licenses []string, s *Step) DError {
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
		result := namedSubexp(licenseURLRegex, l)
		if exists, err := licenseExists(s.w.ComputeClient, result["project"], result["license"]); err != nil {
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
	m := namedSubexp(imageURLRgx, res.link)
	err := ir.w.ComputeClient.DeleteImage(m["project"], m["image"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, "failed to delete image", err)
	}
	return newErr("failed to delete image", err)
}
