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
	disks        = map[*Workflow]*resourceMap{}
	diskURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[1]s)/disks/(?P<disk>%[1]s)$`, rfc1035))
)

func initDisksMap(w *Workflow) {
	m := &resourceMap{}
	disks[w] = m
	m.usageRegistrationHook = diskUsageRegistrationHook
}

func diskUsageRegistrationHook(name string, s *Step) error {
	return nil
}

func deleteDisk(w *Workflow, r *resource) error {
	if err := w.ComputeClient.DeleteDisk(w.Project, w.Zone, r.real); err != nil {
		return err
	}
	r.deleted = true
	return nil
}
