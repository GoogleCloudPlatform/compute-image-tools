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

package importer

import (
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"log"
)

// processor represents the second (and final) phase of import. For bootable
// disks, this means translation and publishing the final image. For data
// disks, this means publishing the image.
//
// Implementers can expose detailed logs using the traceLogs() method.
type processor interface {
	process() error
	traceLogs() []string
	cancel(reason string) bool
}

// processorProvider allows the processor to be determined after the pd has been inflated.
type processorProvider interface {
	provide(pd persistentDisk) (processor, error)
}

type defaultProcessorProvider struct {
	ImportArguments
	imageClient   createImageClient
	diskInspector disk.Inspector
}

func (d defaultProcessorProvider) provide(pd persistentDisk) (processor, error) {
	if d.DataDisk {
		return newDataDiskProcessor(pd, d.imageClient, d.Project,
			d.Labels, d.StorageLocation, d.Description,
			d.Family, d.ImageName), nil
	}
	if d.Inspect && d.diskInspector != nil {
		log.Printf("Running experimental disk inspections on %v.", pd.uri)
		inspectionResult, err := d.diskInspector.Inspect(pd.uri)
		if err != nil {
			log.Printf("Inspection error=%v", err)
		} else {
			log.Printf("Inspection result=%v", inspectionResult)
		}
	}
	return newBootableDiskProcessor(d.ImportArguments, pd)
}
