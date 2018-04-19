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
	"sync"

	"google.golang.org/api/compute/v1"
)

// DeleteResources deletes GCE resources.
type DeleteResources struct {
	Disks     []string `json:",omitempty"`
	Images    []string `json:",omitempty"`
	Instances []string `json:",omitempty"`
	Networks  []string `json:",omitempty"`
}

func (d *DeleteResources) populate(ctx context.Context, s *Step) dErr {
	for i, disk := range d.Disks {
		if diskURLRgx.MatchString(disk) {
			d.Disks[i] = extendPartialURL(disk, s.w.Project)
		}
	}
	for i, image := range d.Images {
		if imageURLRgx.MatchString(image) {
			d.Images[i] = extendPartialURL(image, s.w.Project)
		}
	}
	for i, instance := range d.Instances {
		if instanceURLRgx.MatchString(instance) {
			d.Instances[i] = extendPartialURL(instance, s.w.Project)
		}
	}
	for i, network := range d.Networks {
		if networkURLRegex.MatchString(network) {
			d.Networks[i] = extendPartialURL(network, s.w.Project)
		}
	}
	return nil
}

func (d *DeleteResources) validateInstance(i string, s *Step) dErr {
	if err := s.w.instances.regDelete(i, s); err != nil {
		return err
	}
	ir, _ := s.w.instances.get(i)

	// Get the Instance that created this instance, if any.
	var attachedDisks []*compute.AttachedDisk
	if ir.creator != nil {
		for _, createI := range *ir.creator.CreateInstances {
			if createI.daisyName == i {
				attachedDisks = createI.Disks
			}
		}
	}
	for _, ad := range attachedDisks {
		if ad.AutoDelete {
			if err := s.w.disks.regDelete(ad.Source, s); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *DeleteResources) checkError(err dErr, s *Step) dErr {
	if err != nil && err.Type() == resourceDNEError {
		s.w.Logger.StepInfo(s.w, "DeleteResources", "WARNING: Error validating deletion: %v", err)
		return nil
	} else if err != nil && err.Type() == imageObsoleteDeletedError {
		return nil
	}
	return err
}

func (d *DeleteResources) validate(ctx context.Context, s *Step) dErr {
	// Instance checking.
	for _, i := range d.Instances {
		if err := d.validateInstance(i, s); d.checkError(err, s) != nil {
			return err
		}
	}

	// Disk checking.
	for _, disk := range d.Disks {
		if err := s.w.disks.regDelete(disk, s); d.checkError(err, s) != nil {
			return err
		}
	}

	// Image checking.
	for _, i := range d.Images {
		if err := s.w.images.regDelete(i, s); d.checkError(err, s) != nil {
			return err
		}
	}

	// Network checking.
	for _, n := range d.Networks {
		if err := s.w.networks.regDelete(n, s); d.checkError(err, s) != nil {
			return err
		}
	}

	return nil
}

func (d *DeleteResources) run(ctx context.Context, s *Step) dErr {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan dErr)

	for _, i := range d.Instances {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			w.Logger.StepInfo(w, "DeleteResources", "deleting instance %q.", i)
			if err := w.instances.delete(i); err != nil {
				if err.Type() == resourceDNEError {
					w.Logger.StepInfo(w, "DeleteResources", "WARNING: Error deleting instance %q: %v", i, err)
					return
				}
				e <- err
			}
		}(i)
	}

	for _, i := range d.Images {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			w.Logger.StepInfo(w, "DeleteResources", "deleting image %q.", i)
			if err := w.images.delete(i); err != nil {
				if err.Type() == resourceDNEError {
					w.Logger.StepInfo(w, "DeleteResources", "WARNING: Error deleting image %q: %v", i, err)
					return
				}
				e <- err
			}
		}(i)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		if err != nil {
			return err
		}
	case <-w.Cancel:
		return nil
	}

	// Delete disks only after instances have been deleted.
	e = make(chan dErr)
	for _, d := range d.Disks {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			w.Logger.StepInfo(w, "DeleteResources", "deleting disk %q.", d)
			if err := w.disks.delete(d); err != nil {
				if err.Type() == resourceDNEError {
					w.Logger.StepInfo(w, "DeleteResources", "WARNING: Error deleting disk %q: %v", d, err)
					return
				}
				e <- err
			}
		}(d)
	}

	// Delete disks after instances.
	for _, n := range d.Networks {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			w.Logger.StepInfo(w, "DeleteResources", "deleting network %q.", n)
			if err := w.networks.delete(n); err != nil {
				if err.Type() == resourceDNEError {
					w.Logger.StepInfo(w, "DeleteResources", "WARNING: Error deleting network %q: %v", n, err)
				}
				e <- err
			}
		}(n)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		return nil
	}
}
