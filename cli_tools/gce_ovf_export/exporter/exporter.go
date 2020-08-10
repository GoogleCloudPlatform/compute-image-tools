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
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
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
	ovfExportWorkflowDir       = "daisy_workflows/ovf_export/"
	instanceExportWorkflow     = ovfExportWorkflowDir + "export_instance_to_ovf.wf.json"
	machineImageExportWorkflow = ovfExportWorkflowDir + "export_machine_image_to_ovf.wf.json"
	logPrefix                  = "[export-ovf]"
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
	ctx                   context.Context
	storageClient         domain.StorageClientInterface
	computeClient         daisycompute.Client
	mgce                  domain.MetadataGCEInterface
	bucketIteratorCreator domain.BucketIteratorCreatorInterface
	Logger                logging.LoggerInterface
	zoneValidator         domain.ZoneValidatorInterface
	workflowPath          string
	params                *OVFExportParams
	paramPopulator        param.Populator
	instancePath          string

	// BuildID is ID of Cloud Build in which this OVF export runs in
	BuildID string

	region                 string
	bucketName             string
	gcsDirectoryPath       string
	instance               *compute.Instance
	ovfDescriptorGenerator *OvfDescriptorGenerator
	workflowGenerator      *OVFExportWorkflowGenerator
	exportedDisks          []*ExportedDisk

	prepareFn            func() (*daisy.Workflow, error)
	exportDisksFn        func() (*daisy.Workflow, error)
	generateDescriptorFn func() error
	cleanupFn            func() (*daisy.Workflow, error)
}

// ExportedDisk represents GCE disks exported to GCS disk files.
type ExportedDisk struct {
	attachedDisk *compute.AttachedDisk
	disk         *compute.Disk
	gcsPath      string
	gcsFileAttrs *storage.ObjectAttrs
}

// newOVFExporter creates an OVF exporter, including automatically populating dependencies,
// such as compute/storage clients.
func newOVFExporter(params *OVFExportParams) (*OVFExporter, error) {
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
	workingDirOVFExportWorkflow := toWorkingDir(getExportWorkflowPath(params), params)
	bic := &storageutils.BucketIteratorCreator{}

	ovfExporter := &OVFExporter{ctx: ctx, storageClient: storageClient, computeClient: computeClient,
		workflowPath: workingDirOVFExportWorkflow, BuildID: getBuildID(params),
		mgce: metadataGCE, bucketIteratorCreator: bic, Logger: logger,
		zoneValidator: &computeutils.ZoneValidator{ComputeClient: computeClient}, params: params, paramPopulator: paramPopulator}

	if err := ovfExporter.init(); err != nil {
		return nil, err
	}
	return ovfExporter, nil
}

func (oe *OVFExporter) init() error {
	var err error
	if err := ValidateAndParseParams(oe.params, []string{GA, Beta, Alpha}); err != nil {
		return err
	}
	if err := oe.paramPopulator.PopulateMissingParameters(oe.params.Project, &oe.params.Zone, &oe.region,
		&oe.params.ScratchBucketGcsPath, oe.params.DestinationURI, nil); err != nil {
		return err
	}

	//TODO: determine zone based on VM zone if instance export

	if oe.bucketName, oe.gcsDirectoryPath, err = storageutils.GetGCSObjectPathElements(oe.params.DestinationURI); err != nil {
		oe.Logger.Log(err.Error())
		return err
	}
	if oe.params.IsInstanceExport() {
		oe.instancePath = fmt.Sprintf("projects/%s/zones/%s/instances/%s", *oe.params.Project, oe.params.Zone, strings.ToLower(oe.params.InstanceName))
	}
	oe.instance, err = oe.computeClient.GetInstance(*oe.params.Project, oe.params.Zone, oe.params.InstanceName)
	if err != nil {
		return daisy.Errf("Error retrieving instance `%v`: %v", oe.params.InstanceName, err)
	}
	oe.ovfDescriptorGenerator = &OvfDescriptorGenerator{ComputeClient: oe.computeClient, StorageClient: oe.storageClient, Project: *oe.params.Project, Zone: oe.params.Zone}
	oe.workflowGenerator = &OVFExportWorkflowGenerator{
		Instance:               oe.instance,
		Project:                *oe.params.Project,
		Zone:                   oe.params.Zone,
		OvfGcsDirectoryPath:    oe.params.DestinationURI,
		ExportedDiskFileFormat: oe.params.DiskExportFormat,
		Network:                oe.params.Network,
		Subnet:                 oe.params.Subnet,
		InstancePath:           oe.instancePath,
	}
	return nil
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

//TODO: consolidate with ovf_importer.toWorkingDir
func toWorkingDir(dir string, params *OVFExportParams) string {
	wd, err := filepath.Abs(filepath.Dir(params.CurrentExecutablePath))
	if err == nil {
		return path.Join(wd, dir)
	}
	return dir
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

// Export runs OVF export
func (oe *OVFExporter) run() (*daisy.Workflow, error) {
	oe.Logger.Log("Starting OVF export workflow.")

	defer func() {
		oe.Logger.Log("Cleaning up.")
		oe.cleanup()
		oe.Logger.Log("OVF export finished successfully.")
	}()

	var err error
	oe.Logger.Log("Stopping the instance and detaching the disks.")
	prepareWf, err := oe.prepare()
	if err != nil {
		return prepareWf, err
	}

	oe.Logger.Log("Exporting the disks.")
	exportDisksWf, err := oe.exportDisks()
	if err != nil {
		return exportDisksWf, err
	}

	oe.Logger.Log("Generating OVF descriptor.")
	if err = oe.generateDescriptor(); err != nil {
		return exportDisksWf, err
	}

	return exportDisksWf, nil
}

// Run runs OVF export.
func Run(params *OVFExportParams) (service.Loggable, error) {
	var ovfExporter *OVFExporter
	var err error
	if ovfExporter, err = newOVFExporter(params); err != nil {
		return nil, err
	}

	w, err := ovfExporter.run()
	return service.NewLoggableFromWorkflow(w), err
}
