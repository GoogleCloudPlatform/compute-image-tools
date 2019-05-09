//  Copyright 2018 Google Inc. All Rights Reserved.
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

// GCE VM image import tool
package main

import (
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/parse"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/util"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/option"
)

const (
	workflowDir                = "daisy_workflows/image_import/"
	importWorkflow             = workflowDir + "import_image.wf.json"
	importFromImageWorkflow    = workflowDir + "import_from_image.wf.json"
	importAndTranslateWorkflow = workflowDir + "import_and_translate.wf.json"
	imageNameFlagKey           = "image_name"
	clientIDFlagKey            = "client_id"
)

var (
	imageName            = flag.String(imageNameFlagKey, "", "Image name to be imported.")
	clientID             = flag.String(clientIDFlagKey, "", "Identifies the client of the importer, e.g. `gcloud` or `pantheon`")
	dataDisk             = flag.Bool("data_disk", false, "Specifies that the disk has no bootable OS installed on it.	Imports the disk without making it bootable or installing Google tools on it. ")
	osID                 = flag.String("os", "", "Specifies the OS of the image being imported. OS must be one of: centos-6, centos-7, debian-8, debian-9, rhel-6, rhel-6-byol, rhel-7, rhel-7-byol, ubuntu-1404, ubuntu-1604, windows-10-byol, windows-2008r2, windows-2008r2-byol, windows-2012, windows-2012-byol, windows-2012r2, windows-2012r2-byol, windows-2016, windows-2016-byol, windows-7-byol.")
	customTranWorkflow   = flag.String("custom_translate_workflow", "", "Specifies the custom workflow used to do translation")
	sourceFile           = flag.String("source_file", "", "Google Cloud Storage URI of the virtual disk file	to import. For example: gs://my-bucket/my-image.vmdk")
	sourceImage          = flag.String("source_image", "", "Compute Engine image from which to import")
	noGuestEnvironment   = flag.Bool("no_guest_environment", false, "Google Guest Environment will not be installed on the image.")
	family               = flag.String("family", "", "Family to set for the translated image")
	description          = flag.String("description", "", "Description to set for the translated image")
	network              = flag.String("network", "", "Name of the network in your project to use for the image import. The network must have access to Google Cloud Storage. If not specified, the network named default is used.")
	subnet               = flag.String("subnet", "", "Name of the subnetwork in your project to use for the image import. If	the network resource is in legacy mode, do not provide this property. If the network is in auto subnet mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this field should be specified. Zone should be specified if this field is specified.")
	zone                 = flag.String("zone", "", "Zone of the image to import. The zone in which to do the work of importing the image. Overrides the default compute/zone property value for this command invocation.")
	timeout              = flag.String("timeout", "", "Maximum time a build can last before it is failed as TIMEOUT. For example, specifying 2h will fail the process after 2 hours. See $ gcloud topic datetimes for information on duration formats.")
	project              = flag.String("project", "", "project to run in, overrides what is set in workflow")
	scratchBucketGcsPath = flag.String("scratch_bucket_gcs_path", "", "GCS scratch bucket to use, overrides what is set in workflow")
	oauth                = flag.String("oauth", "", "path to oauth json file, overrides what is set in workflow")
	ce                   = flag.String("compute_endpoint_override", "", "API endpoint to override default")
	gcsLogsDisabled      = flag.Bool("disable_gcs_logging", false, "do not stream logs to GCS")
	cloudLogsDisabled    = flag.Bool("disable_cloud_logging", false, "do not stream logs to Cloud Logging")
	stdoutLogsDisabled   = flag.Bool("disable_stdout_logging", false, "do not display individual workflow logs on stdout")
	kmsKey               = flag.String("kms_key", "", "ID of the key or fully qualified identifier for the key. This flag must be specified if any of the other arguments below are specified.")
	kmsKeyring           = flag.String("kms_keyring", "", "The KMS keyring of the key.")
	kmsLocation          = flag.String("kms_location", "", "The Cloud location for the key.")
	kmsProject           = flag.String("kms_project", "", "The Cloud project for the key")
	noExternalIP         = flag.Bool("no_external_ip", false, "VPC doesn't allow external IPs")
	labels               = flag.String("labels", "", "List of label KEY=VALUE pairs to add. Keys must start with a lowercase character and contain only hyphens (-), underscores (_), lowercase characters, and numbers. Values must contain only hyphens (-), underscores (_), lowercase characters, and numbers.")

	region  *string
	buildID = os.Getenv("BUILD_ID")

	userLabels            map[string]string
	currentExecutablePath *string

	sourceBucketName string
	sourceObjectName string
)

func init() {
	currentExecutablePathStr := string(os.Args[0])
	currentExecutablePath = &currentExecutablePathStr
}

