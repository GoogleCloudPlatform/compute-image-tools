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

// GCE OVF import tool
package main

import (
	"cloud.google.com/go/storage"

	"context"
	"flag"
	"fmt"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/parse"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/daisy_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/gce_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	"github.com/vmware/govmomi/ovf"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ovfWorkflowDir      = "daisy_workflows/ovf_import/"
	ovfImportWorkflow   = ovfWorkflowDir + "import_ovf.wf.json"
	instanceNameFlagKey = "instance-names"
	clientIDFlagKey     = "client-id"
	ovfGcsPathFlagKey   = "ovf-gcs-path"
)

var (
	instanceNames               = flag.String(instanceNameFlagKey, "", "VM Instance names to be created, separated by commas.")
	clientID                    = flag.String(clientIDFlagKey, "", "Identifies the client of the importer, e.g. `gcloud` or `pantheon`")
	ovfOvaGcsPath               = flag.String(ovfGcsPathFlagKey, "", " Google Cloud Storage URI of the OVF or OVA file to import. For example: gs://my-bucket/my-vm.ovf.")
	noGuestEnvironment          = flag.Bool("no-guest-environment", false, "Google Guest Environment will not be installed on the image.")
	canIPForward                = flag.Bool("can-ip-forward", false, "If provided, allows the instances to send and receive packets with non-matching destination or source IP addresses.")
	deletionProtection          = flag.Bool("deletion-protection", false, "Enables deletion protection for the instance.")
	description                 = flag.String("description", "", "Specifies a textual description of the instances.")
	labels                      = flag.String("labels", "", "List of label KEY=VALUE pairs to add. Keys must start with a lowercase character and contain only hyphens (-), underscores (_), lowercase characters, and numbers. Values must contain only hyphens (-), underscores (_), lowercase characters, and numbers.")
	machineType                 = flag.String("machine-type", "", "Specifies the machine type used for the instances. To get a list of available machine types, run 'gcloud compute machine-types list'. If unspecified, the default type is n1-standard-1.")
	network                     = flag.String("network", "", "Name of the network in your project to use for the image import. The network must have access to Google Cloud Storage. If not specified, the network named default is used. If -subnet is also specified subnet must be a subnetwork of network specified by -network.")
	networkInterface            = flag.String("network-interface", "", "Adds a network interface to the instance. Mutually exclusive with any of these flags: --network, --network-tier, --subnet, --private-network-ip. ")
	networkTier                 = flag.String("network-tier", "", "Specifies the network tier that will be used to configure the instance. NETWORK_TIER must be one of: PREMIUM, STANDARD. The default value is PREMIUM.")
	subnet                      = flag.String("subnet", "", "Name of the subnetwork in your project to use for the image import. If	the network resource is in legacy mode, do not provide this property. If the network is in auto subnet mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this field should be specified. Zone should be specified if this field is specified.")
	privateNetworkIP            = flag.String("private-network-ip", "", "Specifies the RFC1918 IP to assign to the instance. The IP should be in the subnet or legacy network IP range.")
	noExternalIP                = flag.Bool("no-external-ip", false, "Specifies that VPC into which instances is being imported doesn't allow external IPs.")
	noRestartOnFailure          = flag.Bool("no-restart-on-failure", false, "the instance will not be restarted if it’s terminated by Compute Engine. This does not affect terminations performed by the user.")
	osID                        = flag.String("os", "", "Specifies the OS of the image being imported. OS must be one of: centos-6, centos-7, debian-8, debian-9, rhel-6, rhel-6-byol, rhel-7, rhel-7-byol, ubuntu-1404, ubuntu-1604, windows-10-byol, windows-2008r2, windows-2008r2-byol, windows-2012, windows-2012-byol, windows-2012r2, windows-2012r2-byol, windows-2016, windows-2016-byol, windows-7-byol.")
	shieldedIntegrityMonitoring = flag.Bool("shielded-integrity-monitoring", false, "Enables monitoring and attestation of the boot integrity of the instance. The attestation is performed against the integrity policy baseline. This baseline is initially derived from the implicitly trusted boot image when the instance is created. This baseline can be updated by using --shielded-vm-learn-integrity-policy.")
	shieldedSecureBoot          = flag.Bool("shielded-secure-boot", false, "The instance will boot with secure boot enabled.")
	shieldedVtpm                = flag.Bool("shielded-vtpm", false, "The instance will boot with the TPM (Trusted Platform Module) enabled. A TPM is a hardware module that can be used for different security operations such as remote attestation, encryption and sealing of keys.")
	tags                        = flag.String("tags", "", "Specifies a list of tags to apply to the instance. These tags allow network firewall rules and routes to be applied to specified VM instances. See `gcloud compute firewall-rules create` for more details.")
	zone                        = flag.String("zone", "", "Zone of the image to import. The zone in which to do the work of importing the image. Overrides the default compute/zone property value for this command invocation")
	bootDiskKmskey              = flag.String("boot-disk-kms-key", "", "The Cloud KMS (Key Management Service) cryptokey that will be used to protect the disk. The arguments in this group can be used to specify the attributes of this resource. ID of the key or fully qualified identifier for the key. This flag must be specified if any of the other arguments in this group are specified.")
	bootDiskKmsKeyring          = flag.String("boot-disk-kms-keyring", "", "The KMS keyring of the key.")
	bootDiskKmsLocation         = flag.String("boot-disk-kms-location", "", "The Cloud location for the key.")
	bootDiskKmsProject          = flag.String("boot-disk-kms-project", "", "The Cloud project for the key.")
	timeout                     = flag.String("timeout", "", "Maximum time a build can last before it is failed as TIMEOUT. For example, specifying 2h will fail the process after 2 hours. See `gcloud topic datetimes` for information on duration formats")
	project                     = flag.String("project", "", "project to run in, overrides what is set in workflow")
	scratchBucketGcsPath        = flag.String("scratch-bucket-gcs-path", "", "GCS scratch bucket to use, overrides what is set in workflow")
	oauth                       = flag.String("oauth", "", "path to oauth json file, overrides what is set in workflow")
	ce                          = flag.String("compute-endpoint-override", "", "API endpoint to override default")
	gcsLogsDisabled             = flag.Bool("disable-gcs-logging", false, "do not stream logs to GCS")
	cloudLogsDisabled           = flag.Bool("disable-cloud-logging", false, "do not stream logs to Cloud Logging")
	stdoutLogsDisabled          = flag.Bool("disable-stdout-logging", false, "do not display individual workflow logs on stdout")

	region                *string
	userLabels            map[string]string
	currentExecutablePath *string
)

