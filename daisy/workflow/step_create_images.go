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
	"errors"
	"fmt"
	"path"
	"sync"
)

// CreateImages is a Daisy CreateImages workflow step.
type CreateImages []CreateImage

// CreateImage creates a GCE image in a project.
// Supported sources are a GCE disk or a RAW image listed in Workflow.Sources.
type CreateImage struct {
	// Name if the image.
	Name string
	// Project to import image into. If this is unset Workflow.Project is
	// used.
	Project string `json:",omitempty"`
	// Image family
	Family string `json:",omitempty"`
	// Image licenses
	Licenses []string `json:",omitempty"`
	// A list of features to enable on the guest OS.
	// https://godoc.org/google.golang.org/api/compute/v1#GuestOsFeature
	GuestOsFeatures []string `json:",omitempty"`
	// Only one of these source types should be specified.
	SourceDisk string `json:",omitempty"`
	SourceFile string `json:",omitempty"`
	// Optional description of the resource, if not specified Daisy will
	// create one with the name of the project.
	Description string `json:",omitempty"`
	// Should this resource be cleaned up after the workflow?
	NoCleanup bool
	// Should we use the user-provided reference name as the actual
	// resource name?
	ExactName bool
}

func (c *CreateImages) validate(s *Step) error {
	w := s.w
	for _, ci := range *c {
		// File/Disk checking.
		if !xor(ci.SourceDisk == "", ci.SourceFile == "") {
			return errors.New("must provide either Disk or File, exclusively")
		}
		if ci.SourceDisk != "" && !diskValid(w, ci.SourceDisk) {
			return fmt.Errorf("cannot create image: disk not found: %s", ci.SourceDisk)
		}
		if _, _, err := splitGCSPath(ci.SourceFile); err != nil && ci.SourceFile != "" && !w.sourceExists(ci.SourceFile) {
			return fmt.Errorf("cannot create image: file not in sources or valid GCS path: %s", ci.SourceFile)
		}

		// Project checking.
		if ci.Project != "" && !projectExists(ci.Project) {
			return fmt.Errorf("cannot create image: project not found: %s", ci.Project)
		}

		// Try adding image name.
		if err := validatedImages.add(w, ci.Name); err != nil {
			return fmt.Errorf("error adding image: %s", err)
		}
	}

	return nil
}

func (c *CreateImages) run(s *Step) error {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan error)
	for _, ci := range *c {
		wg.Add(1)
		go func(ci CreateImage) {
			defer wg.Done()
			name := ci.Name
			if !ci.ExactName {
				name = w.genName(ci.Name)
			}

			project := stringOr(ci.Project, w.Project)

			// Get source disk link, if applicable.
			var diskLink string
			if ci.SourceDisk != "" {
				if diskURLRegex.MatchString(ci.SourceDisk) {
					diskLink = ci.SourceDisk
				} else if disk, ok := disks[w].get(ci.SourceDisk); ok {
					diskLink = disk.link
				} else {
					e <- fmt.Errorf("invalid or missing reference to SourceDisk %q", ci.SourceDisk)
					return
				}
			}

			// Resolve SourceFile, if applicable.
			var sourceFile string
			if ci.SourceFile != "" {
				if w.sourceExists(ci.SourceFile) {
					sourceFile = fmt.Sprintf("https://storage.cloud.google.com/%s", path.Join(w.bucket, w.sourcesPath, ci.SourceFile))
				} else if bkt, obj, err := splitGCSPath(ci.SourceFile); err == nil {
					sourceFile = fmt.Sprintf("https://storage.cloud.google.com/%s", path.Join(bkt, obj))
				} else {
					e <- fmt.Errorf("%q is not in Sources and is not a valid GCS path", ci.SourceFile)
					return
				}
			}

			w.logger.Printf("CreateImages: creating image %q.", name)
			description := ci.Description
			if description == "" {
				description = fmt.Sprintf("Image created by Daisy in workflow %q on behalf of %s.", w.Name, w.username)
			}
			i, err := w.ComputeClient.CreateImage(name, project, diskLink, sourceFile, ci.Family, description, ci.Licenses, ci.GuestOsFeatures)
			if err != nil {
				e <- err
				return
			}
			images[w].add(ci.Name, &resource{ci.Name, name, i.SelfLink, ci.NoCleanup, false})
		}(ci)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so images being created now will complete.
		wg.Wait()
		return nil
	}
}
