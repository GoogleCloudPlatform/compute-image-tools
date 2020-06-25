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
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/google/logger"
	"google.golang.org/api/googleapi"
)

// Importer runs the end-to-end import workflow, and exposes the results
// via an error and Loggable.
type Importer interface {
	Run(ctx context.Context) (service.Loggable, error)
}

// NewImporter constructs an Importer instance.
func NewImporter(args ImportArguments, client daisycompute.Client) (Importer, error) {

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
	processor         processor
}

func (i *importer) Run(ctx context.Context) (loggable service.Loggable, err error) {
	if i.timeout.Nanoseconds() > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, i.timeout)
		defer cancel()
	}
	if err = i.preValidator.validate(); err != nil {
		return i.buildLoggable(), err
	}

	defer i.cleanupDisk()

	if err := i.runInflate(ctx); err != nil {
		return i.buildLoggable(), err
	}

	err = i.runProcess(ctx)
	if err != nil {
		return i.buildLoggable(), err
	}

	return i.buildLoggable(), err
}

func (i *importer) runInflate(ctx context.Context) (err error) {
	return i.runStep(ctx, func(ctx context.Context) error {
		var err error
		i.pd, err = i.inflater.inflate(ctx)
		return err
	}, i.inflater.cancel, i.inflater.traceLogs)
}

func (i *importer) runProcess(ctx context.Context) (err error) {
	i.processor, err = i.processorProvider.provide(i.pd)
	if err != nil {
		return err
	}
	return i.runStep(ctx, func(ctx context.Context) error { return i.processor.process(ctx) }, i.processor.cancel, i.processor.traceLogs)
}

func (i *importer) runStep(ctx context.Context, step func(context.Context) error, cancel func(string), getTraceLogs func() []string) (err error) {
	e := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		//this select checks if context expired prior to runStep being called
		//if not, step is run
		select {
		case <-ctx.Done():
			e <- i.getCtxError(ctx)
		default:
			stepErr := step(ctx)
			wg.Done()
			e <- stepErr
		}
	}()

	// this select waits for either context expiration or step to finish (with either an error or success)
	select {
	case <-ctx.Done():
		err = i.getCtxError(ctx)
		cancel("timed-out")
		wg.Wait()
	case inflaterErr := <-e:
		err = inflaterErr
	}
	i.traceLogs = append(i.traceLogs, getTraceLogs()...)
	return err
}

func (i *importer) getCtxError(ctx context.Context) (err error) {
	if ctxErr := ctx.Err(); ctxErr == context.DeadlineExceeded {
		err = daisy.Errf("Import did not complete within the specified timeout of %s", i.timeout)
	} else {
		err = ctxErr
	}
	return err
}

func (i *importer) cleanupDisk() {
	if i.pd.uri == "" {
		return
	}

	diskName := path.Base(i.pd.uri)

	if err := i.diskClient.DeleteDisk(i.project, i.zone, diskName); err != nil {
		gAPIErr, isGAPIErr := err.(*googleapi.Error)
		if isGAPIErr && gAPIErr.Code != 404 {
			logger.Errorf("Failed to remove temporary disk %v: %e", i.pd, err)
		}
	}
}

func (i *importer) buildLoggable() service.Loggable {
	return service.SingleImageImportLoggable(i.pd.sourceType, i.pd.sourceGb, i.pd.sizeGb, i.traceLogs)
}

// diskClient is the subset of the GCP API that is used by importer.
type diskClient interface {
	DeleteDisk(project, zone, uri string) error
}
