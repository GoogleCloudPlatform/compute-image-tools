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
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	computeutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/daisy_utils"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/gce_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_import_params"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/vmware/govmomi/ovf"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	ovfWorkflowDir    = "daisy_workflows/ovf_import/"
	ovfImportWorkflow = ovfWorkflowDir + "import_ovf.wf.json"
)

const (
	//Alpha represents alpha release track
	Alpha = "alpha"

	//Beta represents beta release track
	Beta = "beta"

	//GA represents GA release track
	GA = "ga"
)

// OVFImporter is responsible for importing OVF into GCE
type OVFImporter struct {
	ctx                   context.Context
	storageClient         domain.StorageClientInterface
	computeClient         daisycompute.Client
	tarGcsExtractor       domain.TarGcsExtractorInterface
	mgce                  domain.MetadataGCEInterface
	ovfDescriptorLoader   ovfdomain.OvfDescriptorLoaderInterface
	bucketIteratorCreator domain.BucketIteratorCreatorInterface
	Logger                logging.LoggerInterface
	zoneValidator         domain.ZoneValidatorInterface
	gcsPathToClean        string
	workflowPath          string
	diskInfos             *[]ovfutils.DiskInfo
	params                *ovfimportparams.OVFImportParams
	imageLocation         string

	// BuildID is ID of Cloud Build in which this OVF import runs in
	BuildID string
}

// NewOVFImporter creates an OVF importer, including automatically populating dependencies,
// such as compute/storage clients.
func NewOVFImporter(params *ovfimportparams.OVFImportParams) (*OVFImporter, error) {
	ctx := context.Background()
	logger := logging.NewLogger("[import-ovf]")
	storageClient, err := storageutils.NewStorageClient(ctx, logger, "")
	if err != nil {
		return nil, err
	}
	computeClient, err := createComputeClient(&ctx, params)
	if err != nil {
		return nil, err
	}
	tarGcsExtractor := storageutils.NewTarGcsExtractor(ctx, storageClient, logger)
	buildID := os.Getenv("BUILD_ID")

	if buildID == "" {
		buildID = pathutils.RandString(5)
	}
	workingDirOVFImportWorkflow := toWorkingDir(ovfImportWorkflow, params)
	bic := &storageutils.BucketIteratorCreator{}

	ovfImporter := &OVFImporter{ctx: ctx, storageClient: storageClient, computeClient: computeClient,
		tarGcsExtractor: tarGcsExtractor, workflowPath: workingDirOVFImportWorkflow, BuildID: buildID,
		ovfDescriptorLoader: ovfutils.NewOvfDescriptorLoader(storageClient),
		mgce:                &computeutils.MetadataGCE{}, bucketIteratorCreator: bic, Logger: logger,
		zoneValidator: &computeutils.ZoneValidator{ComputeClient: computeClient}, params: params}
	return ovfImporter, nil
}

