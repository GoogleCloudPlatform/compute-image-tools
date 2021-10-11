//  Copyright 2020 Google Inc. All Rights Reserved.
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

package ovfexporter

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"

	commondisk "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	computeutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

const (
	// LogPrefix is prefix for OVF export log lines
	LogPrefix  = "[ovf-export]"
	bytesPerGB = int64(1024 * 1024 * 1024)
)

// OVFExporter is responsible for exporting GCE VMs/GMIs to OVF/OVA
type OVFExporter struct {
	storageClient          domain.StorageClientInterface
	computeClient          daisycompute.Client
	mgce                   domain.MetadataGCEInterface
	bucketIteratorCreator  domain.BucketIteratorCreatorInterface
	Logger                 logging.Logger
	instanceDisksExporter  ovfexportdomain.InstanceDisksExporter
	instanceExportPreparer ovfexportdomain.InstanceExportPreparer
	instanceExportCleaner  ovfexportdomain.InstanceExportCleaner
	params                 *ovfexportdomain.OVFExportArgs

	// BuildID is ID of Cloud Build in which this OVF export runs in
	BuildID string

	inspector                 commondisk.Inspector
	ovfDescriptorGenerator    ovfexportdomain.OvfDescriptorGenerator
	manifestFileGenerator     ovfexportdomain.OvfManifestGenerator
	exportedDisks             []*ovfexportdomain.ExportedDisk
	bootDiskInspectionResults *pb.InspectionResults
	loggableBuilder           *service.OvfExportLoggableBuilder
}

// NewOVFExporter creates an OVF exporter, including automatically populating dependencies,
// such as compute/storage clients.
func NewOVFExporter(params *ovfexportdomain.OVFExportArgs, logger logging.ToolLogger) (*OVFExporter, error) {
	ctx := context.Background()

	storageClient, err := storageutils.NewStorageClient(ctx, logger)
	if err != nil {
		return nil, err
	}
	computeClient, err := createComputeClient(&ctx, params)
	if err != nil {
		return nil, err
	}
	metadataGCE := &computeutils.MetadataGCE{}

	paramValidator := NewOvfExportParamValidator(computeClient)
	paramPopulator := NewPopulator(computeClient, metadataGCE, storageClient,
		storageutils.NewResourceLocationRetriever(metadataGCE, computeClient),
		storageutils.NewScratchBucketCreator(ctx, storageClient),
	)
	if err := validateAndPopulateParams(params, paramValidator, paramPopulator); err != nil {
		return nil, err
	}
	inspector, err := commondisk.NewInspector(params.EnvironmentSettings("ovf-export-disk-inspect"), logger)
	if err != nil {
		return nil, daisy.Errf("Error creating disk inspector: %v", err)
	}
	return &OVFExporter{
		storageClient:          storageClient,
		computeClient:          computeClient,
		mgce:                   metadataGCE,
		bucketIteratorCreator:  &storageutils.BucketIteratorCreator{},
		Logger:                 logger,
		params:                 params,
		loggableBuilder:        service.NewOvfExportLoggableBuilder(),
		ovfDescriptorGenerator: NewOvfDescriptorGenerator(computeClient, storageClient, params.Project, params.Zone),
		manifestFileGenerator:  NewManifestFileGenerator(storageClient),
		inspector:              inspector,
		instanceDisksExporter:  NewInstanceDisksExporter(computeClient, storageClient, logger),
		instanceExportPreparer: NewInstanceExportPreparer(logger),
		instanceExportCleaner:  NewInstanceExportCleaner(logger),
	}, nil
}

// validateAndPopulateParams validate and populate OVF export params
func validateAndPopulateParams(params *ovfexportdomain.OVFExportArgs,
	paramValidator ovfexportdomain.OvfExportParamValidator,
	paramPopulator ovfexportdomain.OvfExportParamPopulator) error {
	if err := paramValidator.ValidateAndParseParams(params); err != nil {
		return err
	}
	if err := paramPopulator.Populate(params); err != nil {
		return err
	}
	return nil
}

// creates a new Daisy Compute client
//TODO: consolidate with ovf_importer.createComputeClient
func createComputeClient(ctx *context.Context, params *ovfexportdomain.OVFExportArgs) (daisycompute.Client, error) {
	computeOptions := []option.ClientOption{option.WithCredentialsFile(params.Oauth)}
	if params.Ce != "" {
		computeOptions = append(computeOptions, option.WithEndpoint(params.Ce))
	}

	computeClient, err := daisycompute.NewClient(*ctx, computeOptions...)
	if err != nil {
		return nil, err
	}
	return computeClient, nil
}

func getInstancePath(instance *compute.Instance, project string) string {
	return fmt.Sprintf("projects/%s/zones/%s/instances/%s", project, instance.Zone[strings.LastIndex(instance.Zone, "/")+1:], strings.ToLower(instance.Name))
}

// Export runs OVF export
func (oe *OVFExporter) run(ctx context.Context) error {
	oe.Logger.User("Starting OVF export.")
	if oe.params.Timeout.Nanoseconds() > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, oe.params.Timeout)
		defer cancel()
	}

	//TODO: if machine image export, create instance from machine image and add it to cleanup

	instance, err := oe.computeClient.GetInstance(oe.params.Project, oe.params.Zone, oe.params.InstanceName)
	if err != nil {
		return daisy.Errf("Error retrieving instance `%v`: %v", oe.params.InstanceName, err)
	}
	defer func() {
		oe.cleanup(instance, err)
	}()
	if err = oe.prepare(ctx, instance); err != nil {
		return err
	}
	if err := oe.exportDisks(ctx, instance); err != nil {
		return err
	}
	if err := oe.inspectBootDisk(ctx); err != nil {
		return err
	}
	if err = oe.generateDescriptor(ctx, instance); err != nil {
		return err
	}
	if err = oe.generateManifest(ctx); err != nil {
		return err
	}
	return nil
}

// Run runs OVF export.
func (oe *OVFExporter) Run(ctx context.Context) error {
	var err error
	err = oe.run(ctx)
	return err
}
