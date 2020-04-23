//  Copyright 2020 Google Inc. All Rights Reserved.
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
	"fmt"
	"sync"
)

// CreateSnapshots is a Daisy CreateSnapshots workflow step.
type CreateSnapshots []*Snapshot

func (c *CreateSnapshots) populate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, ss := range *c {
		errs = addErrs(errs, ss.populate(ctx, s))
	}
	return errs
}

func (c *CreateSnapshots) validate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, d := range *c {
		errs = addErrs(errs, d.validate(ctx, s))
	}
	return errs
}

func (c *CreateSnapshots) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)

	createSnapshot := func(ss *Snapshot) {
		defer wg.Done()
		// Get source disk link if SourceDisk is a daisy reference to a disk.
		if d, ok := w.disks.get(ss.SourceDisk); ok {
			ss.SourceDisk = d.link

			// Override snapshot link due that disk may be from a different project
			m := NamedSubexp(diskURLRgx, d.link)
			if ss.Project != m["project"] {
				ss.link = fmt.Sprintf("projects/%s/global/snapshots/%s", m["project"], ss.Name)
			}
		}

		m := NamedSubexp(diskURLRgx, ss.SourceDisk)
		w.LogStepInfo(s.name, "CreateSnapshots", "Creating snapshot %q.", ss.Name)
		if err := w.ComputeClient.CreateSnapshot(m["project"], m["zone"], m["disk"], &ss.Snapshot); err != nil {
			e <- newErr("failed to create snapshots", err)
			return
		}
		ss.createdInWorkflow = true
	}

	for _, ss := range *c {
		wg.Add(1)
		go createSnapshot(ss)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so Snapshots being created now will complete before we try to clean them up.
		wg.Wait()
		return nil
	}
}
