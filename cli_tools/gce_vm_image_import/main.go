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
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisy_common"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
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
	osID                 = flag.String("os", "", "Specifies the OS of the image being imported. Must be one of: centos-6, centos-7, debian-8, debian-9, rhel-6, rhel-6-byol, rhel-7, rhel-7-byol, ubuntu-1404, ubuntu-1604, windows-2008r2, windows-2012r2, windows-2016.")
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

	region    *string
	buildID   = os.Getenv("BUILD_ID")
	gsRegex   = regexp.MustCompile(`^gs://([a-z0-9][-_.a-z0-9]*)/(.+)$`)
	osChoices = map[string]string{
		"debian-8":       "debian/translate_debian_8.wf.json",
		"debian-9":       "debian/translate_debian_9.wf.json",
		"centos-6":       "enterprise_linux/translate_centos_6.wf.json",
		"centos-7":       "enterprise_linux/translate_centos_7.wf.json",
		"rhel-6":         "enterprise_linux/translate_rhel_6_licensed.wf.json",
		"rhel-7":         "enterprise_linux/translate_rhel_7_licensed.wf.json",
		"rhel-6-byol":    "enterprise_linux/translate_rhel_6_byol.wf.json",
		"rhel-7-byol":    "enterprise_linux/translate_rhel_7_byol.wf.json",
		"ubuntu-1404":    "ubuntu/translate_ubuntu_1404.wf.json",
		"ubuntu-1604":    "ubuntu/translate_ubuntu_1604.wf.json",
		"windows-2008r2": "windows/translate_windows_2008_r2.wf.json",
		"windows-2012r2": "windows/translate_windows_2012_r2.wf.json",
		"windows-2016":   "windows/translate_windows_2016.wf.json",
	}
	userLabels *map[string]string
)

