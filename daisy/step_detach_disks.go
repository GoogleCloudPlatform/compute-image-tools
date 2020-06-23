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
)

// DetachDisks is a Daisy DetachDisks workflow step.
type DetachDisks []*DetachDisk

// DetachDisk is used to detach a GCE disk from an instance.
type DetachDisk struct {
	// Instance to detach diskName.
	Instance                string
	DeviceName              string
	realName, project, zone string
}

func (a *DetachDisks) populate(ctx context.Context, s *Step) DError {
	for _, dd := range *a {
		dd.realName = path.Base(dd.DeviceName)
	}
	return nil
}

func (a *DetachDisks) validate(ctx context.Context, s *Step) (errs DError) {
	for _, dd := range *a {
		if dd.DeviceName == "" {
			errs = addErrs(errs, Errf("cannot detach disk: DeviceName is empty"))
		}

		ir, err := s.w.instances.regUse(dd.Instance, s)
		if ir == nil {
			// Return now, the rest of this function can't be run without ir.
			return addErrs(errs, Errf("cannot detach disk: %v", err))
		}
		addErrs(errs, err)

		instance := NamedSubexp(instanceURLRgx, ir.link)

		res, isAttached, err := s.w.disks.regUseDeviceName(dd.DeviceName, instance["project"], instance["zone"], instance["instance"], dd.Instance, s)
		if res == nil {
			// Return now, the rest of this function can't be run without resource.
			return addErrs(errs, Errf("cannot detach disk: %v", err))
		}
		addErrs(errs, err)

		if deviceNameURLRgx.MatchString(dd.DeviceName) {
			// While it's a device URL, no need to do more validation about project/zone
			// since it has been validated in regUseDeviceName.
			device := NamedSubexp(deviceNameURLRgx, res.link)
			dd.project = device["project"]
			dd.zone = device["zone"]
		} else {
			// Ensure disk is in the same project and zone.
			disk := NamedSubexp(diskURLRgx, res.link)
			if disk["project"] != instance["project"] {
				errs = addErrs(errs, Errf("cannot detach disk in project %q from instance in project %q: %q", disk["project"], instance["project"], dd.DeviceName))
			}
			if disk["zone"] != instance["zone"] {
				errs = addErrs(errs, Errf("cannot detach disk in zone %q from instance in zone %q: %q", disk["zone"], instance["zone"], dd.DeviceName))
			}

			dd.project = disk["project"]
			dd.zone = disk["zone"]
		}

		// Register disk detachments.
		errs = addErrs(errs, s.w.instances.w.disks.regDetach(dd.DeviceName, dd.Instance, isAttached, s))
	}
	return errs
}

func (a *DetachDisks) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, dd := range *a {
		wg.Add(1)
		go func(dd *DetachDisk) {
			defer wg.Done()

			inst := dd.Instance
			if instRes, ok := w.instances.get(dd.Instance); ok {
				dd.Instance = instRes.RealName
			}

			w.LogStepInfo(s.name, "DetachDisks", "Detaching disk %q from instance %q.", dd.DeviceName, inst)
			if err := w.ComputeClient.DetachDisk(dd.project, dd.zone, dd.Instance, dd.realName); err != nil {
				e <- newErr("failed to detach disks", err)
				return
			}
		}(dd)
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
