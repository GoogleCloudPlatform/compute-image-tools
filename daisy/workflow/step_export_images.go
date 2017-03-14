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

// ExportImages is a Daisy ExportImages workflow step.
type ExportImages []ExportImage

// ExportImage exports a GCE image or disk to a RAW image file in GCS.
type ExportImage struct {
	// Only one of these source types should be specified
	Image, Disk string
	Bucket      string
}

func (e *ExportImages) validate() error {
	for _, ei := range *e {
		// Image/Disk checking.
		if !xor(ei.Image == "", ei.Disk == "") {
			return fmt.Errorf("cannot export image. Must provide either Disk or Image, exclusively")
		}
		if ei.Image != "" && !imageExists(ei.Image) {
			return fmt.Errorf("cannot export image. Image not found: %s", ei.Image)
		}
		if ei.Disk != "" && !diskExists(ei.Disk) {
			return fmt.Errorf("cannot export image. Disk not found: %s", ei.Disk)
		}
	}

	return nil
}

func (e *ExportImages) run(wf *Workflow) error {
	return nil
}
