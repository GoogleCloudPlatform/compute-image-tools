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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

// NewMultiImageImporter constructs an implementation of MultiImageImporterInterface that can
// import disk files from GCS.
func NewMultiImageImporter(workflowDir string, computeClient daisycompute.Client,
	storageClient domain.StorageClientInterface, logger logging.ToolLogger) ovfdomain.MultiImageImporterInterface {
	return &multiImageImporter{
		builder: &requestBuilder{workflowDir, importer.NewSourceFactory(storageClient)},
		executor: &requestExecutor{
			&importAdapter{computeClient, storageClient},
			computeClient,
			logger,
		},
	}
}

type multiImageImporter struct {
	builder  *requestBuilder
	executor *requestExecutor
}

func (m *multiImageImporter) Import(ctx context.Context, params *ovfdomain.OVFImportParams, fileURIs []string) (images []domain.Image, err error) {
	var requests []importer.ImageImportRequest
	if requests, err = m.builder.buildRequests(params, fileURIs); err != nil {
		return nil, err
	}
	return m.executor.executeRequests(ctx, requests)
}

// importAdapter exposes a simplified interface for image import to facilitate testing.
type importAdapter struct {
	computeClient daisycompute.Client
	storageClient domain.StorageClientInterface
}

func (adapter *importAdapter) Import(ctx context.Context, request importer.ImageImportRequest, logger logging.Logger) error {
	imageImporter, err := importer.NewImporter(request, adapter.computeClient, adapter.storageClient, logger)
	if err != nil {
		return err
	}
	return imageImporter.Run(ctx)
}
