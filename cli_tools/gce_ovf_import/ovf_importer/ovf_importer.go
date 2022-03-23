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

package ovfimporter

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-daisy/compute"
	"github.com/vmware/govmomi/ovf"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/deleter"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	computeutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	daisyovfutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/daisy_utils"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	ovfgceutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/gce_utils"
	multidiskimporter "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/multi_disk_importer"
	ovfutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_utils"
)

const (
	ovfWorkflowDir         = "ovf_import/"
	createInstanceWorkflow = ovfWorkflowDir + "create_instance.wf.json"
	createGMIWorkflow      = ovfWorkflowDir + "create_gmi.wf.json"

	// Amount of time required after disk files have been imported. Used to calculate the
	// timeout budget for disk file import.
	instanceConstructionTime = 10 * time.Minute
)

var (
	// DefaultInstanceAccessScopes hold default instance access scopes
	// https://cloud.google.com/sdk/gcloud/reference/compute/instances/create#--scopes
	DefaultInstanceAccessScopes = []string{
		"https://www.googleapis.com/auth/devstorage.read_only",
		"https://www.googleapis.com/auth/logging.write",
		"https://www.googleapis.com/auth/monitoring.write",
		"https://www.googleapis.com/auth/pubsub",
		"https://www.googleapis.com/auth/service.management.readonly",
		"https://www.googleapis.com/auth/servicecontrol",
		"https://www.googleapis.com/auth/trace.append",
	}
)

// OVFImporter is responsible for importing OVF into GCE
type OVFImporter struct {
	ctx                 context.Context
	storageClient       domain.StorageClientInterface
	computeClient       daisyCompute.Client
	multiDiskImporter   ovfdomain.MultiDiskImporterInterface
	imageImporter       importer.Importer
	tarGcsExtractor     domain.TarGcsExtractorInterface
	ovfDescriptorLoader ovfdomain.OvfDescriptorLoaderInterface
	resourceDeleter     domain.ResourceDeleter
	Logger              logging.Logger
	gcsPathToClean      string
	workflowPath        string
	params              *ovfdomain.OVFImportParams
	imageLocation       string
	paramValidator      *ParamValidatorAndPopulator

	// Populated when disk file import finishes.
	images []domain.Image

	// Populated after creating the data disks from vmdk files.
	disks []domain.Disk

	// Populated after reading OVF file descriptor.
	machineTypeString string
}

// NewOVFImporter creates an OVF importer, including automatically populating dependencies,
// such as compute/storage clients. workflowDir is the filesystem path to `daisy_workflows`.
func NewOVFImporter(params *ovfdomain.OVFImportParams, logger logging.ToolLogger) (*OVFImporter, error) {
	ctx := context.Background()
	storageClient, err := storageutils.NewStorageClient(ctx, logger)
	if err != nil {
		return nil, err
	}
	computeClient, err := createComputeClient(&ctx, params)
	if err != nil {
		return nil, err
	}
	tarGcsExtractor := storageutils.NewTarGcsExtractor(ctx, storageClient, logger)
	workingDirOVFImportWorkflow := toWorkingDir(getImportWorkflowPath(params), params)
	ovfImporter := &OVFImporter{
		ctx:                 ctx,
		storageClient:       storageClient,
		computeClient:       computeClient,
		multiDiskImporter:   multidiskimporter.NewMultiDiskImporter(params.WorkflowDir, computeClient, storageClient, logger),
		imageImporter:       nil,
		resourceDeleter:     deleter.NewResourceDeleter(computeClient, logger),
		tarGcsExtractor:     tarGcsExtractor,
		workflowPath:        workingDirOVFImportWorkflow,
		ovfDescriptorLoader: ovfutils.NewOvfDescriptorLoader(storageClient),
		Logger:              logger,
		params:              params,
		paramValidator: &ParamValidatorAndPopulator{
			&computeutils.MetadataGCE{},
			&computeutils.ZoneValidator{ComputeClient: computeClient},
			&storageutils.BucketIteratorCreator{},
			storageClient,
			param.NewNetworkResolver(computeClient),
			logger,
		},
	}
	return ovfImporter, nil
}

