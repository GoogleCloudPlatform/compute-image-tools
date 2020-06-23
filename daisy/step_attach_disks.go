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
	"path"
	"sync"

	"google.golang.org/api/compute/v1"
)

// AttachDisks is a Daisy AttachDisks workflow step.
type AttachDisks []*AttachDisk

// AttachDisk is used to attach a GCE disk into an instance.
type AttachDisk struct {
	compute.AttachedDisk

	// Instance to attach to.
	Instance      string
	project, zone string
}

func (a *AttachDisks) populate(ctx context.Context, s *Step) DError {
	for _, ad := range *a {
		ad.Mode = strOr(ad.Mode, defaultDiskMode)
		if ad.DeviceName == "" {
			ad.DeviceName = path.Base(ad.Source)
		}
		if diskURLRgx.MatchString(ad.Source) {
			ad.Source = extendPartialURL(ad.Source, s.w.Project)
		}
	}

	return nil
}

func (a *AttachDisks) validate(ctx context.Context, s *Step) (errs DError) {
	for _, ad := range *a {
		if !checkDiskMode(ad.Mode) {
			errs = addErrs(errs, Errf("cannot attach disk: bad disk mode: %q", ad.Mode))
		}
		if ad.Source == "" {
			errs = addErrs(errs, Errf("cannot attach disk: AttachedDisk.Source is empty"))
		}

		ir, err := s.w.instances.regUse(ad.Instance, s)
		if ir == nil {
			// Return now, the rest of this function can't be run without ir.
			return addErrs(errs, Errf("cannot attach disk: %v", err))
		}
		addErrs(errs, err)

		dr, err := s.w.disks.regUse(ad.Source, s)
		if dr == nil {
			// Return now, the rest of this function can't be run without dr.
			return addErrs(errs, Errf("cannot attach disk: %v", err))
		}
		addErrs(errs, err)

		// Ensure disk is in the same project and zone.
		disk := NamedSubexp(diskURLRgx, dr.link)
		instance := NamedSubexp(instanceURLRgx, ir.link)
		if disk["project"] != instance["project"] {
			errs = addErrs(errs, Errf("cannot attach disk in project %q to instance in project %q: %q", disk["project"], instance["project"], ad.Source))
		}
		if disk["zone"] != instance["zone"] {
			errs = addErrs(errs, Errf("cannot attach disk in zone %q to instance in zone %q: %q", disk["zone"], instance["zone"], ad.Source))
		}

		ad.project = disk["project"]
		ad.zone = disk["zone"]

		// Register disk attachments.
		errs = addErrs(errs, s.w.instances.w.disks.regAttach(ad.DeviceName, ad.Source, ad.Instance, ad.Mode, s))
	}
	return errs
}

func (a *AttachDisks) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, ad := range *a {
		wg.Add(1)
		go func(ad *AttachDisk) {
			defer wg.Done()

			if diskRes, ok := w.disks.get(ad.Source); ok {
				ad.Source = diskRes.link
			}

			inst := ad.Instance
			if instRes, ok := w.instances.get(ad.Instance); ok {
				inst = instRes.link
				ad.Instance = instRes.RealName
			}

			w.LogStepInfo(s.name, "AttachDisks", "Attaching disk %q to instance %q.", ad.AttachedDisk.Source, inst)
			if err := w.ComputeClient.AttachDisk(ad.project, ad.zone, ad.Instance, &ad.AttachedDisk); err != nil {
				e <- newErr("failed to attach disk", err)
				return
			}
		}(ad)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		wg.Wait()
		return nil
	}
}