func init() {
	currentExecutablePathStr := string(os.Args[0])
	currentExecutablePath = &currentExecutablePathStr
}

func validateAndParseFlags() error {
	flag.Parse()

	if err := validationutils.ValidateStringFlagNotEmpty(*instanceNames, instanceNameFlagKey); err != nil {
		return err
	}

	instanceNameSplits := strings.Split(*instanceNames, ",")
	if len(instanceNameSplits) > 1 {
		return fmt.Errorf("OVF import doesn't support multi instance import at this time")
	}

	if err := validationutils.ValidateStringFlagNotEmpty(*ovfOvaGcsPath, ovfGcsPathFlagKey); err != nil {
		return err
	}

	if err := validationutils.ValidateStringFlagNotEmpty(*clientID, clientIDFlagKey); err != nil {
		return err
	}

	if _, _, err := storageutils.SplitGCSPath(*ovfOvaGcsPath); err != nil {
		return fmt.Errorf("%v should be a path to OVF or OVA package in GCS", ovfGcsPathFlagKey)
	}

	//TODO: extract OS info from OVF making `-os` flag optional
	if err := daisyutils.ValidateOS(*osID); err != nil {
		return err
	}

	if *labels != "" {
		var err error
		userLabels, err = parseutils.ParseKeyValues(*labels)
		if err != nil {
			return err
		}
	}

	if *networkInterface != "" && (*privateNetworkIP != "" || *network != "" || *subnet != "" || *networkTier != "") {
		return fmt.Errorf("-network-interface is mutually exclusive with any of these flags: --network, --network-tier, --subnet, --private-network-ip")
	}

	return nil
}