func validateAndParseFlags() error {
	flag.Parse()

	if err := validationutils.ValidateStringFlagNotEmpty(*imageName, imageNameFlagKey); err != nil {
		return err
	}
	if err := validationutils.ValidateStringFlagNotEmpty(*clientID, clientIDFlagKey); err != nil {
		return err
	}

	if !*dataDisk && *osID == "" && *customTranWorkflow == "" {
		return fmt.Errorf("-data_disk, or -os, or -custom_translate_workflow has to be specified")
	}

	if *dataDisk && (*osID != "" || *customTranWorkflow != "") {
		return fmt.Errorf("when -data_disk is specified, -os and -custom_translate_workflow should be empty")
	}

	if *osID != "" && *customTranWorkflow != "" {
		return fmt.Errorf("-os and -custom_translate_workflow can't be both specified")
	}

	if *sourceFile == "" && *sourceImage == "" {
		return fmt.Errorf("-source_file or -source_image has to be specified")
	}

	if *sourceFile != "" && *sourceImage != "" {
		return fmt.Errorf("either -source_file or -source_image has to be specified, but not both %v %v", *sourceFile, *sourceImage)
	}

	if *osID != "" {
		if err := daisyutils.ValidateOS(*osID); err != nil {
			return err
		}
	}

	if *sourceFile != "" {
		var err error
		sourceBucketName, sourceObjectName, err = storageutils.SplitGCSPath(*sourceFile)
		if err != nil {
			return err
		}
	}

	if *labels != "" {
		var err error
		userLabels, err = parseutils.ParseKeyValues(*labels)
		if err != nil {
			return err
		}
	}

	return nil
}

// Validate source file is not a compression file by checking file header.
func validateSourceFile(storageClient commondomain.StorageClientInterface) error {
	if sourceFile == nil {
		return nil
	}

	rc, err := storageClient.GetObjectReader(sourceBucketName, sourceObjectName)
	if err != nil {
		return fmt.Errorf("readFile: unable to open file from bucket %q, file %q: %v", sourceBucketName, sourceObjectName, err)
	}
	defer rc.Close()

	// Detect whether it's a compressed file by extracting compressed file header
	if _, err = gzip.NewReader(rc); err == nil {
		return fmt.Errorf("cannot import an image from a compressed file. Please provide a path to an uncompressed image file. If the compressed file is an image exported from Google Compute Engine, please use 'images create' instead")
	}

	return nil
}

// Returns main workflow and translate workflow paths (if any)
func getWorkflowPaths() (string, string) {
	if *sourceImage != "" {
		return toWorkingDir(importFromImageWorkflow), getTranslateWorkflowPath()
	}
	if *dataDisk {
		return toWorkingDir(importWorkflow), ""
	}
	return toWorkingDir(importAndTranslateWorkflow), getTranslateWorkflowPath()
}

func getTranslateWorkflowPath() string {
	if *customTranWorkflow != "" {
		return *customTranWorkflow
	}
	return daisyutils.GetTranslateWorkflowPath(osID)
}

func toWorkingDir(dir string) string {
	wd, err := filepath.Abs(filepath.Dir(*currentExecutablePath))
	if err == nil {
		return path.Join(wd, dir)
	}
	return dir
}

func populateMissingParameters(mgce commondomain.MetadataGCEInterface,
	scratchBucketCreator domain.ScratchBucketCreatorInterface,
	zoneRetriever domain.ZoneRetrieverInterface, storageClient commondomain.StorageClientInterface) error {

	if err := populateProjectIfMissing(mgce); err != nil {
		return err
	}

	scratchBucketRegion := ""
	if *scratchBucketGcsPath == "" {
		scratchBucketName, sbr, err := scratchBucketCreator.CreateScratchBucket(*sourceFile, *project)
		scratchBucketRegion = sbr
		if err != nil {
			return err
		}

		newScratchBucketGcsPath := fmt.Sprintf("gs://%v/", scratchBucketName)
		scratchBucketGcsPath = &newScratchBucketGcsPath
	} else {
		scratchBucketName, err := storageutils.GetBucketNameFromGCSPath(*scratchBucketGcsPath)
		if err != nil {
			return fmt.Errorf("invalid scratch bucket GCS path %v", *scratchBucketGcsPath)
		}
		scratchBucketAttrs, err := storageClient.GetBucketAttrs(scratchBucketName)
		if err == nil {
			scratchBucketRegion = scratchBucketAttrs.Location
		}
	}

	if *zone == "" {
		if aZone, err := zoneRetriever.GetZone(scratchBucketRegion, *project); err == nil {
			zone = &aZone
		} else {
			return err
		}
	}

	if err := populateRegion(); err != nil {
		return err
	}
	return nil
}

func populateProjectIfMissing(mgce commondomain.MetadataGCEInterface) error {
	aProject := *project
	if aProject == "" {
		if !mgce.OnGCE() {
			return fmt.Errorf("project cannot be determined because build is not running on GCE")
		}
		var err error
		aProject, err = mgce.ProjectID()
		if err != nil || aProject == "" {
			return fmt.Errorf("project cannot be determined %v", err)
		}
		project = &aProject
	}
	return nil
}

func populateRegion() error {
	aRegion, err := getRegion()
	if err != nil {
		return err
	}
	region = &aRegion
	return nil
}

