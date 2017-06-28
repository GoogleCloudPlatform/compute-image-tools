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
	"fmt"
	"sync"
)

// DeleteResources deletes GCE resources.
type DeleteResources struct {
	Instances, Disks, Images []string `json:",omitempty"`
}

func (d *DeleteResources) populate(ctx context.Context,s *Step) error {
	return nil
}

func (d *DeleteResources) validate(ctx context.Context,s *Step) error {
	w := s.w
	// Disk checking.
	for _, disk := range d.Disks {
		if !diskValid(w, disk) {
			return fmt.Errorf("cannot delete disk, disk not found: %s", disk)
		}
		if err := validatedDiskDeletions.add(w, disk); err != nil {
			return fmt.Errorf("error scheduling disk for deletion: %s", err)
		}
	}

	// Instance checking.
	for _, i := range d.Instances {
		if !instanceValid(w, i) {
			return fmt.Errorf("cannot delete instance, instance not found: %s", i)
		}
		if err := validatedInstanceDeletions.add(w, i); err != nil {
			return fmt.Errorf("error scheduling instance for deletion: %s", err)
		}
	}

	// Instance checking.
	for _, i := range d.Images {
		if !imageValid(w, i) {
			return fmt.Errorf("cannot delete image, image not found: %s", i)
		}
		if err := validatedImageDeletions.add(w, i); err != nil {
			return fmt.Errorf("error scheduling image for deletion: %s", err)
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
			r, ok := instances[w].get(i)
			if !ok {
				e <- fmt.Errorf("unresolved instance %q", i)
				return
			}
			w.logger.Printf("DeleteResources: deleting instance %q.", r.real)
			if err := deleteInstance(w, r); err != nil {
				e <- err
			}
			r.deleter = s
		}(i)
	}

	for _, i := range d.Images {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			r, ok := images[w].get(i)
			if !ok {
				e <- fmt.Errorf("unresolved image %q", i)
				return
			}
			w.logger.Printf("DeleteResources: deleting image %q.", r.real)
			if err := deleteImage(w, r); err != nil {
				e <- err
			}
			r.deleter = s
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
			r, ok := disks[w].get(d)
			if !ok {
				e <- fmt.Errorf("unresolved disk %q", d)
				return
			}
			w.logger.Printf("DeleteResources: deleting disk %q.", r.real)
			if err := deleteDisk(w, r); err != nil {
				e <- err
			}
			r.deleter = s
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
