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

package daisyutils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"

	stringutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/string"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const (
	// BuildIDOSEnvVarName is the os env var name to get build id
	BuildIDOSEnvVarName   = "BUILD_ID"
	translateFailedPrefix = "TranslateFailed"
)

// TranslationSettings includes information that needs to be added to a disk or image after it is imported,
// for a particular OS and version.
type TranslationSettings struct {
	// GcloudOsFlag is the user-facing string corresponding to this OS, version, and licensing mode.
	// It is passed as a value of the `--os` flag.
	GcloudOsFlag string

	// LicenseURI is the GCP Compute license corresponding to this OS, version, and licensing mode:
	//  https://cloud.google.com/compute/docs/reference/rest/v1/licenses
	LicenseURI string

	// WorkflowPath is the path to a Daisy json workflow, relative to the
	// `daisy_workflows/image_import` directory.
	WorkflowPath string
}

var (
	supportedOS = []TranslationSettings{
		// Enterprise Linux
		{
			GcloudOsFlag: "centos-7",
			WorkflowPath: "enterprise_linux/translate_centos_7.wf.json",
			LicenseURI:   "projects/centos-cloud/global/licenses/centos-7",
		}, {
			GcloudOsFlag: "centos-8",
			WorkflowPath: "enterprise_linux/translate_centos_8.wf.json",
			LicenseURI:   "projects/centos-cloud/global/licenses/centos-8",
		}, {
			GcloudOsFlag: "rhel-6",
			WorkflowPath: "enterprise_linux/translate_rhel_6_licensed.wf.json",
			LicenseURI:   "projects/rhel-cloud/global/licenses/rhel-6-server",
		}, {
			GcloudOsFlag: "rhel-6-byol",
			WorkflowPath: "enterprise_linux/translate_rhel_6_byol.wf.json",
			LicenseURI:   "projects/rhel-cloud/global/licenses/rhel-6-byol",
		}, {
			GcloudOsFlag: "rhel-7",
			WorkflowPath: "enterprise_linux/translate_rhel_7_licensed.wf.json",
			LicenseURI:   "projects/rhel-cloud/global/licenses/rhel-7-server",
		}, {
			GcloudOsFlag: "rhel-7-byol",
			WorkflowPath: "enterprise_linux/translate_rhel_7_byol.wf.json",
			LicenseURI:   "projects/rhel-cloud/global/licenses/rhel-7-byol",
		}, {
			GcloudOsFlag: "rhel-8",
			WorkflowPath: "enterprise_linux/translate_rhel_8_licensed.wf.json",
			LicenseURI:   "projects/rhel-cloud/global/licenses/rhel-8-server",
		}, {
			GcloudOsFlag: "rhel-8-byol",
			WorkflowPath: "enterprise_linux/translate_rhel_8_byol.wf.json",
			LicenseURI:   "projects/rhel-cloud/global/licenses/rhel-8-byos",
		}, {
			GcloudOsFlag: "rocky-8",
			WorkflowPath: "enterprise_linux/translate_rocky_8.wf.json",
			LicenseURI:   "projects/rocky-linux-cloud/global/licenses/rocky-linux-8",
		},

		// SUSE
		{
			GcloudOsFlag: "opensuse-15",
			WorkflowPath: "suse/translate_opensuse_15.wf.json",
			LicenseURI:   "projects/opensuse-cloud/global/licenses/opensuse-leap-42",
		}, {
			GcloudOsFlag: "sles-12",
			WorkflowPath: "suse/translate_sles_12.wf.json",
			LicenseURI:   "projects/suse-cloud/global/licenses/sles-12",
		}, {
			GcloudOsFlag: "sles-12-byol",
			WorkflowPath: "suse/translate_sles_12_byol.wf.json",
			LicenseURI:   "projects/suse-byos-cloud/global/licenses/sles-12-byos",
		}, {
			GcloudOsFlag: "sles-sap-12",
			WorkflowPath: "suse/translate_sles_sap_12.wf.json",
			LicenseURI:   "projects/suse-sap-cloud/global/licenses/sles-sap-12",
		}, {
			GcloudOsFlag: "sles-sap-12-byol",
			WorkflowPath: "suse/translate_sles_sap_12_byol.wf.json",
			LicenseURI:   "projects/suse-byos-cloud/global/licenses/sles-sap-12-byos",
		}, {
			GcloudOsFlag: "sles-15",
			WorkflowPath: "suse/translate_sles_15.wf.json",
			LicenseURI:   "projects/suse-cloud/global/licenses/sles-15",
		}, {
			GcloudOsFlag: "sles-15-byol",
			WorkflowPath: "suse/translate_sles_15_byol.wf.json",
			LicenseURI:   "projects/suse-byos-cloud/global/licenses/sles-15-byos",
		}, {
			GcloudOsFlag: "sles-sap-15",
			WorkflowPath: "suse/translate_sles_sap_15.wf.json",
			LicenseURI:   "projects/suse-sap-cloud/global/licenses/sles-sap-15",
		}, {
			GcloudOsFlag: "sles-sap-15-byol",
			WorkflowPath: "suse/translate_sles_sap_15_byol.wf.json",
			LicenseURI:   "projects/suse-byos-cloud/global/licenses/sles-sap-15-byos",
		},

		// Debian
		{
			GcloudOsFlag: "debian-8",
			WorkflowPath: "debian/translate_debian_8.wf.json",
			LicenseURI:   "projects/debian-cloud/global/licenses/debian-8-jessie",
		}, {
			GcloudOsFlag: "debian-9",
			WorkflowPath: "debian/translate_debian_9.wf.json",
			LicenseURI:   "projects/debian-cloud/global/licenses/debian-9-stretch",
		}, {
			GcloudOsFlag: "debian-10",
			WorkflowPath: "debian/translate_debian_10.wf.json",
			LicenseURI:   "projects/debian-cloud/global/licenses/debian-10-buster",
		}, {
			GcloudOsFlag: "debian-11",
			WorkflowPath: "debian/translate_debian_11.wf.json",
			LicenseURI:   "projects/debian-cloud/global/licenses/debian-11-bullseye",
		},

		// Ubuntu
		{
			GcloudOsFlag: "ubuntu-1404",
			WorkflowPath: "ubuntu/translate_ubuntu_1404.wf.json",
			LicenseURI:   "projects/ubuntu-os-cloud/global/licenses/ubuntu-1404-trusty",
		}, {
			GcloudOsFlag: "ubuntu-1604",
			WorkflowPath: "ubuntu/translate_ubuntu_1604.wf.json",
			LicenseURI:   "projects/ubuntu-os-cloud/global/licenses/ubuntu-1604-xenial",
		}, {
			GcloudOsFlag: "ubuntu-1804",
			WorkflowPath: "ubuntu/translate_ubuntu_1804.wf.json",
			LicenseURI:   "projects/ubuntu-os-cloud/global/licenses/ubuntu-1804-lts",
		}, {
			GcloudOsFlag: "ubuntu-2004",
			WorkflowPath: "ubuntu/translate_ubuntu_2004.wf.json",
			LicenseURI:   "projects/ubuntu-os-cloud/global/licenses/ubuntu-2004-lts",
		},

		// Windows
		{
			GcloudOsFlag: "windows-7-x64-byol",
			WorkflowPath: "windows/translate_windows_7_x64_byol.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-7-x64-byol",
		}, {
			GcloudOsFlag: "windows-7-x86-byol",
			WorkflowPath: "windows/translate_windows_7_x86_byol.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-7-x86-byol",
		}, {
			GcloudOsFlag: "windows-8-x64-byol",
			WorkflowPath: "windows/translate_windows_8_x64_byol.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-8-x64-byol",
		}, {
			GcloudOsFlag: "windows-8-x86-byol",
			WorkflowPath: "windows/translate_windows_8_x86_byol.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-8-x86-byol",
		}, {
			GcloudOsFlag: "windows-10-x64-byol",
			WorkflowPath: "windows/translate_windows_10_x64_byol.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-10-x64-byol",
		}, {
			GcloudOsFlag: "windows-10-x86-byol",
			WorkflowPath: "windows/translate_windows_10_x86_byol.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-10-x86-byol",
		}, {
			GcloudOsFlag: "windows-2008r2",
			WorkflowPath: "windows/translate_windows_2008_r2.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-server-2008-r2-dc",
		}, {
			GcloudOsFlag: "windows-2008r2-byol",
			WorkflowPath: "windows/translate_windows_2008_r2_byol.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-server-2008-r2-byol",
		}, {
			GcloudOsFlag: "windows-2012",
			WorkflowPath: "windows/translate_windows_2012.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-server-2012-dc",
		}, {
			GcloudOsFlag: "windows-2012-byol",
			WorkflowPath: "windows/translate_windows_2012_byol.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-server-2012-byol",
		}, {
			GcloudOsFlag: "windows-2012r2",
			WorkflowPath: "windows/translate_windows_2012_r2.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-server-2012-r2-dc",
		}, {
			GcloudOsFlag: "windows-2012r2-byol",
			WorkflowPath: "windows/translate_windows_2012_r2_byol.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-server-2012-r2-byol",
		}, {
			GcloudOsFlag: "windows-2016",
			WorkflowPath: "windows/translate_windows_2016.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-server-2016-dc",
		}, {
			GcloudOsFlag: "windows-2016-byol",
			WorkflowPath: "windows/translate_windows_2016_byol.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-server-2016-byol",
		}, {
			GcloudOsFlag: "windows-2019",
			WorkflowPath: "windows/translate_windows_2019.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-server-2019-dc",
		}, {
			GcloudOsFlag: "windows-2019-byol",
			WorkflowPath: "windows/translate_windows_2019_byol.wf.json",
			LicenseURI:   "projects/windows-cloud/global/licenses/windows-server-2019-byol",
		},
	}

	// legacyIDs maps a legacy identifier to its replacement.
	legacyIDs = map[string]string{
		"windows-7-byol":       "windows-7-x64-byol",
		"windows-8-1-x64-byol": "windows-8-x64-byol",
		"windows-10-byol":      "windows-10-x64-byol",
	}

	privacyRegex    = regexp.MustCompile(`\[Privacy\->.*?<\-Privacy\]`)
	privacyTagRegex = regexp.MustCompile(`(\[Privacy\->)|(<\-Privacy\])`)

	debianWorkerRegex = regexp.MustCompile("projects/compute-image-tools/global/images/family/debian-\\d+-worker")
)

