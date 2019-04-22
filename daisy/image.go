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

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var (
	imageCache struct {
		exists map[string][]*computeAlpha.Image
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
func imageExists(client daisyCompute.Client, project, family, name string) (bool, dErr) {
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
			return false, typedErr(apiError, err)
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
		return false, errf("must provide either family or name")
	}
	imageCache.mu.Lock()
	defer imageCache.mu.Unlock()
	if imageCache.exists == nil {
		imageCache.exists = map[string][]*computeAlpha.Image{}
	}
	if _, ok := imageCache.exists[project]; !ok {
		il, err := client.ListImages(project)
		if err != nil {
			return false, errf("error listing images for project %q: %v", project, err)
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

// Image is used to create a GCE image.
// Supported sources are a GCE disk or a RAW image listed in Workflow.Sources.
type Image struct {
	computeAlpha.Image
	Resource

	// GuestOsFeatures to set for the image.
	GuestOsFeatures guestOsFeatures `json:"guestOsFeatures,omitempty"`
	// Should an existing image of the same name be deleted, defaults to false
	// which will fail validation.
	OverWrite bool `json:",omitempty"`

	//Ignores license validation if 403/forbidden returned
	IgnoreLicenseValidationIfForbidden bool `json:",omitempty"`
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

func (i *Image) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	i.Name, errs = i.Resource.populateWithGlobal(ctx, s, i.Name)

	i.Description = strOr(i.Description, fmt.Sprintf("Image created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username))

	if diskURLRgx.MatchString(i.SourceDisk) {
		i.SourceDisk = extendPartialURL(i.SourceDisk, i.Project)
	}

	if imageURLRgx.MatchString(i.SourceImage) {
		i.SourceImage = extendPartialURL(i.SourceImage, i.Project)
	}

	if i.RawDisk != nil {
		if s.w.sourceExists(i.RawDisk.Source) {
			i.RawDisk.Source = s.w.getSourceGCSAPIPath(i.RawDisk.Source)
		} else if p, err := getGCSAPIPath(i.RawDisk.Source); err == nil {
			i.RawDisk.Source = p
		} else {
			errs = addErrs(errs, errf("bad value for RawDisk.Source: %q", i.RawDisk.Source))
		}
	}
	i.link = fmt.Sprintf("projects/%s/global/images/%s", i.Project, i.Name)
	i.populateGuestOSFeatures(s.w)
	return errs
}

func (i *Image) populateGuestOSFeatures(w *Workflow) {
	if i.GuestOsFeatures == nil {
		return
	}
	for _, f := range i.GuestOsFeatures {
		i.Image.GuestOsFeatures = append(i.Image.GuestOsFeatures, &computeAlpha.GuestOsFeature{Type: f})
	}
	return
}

func (i *Image) validate(ctx context.Context, s *Step) dErr {
	pre := fmt.Sprintf("cannot create image %q", i.daisyName)
	errs := i.Resource.validate(ctx, s, pre)

	if !xor(!xor(i.SourceDisk == "", i.SourceImage == ""), i.RawDisk == nil) {
		errs = addErrs(errs, errf("%s: must provide either SourceImage, SourceDisk or RawDisk, exclusively", pre))
	}

	// Source disk checking.
	if i.SourceDisk != "" {
		if _, err := s.w.disks.regUse(i.SourceDisk, s); err != nil {
			errs = addErrs(errs, newErr(err))
		}
	}

	// Source image checking.
	if i.SourceImage != "" {
		_, err := s.w.images.regUse(i.SourceImage, s)
		errs = addErrs(errs, err)
	}

	// RawDisk.Source checking.
	if i.RawDisk != nil {
		sBkt, sObj, err := splitGCSPath(i.RawDisk.Source)
		errs = addErrs(errs, err)

		// Check if this image object is created by this workflow, otherwise check if object exists.
		if !strIn(path.Join(sBkt, sObj), s.w.objects.created) {
			if _, err := s.w.StorageClient.Bucket(sBkt).Object(sObj).Attrs(ctx); err != nil {
				errs = addErrs(errs, errf("error reading object %s/%s: %v", sBkt, sObj, err))
			}
		}
	}

	// License checking.
	for _, l := range i.Licenses {
		result := namedSubexp(licenseURLRegex, l)
		if exists, err := licenseExists(s.w.ComputeClient, result["project"], result["license"]); err != nil {
			if !(isGoogleAPIForbiddenError(err) && i.IgnoreLicenseValidationIfForbidden) {
				errs = addErrs(errs, errf("%s: bad license lookup: %q, error: %v", pre, l, err))
			}
		} else if !exists {
			errs = addErrs(errs, errf("%s: license does not exist: %q", pre, l))
		}
	}

	// Register image creation.
	errs = addErrs(errs, s.w.images.regCreate(i.daisyName, &i.Resource, s, i.OverWrite))
	return errs
}

func isGoogleAPIForbiddenError(err dErr) bool {
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

func (ir *imageRegistry) deleteFn(res *Resource) dErr {
	m := namedSubexp(imageURLRgx, res.link)
	err := ir.w.ComputeClient.DeleteImage(m["project"], m["image"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}