func getImportWorkflowPath(params *ovfdomain.OVFImportParams) (workflow string) {
	if params.IsInstanceImport() {
		workflow = createInstanceWorkflow
	} else {
		workflow = createGMIWorkflow
	}
	return path.Join(params.WorkflowDir, workflow)
}

func (oi *OVFImporter) buildDaisyVars(bootDiskImage domain.Image, machineType string) map[string]string {
	varMap := map[string]string{}
	varMap["boot_disk_image_uri"] = bootDiskImage.GetURI()
	if oi.params.IsInstanceImport() {
		varMap["instance_name"] = oi.params.InstanceNames
	} else {
		varMap["machine_image_name"] = oi.params.MachineImageName
	}
	varMap["instance_service_account"] = oi.params.InstanceServiceAccount
	if oi.params.Subnet != "" {
		varMap["subnet"] = oi.params.Subnet
		// When subnet is set, we need to grant a value to network to avoid fallback to default
		if oi.params.Network == "" {
			varMap["network"] = ""
		}
	}
	if oi.params.Network != "" {
		varMap["network"] = oi.params.Network
	}
	if machineType != "" {
		varMap["machine_type"] = machineType
	}
	if oi.params.Description != "" {
		varMap["description"] = oi.params.Description
	}
	if oi.params.PrivateNetworkIP != "" {
		varMap["private_network_ip"] = oi.params.PrivateNetworkIP
	}
	if oi.params.NetworkTier != "" {
		varMap["network_tier"] = oi.params.NetworkTier
	}

	return varMap
}

func (oi *OVFImporter) updateImportedInstance(w *daisy.Workflow) {
	instance := (*w.Steps["create-instance"].CreateInstances).Instances[0]
	instanceBeta := (*w.Steps["create-instance"].CreateInstances).InstancesBeta[0]

	instance.CanIpForward = oi.params.CanIPForward
	instanceBeta.CanIpForward = oi.params.CanIPForward
	instance.DeletionProtection = oi.params.DeletionProtection
	instanceBeta.DeletionProtection = oi.params.DeletionProtection
	if instance.Scheduling == nil {
		instance.Scheduling = &compute.Scheduling{}
		instanceBeta.Scheduling = &computeBeta.Scheduling{}
	}
	if oi.params.NoRestartOnFailure {
		vFalse := false
		instance.Scheduling.AutomaticRestart = &vFalse
		instanceBeta.Scheduling.AutomaticRestart = &vFalse
	}
	if oi.params.NodeAffinities != nil {
		instance.Scheduling.NodeAffinities = oi.params.NodeAffinities
		instanceBeta.Scheduling.NodeAffinities = oi.params.NodeAffinitiesBeta
	}
	if len(oi.params.UserTags) > 0 {
		instance.Tags = &compute.Tags{
			Items: oi.params.UserTags,
		}
		instanceBeta.Tags = &computeBeta.Tags{
			Items: oi.params.UserTags,
		}
	}
	if oi.params.Hostname != "" {
		instance.Hostname = oi.params.Hostname
		instanceBeta.Hostname = oi.params.Hostname
	}
	if oi.params.InstanceServiceAccount == "" {
		instance.ServiceAccounts = []*compute.ServiceAccount{}
		instanceBeta.ServiceAccounts = []*computeBeta.ServiceAccount{}
	}
	if len(oi.params.InstanceAccessScopes) > 0 {
		for _, serviceAccount := range instance.ServiceAccounts {
			serviceAccount.Scopes = oi.params.InstanceAccessScopes
		}
		for _, serviceAccount := range instanceBeta.ServiceAccounts {
			serviceAccount.Scopes = oi.params.InstanceAccessScopes
		}
	}
}

func (oi *OVFImporter) updateMachineImage(w *daisy.Workflow) {
	if oi.params.MachineImageStorageLocation != "" {
		(*w.Steps["create-machine-image"].CreateMachineImages)[0].StorageLocations =
			[]string{oi.params.MachineImageStorageLocation}
	}
}

func toWorkingDir(dir string, params *ovfdomain.OVFImportParams) string {
	if path.IsAbs(dir) {
		return dir
	}
	wd, err := filepath.Abs(filepath.Dir(params.CurrentExecutablePath))
	if err == nil {
		return path.Join(wd, dir)
	}
	return dir
}

