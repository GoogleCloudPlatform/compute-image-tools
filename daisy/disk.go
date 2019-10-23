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
	diskURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[2]s)/disks/(?P<disk>%[2]s)(/resize)?$`, projectRgxStr, rfc1035))
)

// diskExists should only be used during validation for existing GCE disks
// and should not be relied or populated for daisy created resources.
func diskExists(client daisyCompute.Client, project, zone, disk string) (bool, DError) {
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
			return false, Errf("error listing disks for project %q: %v", project, err)
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

	// If this is enabled, then WINDOWS will be added to the
	// disk's guestOsFeatures. This is a string since daisy
	// replaces variables after JSON has been parsed.
	// (If it were boolean, the JSON marshaller throws
	// an error when it sees something like `${is_windows}`)
	IsWindows string `json:"isWindows,omitempty"`

	// Size of this disk.
	SizeGb string `json:"sizeGb,omitempty"`
}

// MarshalJSON is a hacky workaround to prevent Disk from using compute.Disk's implementation.
func (d *Disk) MarshalJSON() ([]byte, error) {
	return json.Marshal(*d)
}

func (d *Disk) populate(ctx context.Context, s *Step) DError {
	var errs DError
	d.Name, d.Zone, errs = d.Resource.populateWithZone(ctx, s, d.Name, d.Zone)

	d.Description = strOr(d.Description, fmt.Sprintf("Disk created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username))
	if d.SizeGb != "" {
		size, err := strconv.ParseInt(d.SizeGb, 10, 64)
		if err != nil {
			errs = addErrs(errs, Errf("cannot parse SizeGb: %s, err: %v", d.SizeGb, err))
		}
		d.Disk.SizeGb = size
	}

	if d.IsWindows != "" {
		isWindows, err := strconv.ParseBool(d.IsWindows)
		if err != nil {
			errs = addErrs(errs, Errf("cannot parse IsWindows as boolean: %s, err: %v", d.IsWindows, err))
		}
		if isWindows {
			d.GuestOsFeatures = CombineGuestOSFeatures(d.GuestOsFeatures, "WINDOWS")
		}
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

func (d *Disk) validate(ctx context.Context, s *Step) DError {
	pre := fmt.Sprintf("cannot create disk %q", d.daisyName)
	errs := d.Resource.validateWithZone(ctx, s, d.Zone, pre)

	if !diskTypeURLRgx.MatchString(d.Type) {
		errs = addErrs(errs, Errf("%s: bad disk type: %q", pre, d.Type))
	}

	if d.SourceImage != "" {
		if _, err := s.w.images.regUse(d.SourceImage, s); err != nil {
			errs = addErrs(errs, Errf("%s: can't use image %q: %v", pre, d.SourceImage, err))
		}
	} else if d.Disk.SizeGb == 0 {
		errs = addErrs(errs, Errf("%s: SizeGb and SourceImage not set", pre))
	}

	// Register creation.
	errs = addErrs(errs, s.w.disks.regCreate(d.daisyName, &d.Resource, s, false))
	return errs
}

type diskAttachment struct {
	mode               string
	attacher, detacher *Step
}

type diskRegistry struct {
	baseResourceRegistry
	attachments      map[string]map[string]*diskAttachment // map (disk, instance) -> attachment
	testDetachHelper func(dName, iName string, s *Step) DError
}

func newDiskRegistry(w *Workflow) *diskRegistry {
	dr := &diskRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "disk", urlRgx: diskURLRgx}}
	dr.baseResourceRegistry.deleteFn = dr.deleteFn
	dr.init()
	return dr
}

func (dr *diskRegistry) init() {
	dr.baseResourceRegistry.init()
	dr.attachments = map[string]map[string]*diskAttachment{}
}

func (dr *diskRegistry) deleteFn(res *Resource) DError {
	m := namedSubexp(diskURLRgx, res.link)
	err := dr.w.ComputeClient.DeleteDisk(m["project"], m["zone"], m["disk"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, "failed to delete disk", err)
	}
	return newErr("failed to delete disk", err)
}

// detachHelper marks s as the detacher between dName and iName.
// Returns an error if dName and iName aren't attached or if the detacher doesn't depend on the attacher.
func (dr *diskRegistry) detachHelper(dName, iName string, s *Step) DError {
	if dr.testDetachHelper != nil {
		return dr.testDetachHelper(dName, iName, s)
	}
	pre := fmt.Sprintf("step %q cannot detach disk %q from instance %q", s.name, dName, iName)
	var att *diskAttachment

	if im, _ := dr.attachments[dName]; im == nil {
		return Errf("%s: not attached", pre)
	} else if att, _ = im[iName]; att == nil {
		return Errf("%s: not attached", pre)
	} else if att.detacher != nil {
		return Errf("%s: already detached or concurrently detached by step %q", pre, att.detacher.name)
	} else if !s.nestedDepends(att.attacher) {
		return Errf("%s: step %q does not depend on attaching step %q", pre, s.name, att.attacher.name)
	}
	att.detacher = s
	return nil
}

// registerAttachment is called by Instance.regCreate and AttachDisks.validate and marks a disk as attached to an instance by Step s.
func (dr *diskRegistry) regAttach(dName, iName, mode string, s *Step) DError {
	dr.mx.Lock()
	defer dr.mx.Unlock()

	pre := fmt.Sprintf("step %q cannot attach disk %q to instance %q", s.name, dName, iName)
	var errs DError
	// Iterate over disk's attachments. Check for concurrent conflicts.
	// Step s is concurrent with other attachments if the attachment detacher == nil
	// or s does not depend on the detacher.
	// If this is a repeat attachment (same disk and instance already attached), do nothing and return.
	for attIName, att := range dr.attachments[dName] {
		// Is this a concurrent attachment?
		if att.detacher == nil || !s.nestedDepends(att.detacher) {
			if attIName == iName {
				errs = addErrs(errs, Errf("%s: concurrently attached by step %q", pre, att.attacher.name))
				return nil // this is a repeat attachment to the same instance -- does nothing
			} else if strIn(diskModeRW, []string{mode, att.mode}) {
				// Can't have concurrent attachment in RW mode.
				return Errf(
					"%s: concurrent RW attachment of disk %q between instances %q (%s) and %q (%s)",
					pre, dName, iName, mode, attIName, att.mode)
			}
		}
	}

	var im map[string]*diskAttachment
	if im, _ = dr.attachments[dName]; im == nil {
		im = map[string]*diskAttachment{}
		dr.attachments[dName] = im
	}
	im[iName] = &diskAttachment{mode: mode, attacher: s}
	return nil
}

// regDetach marks s as the detacher for the dName disk and iName instance.
// Returns an error if dName or iName don't exist or if detachHelper returns an error.
func (dr *diskRegistry) regDetach(dName, iName string, s *Step) DError {
	dr.mx.Lock()
	defer dr.mx.Unlock()

	return dr.detachHelper(dName, iName, s)
}

// regDetachAll is called by Instance.regDelete and registers Step s as the detacher for all disks currently attached to iName.
func (dr *diskRegistry) regDetachAll(iName string, s *Step) DError {
	dr.mx.Lock()
	defer dr.mx.Unlock()

	var errs DError
	// For every disk.
	for dName, im := range dr.attachments {
		// Check if instance attached.
		if att, _ := im[iName]; att == nil || att.detacher != nil {
			continue
		}
		// If yes, detach.
		errs = addErrs(dr.detachHelper(dName, iName, s))
	}
	return errs
}
