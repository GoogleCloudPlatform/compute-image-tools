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

package daisy

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strings"

	stringutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/string"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

const (
	// BuildIDOSEnvVarName is the os env var name to get build id
	BuildIDOSEnvVarName   = "BUILD_ID"
	translateFailedPrefix = "TranslateFailed"
)

var (
	osChoices = map[string]string{
		"debian-8":            "debian/translate_debian_8.wf.json",
		"debian-9":            "debian/translate_debian_9.wf.json",
		"centos-6":            "enterprise_linux/translate_centos_6.wf.json",
		"centos-7":            "enterprise_linux/translate_centos_7.wf.json",
		"centos-8":            "enterprise_linux/translate_centos_8.wf.json",
		"opensuse-15":         "suse/translate_opensuse_15.wf.json",
		"sles-sap-12-byol":    "suse/translate_sles_sap_12_byol.wf.json",
		"sles-12-byol":        "suse/translate_sles_12_byol.wf.json",
		"sles-15-byol":        "suse/translate_sles_15_byol.wf.json",
		"rhel-6":              "enterprise_linux/translate_rhel_6_licensed.wf.json",
		"rhel-7":              "enterprise_linux/translate_rhel_7_licensed.wf.json",
		"rhel-8":              "enterprise_linux/translate_rhel_8_licensed.wf.json",
		"rhel-6-byol":         "enterprise_linux/translate_rhel_6_byol.wf.json",
		"rhel-7-byol":         "enterprise_linux/translate_rhel_7_byol.wf.json",
		"rhel-8-byol":         "enterprise_linux/translate_rhel_8_byol.wf.json",
		"ubuntu-1404":         "ubuntu/translate_ubuntu_1404.wf.json",
		"ubuntu-1604":         "ubuntu/translate_ubuntu_1604.wf.json",
		"ubuntu-1804":         "ubuntu/translate_ubuntu_1804.wf.json",
		"windows-2008r2":      "windows/translate_windows_2008_r2.wf.json",
		"windows-2008r2-byol": "windows/translate_windows_2008_r2_byol.wf.json",
		"windows-2012":        "windows/translate_windows_2012.wf.json",
		"windows-2012-byol":   "windows/translate_windows_2012_byol.wf.json",
		"windows-2012r2":      "windows/translate_windows_2012_r2.wf.json",
		"windows-2012r2-byol": "windows/translate_windows_2012_r2_byol.wf.json",
		"windows-2016":        "windows/translate_windows_2016.wf.json",
		"windows-2016-byol":   "windows/translate_windows_2016_byol.wf.json",
		"windows-2019":        "windows/translate_windows_2019.wf.json",
		"windows-2019-byol":   "windows/translate_windows_2019_byol.wf.json",
		"windows-7-x64-byol":  "windows/translate_windows_7_x64_byol.wf.json",
		"windows-7-x86-byol":  "windows/translate_windows_7_x86_byol.wf.json",
		"windows-8-x64-byol":  "windows/translate_windows_8_x64_byol.wf.json",
		"windows-8-x86-byol":  "windows/translate_windows_8_x86_byol.wf.json",
		"windows-10-x64-byol": "windows/translate_windows_10_x64_byol.wf.json",
		"windows-10-x86-byol": "windows/translate_windows_10_x86_byol.wf.json",

		// for backward compatibility, to be removed once clients (gcloud and UI) and docs are all updated
		"windows-7-byol":       "windows/translate_windows_7_x64_byol.wf.json",
		"windows-8-1-x64-byol": "windows/translate_windows_8_x64_byol.wf.json",
		"windows-10-byol":      "windows/translate_windows_10_x64_byol.wf.json",
	}
	privacyRegex    = regexp.MustCompile(`\[Privacy\->.*?<\-Privacy\]`)
	privacyTagRegex = regexp.MustCompile(`(\[Privacy\->)|(<\-Privacy\])`)
)

// ValidateOS validates that osID is supported by Daisy image import
func ValidateOS(osID string) error {
	if osID == "" {
		return daisy.Errf("osID is empty")
	}
	if _, osValid := osChoices[osID]; !osValid {
		// Expose osID and osChoices in the anonymized error message since they are not sensitive values.
		allowedValuesMsg := fmt.Sprintf("Allowed values: %v", reflect.ValueOf(osChoices).MapKeys())
		return daisy.Errf("os `%v` is invalid. "+allowedValuesMsg, osID)
	}
	return nil
}

// GetTranslateWorkflowPath returns path to image translate workflow path for given OS
func GetTranslateWorkflowPath(os string) string {
	return osChoices[os]
}

// UpdateAllInstanceNoExternalIP updates all Create Instance steps in the workflow to operate
// when no external IP access is allowed by the VPC Daisy workflow is running in.
func UpdateAllInstanceNoExternalIP(workflow *daisy.Workflow, noExternalIP bool) {
	if !noExternalIP {
		return
	}
	workflow.IterateWorkflowSteps(func(step *daisy.Step) {
		if step.CreateInstances != nil {
			for _, instance := range step.CreateInstances.Instances {
				if instance.Instance.NetworkInterfaces == nil {
					continue
				}
				for _, networkInterface := range instance.Instance.NetworkInterfaces {
					networkInterface.AccessConfigs = []*compute.AccessConfig{}
				}
			}
			for _, instance := range step.CreateInstances.InstancesBeta {
				if instance.Instance.NetworkInterfaces == nil {
					continue
				}
				for _, networkInterface := range instance.Instance.NetworkInterfaces {
					networkInterface.AccessConfigs = []*computeBeta.AccessConfig{}
				}
			}

		}
	})
}

