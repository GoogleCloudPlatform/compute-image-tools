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
	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	daisyovfutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/daisy_utils"
	ovfexportparams "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/ovf_export_params"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	ovfutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
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
	ovfDescriptorLoader   ovfdomain.OvfDescriptorLoaderInterface
	bucketIteratorCreator domain.BucketIteratorCreatorInterface
	Logger                logging.LoggerInterface
	zoneValidator         domain.ZoneValidatorInterface
	workflowPath          string
	params                *ovfexportparams.OVFExportParams
	instancePath          string

	// BuildID is ID of Cloud Build in which this OVF export runs in
	BuildID string
	Zone    string
}

// NewOVFExporter creates an OVF exporter, including automatically populating dependencies,
// such as compute/storage clients.
func NewOVFExporter(params *ovfexportparams.OVFExportParams) (*OVFExporter, error) {
	ctx := context.Background()
	log.SetPrefix(logPrefix + " ")
	logger := logging.NewStdoutLogger(logPrefix)
	storageClient, err := storageutils.NewStorageClient(ctx, logger, "")
	if err != nil {
		return nil, err
	}
	computeClient, err := createComputeClient(&ctx, params)
	if err != nil {
		return nil, err
	}
	workingDirOVFImportWorkflow := toWorkingDir(getExportWorkflowPath(params), params)
	bic := &storageutils.BucketIteratorCreator{}

	ovfImporter := &OVFExporter{ctx: ctx, storageClient: storageClient, computeClient: computeClient,
		workflowPath: workingDirOVFImportWorkflow, BuildID: getBuildID(params),
		ovfDescriptorLoader: ovfutils.NewOvfDescriptorLoader(storageClient),
		mgce:                &computeutils.MetadataGCE{}, bucketIteratorCreator: bic, Logger: logger,
		zoneValidator: &computeutils.ZoneValidator{ComputeClient: computeClient}, params: params}
	return ovfImporter, nil
}

// creates a new Daisy Compute client
//TODO: consolidate with ovf_importer.createComputeClient
func createComputeClient(ctx *context.Context, params *ovfexportparams.OVFExportParams) (daisycompute.Client, error) {
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
func toWorkingDir(dir string, params *ovfexportparams.OVFExportParams) string {
	wd, err := filepath.Abs(filepath.Dir(params.CurrentExecutablePath))
	if err == nil {
		return path.Join(wd, dir)
	}
	return dir
}

func getExportWorkflowPath(params *ovfexportparams.OVFExportParams) string {
	if params.IsInstanceExport() {
		return instanceExportWorkflow
	}
	return machineImageExportWorkflow
}

func getBuildID(params *ovfexportparams.OVFExportParams) string {
	if params != nil && params.BuildID != "" {
		return params.BuildID
	}
	buildID := os.Getenv("BUILD_ID")
	if buildID == "" {
		buildID = pathutils.RandString(5)
	}
	return buildID
}

func (oe *OVFExporter) getZone(project string) (string, error) {
	if oe.params.Zone != "" {
		if err := oe.zoneValidator.ZoneValid(project, oe.params.Zone); err != nil {
			return "", err
		}
		return oe.params.Zone, nil
	}

	//TODO: determine zone based on VM zone if instance export

	if !oe.mgce.OnGCE() {
		return "", fmt.Errorf("zone cannot be determined because build is not running on GCE")
	}
	// determine zone based on the zone Cloud Build is running in
	zone, err := oe.mgce.Zone()
	if err != nil || zone == "" {
		return "", fmt.Errorf("can't infer zone: %v", err)
	}
	return zone, nil
}

//TODO: consolidate with OVFImporter.getRegion
func (oe *OVFExporter) getRegion(zone string) (string, error) {
	zoneSplits := strings.Split(zone, "-")
	if len(zoneSplits) < 2 {
		return "", fmt.Errorf("%v is not a valid zone", zone)
	}
	return strings.Join(zoneSplits[:len(zoneSplits)-1], "-"), nil
}

func (oe *OVFExporter) buildDaisyVars(region string) map[string]string {

	varMap := map[string]string{}
	if oe.params.IsInstanceExport() {
		// instance import specific vars
		oe.instancePath = fmt.Sprintf("projects/%s/zones/%s/instances/%s", *oe.params.Project, oe.params.Zone, strings.ToLower(oe.params.InstanceName))
		varMap["instance_name"] = oe.instancePath
	} else {
		// machine image import specific vars
		varMap["machine_image_name"] = strings.ToLower(oe.params.MachineImageName)
	}

	if oe.params.Subnet != "" {
		varMap["subnet"] = param.GetRegionalResourcePath(region, "subnetworks", oe.params.Subnet)
		// When subnet is set, we need to grant a value to network to avoid fallback to default
		if oe.params.Network == "" {
			varMap["network"] = ""
		}
	}
	if oe.params.Network != "" {
		varMap["network"] = param.GetGlobalResourcePath("networks", oe.params.Network)
	}
	return varMap
}

func isInstanceRunning(instance *compute.Instance) bool {
	return !(instance == nil || instance.Status == "STOPPED" || instance.Status == "STOPPING" ||
		instance.Status == "SUSPENDED" || instance.Status == "SUSPENDING")
}

func (oe *OVFExporter) modifyWorkflowPostValidate(w *daisy.Workflow) {
	w.LogWorkflowInfo("OVF import flags: %s", oe.params)
	w.LogWorkflowInfo("Cloud Build ID: %s", oe.BuildID)
	rl := &daisyutils.ResourceLabeler{
		BuildID:         oe.BuildID,
		BuildIDLabelKey: "gce-ovf-export-build-id",
		InstanceLabelKeyRetriever: func(instanceName string) string {
			return "gce-ovf-export-tmp"
		},
		DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
			return "gce-ovf-export-tmp"
		},
		ImageLabelKeyRetriever: func(imageName string) string {
			return "gce-ovf-export-tmp"
		}}
	rl.LabelResources(w)
	daisyutils.UpdateAllInstanceNoExternalIP(w, oe.params.NoExternalIP)
}

