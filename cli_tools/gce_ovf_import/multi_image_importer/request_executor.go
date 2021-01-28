//  Copyright 2021 Google Inc. All Rights Reserved.
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

package multiimageimporter

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

type requestExecutor struct {
	singleImporter ovfdomain.ImageImporterInterface
	computeClient  daisycompute.Client
	logger         logging.ToolLogger
}

// executeRequests performs multiple image import requests in parallel, and blocks until
// all requests are finished. If a request fails, all requests are stopped, and finished
// images are deleted.
//
// On success, returns the URIs of the imported images, in the same order as requests.
func (r *requestExecutor) executeRequests(parentContext context.Context, requests []importer.ImageImportRequest) (imageURIs []string, err error) {
	group, ctx := errgroup.WithContext(parentContext)
	// Check whether any of the proposed image names exist, and exit if so. Pre-checking to
	// avoid deleting the pre-existing image during cleanup.
	for _, request := range requests {
		if request.Timeout <= 0 {
			return nil, errors.New("Timeout exceeded")
		}

		if _, err := r.computeClient.GetImage(request.Project, request.ImageName); err == nil {
			return imageURIs, daisy.Errf("Intermediate image %s already exists. Re-run import.", request.ImageName)
		}
	}
	for _, request := range requests {
		req := request
		imageURIs = append(imageURIs, param.GetImageResourcePath(request.Project, request.ImageName))
		logPrefix := fmt.Sprintf("[import-%s]", req.DaisyLogLinePrefix)
		group.Go(func() error {
			return r.singleImporter.Import(ctx, req, r.logger.NewLogger(logPrefix))
		})
	}
	err = group.Wait()
	if err != nil {
		r.deleteImagesIfExist(requests)
	}
	return imageURIs, err
}

func (r *requestExecutor) deleteImagesIfExist(requests []importer.ImageImportRequest) {
	for _, request := range requests {
		if _, err := r.computeClient.GetImage(request.Project, request.ImageName); err == nil {
			r.logger.Debug("Found image " + request.ImageName)
			if err = r.computeClient.DeleteImage(request.Project, request.ImageName); err != nil {
				r.logger.User(fmt.Sprintf("Failed to delete %q. Manual deletion required.",
					param.GetImageResourcePath(request.Project, request.ImageName)))
			} else {
				r.logger.Debug("Deleted image " + request.ImageName)
			}
		}
	}
}
