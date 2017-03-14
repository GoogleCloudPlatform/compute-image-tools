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

import "fmt"

// ImportDisks is a Daisy ImportDisks workflow step.
type ImportDisks []ImportDisk

// ImportDisk creates a GCE disk from a external non GCE image file.
// This step is used to import vmdk, vhd or RAW disks.
type ImportDisk struct {
	Name, File string
}

func (i *ImportDisks) validate() error {
	for _, id := range *i {
		if err := diskNames.add(id.Name); err != nil {
			return fmt.Errorf("error adding disk: %s", err)
		}

		// File checking.
		if !sourceExists(id.File) {
			return fmt.Errorf("file not found: %s", id.File)
		}
	}
	return nil
}

func (i *ImportDisks) run(wf *Workflow) error {
	return nil
}