func buildDaisyVars(translateWorkflowPath string, bootDiskGcsPath string, machineType string) map[string]string {
	varMap := map[string]string{}

	varMap["instance_name"] = strings.ToLower(*instanceNames)
	if translateWorkflowPath != "" {
		varMap["translate_workflow"] = translateWorkflowPath
		varMap["install_gce_packages"] = strconv.FormatBool(!*noGuestEnvironment)
	}
	if bootDiskGcsPath != "" {
		varMap["boot_disk_file"] = bootDiskGcsPath
	}
	if *network != "" {
		varMap["network"] = fmt.Sprintf("global/networks/%v", *network)
	}
	if *subnet != "" {
		varMap["subnet"] = fmt.Sprintf("regions/%v/subnetworks/%v", *region, *subnet)
	}
	if machineType != "" {
		varMap["machine_type"] = machineType
	}
	if *description != "" {
		varMap["description"] = *description
	}
	if *privateNetworkIP != "" {
		varMap["private_network_ip"] = *privateNetworkIP
	}

	if *networkTier != "" {
		varMap["network_tier"] = *networkTier
	}
	//TODO: networkInterface
	return varMap
}

// Daisy workflow files don't support boolean variables so we have to do it here.
func updateInstanceWithBooleanFlagValues(w *daisy.Workflow) {
	instance := (*w.Steps["create-instance"].CreateInstances)[0]
	instance.CanIpForward = *canIPForward
	instance.DeletionProtection = *deletionProtection
	if instance.Scheduling == nil {
		instance.Scheduling = &compute.Scheduling{}
	}
	if *noRestartOnFailure {
		vFalse := false
		instance.Scheduling.AutomaticRestart = &vFalse
	}
}

func toWorkingDir(dir string) string {
	wd, err := filepath.Abs(filepath.Dir(*currentExecutablePath))
	if err == nil {
		return path.Join(wd, dir)
	}
	return dir
}

// creates a new Daisy Compute client
func createComputeClient(ctx *context.Context) (daisycompute.Client, error) {
	computeOptions := []option.ClientOption{option.WithCredentialsFile(*oauth)}
	if *ce != "" {
		computeOptions = append(computeOptions, option.WithEndpoint(*ce))
	}

	computeClient, err := daisycompute.NewClient(*ctx, computeOptions...)
	if err != nil {
		return nil, err
	}
	return computeClient, nil
}

// OVFImporter is responsible for importing OVF into GCE
type OVFImporter struct {
	ctx                   context.Context
	storageClient         commondomain.StorageClientInterface
	computeClient         daisycompute.Client
	tarGcsExtractor       commondomain.TarGcsExtractorInterface
	mgce                  commondomain.MetadataGCEInterface
	ovfDescriptorLoader   domain.OvfDescriptorLoaderInterface
	bucketIteratorCreator commondomain.BucketIteratorCreatorInterface
	gcsPathToClean        string
	workflowPath          string
	buildID               string
	diskInfos             *[]ovfutils.DiskInfo
}

// NewOVFImporter creates an OVF importer, including automatically poulating dependencies,
// such as compute/storage clients.
func NewOVFImporter() (*OVFImporter, error) {
	ctx := context.Background()
	sc, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	storageClient, err := storageutils.NewStorageClient(ctx, sc)
	if err != nil {
		return nil, err
	}
	computeClient, err := createComputeClient(&ctx)
	if err != nil {
		return nil, err
	}
	tarGcsExtractor := storageutils.NewTarGcsExtractor(ctx, storageClient)
	buildID := os.Getenv("BUILD_ID")
	if buildID == "" {
		buildID = utils.RandString(5)
	}
	workingDirOVFImportWorkflow := toWorkingDir(ovfImportWorkflow)
	bic := &storageutils.BucketIteratorCreator{}

	ovfImporter := &OVFImporter{ctx: ctx, storageClient: storageClient, computeClient: computeClient,
		tarGcsExtractor: tarGcsExtractor, workflowPath: workingDirOVFImportWorkflow, buildID: buildID,
		ovfDescriptorLoader: ovfutils.NewOvfDescriptorLoader(storageClient),
		mgce:                &computeutils.MetadataGCE{}, bucketIteratorCreator: bic}
	return ovfImporter, nil
}

