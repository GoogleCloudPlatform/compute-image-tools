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
	"strings"
)

const (
	defaultAccessConfigType = "ONE_TO_ONE_NAT"
	defaultDiskMode         = diskModeRW
	defaultDiskType         = "pd-standard"
	diskModeRO              = "READ_ONLY"
	diskModeRW              = "READ_WRITE"
)

var (
	instances      = map[*Workflow]*instanceMap{}
	instanceURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[1]s)/instances/(?P<instance>%[1]s)$`, rfc1035))
	validDiskModes = []string{diskModeRO, diskModeRW}
)

type instanceMap struct {
	baseResourceMap
}

func initInstanceMap(w *Workflow) {
	im := &instanceMap{baseResourceMap: baseResourceMap{w: w, typeName: "instance", urlRgx: instanceURLRgx}}
	im.baseResourceMap.deleteFn = im.deleteFn
	im.init()
	instances[w] = im
}

func (im *instanceMap) deleteFn(r *resource) error {
	m := namedSubexp(instanceURLRgx, r.link)
	if err := im.w.ComputeClient.DeleteInstance(m["project"], m["zone"], m["instance"]); err != nil {
		return err
	}
	return nil
}

func (im *instanceMap) registerCreation(name string, r *resource, s *Step) error {
	// Base creation logic.
	if err := im.baseResourceMap.registerCreation(name, r, s); err != nil {
		return err
	}

	// Find the CreateInstance responsible for this.
	var ci *CreateInstance
	for _, ci = range *s.CreateInstances {
		if ci.daisyName == name {
			break
		}
	}
	// Register attachments.
	for _, d := range ci.Disks {
		dName := d.Source
		if d.InitializeParams != nil {
			dName = d.InitializeParams.DiskName
		}
		if err := disks[im.w].registerAttachment(dName, ci.daisyName, d.Mode, s); err != nil {
			return err
		}
	}
	return nil
}

func (im *instanceMap) registerDeletion(name string, s *Step) error {
	if err := im.baseResourceMap.registerDeletion(name, s); err != nil {
		return err
	}
	return disks[im.w].registerAllDetachments(name, s)
}

func checkDiskMode(m string) bool {
	parts := strings.Split(m, "/")
	m = parts[len(parts)-1]
	return strIn(m, validDiskModes)
}
