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
	"fmt"
	"regexp"
)

var (
	disks      = map[*Workflow]*diskMap{}
	diskURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[1]s)/disks/(?P<disk>%[1]s)$`, rfc1035))
)

type diskMap struct {
	baseResourceMap
	attachments map[*resource]map[*resource]diskAttachment
}

type diskAttachment struct {
	mode               string
	attacher, detacher *Step
}

func initDiskMap(w *Workflow) {
	dm := &diskMap{baseResourceMap: baseResourceMap{w: w, typeName: "disk", urlRgx: diskURLRgx}}
	dm.baseResourceMap.deleteFn = dm.deleteFn
	disks[w] = dm
}

func (dm *diskMap) deleteFn(r *resource) error {
	m := namedSubexp(diskURLRgx, r.link)
	if err := dm.w.ComputeClient.DeleteDisk(m["project"], m["zone"], m["disk"]); err != nil {
		return err
	}
	r.deleted = true
	return nil
}

func (dm *diskMap) registerAttachment(dName, iName, mode string, s *Step) error {
	return nil
}

func (dm *diskMap) registerDetachment(dName, iName, mode string, s *Step) error {
	return nil
}
