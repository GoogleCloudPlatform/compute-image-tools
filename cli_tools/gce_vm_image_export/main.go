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

// GCE VM image export tool
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/arg"
	"log"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/option"
)

const (
	workflowDir                = "daisy_workflows/export/"
	exportWorkflow             = workflowDir + "image_export.wf.json"
	exportAndConvertWorkflow   = workflowDir + "image_export_ext.wf.json"
	clientIDFlagKey            = "client_id"
	destinationUriFlagKey      = "destination_uri"
)

var (
	clientID             = flag.String(clientIDFlagKey, "", "Identifies the client of the importer, e.g. `gcloud` or `pantheon`.")
	destinationUri       = flag.String(destinationUriFlagKey, "", "The Google Cloud Storage URI destination for the exported virtual disk file. For example: gs://my-bucket/my-exported-image.vmdk.")
	image                = flag.String("image", "", "The name of the disk image to export")
	imageFamily          = flag.String("image_family", "", "The family of the disk image to be exported. When a family is used instead of an image, the latest non-deprecated image associated with that family is used.")
	format               = flag.String("format", "", "Specify the format to export to, such as vmdk, vhdx, vpc, or qcow2.")
	project              = flag.String("project", "", "Project to run in, overrides what is set in workflow.")
	network              = flag.String("network", "", "Name of the network in your project to use for the image import. The network must have access to Google Cloud Storage. If not specified, the network named default is used.")
	subnet               = flag.String("subnet", "", "Name of the subnetwork in your project to use for the image import. If	the network resource is in legacy mode, do not provide this property. If the network is in auto subnet mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this field should be specified. Zone should be specified if this field is specified.")
	zone                 = flag.String("zone", "", "Zone of the image to import. The zone in which to do the work of importing the image. Overrides the default compute/zone property value for this command invocation.")
	timeout              = flag.String("timeout", "", "Maximum time a build can last before it is failed as TIMEOUT. For example, specifying 2h will fail the process after 2 hours. See $ gcloud topic datetimes for information on duration formats.")
	scratchBucketGcsPath = flag.String("scratch_bucket_gcs_path", "", "GCS scratch bucket to use, overrides what is set in workflow.")
	oauth                = flag.String("oauth", "", "path to oauth json file, overrides what is set in workflow.")
	ce                   = flag.String("compute_endpoint_override", "", "API endpoint to override default.")
	gcsLogsDisabled      = flag.Bool("disable_gcs_logging", false, "do not stream logs to GCS.")
	cloudLogsDisabled    = flag.Bool("disable_cloud_logging", false, "do not stream logs to Cloud Logging.")
	stdoutLogsDisabled   = flag.Bool("disable_stdout_logging", false, "do not display individual workflow logs on stdout.")
	labels               = flag.String("labels", "", "List of label KEY=VALUE pairs to add. Keys must start with a lowercase character and contain only hyphens (-), underscores (_), lowercase characters, and numbers. Values must contain only hyphens (-), underscores (_), lowercase characters, and numbers.")

	region  *string
	buildID = os.Getenv("BUILD_ID")

	userLabels            map[string]string
	currentExecutablePath *string

	destinationBucketName *string
	destinationObjectName *string
)

func init() {
	currentExecutablePathStr := string(os.Args[0])
	currentExecutablePath = &currentExecutablePathStr
}

func validateAndParseFlags() error {
	flag.Parse()

	if err := validationutils.ValidateStringFlagNotEmpty(*clientID, clientIDFlagKey); err != nil {
		return err
	}
	if err := validationutils.ValidateStringFlagNotEmpty(*destinationUri, destinationUriFlagKey); err != nil {
		return err
	}

	if *image == "" && *imageFamily == "" {
		return fmt.Errorf("-image or -image_family has to be specified")
	}

	if *image != "" && *imageFamily != "" {
		return fmt.Errorf("-image and -image_family can't be both specified")
	}


	if *destinationUri != "" {
		bucketName, objectName, err := storageutils.SplitGCSPath(*destinationUri)
		if err != nil {
			return err
		}

		destinationBucketName = &bucketName
		destinationObjectName = &objectName
	}

	if *labels != "" {
		var err error
		userLabels, err = argutils.ParseKeyValues(*labels)
		if err != nil {
			return err
		}
	}

	return nil
}


func getWorkflowPath() string {
	if *format == "" {
		return pathutils.ToWorkingDir(exportWorkflow, *currentExecutablePath)
	} else {
		return pathutils.ToWorkingDir(exportAndConvertWorkflow, *currentExecutablePath)
	}
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
	var err error
	project, err = argutils.GetProjectID(mgce, project)
	return err
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

func runExport(ctx context.Context, metadataGCEHolder computeutils.MetadataGCE,
	scratchBucketCreator *storageutils.ScratchBucketCreator,
	zoneRetriever *storageutils.ZoneRetriever, storageClient commondomain.StorageClientInterface) error {

	err := populateMissingParameters(&metadataGCEHolder, scratchBucketCreator, zoneRetriever, storageClient)
	if err != nil {
		return err
	}
	exportWorkflowPath := getWorkflowPath()
	varMap := buildDaisyVars()
	workflow, err := daisyutils.ParseWorkflow(&computeutils.MetadataGCE{}, exportWorkflowPath, varMap,
		*project, *zone, *scratchBucketGcsPath, *oauth, *timeout, *ce, *gcsLogsDisabled,
		*cloudLogsDisabled, *stdoutLogsDisabled)
	if err != nil {
		return err
	}

	workflowModifier := func(w *daisy.Workflow) {
		rl := &daisyutils.ResourceLabeler{
			BuildID: buildID, UserLabels: userLabels, BuildIDLabelKey: "gce-image-export-build-id",
			InstanceLabelKeyRetriever: func(instance *daisy.Instance) string {
				return "gce-image-export-tmp"
			},
			DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
				return "gce-image-export-tmp"
			},
			ImageLabelKeyRetriever: func(image *daisy.Image) string {
				return "gce-image-export"
			}}
		rl.LabelResources(w)
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

	scratchBucketCreator := storageutils.NewScratchBucketCreator(ctx, storageClient)
	zoneRetriever, err := storageutils.NewZoneRetriever(
		&metadataGCEHolder, createComputeClient(&ctx))

	if err != nil {
		log.Fatalf(err.Error())
	}

	if err := runExport(ctx, metadataGCEHolder, scratchBucketCreator, zoneRetriever, storageClient); err != nil {
		log.Fatalf(err.Error())
	}
}