func (oi *OVFImporter) buildDaisyVars(
	translateWorkflowPath string,
	bootDiskGcsPath string,
	machineType string,
	region string) map[string]string {
	varMap := map[string]string{}

	varMap["instance_name"] = strings.ToLower(oi.params.InstanceNames)
	if translateWorkflowPath != "" {
		varMap["translate_workflow"] = translateWorkflowPath
		varMap["install_gce_packages"] = strconv.FormatBool(!oi.params.NoGuestEnvironment)
	}
	if bootDiskGcsPath != "" {
		varMap["boot_disk_file"] = bootDiskGcsPath
	}
	if oi.params.Network != "" {
		varMap["network"] = fmt.Sprintf("global/networks/%v", oi.params.Network)
	}
	if oi.params.Subnet != "" {
		varMap["subnet"] = fmt.Sprintf("regions/%v/subnetworks/%v", region, oi.params.Subnet)
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

func (oi *OVFImporter) updateInstance(w *daisy.Workflow) {
	instance := (*w.Steps["create-instance"].CreateInstances)[0]
	instance.CanIpForward = oi.params.CanIPForward
	instance.DeletionProtection = oi.params.DeletionProtection
	if instance.Scheduling == nil {
		instance.Scheduling = &compute.Scheduling{}
	}
	if oi.params.NoRestartOnFailure {
		vFalse := false
		instance.Scheduling.AutomaticRestart = &vFalse
	}
	if oi.params.NodeAffinities != nil {
		instance.Scheduling.NodeAffinities = oi.params.NodeAffinities
	}
}

func toWorkingDir(dir string, params *ovfimportparams.OVFImportParams) string {
	wd, err := filepath.Abs(filepath.Dir(params.CurrentExecutablePath))
	if err == nil {
		return path.Join(wd, dir)
	}
	return dir
}

// creates a new Daisy Compute client
func createComputeClient(ctx *context.Context, params *ovfimportparams.OVFImportParams) (daisycompute.Client, error) {
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

func (oi *OVFImporter) getProject() (string, error) {
	return param.GetProjectID(oi.mgce, oi.params.Project)
}

func (oi *OVFImporter) getZone(project string) (string, error) {
	if oi.params.Zone != "" {
		if err := oi.zoneValidator.ZoneValid(project, oi.params.Zone); err != nil {
			return "", err
		}
		return oi.params.Zone, nil
	}

	if !oi.mgce.OnGCE() {
		return "", fmt.Errorf("zone cannot be determined because build is not running on GCE")
	}
	// determine zone based on the zone Cloud Build is running in
	zone, err := oi.mgce.Zone()
	if err != nil || zone == "" {
		return "", fmt.Errorf("can't infer zone: %v", err)
	}
	return zone, nil
}

func (oi *OVFImporter) getRegion(zone string) (string, error) {
	zoneSplits := strings.Split(zone, "-")
	if len(zoneSplits) < 2 {
		return "", fmt.Errorf("%v is not a valid zone", zone)
	}
	return strings.Join(zoneSplits[:len(zoneSplits)-1], "-"), nil
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
		oi.Logger.Log(
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

func (oi *OVFImporter) createScratchBucketBucket(project string, region string) error {
	safeProjectName := strings.Replace(project, "google", "elgoog", -1)
	safeProjectName = strings.Replace(safeProjectName, ":", "-", -1)
	if strings.HasPrefix(safeProjectName, "goog") {
		safeProjectName = strings.Replace(safeProjectName, "goog", "ggoo", 1)
	}
	bucket := strings.ToLower(safeProjectName + "-ovf-import-bkt-" + region)
	it := oi.bucketIteratorCreator.CreateBucketIterator(oi.ctx, oi.storageClient, project)
	for itBucketAttrs, err := it.Next(); err != iterator.Done; itBucketAttrs, err = it.Next() {
		if err != nil {
			return err
		}
		if itBucketAttrs.Name == bucket {
			oi.params.ScratchBucketGcsPath = fmt.Sprintf("gs://%v/", bucket)
			return nil
		}
	}

	oi.Logger.Log(fmt.Sprintf("Creating scratch bucket `%v` in %v region", bucket, region))
	if err := oi.storageClient.CreateBucket(
		bucket, project,
		&storage.BucketAttrs{Name: bucket, Location: region}); err != nil {
		return err
	}
	oi.params.ScratchBucketGcsPath = fmt.Sprintf("gs://%v/", bucket)
	return nil
}

func (oi *OVFImporter) buildTmpGcsPath(project string, region string) (string, error) {
	if oi.params.ScratchBucketGcsPath == "" {
		if err := oi.createScratchBucketBucket(project, region); err != nil {
			return "", err
		}
	}
	return pathutils.JoinURL(oi.params.ScratchBucketGcsPath,
		fmt.Sprintf("ovf-import-%v", oi.BuildID)), nil
}

func (oi *OVFImporter) modifyWorkflowPostValidate(w *daisy.Workflow) {
	w.LogWorkflowInfo("OVF import flags: %s", oi.params)
	w.LogWorkflowInfo("Cloud Build ID: %s", oi.BuildID)
	rl := &daisyutils.ResourceLabeler{
		BuildID:         oi.BuildID,
		UserLabels:      oi.params.UserLabels,
		BuildIDLabelKey: "gce-ovf-import-build-id",
		ImageLocation:   oi.imageLocation,
		InstanceLabelKeyRetriever: func(instance *daisy.Instance) string {
			if strings.ToLower(oi.params.InstanceNames) == instance.Name {
				return "gce-ovf-import"
			}
			return "gce-ovf-import-tmp"
		},
		DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
			return "gce-ovf-import-tmp"
		},
		ImageLabelKeyRetriever: func(imageName string) string {
			return "gce-ovf-import-tmp"
		}}
	rl.LabelResources(w)
	daisyutils.UpdateAllInstanceNoExternalIP(w, oi.params.NoExternalIP)
}

func (oi *OVFImporter) modifyWorkflowPreValidate(w *daisy.Workflow) {
	daisyovfutils.AddDiskImportSteps(w, (*oi.diskInfos)[1:])
	oi.updateInstance(w)
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

func (oi *OVFImporter) setUpImportWorkflow() (*daisy.Workflow, error) {
	if err := ovfimportparams.ValidateAndParseParams(oi.params); err != nil {
		return nil, err
	}
	var (
		project string
		zone    string
		region  string
		err     error
	)
	if project, err = param.GetProjectID(oi.mgce, oi.params.Project); err != nil {
		return nil, err
	}
	if zone, err = oi.getZone(project); err != nil {
		return nil, err
	}
	if region, err = oi.getRegion(zone); err != nil {
		return nil, err
	}
	if err := validateReleaseTrack(oi.params.ReleaseTrack); err != nil {
		return nil, err
	}
	if oi.params.ReleaseTrack == Alpha || oi.params.ReleaseTrack == Beta {
		oi.imageLocation = region
	}

	tmpGcsPath, err := oi.buildTmpGcsPath(project, region)
	if err != nil {
		return nil, err
	}

	ovfGcsPath, shouldCleanup, err := oi.getOvfGcsPath(tmpGcsPath)
	if shouldCleanup {
		oi.gcsPathToClean = ovfGcsPath
	}
	if err != nil {
		return nil, err
	}

	ovfDescriptor, diskInfos, err := ovfutils.GetOVFDescriptorAndDiskPaths(
		oi.ovfDescriptorLoader, ovfGcsPath)
	if err != nil {
		return nil, err
	}
	oi.diskInfos = &diskInfos

	var osIDValue string
	if oi.params.OsID == "" {
		if osIDValue, err = ovfutils.GetOSId(ovfDescriptor); err != nil {
			return nil, err
		}
		oi.Logger.Log(
			fmt.Sprintf("Found valid osType in OVF descriptor, importing VM with `%v` as OS.",
				osIDValue))
	} else if err = daisyutils.ValidateOS(oi.params.OsID); err != nil {
		return nil, err
	} else {
		osIDValue = oi.params.OsID
	}

	translateWorkflowPath := "../image_import/" + daisyutils.GetTranslateWorkflowPath(osIDValue)
	machineTypeStr, err := oi.getMachineType(ovfDescriptor, project, zone)
	if err != nil {
		return nil, err
	}

	oi.Logger.Log(fmt.Sprintf("Will create instance of `%v` machine type.", machineTypeStr))

	varMap := oi.buildDaisyVars(translateWorkflowPath, diskInfos[0].FilePath, machineTypeStr, region)

	workflow, err := daisycommon.ParseWorkflow(oi.workflowPath, varMap, project,
		zone, oi.params.ScratchBucketGcsPath, oi.params.Oauth, oi.params.Timeout, oi.params.Ce,
		oi.params.GcsLogsDisabled, oi.params.CloudLogsDisabled, oi.params.StdoutLogsDisabled)

	if err != nil {
		return nil, fmt.Errorf("error parsing workflow %q: %v", ovfImportWorkflow, err)
	}
	workflow.ForceCleanupOnError = true
	return workflow, nil
}

func validateReleaseTrack(releaseTrack string) error {
	if releaseTrack != "" && releaseTrack != Alpha && releaseTrack != Beta && releaseTrack != GA {
		return fmt.Errorf("invalid value for release-track flag: %v", releaseTrack)
	}
	return nil
}

// Import runs OVF import
func (oi *OVFImporter) Import() error {
	oi.Logger.Log("Starting OVF import workflow.")
	w, err := oi.setUpImportWorkflow()
	if err != nil {
		oi.Logger.Log(err.Error())
		return err
	}

	if err := w.RunWithModifiers(oi.ctx, oi.modifyWorkflowPreValidate, oi.modifyWorkflowPostValidate); err != nil {
		oi.Logger.Log(err.Error())
		return err
	}
	oi.Logger.Log("OVF import workflow finished successfully.")
	return nil
}

// CleanUp performs clean up of any temporary resources or connections used for OVF import
func (oi *OVFImporter) CleanUp() {
	oi.Logger.Log("Cleaning up.")
	if oi.storageClient != nil {
		if oi.gcsPathToClean != "" {
			err := oi.storageClient.DeleteGcsPath(oi.gcsPathToClean)
			if err != nil {
				oi.Logger.Log(
					fmt.Sprintf("couldn't delete GCS path %v: %v", oi.gcsPathToClean, err.Error()))
			}
		}

		err := oi.storageClient.Close()
		if err != nil {
			oi.Logger.Log(fmt.Sprintf("couldn't close storage client: %v", err.Error()))
		}
	}
}