func getRegion() (string, error) {
	if *zone == "" {
		return "", fmt.Errorf("zone is empty. Can't determine region")
	}
	zoneStrs := strings.Split(*zone, "-")
	if len(zoneStrs) < 2 {
		return "", fmt.Errorf("%v is not a valid zone", *zone)
	}
	return strings.Join(zoneStrs[:len(zoneStrs)-1], "-"), nil
}

func buildDaisyVars(translateWorkflowPath string) map[string]string {
	varMap := map[string]string{}

	varMap["image_name"] = strings.ToLower(*imageName)
	if translateWorkflowPath != "" {
		varMap["translate_workflow"] = translateWorkflowPath
		varMap["install_gce_packages"] = strconv.FormatBool(!*noGuestEnvironment)
	}
	if *sourceFile != "" {
		varMap["source_disk_file"] = *sourceFile
	}
	if *sourceImage != "" {
		varMap["source_image"] = fmt.Sprintf("global/images/%v", *sourceImage)
	}
	varMap["family"] = *family
	varMap["description"] = *description
	if *network != "" {
		varMap["import_network"] = fmt.Sprintf("global/networks/%v", *network)
	}
	if *subnet != "" {
		varMap["import_subnet"] = fmt.Sprintf("regions/%v/subnetworks/%v", *region, *subnet)
	}
	return varMap
}

func createStorageClient(ctx context.Context) *storage.Client {
	storageOptions := []option.ClientOption{option.WithCredentialsFile(*oauth)}
	storageClient, err := storage.NewClient(ctx, storageOptions...)
	if err != nil {
		log.Fatalf("error creating storage client %v", err)
	}
	return storageClient
}

// creates a new Daisy Compute client
func createComputeClient(ctx *context.Context) daisycompute.Client {
	computeOptions := []option.ClientOption{option.WithCredentialsFile(*oauth)}
	if *ce != "" {
		computeOptions = append(computeOptions, option.WithEndpoint(*ce))
	}

	computeClient, err := daisycompute.NewClient(*ctx, computeOptions...)
	if err != nil {
		log.Fatalf("compute client: %v", err)
	}
	return computeClient
}

func runImport(ctx context.Context, metadataGCEHolder computeutils.MetadataGCE,
	scratchBucketCreator *gcevmimageimportutil.ScratchBucketCreator,
	zoneRetriever *gcevmimageimportutil.ZoneRetriever, storageClient commondomain.StorageClientInterface) error {

	err := populateMissingParameters(&metadataGCEHolder, scratchBucketCreator, zoneRetriever, storageClient)
	if err != nil {
		return err
	}
	importWorkflowPath, translateWorkflowPath := getWorkflowPaths()
	varMap := buildDaisyVars(translateWorkflowPath)
	workflow, err := daisyutils.ParseWorkflow(&computeutils.MetadataGCE{}, importWorkflowPath, varMap,
		*project, *zone, *scratchBucketGcsPath, *oauth, *timeout, *ce, *gcsLogsDisabled,
		*cloudLogsDisabled, *stdoutLogsDisabled)
	if err != nil {
		return err
	}

	workflowModifier := func(w *daisy.Workflow) {
		rl := &daisyutils.ResourceLabeler{
			BuildID: buildID, UserLabels: userLabels, BuildIDLabelKey: "gce-image-import-build-id",
			InstanceLabelKeyRetriever: func(instance *daisy.Instance) string {
				return "gce-image-import-tmp"
			},
			DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
				return "gce-image-import-tmp"
			},
			ImageLabelKeyRetriever: func(image *daisy.Image) string {
				imageTypeLabel := "gce-image-import"
				if strings.Contains(image.Image.Name, "untranslated") {
					imageTypeLabel = "gce-image-import-tmp"
				}
				return imageTypeLabel
			}}
		rl.LabelResources(w)
		daisyutils.UpdateAllInstanceNoExternalIP(w, *noExternalIP)
	}

	return workflow.RunWithModifiers(ctx, nil, workflowModifier)
}

func main() {
	if err := validateAndParseFlags(); err != nil {
		log.Fatalf(err.Error())
	}

	ctx := context.Background()
	metadataGCEHolder := computeutils.MetadataGCE{}
	storageClient, err := storageutils.NewStorageClient(
		ctx, createStorageClient(ctx), logging.NewLogger("[image-import]"))
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer storageClient.Close()

	err = validateSourceFile(storageClient)
	if err != nil {
		log.Fatalf(err.Error())
	}

	scratchBucketCreator := gcevmimageimportutil.NewScratchBucketCreator(ctx, storageClient)
	zoneRetriever, err := gcevmimageimportutil.NewZoneRetriever(
		&metadataGCEHolder, createComputeClient(&ctx))

	if err != nil {
		log.Fatalf(err.Error())
	}

	if err := runImport(ctx, metadataGCEHolder, scratchBucketCreator, zoneRetriever, storageClient); err != nil {
		log.Fatalf(err.Error())
	}
}
