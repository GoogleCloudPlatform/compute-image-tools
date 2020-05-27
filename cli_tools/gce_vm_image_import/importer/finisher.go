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
	"context"
)

// finisher represents the second (and final) phase of import. For bootable
// disks, this means translation and publishing the final image. For data
// disks, this means publishing the image.
type finisher interface {
	run(ctx context.Context) error
	serials() []string
}

// finisherProvider allows the translator to be
// determined after the pd has been inflated.
type finisherProvider interface {
	provide(pd pd) (finisher, error)
}

type defaultFinisherProvider struct {
	ImportArguments
	imageClient imageClient
}

func (d defaultFinisherProvider) provide(pd pd) (finisher, error) {
	if d.DataDisk {
		return newDataDiskFinisher(pd, d.imageClient, d.Project,
			d.Labels, d.StorageLocation, d.Description,
			d.Family, d.ImageName), nil
	}
	return newBootableFinisher(d.ImportArguments, pd, WorkflowDir)
}
