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
	"log"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	commondisk "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	computeutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

const (
	logPrefix                  = "[export-ovf]"
	instanceExportWorkflow     = "ovf_export/export_instance_to_ovf.wf.json"
	machineImageExportWorkflow = "ovf_export/export_machine_image_to_ovf.wf.json"
)

const (
	//Alpha represents alpha release track
	Alpha = "alpha"

	//Beta represents beta release track
	Beta = "beta"

	//GA represents GA release track
	GA = "ga"
)

// OVFExporter is responsible for exporting GCE VMs/GMIs to OVF/OVA
type OVFExporter struct {
	ctx                    context.Context
	storageClient          domain.StorageClientInterface
	computeClient          daisycompute.Client
	mgce                   domain.MetadataGCEInterface
	bucketIteratorCreator  domain.BucketIteratorCreatorInterface
	Logger                 logging.LoggerInterface
	zoneValidator          domain.ZoneValidatorInterface
	instanceDisksExporter  InstanceDisksExporter
	instanceExportPreparer InstanceExportPreparer
	paramPopulator         param.Populator
	workflowPath           string
	params                 *OVFExportParams

	// BuildID is ID of Cloud Build in which this OVF export runs in
	BuildID string

	region                    string
	inspector                 commondisk.Inspector
	ovfDescriptorGenerator    OvfDescriptorGenerator
	workflowGenerator         *OVFExportWorkflowGenerator
	manifestFileGenerator     ManifestFileGenerator
	exportedDisks             []*ExportedDisk
	bootDiskInspectionResults commondisk.InspectionResult
	traceLogs                 []string
	timeout                   time.Duration
	loggableBuilder           *service.OVFExportLoggableBuilder
}

// ExportedDisk represents GCE disks exported to GCS disk files.
type ExportedDisk struct {
	attachedDisk *compute.AttachedDisk
	disk         *compute.Disk
	gcsPath      string
	gcsFileAttrs *storage.ObjectAttrs
}

// NewOVFExporter creates an OVF exporter, including automatically populating dependencies,
// such as compute/storage clients.
func NewOVFExporter(params *OVFExportParams) (*OVFExporter, error) {
	ctx := context.Background()
	log.SetPrefix(logPrefix + " ")
	logger := logging.NewStdoutLogger(logPrefix)
	storageClient, err := storageutils.NewStorageClient(ctx, logger)
	if err != nil {
		return nil, err
	}
	computeClient, err := createComputeClient(&ctx, params)
	if err != nil {
		return nil, err
	}
	metadataGCE := &computeutils.MetadataGCE{}
	paramPopulator := param.NewPopulator(
		metadataGCE,
		storageClient,
		storageutils.NewResourceLocationRetriever(metadataGCE, computeClient),
		storageutils.NewScratchBucketCreator(ctx, storageClient),
	)
	ovfExportWorkflowPath := filepath.Join(params.WorkflowDir, getExportWorkflowPath(params))
	bic := &storageutils.BucketIteratorCreator{}

	oe := &OVFExporter{
		ctx:                   ctx,
		storageClient:         storageClient,
		computeClient:         computeClient,
		workflowPath:          ovfExportWorkflowPath,
		BuildID:               getBuildID(params),
		mgce:                  metadataGCE,
		bucketIteratorCreator: bic,
		Logger:                logger,
		zoneValidator:         &computeutils.ZoneValidator{ComputeClient: computeClient},
		params:                params,
		paramPopulator:        paramPopulator,
		loggableBuilder:       service.NewOVFExportLoggableBuilder(),
	}

	if err := ValidateAndParseParams(oe.params, []string{GA, Beta, Alpha}); err != nil {
		return oe, err
	}
	if err := oe.paramPopulator.PopulateMissingParameters(oe.params.Project, oe.params.ClientID, &oe.params.Zone,
		&oe.region, &oe.params.ScratchBucketGcsPath, oe.params.DestinationURI, nil); err != nil {
		return oe, err
	}

	oe.ovfDescriptorGenerator = NewOvfDescriptorGenerator(oe.computeClient, oe.storageClient, *oe.params.Project, oe.params.Zone)
	oe.manifestFileGenerator = NewManifestFileGenerator(oe.storageClient)
	oe.inspector, err = commondisk.NewInspector(oe.params.DaisyAttrs())
	if err != nil {
		return oe, daisy.Errf("Error creating disk inspector: %v", err)
	}
	oe.workflowGenerator = &OVFExportWorkflowGenerator{
		Project:                *oe.params.Project,
		Zone:                   oe.params.Zone,
		OvfGcsDirectoryPath:    oe.params.DestinationURI,
		ExportedDiskFileFormat: oe.params.DiskExportFormat,
		Network:                oe.params.Network,
		Subnet:                 oe.params.Subnet,
	}
	oe.instanceDisksExporter = NewInstanceDisksExporter(oe.params, oe.workflowGenerator, oe.workflowPath, oe.computeClient, oe.storageClient)
	oe.instanceExportPreparer = NewInstanceExportPreparer(oe.params, oe.workflowGenerator, oe.workflowPath)
	return oe, nil
}