// UpdateToUEFICompatible marks workflow resources (disks and images) to be UEFI
// compatible by adding "UEFI_COMPATIBLE" to GuestOSFeatures. Debian 9 workers
// are excluded until UEFI becomes the default boot method.
func UpdateToUEFICompatible(workflow *daisy.Workflow) {
	workflow.IterateWorkflowSteps(func(step *daisy.Step) {
		if step.CreateDisks != nil {
			for _, disk := range *step.CreateDisks {
				// for the time being, don't run Debian 9 worker in UEFI mode
				if strings.Contains(disk.SourceImage, "projects/compute-image-tools/global/images/family/debian-9-worker") {
					continue
				}
				// also, don't run Windows bootstrap worker in UEFI mode
				if strings.Contains(disk.SourceImage, "projects/windows-cloud/global/images/family/windows-2019-core") && strings.Contains(disk.Name, "disk-bootstrap") {
					continue
				}

				disk.Disk.GuestOsFeatures = daisy.CombineGuestOSFeatures(disk.Disk.GuestOsFeatures, "UEFI_COMPATIBLE")
			}
		}
		if step.CreateImages != nil {
			for _, image := range step.CreateImages.Images {
				image.GuestOsFeatures = stringutils.CombineStringSlices(image.GuestOsFeatures, "UEFI_COMPATIBLE")
				image.Image.GuestOsFeatures = daisy.CombineGuestOSFeatures(image.Image.GuestOsFeatures, "UEFI_COMPATIBLE")

			}
			for _, image := range step.CreateImages.ImagesBeta {
				image.GuestOsFeatures = stringutils.CombineStringSlices(image.GuestOsFeatures, "UEFI_COMPATIBLE")
				image.Image.GuestOsFeatures = daisy.CombineGuestOSFeaturesBeta(image.Image.GuestOsFeatures, "UEFI_COMPATIBLE")
			}
		}
	})
}

// RemovePrivacyLogInfo removes privacy log information.
func RemovePrivacyLogInfo(message string) string {
	// Since translation scripts vary and is hard to predict the output, we have to hide the
	// details and only remain "TranslateFailed"
	if strings.Contains(message, translateFailedPrefix) {
		return translateFailedPrefix
	}

	// All import/export bash scripts enclose privacy info inside "[Privacy-> XXX <-Privacy]". Let's
	// remove it for privacy.
	message = privacyRegex.ReplaceAllString(message, "")

	return message
}

// RemovePrivacyLogTag removes privacy log tag.
func RemovePrivacyLogTag(message string) string {
	// All import/export bash scripts enclose privacy info inside a pair of tag "[Privacy->XXX<-Privacy]".
	// Let's remove the tag to improve the readability.
	message = privacyTagRegex.ReplaceAllString(message, "")

	return message
}

// PostProcessDErrorForNetworkFlag determines whether to show more hints for network flag
func PostProcessDErrorForNetworkFlag(action string, err error, network string, w *daisy.Workflow) {
	if derr, ok := err.(daisy.DError); ok {
		if derr.CausedByErrType("networkResourceDoesNotExist") && network == "" {
			w.LogWorkflowInfo("A VPC network is required for running %v,"+
				" and the default VPC network does not exist in your project. You will need to"+
				" specify a VPC network with the --network flag. For more information about"+
				" VPC networks, see https://cloud.google.com/vpc.", action)
		}
	}
}

// RunWorkflowWithCancelSignal runs Daisy workflow with accepting Ctrl-C signal
func RunWorkflowWithCancelSignal(ctx context.Context, w *daisy.Workflow) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(w *daisy.Workflow) {
		select {
		case <-c:
			w.LogWorkflowInfo("\nCtrl-C caught, sending cancel signal to %q...\n", w.Name)
			close(w.Cancel)
		case <-w.Cancel:
		}
	}(w)
	return w.Run(ctx)
}

// NewStep creates a new step for the workflow along with dependencies.
func NewStep(w *daisy.Workflow, name string, dependencies ...*daisy.Step) (*daisy.Step, error) {
	s, err := w.NewStep(name)
	if err != nil {
		return nil, err
	}

	err = w.AddDependency(s, dependencies...)
	return s, err
}

// GetResourceID gets resource id from its URI. Definition of resource ID:
// https://cloud.google.com/apis/design/resource_names#resource_id
func GetResourceID(resourceURI string) string {
	dm := strings.Split(resourceURI, "/")
	return dm[len(dm)-1]
}

// GetDeviceURI gets a URI for a device based on its attributes. A device is a disk
// attached to a instance.
func GetDeviceURI(project, zone, name string) string {
	return fmt.Sprintf("projects/%v/zones/%v/devices/%v", project, zone, name)
}

// GetDiskURI gets a URI for a disk based on its attributes. Introduction
// to a disk resource: https://cloud.google.com/compute/docs/reference/rest/v1/disks
func GetDiskURI(project, zone, name string) string {
	return fmt.Sprintf("projects/%v/zones/%v/disks/%v", project, zone, name)
}

// GetInstanceURI gets a URI for a instance based on its attributes. Introduction
// to a instance resource: https://cloud.google.com/compute/docs/reference/rest/v1/instances
func GetInstanceURI(project, zone, name string) string {
	return fmt.Sprintf("projects/%v/zones/%v/instances/%v", project, zone, name)
}
