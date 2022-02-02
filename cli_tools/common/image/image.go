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

package image

import (
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
)

type defaultImage struct {
	Project, ImageName, URI string
}

// NewImage constructs an Image instance. The returned instance is not a GCE resource,
// but rather a convenience object for passing the image's project, image name,
// and URI.
func NewImage(project, imageName string) domain.Image {
	return &defaultImage{
		Project:   project,
		ImageName: imageName,
		URI:       param.GetImageResourcePath(project, imageName),
	}
}

// GetProject returns the project for the image.
func (d *defaultImage) GetProject() string {
	return d.Project
}

// GetImageName returns the image's name.
func (d *defaultImage) GetImageName() string {
	return d.ImageName
}

// GetURI returns the global GCP URI for the image.
func (d *defaultImage) GetURI() string {
	return d.URI
}
