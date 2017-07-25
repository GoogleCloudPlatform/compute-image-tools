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
	"context"
	"sync"

	compute "google.golang.org/api/compute/v1"
)

// DeleteResources deletes GCE resources.
type DeleteResources struct {
	Disks     []string `json:",omitempty"`
	Images    []string `json:",omitempty"`
	Instances []string `json:",omitempty"`
}

func (d *DeleteResources) populate(ctx context.Context, s *Step) error {
	return nil
}

func (d *DeleteResources) validateInstance(i string, s *Step) error {
	if err := instances[s.w].registerDeletion(i, s); err != nil {
		return err
	}
	ir, _ := instances[s.w].get(i)

	// Get the CreateInstance that created this instance, if any.
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
			if err := disks[s.w].registerDeletion(ad.Source, s); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *DeleteResources) validate(ctx context.Context, s *Step) error {
	// Instance checking.
	for _, i := range d.Instances {
		if err := d.validateInstance(i, s); err != nil {
			return err
		}
	}

	// Disk checking.
	for _, disk := range d.Disks {
		if err := disks[s.w].registerDeletion(disk, s); err != nil {
			return err
		}
	}

	// Image checking.
	for _, i := range d.Images {
		if err := images[s.w].registerDeletion(i, s); err != nil {
			return err
		}
	}

	return nil
}

func (d *DeleteResources) run(ctx context.Context, s *Step) error {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan error)

	for _, i := range d.Instances {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			w.logger.Printf("DeleteResources: deleting instance %q.", i)
			if err := instances[w].delete(i); err != nil {
				e <- err
			}
		}(i)
	}

	for _, i := range d.Images {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			w.logger.Printf("DeleteResources: deleting image %q.", i)
			if err := images[w].delete(i); err != nil {
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
	e = make(chan error)
	for _, d := range d.Disks {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			w.logger.Printf("DeleteResources: deleting disk %q.", d)
			if err := disks[w].delete(d); err != nil {
				e <- err
			}
		}(d)
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
