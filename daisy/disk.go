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
	disks      = map[*Workflow]*diskMap{}
	diskURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[1]s)/disks/(?P<disk>%[1]s)$`, rfc1035))
)

type diskMap struct {
	baseResourceMap
	attachments      map[*resource]map[*resource]*diskAttachment // map (disk, instance) -> attachment
	testDetachHelper func(d, i *resource, s *Step) error
}

type diskAttachment struct {
	mode               string
	attacher, detacher *Step
}

func initDiskMap(w *Workflow) {
	dm := &diskMap{baseResourceMap: baseResourceMap{w: w, typeName: "disk", urlRgx: diskURLRgx}}
	dm.baseResourceMap.deleteFn = dm.deleteFn
	dm.init()
	disks[w] = dm
}

func (dm *diskMap) init() {
	dm.baseResourceMap.init()
	dm.attachments = map[*resource]map[*resource]*diskAttachment{}
}

func (dm *diskMap) deleteFn(r *resource) error {
	m := namedSubexp(diskURLRgx, r.link)
	if err := dm.w.ComputeClient.DeleteDisk(m["project"], m["zone"], m["disk"]); err != nil {
		return err
	}
	return nil
}

func (dm *diskMap) registerAttachment(dName, iName, mode string, s *Step) error {
	dm.mx.Lock()
	defer dm.mx.Unlock()
	var d, i *resource
	var ok bool
	if d, ok = dm.m[dName]; !ok {
		return Errorf("cannot attach disk %q, does not exist", dName)
	}
	if i, ok = instances[dm.w].get(iName); !ok {
		return Errorf("cannot attach disk to instance %q, does not exist", iName)
	}
	// Iterate over disk's attachments. Check for concurrent conflicts.
	// Step s is concurrent with other attachments if the attachment detacher == nil
	// or s does not depend on the detacher.
	// If this is a repeat attachment (same disk and instance already attached), do nothing and return.
	for attI, att := range dm.attachments[d] {
		// Is this a concurrent attachment?
		if att.detacher == nil || !s.nestedDepends(att.detacher) {
			if attI == i {
				return nil // this is a repeat attachment to the same instance -- does nothing
			} else if strIn(diskModeRW, []string{mode, att.mode}) {
				// Can't have concurrent attachment in RW mode.
				return Errorf(
					"concurrent attachment of disk %q between instances %q (%s) and %q (%s)",
					dName, i.real, mode, attI.real, att.mode)
			}
		}
	}

	var im map[*resource]*diskAttachment
	if im, ok = dm.attachments[d]; !ok {
		im = map[*resource]*diskAttachment{}
		dm.attachments[d] = im
	}
	im[i] = &diskAttachment{mode: mode, attacher: s}
	return nil
}

// detachHelper marks s as the detacher between d and i.
// Returns an error if d and i aren't attached
// or if the detacher doesn't depend on the attacher.
func (dm *diskMap) detachHelper(d, i *resource, s *Step) error {
	if dm.testDetachHelper != nil {
		return dm.testDetachHelper(d, i, s)
	}
	var att *diskAttachment
	var im map[*resource]*diskAttachment
	var ok bool
	if im, ok = dm.attachments[d]; !ok {
		return Errorf("not attached")
	}
	if att, ok = im[i]; !ok || att.detacher != nil {
		return Errorf("not attached")
	} else if !s.nestedDepends(att.attacher) {
		return Errorf("detacher %q does not depend on attacher %q", s.name, att.attacher.name)
	}
	att.detacher = s
	return nil
}

// registerDetachment marks s as the detacher for the dName disk and iName instance.
// Returns an error if dName or iName don't exist
// or if detachHelper returns an error.
func (dm *diskMap) registerDetachment(dName, iName string, s *Step) error {
	dm.mx.Lock()
	defer dm.mx.Unlock()
	var d, i *resource
	var ok bool
	if d, ok = dm.m[dName]; !ok {
		return Errorf("cannot detach disk %q, does not exist", dName)
	}
	if i, ok = instances[dm.w].get(iName); !ok {
		return Errorf("cannot detach disk from instance %q, does not exist", iName)
	}

	if err := dm.detachHelper(d, i, s); err != nil {
		return Errorf("cannot detach disk %q from instance %q: %v", dName, iName, err)
	}
	return nil
}

// registerAllDetachments marks s as the detacher for all disks attached to the iName instance.
// Returns an error if iName does not exist
// or if detachHelper returns an error.
func (dm *diskMap) registerAllDetachments(iName string, s *Step) error {
	dm.mx.Lock()
	defer dm.mx.Unlock()

	i, ok := instances[dm.w].get(iName)
	if !ok {
		return Errorf("cannot detach disks from instance %q, does not exist", iName)
	}

	var errs Errors
	for d, im := range dm.attachments {
		if att, ok := im[i]; !ok || att.detacher != nil {
			continue
		}
		if err := dm.detachHelper(d, i, s); err != nil {
			errs.add(Errorf("cannot detach disk %q from instance %q: %v", d.real, iName, err))
		}
	}
	return errs.cast()
}
