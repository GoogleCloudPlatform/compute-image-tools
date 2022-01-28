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

package ovfgceutils

import (
	"fmt"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

// NewImageDeleter creates an ImageDeleter.
func NewImageDeleter(computeClient daisycompute.Client, logger logging.Logger) ovfdomain.ImageDeleter {
	return &imageDeleter{computeClient: computeClient, logger: logger}
}

type imageDeleter struct {
	computeClient daisycompute.Client
	logger        logging.Logger
}

// DeleteImagesIfExist iterates over images, and checks whether they exist.
// If so, it removes the image.
func (d *imageDeleter) DeleteImagesIfExist(images []ovfdomain.Image) {
	for _, image := range images {
		if _, err := d.computeClient.GetImage(image.Project, image.ImageName); err == nil {
			d.logger.Debug("Found image " + image.ImageName)
			if err = d.computeClient.DeleteImage(image.Project, image.ImageName); err != nil {
				d.logger.User(fmt.Sprintf("Failed to delete %q. Manual deletion required.",
					image.URI))
			} else {
				d.logger.Debug("Deleted image " + image.ImageName)
			}
		}
	}
}