func (oi *OVFImporter) populateMissingParameters() error {
	aProject := *project
	if aProject == "" {
		if !oi.mgce.OnGCE() {
			return fmt.Errorf("project cannot be determined because build is not running on GCE")
		}
		var err error
		aProject, err = oi.mgce.ProjectID()
		if err != nil || aProject == "" {
			return fmt.Errorf("project cannot be determined %v", err)
		}
		project = &aProject
	}

	if *zone == "" {
		if !oi.mgce.OnGCE() {
			return fmt.Errorf("zone cannot be determined because build is not running on GCE")
		}
		// determine zone based on the zone Cloud Build is running in
		aZone, err := oi.mgce.Zone()
		if err != nil || aZone == "" {
			return fmt.Errorf("can't infer zone: %v", err)
		}
		zone = &aZone
	}

	if region == nil || *region == "" {
		zoneStrs := strings.Split(*zone, "-")
		if len(zoneStrs) < 2 {
			return fmt.Errorf("%v is not a valid zone", *zone)
		}

		aRegion := strings.Join(zoneStrs[:len(zoneStrs)-1], "-")
		region = &aRegion
	}
	return nil
}

// Returns OVF GCS bucket and object path (director). If ovaOvaGcsPath is pointing to an OVA file,
// it extracts it to a temporary GCS folder and returns it's path.
func (oi *OVFImporter) getOvfGcsPath(tmpGcsPath string) (string, bool, error) {
	ovfOvaGcsPathLowered := strings.ToLower(*ovfOvaGcsPath)
	var ovfGcsPath string
	var shouldCleanUp bool
	var err error
	if strings.HasSuffix(ovfOvaGcsPathLowered, ".ova") {
		ovfGcsPath = pathutils.JoinURL(tmpGcsPath, "ovf")
		log.Printf("Extracting %v OVA archive to %v", *ovfOvaGcsPath, ovfGcsPath)
		err = oi.tarGcsExtractor.ExtractTarToGcs(*ovfOvaGcsPath, ovfGcsPath)
		shouldCleanUp = true
	} else if strings.HasSuffix(ovfOvaGcsPathLowered, ".ovf") {
		// ovfOvaGcsPath is pointing to OVF descriptor, no need to unpack, just extract directory path.
		ovfGcsPath = (*ovfOvaGcsPath)[0 : strings.LastIndex(*ovfOvaGcsPath, "/")+1]
	} else {
		ovfGcsPath = *ovfOvaGcsPath
	}

	// assume ovfOvaGcsPath is a GCS folder for the whole OVF package
	return pathutils.ToDirectoryURL(ovfGcsPath), shouldCleanUp, err
}

func (oi *OVFImporter) createScratchBucketBucketIfDoesntExist() error {
	bucket := strings.ToLower(
		strings.Replace(*project, ":", "-", -1) + "-ovf-import-bkt-" + *region)
	*scratchBucketGcsPath = fmt.Sprintf("gs://%v/", bucket)
	it := oi.bucketIteratorCreator.CreateBucketIterator(oi.ctx, oi.storageClient, *project)
	for itBucketAttrs, err := it.Next(); err != iterator.Done; itBucketAttrs, err = it.Next() {
		if err != nil {
			return err
		}
		if itBucketAttrs.Name == bucket {
			return nil
		}
	}

	log.Printf("Creating scratch bucket `%v` in %v region", bucket, *region)
	if err := oi.storageClient.CreateBucket(
		bucket, *project,
		&storage.BucketAttrs{Name: bucket, Location: *region, StorageClass: "REGIONAL"}); err != nil {
		return err
	}
	return nil
}

func (oi *OVFImporter) buildTmpGcsPath() (string, error) {
	if *scratchBucketGcsPath == "" {
		if err := oi.createScratchBucketBucketIfDoesntExist(); err != nil {
			return "", err
		}
	}
	return pathutils.JoinURL(*scratchBucketGcsPath, fmt.Sprintf("ovf-import-%v", oi.buildID)), nil
}

