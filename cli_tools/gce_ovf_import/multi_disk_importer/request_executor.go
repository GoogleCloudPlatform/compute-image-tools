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

package multidiskimporter

import (
	"context"
	"errors"
	"fmt"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-daisy/compute"
	"golang.org/x/sync/errgroup"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
)

type requestExecutor struct {
	singleDiskImporter ovfdomain.DiskImporterInterface
	computeClient      daisyCompute.Client
	logger             logging.ToolLogger
}

// executeRequests create multiple disks in parallel, and blocks until
// all disks are created. If a disk creation fails, all requests are stopped.
//
// On success, returns the URIs of the imported disks, in the same order as requests.
func (r *requestExecutor) executeRequests(parentContext context.Context, requests []importer.ImageImportRequest) (disks []domain.Disk, err error) {
	group, ctx := errgroup.WithContext(parentContext)
	// Check whether any of the proposed disk names exist, and exit if so. Pre-checking to
	// avoid deleting the pre-existing disks during cleanup.
	for _, request := range requests {
		if request.Timeout <= 0 {
			return nil, errors.New("Timeout exceeded")
		}
		if _, err := r.computeClient.GetDisk(request.Project, request.Zone, request.DiskName); err == nil {
			return disks, daisy.Errf("Intermediate disk %s already exists. Re-run create-disk.", request.DiskName)
		}
	}

	for _, request := range requests {
		req := request
		disk, err := disk.NewDisk(req.Project, req.Zone, req.DiskName)
		if err != nil {
			return nil, err
		}

		disks = append(disks, disk)
		logPrefix := fmt.Sprintf("[import-%s]", req.DaisyLogLinePrefix)

		group.Go(func() error {
			return r.singleDiskImporter.Import(ctx, req, r.logger.NewLogger(logPrefix))
		})
	}
	return disks, group.Wait()
}
