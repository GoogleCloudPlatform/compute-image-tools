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
	"fmt"

	daisyCompute "github.com/GoogleCloudPlatform/compute-daisy/compute"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
)

// NewImageDeleter creates an ImageDeleter.
func NewImageDeleter(computeClient daisyCompute.Client, logger logging.Logger) domain.ImageDeleter {
	return &imageDeleter{computeClient: computeClient, logger: logger}
}

type imageDeleter struct {
	computeClient daisyCompute.Client
	logger        logging.Logger
}

// DeleteImagesIfExist iterates over images, and checks whether they exist.
// If so, it removes the image.
func (d *imageDeleter) DeleteImagesIfExist(images []domain.Image) {
	for _, image := range images {
		if _, err := d.computeClient.GetImage(image.GetProject(), image.GetImageName()); err == nil {
			d.logger.Debug("Found image " + image.GetImageName())
			if err = d.computeClient.DeleteImage(image.GetProject(), image.GetImageName()); err != nil {
				d.logger.User(fmt.Sprintf("Failed to delete %q. Manual deletion required.",
					image.GetURI()))
			} else {
				d.logger.Debug("Deleted image " + image.GetImageName())
			}
		}
	}
}