// creates a new Daisy Compute client
func createComputeClient(ctx *context.Context, params *ovfdomain.OVFImportParams) (daisyCompute.Client, error) {
	computeOptions := []option.ClientOption{option.WithCredentialsFile(params.Oauth)}
	if params.Ce != "" {
		computeOptions = append(computeOptions, option.WithEndpoint(params.Ce))
	}

	computeClient, err := daisyCompute.NewClient(*ctx, computeOptions...)
	if err != nil {
		return nil, err
	}
	return computeClient, nil
}

// Returns OVF GCS bucket and object path (director). If ovaOvaGcsPath is pointing to an OVA file,
// it extracts it to a temporary GCS folder and returns it's path.
func (oi *OVFImporter) getOvfGcsPath(tmpGcsPath string) (string, bool, error) {
	ovfOvaGcsPathLowered := strings.ToLower(oi.params.OvfOvaGcsPath)

	var ovfGcsPath string
	var shouldCleanUp bool
	var err error

	if strings.HasSuffix(ovfOvaGcsPathLowered, ".ova") {
		ovfGcsPath = pathutils.JoinURL(tmpGcsPath, "ovf")
		oi.Logger.User(
			fmt.Sprintf("Extracting %v OVA archive to %v", oi.params.OvfOvaGcsPath, ovfGcsPath))
		err = oi.tarGcsExtractor.ExtractTarToGcs(oi.params.OvfOvaGcsPath, ovfGcsPath)
		shouldCleanUp = true

	} else if strings.HasSuffix(ovfOvaGcsPathLowered, ".ovf") {
		// OvfOvaGcsPath is pointing to OVF descriptor, no need to unpack, just extract directory path.
		ovfGcsPath = (oi.params.OvfOvaGcsPath)[0 : strings.LastIndex(oi.params.OvfOvaGcsPath, "/")+1]

	} else {
		ovfGcsPath = oi.params.OvfOvaGcsPath
	}

	// assume OvfOvaGcsPath is a GCS folder for the whole OVF package
	return pathutils.ToDirectoryURL(ovfGcsPath), shouldCleanUp, err
}

func (oi *OVFImporter) getMachineType(
	ovfDescriptor *ovf.Envelope, project string, zone string) (string, error) {
	machineTypeProvider := ovfgceutils.MachineTypeProvider{
		OvfDescriptor: ovfDescriptor,
		MachineType:   oi.params.MachineType,
		ComputeClient: oi.computeClient,
		Project:       project,
		Zone:          zone,
	}
	return machineTypeProvider.GetMachineType()
}

// this function is mainly responsible for importing all the Disks files into GCP
// it creates disks for all data disks files, and import the boot disk file as disk image
func (oi *OVFImporter) importDisksFiles() error {
	oi.imageLocation = oi.params.Region

	ovfGcsPath, shouldCleanup, err := oi.getOvfGcsPath(oi.params.ScratchBucketGcsPath)
	if shouldCleanup {
		oi.gcsPathToClean = ovfGcsPath
	}
	if err != nil {
		return err
	}

	ovfDescriptor, diskInfos, err := ovfutils.GetOVFDescriptorAndDiskPaths(
		oi.ovfDescriptorLoader, ovfGcsPath)
	if err != nil {
		return err
	}

	osIDValue, err := oi.getOsIDValue(ovfDescriptor)
	if err != nil {
		return err
	}

	oi.params.OsID = osIDValue
	oi.params.Deadline = oi.params.Deadline.Add(-1 * instanceConstructionTime)

	oi.machineTypeString, err = oi.getMachineType(ovfDescriptor, *oi.params.Project, oi.params.Zone)
	if err != nil {
		return err
	}

	oi.Logger.User(fmt.Sprintf("Will create instance of `%v` machine type.", oi.machineTypeString))

	var disksNamesPrefix string
	if oi.params.IsInstanceImport() {
		disksNamesPrefix = oi.params.InstanceNames
	} else {
		disksNamesPrefix = oi.params.MachineImageName
	}

	if len(diskInfos) > 1 {
		var dataDiskURIs []string
		for _, info := range diskInfos[1:] {
			dataDiskURIs = append(dataDiskURIs, info.FilePath)
		}
		disks, err := oi.multiDiskImporter.Import(oi.ctx, oi.params, dataDiskURIs)
		if err != nil {
			return err
		}
		oi.disks = disks
	}

	if len(diskInfos) > 0 {
		if err = oi.importBootDiskImage(diskInfos[0], disksNamesPrefix); err != nil {
			return err
		}
	}

	return err
}

