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
	"regexp"
)

var (
	images      = map[*Workflow]*imageMap{}
	imageURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/images/(?P<image>%[2]s)|family/(?P<family>%[2]s)$`, rfc1035Proj, rfc1035))
)

type imageMap struct {
	baseResourceMap
}

func initImageMap(w *Workflow) {
	im := &imageMap{baseResourceMap: baseResourceMap{w: w, typeName: "image", urlRgx: imageURLRgx}}
	im.baseResourceMap.deleteFn = im.deleteFn
	im.init()
	images[w] = im
}

func (im *imageMap) deleteFn(r *resource) error {
	m := namedSubexp(imageURLRgx, r.link)
	if err := im.w.ComputeClient.DeleteImage(m["project"], m["image"]); err != nil {
		return err
	}
	return nil
}