// GetSortedOSIDs returns the supported OS identifiers, sorted.
func GetSortedOSIDs() []string {
	choices := make([]string, 0, len(supportedOS))
	for _, k := range supportedOS {
		choices = append(choices, k.GcloudOsFlag)
	}
	sort.Strings(choices)
	return choices
}

// ValidateOS validates that osID is supported by Daisy image import
func ValidateOS(osID string) error {
	_, err := GetTranslationSettings(osID)
	return err
}

// GetTranslationSettings returns parameters required for translating a particular OS, version,
// and licensing mode to run on GCE.
//
// An error is returned if the OS, version, and licensing mode is not supported for import.
func GetTranslationSettings(osID string) (spec TranslationSettings, err error) {
	if osID == "" {
		return spec, errors.New("osID is empty")
	}

	if replacement := legacyIDs[osID]; replacement != "" {
		osID = replacement
	}
	for _, choice := range supportedOS {
		if choice.GcloudOsFlag == osID {
			return choice, nil
		}
	}
	allowedValuesMsg := fmt.Sprintf("Allowed values: %v", GetSortedOSIDs())
	return spec, daisy.Errf("os `%v` is invalid. "+allowedValuesMsg, osID)
}

// UpdateToUEFICompatible marks workflow resources (disks and images) to be UEFI
// compatible by adding "UEFI_COMPATIBLE" to GuestOSFeatures. Debian workers
// are excluded until UEFI becomes the default boot method.
func UpdateToUEFICompatible(workflow *daisy.Workflow) {
	workflow.IterateWorkflowSteps(func(step *daisy.Step) {
		if step.CreateDisks != nil {
			for _, disk := range *step.CreateDisks {
				// for the time being, don't run Debian worker in UEFI mode
				if debianWorkerRegex.MatchString(disk.SourceImage) {
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

// RunWorkflowWithCancelSignal runs a Daisy workflow, and allows for cancellation from two sources:
//   1. The user types Ctrl-C on their keyboard.
//   2. The caller sends a cancellation reason on the cancel channel (or closes it).
func RunWorkflowWithCancelSignal(w *daisy.Workflow, cancel <-chan string) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(w *daisy.Workflow) {
		select {
		case reason := <-cancel:
			if reason != "" {
				w.CancelWithReason(reason)
			}
			break
		case <-c:
			w.LogWorkflowInfo("\nCtrl-C caught, sending cancel signal to %q...\n", w.Name)
			w.CancelWorkflow()
			break
		case <-w.Cancel:
		}
	}(w)
	return w.Run(context.Background())
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

// ParseWorkflow parses Daisy workflow file and returns Daisy workflow object or error in case of failure
func ParseWorkflow(path string, varMap map[string]string, project, zone, gcsPath, oauth, dTimeout, cEndpoint string, disableGCSLogs, disableCloudLogs, disableStdoutLogs bool) (*daisy.Workflow, error) {
	w, err := daisy.NewFromFile(path)
	if err != nil {
		return nil, err
	}
Loop:
	for k, v := range varMap {
		for wv := range w.Vars {
			if k == wv {
				w.AddVar(k, v)
				continue Loop
			}
		}
		return nil, daisy.Errf("unknown workflow Var %q passed to Workflow %q", k, w.Name)
	}

	EnvironmentSettings{
		Project:           project,
		Zone:              zone,
		GCSPath:           gcsPath,
		OAuth:             oauth,
		Timeout:           dTimeout,
		ComputeEndpoint:   cEndpoint,
		DisableGCSLogs:    disableGCSLogs,
		DisableCloudLogs:  disableCloudLogs,
		DisableStdoutLogs: disableStdoutLogs,
	}.ApplyToWorkflow(w)

	return w, nil
}

// Tool is used to communicate the tool's name ot the user.
type Tool struct {

	// HumanReadableName is used for error messages, for example: "image import".
	HumanReadableName string
	// ResourceLabelName is used when labeling temporary resources, for example: "image-import"
	ResourceLabelName string
}

// EnvironmentSettings controls the resources that are used during tool execution.
type EnvironmentSettings struct {
	// Location of workflows
	WorkflowDirectory string

	// Fields from daisy.Workflow
	Project, Zone, GCSPath, OAuth, Timeout, ComputeEndpoint string
	DisableGCSLogs, DisableCloudLogs, DisableStdoutLogs     bool

	// An optional prefix to include in the bracketed portion of daisy's stdout logs.
	// Gcloud does a prefix match to determine whether to show a log line to a user.
	//
	// With a prefix of `disk-1`, for example, the workflow in `importer.NewDaisyInflater`
	// emits log messages starting with `[disk-1-inflate]`.
	DaisyLogLinePrefix string

	// Worker instance customizations
	Network, Subnet       string
	ComputeServiceAccount string
	NoExternalIP          bool
	Labels                map[string]string
	ExecutionID           string
	StorageLocation       string
	Tool                  Tool
}

// ApplyToWorkflow sets fields on daisy.Workflow from the environment settings.
func (env EnvironmentSettings) ApplyToWorkflow(w *daisy.Workflow) {
	w.Project = env.Project
	w.Zone = env.Zone
	if env.GCSPath != "" {
		w.GCSPath = env.GCSPath
	}
	if env.OAuth != "" {
		w.OAuthPath = env.OAuth
	}
	if env.Timeout != "" {
		w.DefaultTimeout = env.Timeout
	}
	if env.ComputeEndpoint != "" {
		w.ComputeEndpoint = env.ComputeEndpoint
	}
	if env.DisableGCSLogs {
		w.DisableGCSLogging()
	}
	if env.DisableCloudLogs {
		w.DisableCloudLogging()
	}
	if env.DisableStdoutLogs {
		w.DisableStdoutLogging()
	}
}

// UpdateAllInstanceNoExternalIP updates all Create Instance steps in the workflow to operate
// when no external IP access is allowed by the VPC Daisy workflow is running in.
func UpdateAllInstanceNoExternalIP(workflow *daisy.Workflow, noExternalIP bool) {
	if !noExternalIP {
		return
	}
	(&RemoveExternalIPHook{}).PreRunHook(workflow)
}
