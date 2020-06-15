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
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/google/logger"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
)

// Importer runs the end-to-end import workflow, and exposes the results
// via an error and Loggable.
type Importer interface {
	Run(ctx context.Context) (service.Loggable, error)
}

// NewImporter constructs an Importer instance.
func NewImporter(args ImportArguments, client compute.Client) (Importer, error) {

	inflater, err := createDaisyInflater(args)
	if err != nil {
		return nil, err
	}
	return &importer{
		project:           args.Project,
		zone:              args.Zone,
		timeout:           args.Timeout,
		preValidator:      newPreValidator(args, client),
		inflater:          inflater,
		processorProvider: defaultProcessorProvider{ImportArguments: args, imageClient: client},
		traceLogs:         []string{},
		diskClient:        client,
	}, nil
}

// importer is an implementation of Importer that uses a combination of Daisy workflows
// and GCP API calls.
type importer struct {
	project, zone     string
	pd                persistentDisk
	preValidator      validator
	inflater          inflater
	processorProvider processorProvider
	traceLogs         []string
	diskClient        diskClient
	timeout           time.Duration
	isInProcess       bool
	processor         processor
}

func (i importer) Run(ctx context.Context) (loggable service.Loggable, err error) {
	go i.handleTimeout()

	if err = i.preValidator.validate(); err != nil {
		return i.buildLoggable(), err
	}

	if err = i.runInflate(ctx); err != nil {
		return i.buildLoggable(), err
	}

	i.isInProcess = true
	defer i.cleanupDisk()

	err = i.runProcess(ctx)
	if err != nil {
		return i.buildLoggable(), err
	}

	return i.buildLoggable(), err
}

func (i *importer) runInflate(ctx context.Context) (err error) {
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		i.pd, err = i.inflater.inflate(ctx)
		i.traceLogs = append(i.traceLogs, i.inflater.traceLogs()...)
	}
	return err
}

func (i *importer) runProcess(ctx context.Context) error {
	var err error
	i.processor, err = i.processorProvider.provide(i.pd)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		err = i.processor.process(ctx)
		i.traceLogs = append(i.traceLogs, i.processor.traceLogs()...)
	}
	return err
}

func (i *importer) cleanupDisk() {
	if i.pd.uri != "" {
		diskName := path.Base(i.pd.uri)
		err := i.diskClient.DeleteDisk(i.project, i.zone, diskName)
		if err != nil {
			logger.Errorf("Failed to remove temporary disk %v: %e", i.pd, err)
		}
	}
}

func (i *importer) buildLoggable() service.Loggable {
	return service.SingleImageImportLoggable(i.pd.sourceType, i.pd.sourceGb, i.pd.sizeGb, i.traceLogs)
}

func (i *importer) handleTimeout() {
	time.Sleep(i.timeout)
	logger.Errorf("Timeout %v exceeded, stopping the import", i.timeout)
	if i.isInProcess {
		i.processor.cancel("timed-out")
	} else {
		i.inflater.cancel()
	}
}

// diskClient is the subset of the GCP API that is used by importer.
type diskClient interface {
	DeleteDisk(project, zone, uri string) error
}
