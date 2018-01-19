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
	"regexp"
	"sync"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
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
		imageCache.exists = map[string][]*compute.Image{}
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
	compute.Image
	Resource

	// Should an existing image of the same name be deleted, defaults to false
	// which will fail validation.
	OverWrite bool
}

// MarshalJSON is a hacky workaround to prevent Image from using compute.Image's implementation.
func (i *Image) MarshalJSON() ([]byte, error) {
	return json.Marshal(*i)
}

func (i *Image) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	i.Name, _, errs = i.Resource.populate(ctx, s, i.Name, "")

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
	return errs
}

func (i *Image) validate(ctx context.Context, s *Step) dErr {
	errs := i.Resource.validate(ctx, s)

	if !xor(!xor(i.SourceDisk == "", i.SourceImage == ""), i.RawDisk == nil) {
		errs = addErrs(errs, errf("cannot create image %q: must provide either SourceImage, SourceDisk or RawDisk, exclusively", i.daisyName))
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
		if _, err := s.w.StorageClient.Bucket(sBkt).Object(sObj).Attrs(ctx); err != nil {
			errs = addErrs(errs, errf("error reading object %s/%s: %v", sBkt, sObj, err))
		}
	}

	// License checking.
	for _, l := range i.Licenses {
		result := namedSubexp(licenseURLRegex, l)
		if exists, err := licenseExists(s.w.ComputeClient, result["project"], result["license"]); err != nil {
			errs = addErrs(errs, errf("cannot create image %q: bad license lookup: %q, error: %v", i.daisyName, l, err))
		} else if !exists {
			errs = addErrs(errs, errf("cannot create image %q: license does not exist: %q", i.daisyName, l))
		}
	}

	// Register image creation.
	errs = addErrs(errs, s.w.images.regCreate(i.daisyName, &i.Resource, s, i.OverWrite))
	return errs
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