func (oi *OVFImporter) importBootDiskImage(bootDiskInfos ovfutils.DiskInfo, disksNamesPrefix string) error {

	oi.Logger.User(fmt.Sprint("Importing boot Disk Image ..."))

	imageName := fmt.Sprintf("%s-%s", disksNamesPrefix, "boot-image")
	request, err := oi.buildBootDiskImageImportRequest(imageName, bootDiskInfos.FilePath)

	if oi.imageImporter == nil {
		oi.imageImporter, err = importer.NewImporter(request, oi.computeClient, oi.storageClient, oi.Logger)
		if err != nil {
			return err
		}
	}

	err = oi.imageImporter.Run(oi.ctx)
	if err != nil {
		return err
	}

	oi.images = []domain.Image{image.NewImage(request.Project, request.ImageName)}

	return nil
}

func (oi *OVFImporter) createWorkflowForFinalInstance() (workflow *daisy.Workflow, err error) {

	varMap := oi.buildDaisyVars(oi.images[0], oi.machineTypeString)

	workflow, err = daisyutils.ParseWorkflow(oi.workflowPath, varMap, *oi.params.Project,
		oi.params.Zone, oi.params.ScratchBucketGcsPath, oi.params.Oauth, oi.params.Timeout, oi.params.Ce,
		oi.params.GcsLogsDisabled, oi.params.CloudLogsDisabled, oi.params.StdoutLogsDisabled)

	if err != nil {
		return nil, fmt.Errorf("error parsing workflow %q: %v", oi.workflowPath, err)
	}
	workflow.ForceCleanupOnError = true

	// See workflows in `ovfWorkflowDir` for variable name declaration.
	createInstanceStepName := "create-instance"

	// Attach the already created data disks to the final instance
	daisyovfutils.AppendDisksToInstance(
		workflow.Steps[createInstanceStepName].CreateInstances.Instances[0], oi.disks)

	oi.updateImportedInstance(workflow)
	if oi.params.IsMachineImageImport() {
		oi.updateMachineImage(workflow)
	}

	if oi.params.UefiCompatible {
		daisyutils.UpdateToUEFICompatible(workflow)
	}

	return workflow, nil
}

// Import runs OVF import. It first imports all of the disk files to images, and then uses the images
// to create an instance. If the user requested a machine image, then it converts the instance to a machine image.
func (oi *OVFImporter) Import() error {
	oi.Logger.User("Starting OVF import workflow.")
	if err := oi.paramValidator.ValidateAndPopulate(oi.params); err != nil {
		return err
	}
	if err := oi.importDisksFiles(); err != nil {
		oi.resourceDeleter.DeleteImagesIfExist(oi.images)
		oi.resourceDeleter.DeleteDisksIfExist(oi.disks)
		return err
	}
	if err := oi.createWorkerForFinalInstance().Run(map[string]string{}); err != nil {
		oi.Logger.User(err.Error())
		oi.resourceDeleter.DeleteImagesIfExist(oi.images)
		oi.resourceDeleter.DeleteDisksIfExist(oi.disks)
		return err
	}
	oi.Logger.User("OVF import workflow finished successfully.")
	return nil
}

func (oi *OVFImporter) createWorkerForFinalInstance() daisyutils.DaisyWorker {
	return daisyutils.NewDaisyWorker(func() (*daisy.Workflow, error) {
		return oi.createWorkflowForFinalInstance()
	}, oi.params.EnvironmentSettings(), oi.Logger,
		&daisyutils.ResourceLabeler{
			BuildID:         oi.params.BuildID,
			UserLabels:      oi.params.UserLabels,
			BuildIDLabelKey: "gce-ovf-import-build-id",
			ImageLocation:   oi.imageLocation,
			InstanceLabelKeyRetriever: func(instanceName string) string {
				if strings.ToLower(oi.params.InstanceNames) == instanceName {
					return "gce-ovf-import"
				}
				return "gce-ovf-import-tmp"
			},
			DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
				return "gce-ovf-import-tmp"
			},
			ImageLabelKeyRetriever: func(imageName string) string {
				return "gce-ovf-import-tmp"
			}})
}

