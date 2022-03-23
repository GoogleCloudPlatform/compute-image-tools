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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDisk(t *testing.T) {
	disk, err := NewDisk("project-id", "test-zone", "disk-name")
	assert.NoError(t, err)
	assert.Equal(t, "project-id", disk.GetProject())
	assert.Equal(t, "test-zone", disk.GetZone())
	assert.Equal(t, "disk-name", disk.GetDiskName())
	assert.Equal(t, "projects/project-id/zones/test-zone/disks/disk-name", disk.GetURI())
}

func TestNewDisk_Error(t *testing.T) {
	_, err := NewDisk("", "test-zone", "disk-name")
	assert.Error(t, err, errors.New("Error creating new disk: project, zone or diskName cannot be empty"))

	_, err = NewDisk("project-id", "", "disk-name")
	assert.Error(t, err, errors.New("Error creating new disk: project, zone or diskName cannot be empty"))

	_, err = NewDisk("project-id", "test-zone", "")
	assert.Error(t, err, errors.New("Error creating new disk: project, zone or diskName cannot be empty"))
}