func (oi *OVFImporter) modifyWorkflowPostValidate(w *daisy.Workflow) {
	rl := &daisyutils.ResourceLabeler{
		BuildID: oi.buildID, UserLabels: userLabels, BuildIDLabelKey: "gce-ovf-import-build-id",
		InstanceLabelKeyRetriever: func(instance *daisy.Instance) string {
			if strings.ToLower(*instanceNames) == instance.Name {
				return "gce-ovf-import"
			}
			return "gce-ovf-import-tmp"
		},
		DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
			return "gce-ovf-import-tmp"
		},
		ImageLabelKeyRetriever: func(image *daisy.Image) string {
			return "gce-ovf-import-tmp"
		}}
	rl.LabelResources(w)
	daisyutils.UpdateAllInstanceNoExternalIP(w, *noExternalIP)
}

func (oi *OVFImporter) modifyWorkflowPreValidate(w *daisy.Workflow) {
	daisyovfutils.AddDiskImportSteps(w, (*oi.diskInfos)[1:])
	updateInstanceWithBooleanFlagValues(w)
}

func (oi *OVFImporter) getMachineType(ovfDescriptor *ovf.Envelope) (string, error) {
	machineTypeProvider := ovfgceutils.MachineTypeProvider{
		OvfDescriptor: ovfDescriptor,
		MachineType:   *machineType,
		ComputeClient: oi.computeClient,
		Project:       *project,
		Zone:          *zone,
	}
	return machineTypeProvider.GetMachineType()
}

func (oi *OVFImporter) setUpImportWorkflow() (*daisy.Workflow, error) {
	if err := validateAndParseFlags(); err != nil {
		return nil, err
	}
	if err := oi.populateMissingParameters(); err != nil {
		return nil, err
	}
	tmpGcsPath, err := oi.buildTmpGcsPath()
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
	translateWorkflowPath := "../image_import/" + daisyutils.GetTranslateWorkflowPath(osID)
	machineTypeStr, err := oi.getMachineType(ovfDescriptor)
	if err != nil {
		return nil, err
	}
	varMap := buildDaisyVars(translateWorkflowPath, diskInfos[0].FilePath, machineTypeStr)

	workflow, err := daisyutils.ParseWorkflow(oi.mgce, oi.workflowPath, varMap, *project,
		*zone, *scratchBucketGcsPath, *oauth, *timeout, *ce, *gcsLogsDisabled, *cloudLogsDisabled,
		*stdoutLogsDisabled)

	if err != nil {
		return nil, fmt.Errorf("error parsing workflow %q: %v", ovfImportWorkflow, err)
	}
	return workflow, nil
}

// Import runs OVF import
func (oi *OVFImporter) Import() error {
	w, err := oi.setUpImportWorkflow()
	if err != nil {
		return err
	}

	if err := w.RunWithModifiers(oi.ctx, oi.modifyWorkflowPreValidate, oi.modifyWorkflowPostValidate); err != nil {
		return fmt.Errorf("%s: %v", w.Name, err)
	}
	return nil
}

// CleanUp performs clean up of any temporary resources or connections used for OVF import
func (oi *OVFImporter) CleanUp() {
	if oi.storageClient != nil {
		if oi.gcsPathToClean != "" {
			err := oi.storageClient.DeleteGcsPath(oi.gcsPathToClean)
			if err != nil {
				log.Printf("couldn't delete GCS path %v: %v", oi.gcsPathToClean, err.Error())
			}
		}

		err := oi.storageClient.Close()
		if err != nil {
			log.Printf("couldn't close storage client: %v", err.Error())
		}
	}
}

func main() {
	ovfImporter, err := NewOVFImporter()

	if err != nil {
		log.Println(err.Error())
		ovfImporter.CleanUp()
		os.Exit(1)
	}
	err = ovfImporter.Import()
	if err != nil {
		log.Println(err.Error())
		ovfImporter.CleanUp()
		os.Exit(1)
	}
	ovfImporter.CleanUp()
}
