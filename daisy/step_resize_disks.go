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
	"fmt"
	"sync"

	"google.golang.org/api/compute/v1"
)

// ResizeDisks is a Daisy ResizeDisks workflow step.
type ResizeDisks []*ResizeDisk

// ResizeDisk is used to resize a GCE disk.
type ResizeDisk struct {
	compute.DisksResizeRequest
	// Name of the disk to be resized
	Name string
}

func (r *ResizeDisks) populate(ctx context.Context, s *Step) dErr {
	for _, rd := range *r {
		if diskURLRgx.MatchString(rd.Name) {
			rd.Name = extendPartialURL(rd.Name, s.w.Project)
		}
	}
	return nil
}

func (r *ResizeDisks) validate(ctx context.Context, s *Step) dErr {
	var errs dErr
	for _, rd := range *r {
		dr, err := s.w.disks.regUse(rd.Name, s)
		if dr == nil {
			// Return now, the rest of this function can't be run without dr.
			return addErrs(errs, errf("cannot resize disk: %v", err))
		}
		// Reference the actual name of the disk
		rd.Name = dr.RealName

		pre := fmt.Sprintf("cannot resize disk %q", rd.Name)
		if rd.SizeGb <= 0 {
			errs = addErrs(errs, errf("%s: SizeGb can't be zero: it's a mandatory field.", pre))
		}
	}
	return errs
}

func (r *ResizeDisks) run(ctx context.Context, s *Step) dErr {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan dErr)
	for _, rd := range *r {
		wg.Add(1)
		go func(rd *ResizeDisk) {
			defer wg.Done()

			w.LogStepInfo(s.name, "ResizeDisks", "Resizing disk %q to %v GB.", rd.Name, rd.SizeGb)
			if err := w.ComputeClient.ResizeDisk(s.w.Project, s.w.Zone, rd.Name, &rd.DisksResizeRequest); err != nil {
				e <- newErr(err)
				return
			}
		}(rd)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so disks being created now can be deleted.
		wg.Wait()
		return nil
	}
}
