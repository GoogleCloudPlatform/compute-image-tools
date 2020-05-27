//  Copyright 2019 Google Inc. All Rights Reserved.
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

package importer

import (
	"context"
	"path"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/google/logger"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
)

const WorkflowDir = "daisy_workflows/image_import/"

// Importer runs the import workflow.
type Importer interface {
	Run(ctx context.Context) (service.Loggable, error)
}

// NewImporter constructs an Importer instance.
func NewImporter(args ImportArguments, client compute.Client) (Importer, error) {
	inflater, err := createDaisyInflater(args, WorkflowDir)
	if err != nil {
		return nil, err
	}
	return importer{
		project:          args.Project,
		zone:             args.Zone,
		inflater:         inflater,
		finisherProvider: defaultFinisherProvider{ImportArguments: args, imageClient: client},
		serials:          []string{},
		diskClient:       client,
	}, nil
}

type importer struct {
	project, zone    string
	pd               pd
	inflater         inflater
	finisherProvider finisherProvider
	serials          []string
	diskClient       diskClient
}

func (d importer) Run(ctx context.Context) (loggable service.Loggable, err error) {
	if err = d.runInflate(ctx); err != nil {
		return d.buildLoggable(), err
	}

	defer d.cleanupDisk()

	err = d.runFinish(ctx)
	if err != nil {
		return d.buildLoggable(), err
	}

	return d.buildLoggable(), err
}

func (d *importer) runInflate(ctx context.Context) (err error) {
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		d.pd, err = d.inflater.inflate(ctx)
		d.serials = append(d.serials, d.inflater.serials()...)
	}
	return err
}

func (d *importer) runFinish(ctx context.Context) (err error) {
	finisher, err := d.finisherProvider.provide(d.pd)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		err = finisher.run(ctx)
		d.serials = append(d.serials, finisher.serials()...)
	}
	return err
}

func (d *importer) cleanupDisk() {
	if d.pd.uri != "" {
		diskName := path.Base(d.pd.uri)
		err := d.diskClient.DeleteDisk(d.project, d.zone, diskName)
		if err != nil {
			logger.Errorf("Failed to remove temporary disk %v: %e", d.pd, err)
		}
	}
}

func (d importer) buildLoggable() service.Loggable {
	return service.SingleImageImportLoggable(d.pd.sourceType, d.pd.sourceGb, d.pd.sizeGb, d.serials)
}

type diskClient interface {
	DeleteDisk(project, zone, uri string) error
}