func (oe *OVFExporter) createScratchBucket(project string, region string) error {
	safeProjectName := strings.Replace(project, "google", "elgoog", -1)
	safeProjectName = strings.Replace(safeProjectName, ":", "-", -1)
	if strings.HasPrefix(safeProjectName, "goog") {
		safeProjectName = strings.Replace(safeProjectName, "goog", "ggoo", 1)
	}
	bucket := strings.ToLower(safeProjectName + "-ovf-export-bkt-" + region)
	it := oe.bucketIteratorCreator.CreateBucketIterator(oe.ctx, oe.storageClient, project)
	for itBucketAttrs, err := it.Next(); err != iterator.Done; itBucketAttrs, err = it.Next() {
		if err != nil {
			return err
		}
		if itBucketAttrs.Name == bucket {
			oe.params.ScratchBucketGcsPath = fmt.Sprintf("gs://%v/", bucket)
			return nil
		}
	}

	oe.Logger.Log(fmt.Sprintf("Creating scratch bucket `%v` in %v region", bucket, region))
	if err := oe.storageClient.CreateBucket(
		bucket, project,
		&storage.BucketAttrs{Name: bucket, Location: region}); err != nil {
		return err
	}
	oe.params.ScratchBucketGcsPath = fmt.Sprintf("gs://%v/", bucket)
	return nil
}

func (oe *OVFExporter) setUpExportWorkflow() (*daisy.Workflow, error) {
	if err := ovfexportparams.ValidateAndParseParams(oe.params, []string{GA, Beta, Alpha}); err != nil {
		return nil, err
	}
	var (
		project string
		zone    string
		region  string
		err     error
	)
	if project, err = param.GetProjectID(oe.mgce, *oe.params.Project); err != nil {
		return nil, err
	}
	*oe.params.Project = project
	if zone, err = oe.getZone(project); err != nil {
		return nil, err
	}
	oe.Zone = zone
	if region, err = oe.getRegion(zone); err != nil {
		return nil, err
	}
	if oe.params.ScratchBucketGcsPath == "" {
		if err := oe.createScratchBucket(project, region); err != nil {
			return nil, err
		}
	}
	varMap := oe.buildDaisyVars(region)

	w, err := daisycommon.ParseWorkflow(oe.workflowPath, varMap, project,
		zone, oe.params.ScratchBucketGcsPath, oe.params.Oauth, oe.params.Timeout, oe.params.Ce,
		oe.params.GcsLogsDisabled, oe.params.CloudLogsDisabled, oe.params.StdoutLogsDisabled)

	if err != nil {
		return nil, fmt.Errorf("error parsing workflow %q: %v", oe.workflowPath, err)
	}
	w.ForceCleanupOnError = true

	w.SetLogProcessHook(daisyutils.RemovePrivacyLogTag)

	instance, err := oe.computeClient.GetInstance(*oe.params.Project, oe.Zone, oe.params.InstanceName)
	if err != nil {
		return w, err
	}

	workflowGenerator := &daisyovfutils.OVFExportWorkflowGenerator{
		Instance:               instance,
		Project:                *oe.params.Project,
		Zone:                   oe.params.Zone,
		OvfGcsDirectoryPath:    oe.params.DestinationURI,
		ExportedDiskFileFormat: oe.params.DiskExportFormat,
		Network:                oe.params.Network,
		Subnet:                 oe.params.Subnet,
		InstancePath:           oe.instancePath,
		IsInstanceRunning:      isInstanceRunning(instance),
	}

	var previousStepName, nextStepName string
	if isInstanceRunning(instance) {
		previousStepName = "stop-instance"
		nextStepName = "start-instance"
		workflowGenerator.AddStopInstanceStep(w, previousStepName)
	}
	exportedDisksGCSPaths, err := workflowGenerator.AddDiskExportSteps(w, previousStepName, nextStepName)
	//TODO: use in OVF descriptor, remove print
	print(exportedDisksGCSPaths)

	if err != nil {
		return w, err
	}

	if isInstanceRunning(instance) {
		workflowGenerator.AddStartInstanceStep(w, nextStepName)
	}
	return w, nil
}

// Export runs OVF export
func (oe *OVFExporter) Export() (*daisy.Workflow, error) {
	oe.Logger.Log("Starting OVF export workflow.")
	w, err := oe.setUpExportWorkflow()
	if err != nil {
		oe.Logger.Log(err.Error())
		return w, err
	}

	if err := w.RunWithModifiers(oe.ctx, nil, oe.modifyWorkflowPostValidate); err != nil {
		oe.Logger.Log(err.Error())
		daisyutils.PostProcessDErrorForNetworkFlag("instance export", err, oe.params.Network, w)
		return w, err
	}
	oe.Logger.Log("OVF export workflow finished successfully.")
	return w, nil
}

// CleanUp performs clean up of any temporary resources or connections used for OVF export
func (oe *OVFExporter) CleanUp() {
	oe.Logger.Log("Cleaning up.")
	if oe.storageClient != nil {
		err := oe.storageClient.Close()
		if err != nil {
			oe.Logger.Log(fmt.Sprintf("couldn't close storage client: %v", err.Error()))
		}
	}
}
