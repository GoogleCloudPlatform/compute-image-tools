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
	"log"
	"path"
	"sync"
	"time"

	"google.golang.org/api/googleapi"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

// LogPrefix is a string that conforms to gcloud's output filter.
// To ensure that a line is shown by gcloud, emit a line to stdout
// using this string surrounded in brackets.
const LogPrefix = "import-image"

// Importer runs the end-to-end import workflow, and exposes the results
// via an error and Loggable.
type Importer interface {
	Run(ctx context.Context) error
}

// NewImporter constructs an Importer instance.
func NewImporter(request ImageImportRequest, computeClient compute.Client, storageClient storage.Client, logger logging.Logger) (Importer, error) {

	if err := request.validate(); err != nil {
		return nil, err
	}

	inflater, err := newInflater(request, computeClient, storageClient, imagefile.NewGCSInspector(), logger)
	if err != nil {
		return nil, err
	}

	inspector, err := disk.NewInspector(request.EnvironmentSettings(), logger)
	if err != nil {
		return nil, err
	}
	return &importer{
		project:      request.Project,
		zone:         request.Zone,
		timeout:      request.Timeout,
		preValidator: newPreValidator(request, computeClient),
		inflater:     inflater,
		processorProvider: defaultProcessorProvider{
			request,
			computeClient,
			newProcessPlanner(request, inspector),
			logger,
		},
		diskClient: computeClient,
		logger:     logger,
	}, nil
}

// importer is an implementation of Importer that uses a combination of Daisy workflows
// and GCP API calls.
type importer struct {
	project, zone     string
	pd                persistentDisk
	preValidator      validator
	inflater          Inflater
	processorProvider processorProvider
	diskClient        diskClient
	logger            logging.Logger
	timeout           time.Duration
}

func (i *importer) Run(ctx context.Context) error {
	if i.timeout.Nanoseconds() > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, i.timeout)
		defer cancel()
	}
	if err := i.preValidator.validate(); err != nil {
		return err
	}

	defer i.deleteDisk()

	if err := i.runInflate(ctx); err != nil {
		return err
	}

	err := i.runProcess(ctx)
	if err != nil {
		return err
	}

	return err
}

func (i *importer) runInflate(ctx context.Context) (err error) {
	return i.runStep(ctx, func() error {
		var err error
		i.pd, _, err = i.inflater.Inflate()
		if i.pd.sizeGb > 0 {
			i.logger.Metric(&pb.OutputInfo{
				SourcesSizeGb:    []int64{i.pd.sourceGb},
				TargetsSizeGb:    []int64{i.pd.sizeGb},
				ImportFileFormat: i.pd.sourceType,
			})
		}
		return err
	}, i.inflater.Cancel)
}

func (i *importer) runProcess(ctx context.Context) error {
	processors, err := i.processorProvider.provide(i.pd)
	if err != nil {
		return err
	}
	for _, processor := range processors {
		err = i.runStep(ctx, func() error {
			var err error
			i.pd, err = processor.process(i.pd)
			if err != nil {
				return err
			}
			return nil
		}, processor.cancel)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *importer) runStep(ctx context.Context, step func() error, cancel func(string) bool) (err error) {
	e := make(chan error)
	var wg sync.WaitGroup
	go func() {
		//this select checks if context expired prior to runStep being called
		//if not, step is run
		select {
		case <-ctx.Done():
			e <- i.getCtxError(ctx)
		default:
			wg.Add(1)
			var stepErr error
			defer func() {
				// error should only be returned after wg is marked as done. Otherwise,
				// a deadlock can occur when handling a timeout in the select below
				// because cancel() causes step() to finish, then waits for wg, while
				// writing to error chan waits on error chan reader which never happens
				wg.Done()
				e <- stepErr
			}()
			stepErr = step()
		}
	}()

	// this select waits for either context expiration or step to finish (with either an error or success)
	select {
	case <-ctx.Done():
		if cancel("timed-out") {
			//Only return timeout error if step was able to cancel on time-out.
			//Otherwise, step has finished and import succeeded even though it timed out
			err = i.getCtxError(ctx)
		}
		wg.Wait()
	case stepErr := <-e:
		err = stepErr
	}

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

func (i *importer) deleteDisk() {
	deleteDisk(i.diskClient, i.project, i.zone, i.pd)
}

func deleteDisk(diskClient diskClient, project string, zone string, pd persistentDisk) {
	if pd.uri == "" {
		return
	}

	diskName := path.Base(pd.uri)
	if err := diskClient.DeleteDisk(project, zone, diskName); err != nil {
		gAPIErr, isGAPIErr := err.(*googleapi.Error)
		if isGAPIErr && gAPIErr.Code != 404 {
			log.Printf("Failed to remove temporary disk %v: %e", pd, err)
		}
	}
}

// diskClient is the subset of the GCP API that is used by importer.
type diskClient interface {
	DeleteDisk(project, zone, uri string) error
}