// creates a new Daisy Compute client
//TODO: consolidate with ovf_importer.createComputeClient
func createComputeClient(ctx *context.Context, params *OVFExportParams) (daisycompute.Client, error) {
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

func getExportWorkflowPath(params *OVFExportParams) string {
	if params.IsInstanceExport() {
		return instanceExportWorkflow
	}
	return machineImageExportWorkflow
}

func getBuildID(params *OVFExportParams) string {
	if params != nil && params.BuildID != "" {
		return params.BuildID
	}
	buildID := os.Getenv("BUILD_ID")
	if buildID == "" {
		buildID = pathutils.RandString(5)
	}
	return buildID
}

func (oe *OVFExporter) buildLoggable() service.Loggable {
	exportedDisksSourceSizes := make([]int64, len(oe.exportedDisks))
	exportedDisksTargetSizes := make([]int64, len(oe.exportedDisks))
	return oe.loggableBuilder.SetDiskSizes(
		exportedDisksSourceSizes,
		exportedDisksTargetSizes).SetTraceLogs(oe.traceLogs).
		Build()
}

// Export runs OVF export
func (oe *OVFExporter) run(ctx context.Context) error {
	oe.Logger.Log("Starting OVF export workflow.")
	if oe.timeout.Nanoseconds() > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, oe.timeout)
		defer cancel()
	}
	instance, err := oe.computeClient.GetInstance(*oe.params.Project, oe.params.Zone, oe.params.InstanceName)
	if err != nil {
		return daisy.Errf("Error retrieving instance `%v`: %v", oe.params.InstanceName, err)
	}
	defer func() {
		oe.Logger.Log("Cleaning up.")
		oe.cleanup(ctx, instance)
	}()

	oe.Logger.Log("Stopping the instance and detaching the disks.")

	if err = oe.prepare(ctx, instance); err != nil {
		return err
	}

	oe.Logger.Log("Exporting the disks.")
	if err := oe.exportDisks(ctx, instance); err != nil {
		return err
	}

	oe.Logger.Log("Inspecting the boot disk.")
	if err := oe.inspectBootDisk(ctx); err != nil {
		return err
	}

	oe.Logger.Log("Generating OVF descriptor.")
	if err = oe.generateDescriptor(ctx, instance); err != nil {
		return err
	}

	oe.Logger.Log("Generating manifest.")
	if err = oe.generateManifest(ctx); err != nil {
		return err
	}

	oe.Logger.Log("OVF export finished successfully.")
	return nil
}

// Run runs OVF export.
func Run(params *OVFExportParams) (service.Loggable, error) {
	var oe *OVFExporter
	var err error
	if oe, err = NewOVFExporter(params); err != nil {
		return nil, err
	}
	ctx := context.Background()
	err = oe.run(ctx)
	return oe.buildLoggable(), err
}
