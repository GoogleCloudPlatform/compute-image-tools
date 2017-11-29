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
	"fmt"
	"net/http"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/googleapi"
)

var (
	images      = map[*Workflow]*imageRegistry{}
	imagesMu    sync.Mutex
	imageURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/images\/((family/(?P<family>%[2]s))?|(?P<image>%[2]s))$`, projectRgxStr, rfc1035))
)

type imageRegistry struct {
	baseResourceRegistry
}

func initImageRegistry(w *Workflow) {
	ir := &imageRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "image", urlRgx: imageURLRgx}}
	ir.baseResourceRegistry.deleteFn = ir.deleteFn
	ir.init()
	imagesMu.Lock()
	images[w] = ir
	imagesMu.Unlock()
}

func (ir *imageRegistry) deleteFn(res *resource) dErr {
	m := namedSubexp(imageURLRgx, res.link)
	err := ir.w.ComputeClient.DeleteImage(m["project"], m["image"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}

var imagesCache struct {
	exists map[string][]string
	mu     sync.Mutex
}

var familiesCache struct {
	exists map[string][]string
	mu     sync.Mutex
}

// imageExists should only be used during validation for existing GCE images
// and should not be relied or populated for daisy created resources.
func imageExists(client compute.Client, project, family, name string) (bool, dErr) {
	if family != "" {
		familiesCache.mu.Lock()
		defer familiesCache.mu.Unlock()
		if familiesCache.exists == nil {
			familiesCache.exists = map[string][]string{}
		}
		if _, ok := familiesCache.exists[project]; !ok {
			familiesCache.exists[project] = []string{}
		}
		if strIn(name, familiesCache.exists[project]) {
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
				return false, nil
			}
		}
		familiesCache.exists[project] = append(familiesCache.exists[project], name)
		return true, nil
	}

	if name == "" {
		return false, errf("must provide either family or name")
	}
	imagesCache.mu.Lock()
	defer imagesCache.mu.Unlock()
	if imagesCache.exists == nil {
		imagesCache.exists = map[string][]string{}
	}
	if _, ok := imagesCache.exists[project]; !ok {
		il, err := client.ListImages(project)
		if err != nil {
			return false, errf("error listing images for project %q: %v", project, err)
		}
		var images []string
		for _, i := range il {
			if i.Deprecated != nil {
				if i.Deprecated.State == "OBSOLETE" || i.Deprecated.State == "DELETED" {
					continue
				}
			}
			images = append(images, i.Name)
		}
		imagesCache.exists[project] = images
	}
	return strIn(name, imagesCache.exists[project]), nil
}
