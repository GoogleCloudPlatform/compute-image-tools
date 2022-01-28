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
//  limitations under the License.

package domain

import "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"

// Image holds the project, name, and URI of a GCP disk image.
type Image struct {
	Project, ImageName, URI string
}

// NewImage constructs an Image instance.
func NewImage(Project, ImageName string) Image {
	return Image{
		Project:   Project,
		ImageName: ImageName,
		URI:       param.GetImageResourcePath(Project, ImageName),
	}
}
