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
	"fmt"
	"strings"

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
	disks = make([]domain.Disk, len(requests))

	for i, request := range requests {
		req := request
		diskIdx := i
		logPrefix := fmt.Sprintf("[import-%s]", req.DaisyLogLinePrefix)
		group.Go(func() error {
			diskURI, err := r.singleDiskImporter.Import(ctx, req, r.logger.NewLogger(logPrefix))
			if err != nil {
				return err
			}
			disk, err := disk.NewDisk(req.Project, req.Zone, getDiskNameFromURI(diskURI))
			if err != nil {
				return err
			}
			disks[diskIdx] = disk
			return nil
		})
	}
	return disks, group.Wait()
}

func getDiskNameFromURI(URI string) string {
	SplittedDiskURI := strings.Split(URI, "/")
	diskName := SplittedDiskURI[len(SplittedDiskURI)-1]
	return diskName
}