func validateAndParseFlags() error {
	flag.Parse()

	if err := validateStringFlag(*imageName, imageNameFlagKey); err != nil {
		return err
	}
	if err := validateStringFlag(*clientID, clientIDFlagKey); err != nil {
		return err
	}

	if !*dataDisk && *osID == "" {
		return fmt.Errorf("-data_disk or -os has to be specified")
	}

	if *dataDisk && *osID != "" {
		return fmt.Errorf("either -data_disk or -os has to be specified, but not both")
	}

	if *sourceFile == "" && *sourceImage == "" {
		return fmt.Errorf("-source_file or -source_image has to be specified")
	}

	if *sourceFile != "" && *sourceImage != "" {
		return fmt.Errorf("either -source_file or -source_image has to be specified, but not both %v %v", *sourceFile, *sourceImage)
	}

	if *osID != "" {
		if _, osValid := osChoices[*osID]; !osValid {
			return fmt.Errorf("os %v is invalid. Allowed values: %v", *osID, reflect.ValueOf(osChoices).MapKeys())
		}
	}

	if *sourceFile != "" {
		_, _, err := splitGCSPath(*sourceFile)
		if err != nil {
			return err
		}
	}

	if *labels != "" {
		var err error
		userLabels, err = parseUserLabels(labels)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseUserLabels(labelsFlag *string) (*map[string]string, error) {
	labelsMap := make(map[string]string)
	splits := strings.Split(*labelsFlag, ",")
	for _, split := range splits {
		if len(split) == 0 {
			continue
		}
		key, value, err := parseUserLabel(split)
		if err != nil {
			return nil, err
		}
		labelsMap[key] = value
	}
	return &labelsMap, nil
}

func parseUserLabel(labelSplit string) (string, string, error) {
	splits := strings.Split(labelSplit, "=")
	if len(splits) != 2 {
		return "", "", fmt.Errorf("Label specification should be in the following format: LABEL_KEY=LABEL_VALUE, but it's %v", labelSplit)
	}
	key := strings.TrimSpace(splits[0])
	value := strings.TrimSpace(splits[1])
	if len(key) == 0 {
		return "", "", fmt.Errorf("Label key is empty string: %v", labelSplit)
	}
	if len(value) == 0 {
		return "", "", fmt.Errorf("Label value is empty string: %v", labelSplit)
	}
	return key, value, nil
}

func validateStringFlag(flagValue string, flagKey string) error {
	return validateString(flagValue, flagKey, "The flag -%v must be provided")
}

func validateString(value string, key string, errorMessage string) error {
	if value == "" {
		return fmt.Errorf(errorMessage, key)
	}
	return nil
}

func splitGCSPath(p string) (string, string, error) {
	matches := gsRegex.FindStringSubmatch(p)
	if matches != nil {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("%q is not a valid GCS path", p)
}

//Returns main workflow and translate workflow paths (if any)
func getWorkflowPaths() (string, string) {
	if *sourceImage != "" {
		return importFromImageWorkflow, getTranslateWorkflowPath(osID)
	}
	if *dataDisk {
		return importWorkflow, ""
	}
	return importAndTranslateWorkflow, getTranslateWorkflowPath(osID)
}

func getTranslateWorkflowPath(os *string) string {
	return osChoices[*os]
}

func fatalIfError(f func() error) {
	if err := f(); err != nil {
		log.Fatalf(err.Error())
	}
}

func populateMissingParameters() {
	fatalIfError(func() error {
		return populateZoneIfMissing(metadataGCEHolder{})
	})
	fatalIfError(populateRegion)

	//TODO: network, subnetwork, gcsPath (create scratch bucket including regionalization, if possible)
}

type metadataGCEHolder struct{}

type metadataGCE interface {
	OnGCE() bool
	Zone() (string, error)
}

func (m metadataGCEHolder) OnGCE() bool {
	return metadata.OnGCE()
}

func (m metadataGCEHolder) Zone() (string, error) {
	return metadata.Zone()
}

func populateZoneIfMissing(mgce metadataGCE) error {
	if *zone == "" {
		var err error
		var aZone = ""
		if mgce.OnGCE() {
			aZone, err = mgce.Zone()
		}

		if err != nil {
			return fmt.Errorf("can't infer zone: %v", err)
		}
		if aZone == "" {
			return fmt.Errorf("zone is empty")
		}

		zone = &aZone
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

// Updates workflow to support image import:
// - Labels temporary and permanent resources with appropriate labels
// - Updates instance network interfaces to not require external IP if external IP is disabled by
//   org policy
func updateWorkflow(workflow *daisy.Workflow) {
	for _, step := range workflow.Steps {
		if step.IncludeWorkflow != nil {
			//recurse into included workflow
			updateWorkflow(step.IncludeWorkflow.Workflow)
		}
		if step.CreateInstances != nil {
			for _, instance := range *step.CreateInstances {
				instance.Instance.Labels = updateResourceLabels(instance.Instance.Labels, "")
				if *noExternalIP {
					configureInstanceNetworkInterfaceForNoExternalIP(instance)
				}
			}
		}
		if step.CreateDisks != nil {
			for _, disk := range *step.CreateDisks {
				disk.Disk.Labels = updateResourceLabels(disk.Disk.Labels, "")
			}
		}
		if step.CreateImages != nil {
			for _, image := range *step.CreateImages {
				image.Image.Labels = updateResourceLabels(image.Image.Labels, getImageTypeLabelKey(image))
			}
		}
	}
}

func getImageTypeLabelKey(image *daisy.Image) string {
	imageTypeLabel := "gce-image-import"
	if strings.Contains(image.Image.Name, "untranslated") {
		imageTypeLabel = "gce-image-import-tmp"
	}
	return imageTypeLabel
}

func configureInstanceNetworkInterfaceForNoExternalIP(instance *daisy.Instance) {
	if instance.Instance.NetworkInterfaces == nil {
		return
	}
	for _, networkInterface := range instance.Instance.NetworkInterfaces {
		networkInterface.AccessConfigs = []*compute.AccessConfig{}
	}
}

func updateResourceLabels(labels map[string]string, imageTypeLabel string) map[string]string {
	labels = extendWithImageImportLabels(labels, imageTypeLabel)
	labels = extendWithUserLabels(labels)
	return labels
}

//Extend labels with image import related labels
func extendWithImageImportLabels(labels map[string]string, imageTypeLabel string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	if imageTypeLabel == "" {
		imageTypeLabel = "gce-image-import-tmp"
	}
	labels[imageTypeLabel] = "true"
	labels["gce-image-import-build-id"] = buildID

	return labels
}

func extendWithUserLabels(labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}

	if userLabels == nil || len(*userLabels) == 0 {
		return labels
	}

	for key, value := range *userLabels {
		labels[key] = value
	}
	return labels
}

func buildDaisyVars(translateWorkflowPath string) map[string]string {
	varMap := map[string]string{}

	varMap["image_name"] = *imageName
	if translateWorkflowPath != "" {
		varMap["translate_workflow"] = translateWorkflowPath
		varMap["install_gce_packages"] = strconv.FormatBool(!*noGuestEnvironment)
	}
	//TODO: copy sourceFile to gcsPath. NOTE: This maybe has to be done externally due to missing
	//permissions of argo service account
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

func toWorkingDir(dir string) string {
	wd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err == nil {
		return path.Join(wd, dir)
	}
	return dir
}

func main() {
	fatalIfError(validateAndParseFlags)
	populateMissingParameters()

	ctx := context.Background()

	importWorkflowPath, translateWorkflowPath := getWorkflowPaths()
	importWorkflowPath = toWorkingDir(importWorkflowPath)

	varMap := buildDaisyVars(translateWorkflowPath)
	workflow, err := daisycommon.ParseWorkflow(ctx, importWorkflowPath, varMap, *project, *zone,
		*scratchBucketGcsPath, *oauth, *timeout, *ce, *gcsLogsDisabled, *cloudLogsDisabled,
		*stdoutLogsDisabled)
	if err != nil {
		log.Fatalf("Error parsing workflow %q: %v", importWorkflowPath, err)
	}

	if err := workflow.RunWithModifier(ctx, updateWorkflow); err != nil {
		log.Fatalf("%s: %v", workflow.Name, err)
	}
}
