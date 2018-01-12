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
	"strconv"
	"sync"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var (
	diskCache struct {
		exists map[string]map[string][]string
		mu     sync.Mutex
	}
	diskURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[2]s)/disks/(?P<disk>%[2]s)$`, projectRgxStr, rfc1035))
)

// diskExists should only be used during validation for existing GCE disks
// and should not be relied or populated for daisy created resources.
func diskExists(client daisyCompute.Client, project, zone, disk string) (bool, dErr) {
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
			return false, errf("error listing disks for project %q: %v", project, err)
		}
		var disks []string
		for _, d := range dl {
			disks = append(disks, d.Name)
		}
		diskCache.exists[project][zone] = disks
	}
	return strIn(disk, diskCache.exists[project][zone]), nil
}

// Disk is used to create a GCE disk in a project.
type Disk struct {
	compute.Disk
	Resource

	// Size of this disk.
	SizeGb string `json:"sizeGb,omitempty"`
}

// MarshalJSON is a hacky workaround to prevent Disk from using compute.Disk's implementation.
func (d *Disk) MarshalJSON() ([]byte, error) {
	return json.Marshal(*d)
}

func (d *Disk) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	d.Name, d.Zone, errs = d.Resource.populate(ctx, s, d.Name, d.Zone)

	d.Description = strOr(d.Description, fmt.Sprintf("Disk created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username))
	if d.SizeGb != "" {
		size, err := strconv.ParseInt(d.SizeGb, 10, 64)
		if err != nil {
			errs = addErrs(errs, errf("cannot parse SizeGb: %s, err: %v", d.SizeGb, err))
		}
		d.Disk.SizeGb = size
	}
	if imageURLRgx.MatchString(d.SourceImage) {
		d.SourceImage = extendPartialURL(d.SourceImage, d.Project)
	}
	if d.Type == "" {
		d.Type = fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-standard", d.Project, d.Zone)
	} else if diskTypeURLRgx.MatchString(d.Type) {
		d.Type = extendPartialURL(d.Type, d.Project)
	} else {
		d.Type = fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", d.Project, d.Zone, d.Type)
	}
	d.link = fmt.Sprintf("projects/%s/zones/%s/disks/%s", d.Project, d.Zone, d.Name)
	return errs
}

func (d *Disk) validate(ctx context.Context, s *Step) dErr {
	errs := d.Resource.validateWithZone(ctx, s, d.Zone)

	if !diskTypeURLRgx.MatchString(d.Type) {
		errs = addErrs(errs, errf("cannot create disk %q: bad disk type: %q", d.daisyName, d.Type))
	}

	if d.SourceImage != "" {
		if _, err := images[s.w].registerUsage(d.SourceImage, s); err != nil {
			errs = addErrs(errs, errf("cannot create disk %q: can't use image %q: %v", d.daisyName, d.SourceImage, err))
		}
	} else if d.Disk.SizeGb == 0 {
		errs = addErrs(errs, errf("cannot create disk %q: SizeGb and SourceImage not set", d.daisyName))
	}

	// Register creation.
	errs = addErrs(errs, disks[s.w].registerCreation(d.daisyName, &d.Resource, s, false))
	return errs
}

type diskAttachment struct {
	mode               string
	attacher, detacher *Step
}

type diskRegistry struct {
	baseResourceRegistry
	attachments      map[*Resource]map[*Resource]*diskAttachment // map (disk, instance) -> attachment
	testDetachHelper func(d, i *Resource, s *Step) dErr
}

func newDiskRegistry(w *Workflow) *diskRegistry {
	dr := &diskRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "disk", urlRgx: diskURLRgx}}
	dr.baseResourceRegistry.deleteFn = dr.deleteFn
	dr.init()
	return dr
}

func (dr *diskRegistry) init() {
	dr.baseResourceRegistry.init()
	dr.attachments = map[*Resource]map[*Resource]*diskAttachment{}
}

func (dr *diskRegistry) deleteFn(res *Resource) dErr {
	m := namedSubexp(diskURLRgx, res.link)
	err := dr.w.ComputeClient.DeleteDisk(m["project"], m["zone"], m["disk"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}

func (dr *diskRegistry) registerAttachment(dName, iName, mode string, s *Step) dErr {
	dr.mx.Lock()
	defer dr.mx.Unlock()
	var d, i *Resource
	var ok bool
	if d, ok = dr.m[dName]; !ok {
		return errf("cannot attach disk %q, does not exist", dName)
	}
	if i, ok = instances[dr.w].get(iName); !ok {
		return errf("cannot attach disk to instance %q, does not exist", iName)
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
				return errf(
					"concurrent attachment of disk %q between instances %q (%s) and %q (%s)",
					dName, i.RealName, mode, attI.RealName, att.mode)
			}
		}
	}

	var im map[*Resource]*diskAttachment
	if im, ok = dr.attachments[d]; !ok {
		im = map[*Resource]*diskAttachment{}
		dr.attachments[d] = im
	}
	im[i] = &diskAttachment{mode: mode, attacher: s}
	return nil
}

// detachHelper marks s as the detacher between d and i.
// Returns an error if d and i aren't attached
// or if the detacher doesn't depend on the attacher.
func (dr *diskRegistry) detachHelper(d, i *Resource, s *Step) dErr {
	if dr.testDetachHelper != nil {
		return dr.testDetachHelper(d, i, s)
	}
	var att *diskAttachment
	var im map[*Resource]*diskAttachment
	var ok bool
	if im, ok = dr.attachments[d]; !ok {
		return errf("not attached")
	}
	if att, ok = im[i]; !ok || att.detacher != nil {
		return errf("not attached")
	} else if !s.nestedDepends(att.attacher) {
		return errf("detacher %q does not depend on attacher %q", s.name, att.attacher.name)
	}
	att.detacher = s
	return nil
}

// registerDetachment marks s as the detacher for the dName disk and iName instance.
// Returns an error if dName or iName don't exist
// or if detachHelper returns an error.
func (dr *diskRegistry) registerDetachment(dName, iName string, s *Step) dErr {
	dr.mx.Lock()
	defer dr.mx.Unlock()
	var d, i *Resource
	var ok bool
	if d, ok = dr.m[dName]; !ok {
		return errf("cannot detach disk %q, does not exist", dName)
	}
	if i, ok = instances[dr.w].get(iName); !ok {
		return errf("cannot detach disk from instance %q, does not exist", iName)
	}

	if err := dr.detachHelper(d, i, s); err != nil {
		return errf("cannot detach disk %q from instance %q: %v", dName, iName, err)
	}
	return nil
}

// registerAllDetachments marks s as the detacher for all disks attached to the iName instance.
// Returns an error if iName does not exist
// or if detachHelper returns an error.
func (dr *diskRegistry) registerAllDetachments(iName string, s *Step) dErr {
	dr.mx.Lock()
	defer dr.mx.Unlock()

	i, ok := instances[dr.w].get(iName)
	if !ok {
		return errf("cannot detach disks from instance %q, does not exist", iName)
	}

	var errs dErr
	for d, im := range dr.attachments {
		if att, ok := im[i]; !ok || att.detacher != nil {
			continue
		}
		if err := dr.detachHelper(d, i, s); err != nil {
			errs = addErrs(errs, errf("cannot detach disk %q from instance %q: %v", d.RealName, iName, err))
		}
	}
	return errs
}
