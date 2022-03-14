//  Copyright 2022 Google Inc. All Rights Reserved.
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
//  limitations under the License

package disk

import (
	"errors"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
)

type defaultDisk struct {
	project, zone, diskName, uri string
}

// NewDisk constructs a convenience object for passing the disk's project, disk name, disk zone,
// and URI.
func NewDisk(project, zone, diskName string) (disk domain.Disk, err error) {
	if project == "" || zone == "" || diskName == "" {
		return disk, errors.New("Error creating new disk: project, zone or diskName cannot be empty")
	}

	disk = &defaultDisk{
		project:  project,
		diskName: diskName,
		zone:     zone,
		uri:      daisyutils.GetDiskURI(project, zone, diskName),
	}
	return disk, nil
}

// GetProject returns the project for the disk.
func (d *defaultDisk) GetProject() string {
	return d.project
}

// GetZoneName returns the disk's zone.
func (d *defaultDisk) GetZone() string {
	return d.zone
}

// GetDiskName returns the disk's name.
func (d *defaultDisk) GetDiskName() string {
	return d.diskName
}

// GetURI returns the global GCP URI for the image.
func (d *defaultDisk) GetURI() string {
	return d.uri
}
