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
	"sync"
)

// DeleteResources deletes GCE resources.
// TODO(crunkleton) DeleteResources only works on Workflow references right now.
type DeleteResources struct {
	Instances, Disks, Images []string
}

func (d *DeleteResources) validate(w *Workflow) error {
	// Disk checking.
	for _, disk := range d.Disks {
		if !diskValid(w, disk) {
			return fmt.Errorf("cannot delete disk. Disk not found: %s", disk)
		}
		if err := validatedDiskDeletions.add(w, disk); err != nil {
			return fmt.Errorf("error scheduling disk for deletion: %s", err)
		}
	}

	// Instance checking.
	for _, i := range d.Instances {
		if !instanceValid(w, i) {
			return fmt.Errorf("cannot delete instance. Instance not found: %s", i)
		}
		if err := validatedInstanceDeletions.add(w, i); err != nil {
			return fmt.Errorf("error scheduling instance for deletion: %s", err)
		}
	}

	return nil
}

func (d *DeleteResources) run(w *Workflow) error {
	var wg sync.WaitGroup
	e := make(chan error)

	for _, i := range d.Instances {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			r, ok := w.instanceRefs.get(i)
			if !ok {
				e <- fmt.Errorf("unresolved instance %q", i)
			} else if err := w.deleteInstance(r); err != nil {
				e <- err
			}
		}(i)
	}

	for _, i := range d.Images {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			r, ok := w.imageRefs.get(i)
			if !ok {
				e <- fmt.Errorf("unresolved image %q", i)
			} else if err := w.deleteImage(r); err != nil {
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
	case <-w.Ctx.Done():
		return nil
	}

	// Delete disks only after instances have been deleted.
	e = make(chan error)
	for _, d := range d.Disks {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			r, ok := w.diskRefs.get(d)
			if !ok {
				e <- fmt.Errorf("unresolved disk %q", d)
			} else if err := w.deleteDisk(r); err != nil {
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
	case <-w.Ctx.Done():
		return nil
	}
}
