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
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

var (
	disks      = map[*Workflow]*diskRegistry{}
	disksMu    sync.Mutex
	diskURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[2]s)/disks/(?P<disk>%[2]s)$`, projectRgxStr, rfc1035))
)

type diskRegistry struct {
	baseResourceRegistry
	attachments      map[*resource]map[*resource]*diskAttachment // map (disk, instance) -> attachment
	testDetachHelper func(d, i *resource, s *Step) error
}

type diskAttachment struct {
	mode               string
	attacher, detacher *Step
}

func initDiskRegistry(w *Workflow) {
	dr := &diskRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "disk", urlRgx: diskURLRgx}}
	dr.baseResourceRegistry.deleteFn = dr.deleteFn
	dr.init()
	disksMu.Lock()
	disks[w] = dr
	disksMu.Unlock()
}

func (dr *diskRegistry) init() {
	dr.baseResourceRegistry.init()
	dr.attachments = map[*resource]map[*resource]*diskAttachment{}
}

func (dr *diskRegistry) deleteFn(res *resource) error {
	m := namedSubexp(diskURLRgx, res.link)
	return dr.w.ComputeClient.DeleteDisk(m["project"], m["zone"], m["disk"])
}

func (dr *diskRegistry) registerAttachment(dName, iName, mode string, s *Step) error {
	dr.mx.Lock()
	defer dr.mx.Unlock()
	var d, i *resource
	var ok bool
	if d, ok = dr.m[dName]; !ok {
		return errorf("cannot attach disk %q, does not exist", dName)
	}
	if i, ok = instances[dr.w].get(iName); !ok {
		return errorf("cannot attach disk to instance %q, does not exist", iName)
	}
	// Iterate over disk's attachments. Check for concurrent conflicts.
	// Step s is concurrent with other attachments if the attachment detacher == nil
	// or s does not depend on the detacher.
	// If this is a repeat attachment (same disk and instance already attached), do nothing and return.
	for attI, att := range dr.attachments[d] {
		// Is this a concurrent attachment?
		if att.detacher == nil || !s.nestedDepends(att.detacher) {
			if attI == i {
				return nil // this is a repeat attachment to the same instance -- does nothing
			} else if strIn(diskModeRW, []string{mode, att.mode}) {
				// Can't have concurrent attachment in RW mode.
				return errorf(
					"concurrent attachment of disk %q between instances %q (%s) and %q (%s)",
					dName, i.real, mode, attI.real, att.mode)
			}
		}
	}

	var im map[*resource]*diskAttachment
	if im, ok = dr.attachments[d]; !ok {
		im = map[*resource]*diskAttachment{}
		dr.attachments[d] = im
	}
	im[i] = &diskAttachment{mode: mode, attacher: s}
	return nil
}

// detachHelper marks s as the detacher between d and i.
// Returns an error if d and i aren't attached
// or if the detacher doesn't depend on the attacher.
func (dr *diskRegistry) detachHelper(d, i *resource, s *Step) error {
	if dr.testDetachHelper != nil {
		return dr.testDetachHelper(d, i, s)
	}
	var att *diskAttachment
	var im map[*resource]*diskAttachment
	var ok bool
	if im, ok = dr.attachments[d]; !ok {
		return errorf("not attached")
	}
	if att, ok = im[i]; !ok || att.detacher != nil {
		return errorf("not attached")
	} else if !s.nestedDepends(att.attacher) {
		return errorf("detacher %q does not depend on attacher %q", s.name, att.attacher.name)
	}
	att.detacher = s
	return nil
}

// registerDetachment marks s as the detacher for the dName disk and iName instance.
// Returns an error if dName or iName don't exist
// or if detachHelper returns an error.
func (dr *diskRegistry) registerDetachment(dName, iName string, s *Step) error {
	dr.mx.Lock()
	defer dr.mx.Unlock()
	var d, i *resource
	var ok bool
	if d, ok = dr.m[dName]; !ok {
		return errorf("cannot detach disk %q, does not exist", dName)
	}
	if i, ok = instances[dr.w].get(iName); !ok {
		return errorf("cannot detach disk from instance %q, does not exist", iName)
	}

	if err := dr.detachHelper(d, i, s); err != nil {
		return errorf("cannot detach disk %q from instance %q: %v", dName, iName, err)
	}
	return nil
}

// registerAllDetachments marks s as the detacher for all disks attached to the iName instance.
// Returns an error if iName does not exist
// or if detachHelper returns an error.
func (dr *diskRegistry) registerAllDetachments(iName string, s *Step) error {
	dr.mx.Lock()
	defer dr.mx.Unlock()

	i, ok := instances[dr.w].get(iName)
	if !ok {
		return errorf("cannot detach disks from instance %q, does not exist", iName)
	}

	var errs dErrors
	for d, im := range dr.attachments {
		if att, ok := im[i]; !ok || att.detacher != nil {
			continue
		}
		if err := dr.detachHelper(d, i, s); err != nil {
			errs.add(errorf("cannot detach disk %q from instance %q: %v", d.real, iName, err))
		}
	}
	return errs.cast()
}

var diskCache struct {
	exists map[string]map[string][]string
	mu     sync.Mutex
}

// diskExists should only be used during validation for existing GCE disks
// and should not be relied or populated for daisy created resources.
func diskExists(client compute.Client, project, zone, disk string) (bool, error) {
	diskCache.mu.Lock()
	defer diskCache.mu.Unlock()
	if diskCache.exists == nil {
		diskCache.exists = map[string]map[string][]string{}
	}
	if _, ok := diskCache.exists[project]; !ok {
		diskCache.exists[project] = map[string][]string{}
	}
	if _, ok := diskCache.exists[project][zone]; !ok {
		dl, err := client.ListDisks(project, zone)
		if err != nil {
			return false, fmt.Errorf("error listing disks for project %q: %v", project, err)
		}
		var disks []string
		for _, d := range dl {
			disks = append(disks, d.Name)
		}
		diskCache.exists[project][zone] = disks
	}
	return strIn(disk, diskCache.exists[project][zone]), nil
}