// CleanUp performs clean up of any temporary resources or connections used for OVF import
func (oi *OVFImporter) CleanUp() {
	oi.Logger.User("Cleaning up.")
	if oi.storageClient != nil {
		if oi.gcsPathToClean != "" {
			err := oi.storageClient.DeleteGcsPath(oi.gcsPathToClean)
			if err != nil {
				oi.Logger.User(
					fmt.Sprintf("couldn't delete GCS path %v: %v", oi.gcsPathToClean, err.Error()))
			}
		}

		err := oi.storageClient.Close()
		if err != nil {
			oi.Logger.User(fmt.Sprintf("couldn't close storage client: %v", err.Error()))
		}
	}
}

// getOsIDValue determines the osID to use when importing the boot disk.
func (oi *OVFImporter) getOsIDValue(descriptor *ovf.Envelope) (osIDValue string, err error) {
	userOS := oi.params.OsID
	descriptorOS, err := ovfutils.GetOSId(descriptor)
	if err != nil {
		oi.Logger.Debug(fmt.Sprintf("Didn't find valid osID in descriptor. Error=%q", err))
		descriptorOS = ""
	}
	oi.Logger.Debug(fmt.Sprintf("osID candidates: from-user=%q, ovf-descriptor=%q", userOS, descriptorOS))

	if userOS != "" {
		osIDValue = userOS
		if descriptorOS != "" && userOS != descriptorOS {
			oi.Logger.User(
				fmt.Sprintf("WARNING: The OS info in the OVF descriptor was `%v`, "+
					"but you specified `%v`. Continuing import using your specified OS `%v`.",
					descriptorOS, userOS, userOS))
		}
	} else if descriptorOS != "" {
		osIDValue = descriptorOS
	} else {
		oi.Logger.User("Didn't find valid OS info in OVF descriptor. OS will be detected from boot disk.")
		return "", nil
	}

	if err = daisyutils.ValidateOS(osIDValue); err != nil {
		return "", err
	}
	if osIDValue == descriptorOS {
		oi.Logger.User(
			fmt.Sprintf("Found valid OS info in OVF descriptor, importing VM with `%v` as OS.",
				osIDValue))
	}
	return osIDValue, nil
}

func (oi *OVFImporter) buildBootDiskImageImportRequest(imageName string, bootDiskFilePath string) (request importer.ImageImportRequest, err error) {
	bootDiskFilePathSource, err := importer.NewSourceFactory(oi.storageClient).Init(bootDiskFilePath, "")

	if err != nil {
		return request, err
	}

	request = importer.ImageImportRequest{
		ExecutionID:           imageName,
		CloudLogsDisabled:     oi.params.CloudLogsDisabled,
		ComputeEndpoint:       oi.params.Ce,
		ComputeServiceAccount: oi.params.ComputeServiceAccount,
		WorkflowDir:           oi.params.WorkflowDir,
		GcsLogsDisabled:       oi.params.GcsLogsDisabled,
		ImageName:             imageName,
		Network:               oi.params.Network,
		NoExternalIP:          oi.params.NoExternalIP,
		NoGuestEnvironment:    oi.params.NoGuestEnvironment,
		Oauth:                 oi.params.Oauth,
		Project:               *oi.params.Project,
		ScratchBucketGcsPath:  pathutils.JoinURL(oi.params.ScratchBucketGcsPath, imageName),
		Source:                bootDiskFilePathSource,
		StdoutLogsDisabled:    oi.params.StdoutLogsDisabled,
		Subnet:                oi.params.Subnet,
		Timeout:               oi.params.Deadline.Sub(time.Now()),
		Tool:                  oi.params.GetTool(),
		UefiCompatible:        oi.params.UefiCompatible,
		Zone:                  oi.params.Zone,
		DataDisks:             oi.disks,
		OS:                    oi.params.OsID,
		BYOL:                  oi.params.BYOL,
	}

	importer.FixBYOLAndOSArguments(&request.OS, &request.BYOL)

	return request, err
}
