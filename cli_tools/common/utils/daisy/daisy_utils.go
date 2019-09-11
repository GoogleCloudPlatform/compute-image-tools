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
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

const (
	// BuildIDOSEnvVarName is the os env var name to get build id
	BuildIDOSEnvVarName    = "BUILD_ID"
	translateFailedPrefix  = "TranslateFailed"
	privacyInfoReplacement = "[Privacy Info]"
)

var (
	osChoices = map[string]string{
		"debian-8":             "debian/translate_debian_8.wf.json",
		"debian-9":             "debian/translate_debian_9.wf.json",
		"centos-6":             "enterprise_linux/translate_centos_6.wf.json",
		"centos-7":             "enterprise_linux/translate_centos_7.wf.json",
		"rhel-6":               "enterprise_linux/translate_rhel_6_licensed.wf.json",
		"rhel-6-byol":          "enterprise_linux/translate_rhel_6_byol.wf.json",
		"rhel-7":               "enterprise_linux/translate_rhel_7_licensed.wf.json",
		"rhel-7-byol":          "enterprise_linux/translate_rhel_7_byol.wf.json",
		"ubuntu-1404":          "ubuntu/translate_ubuntu_1404.wf.json",
		"ubuntu-1604":          "ubuntu/translate_ubuntu_1604.wf.json",
		"windows-2008r2":       "windows/translate_windows_2008_r2.wf.json",
		"windows-2008r2-byol":  "windows/translate_windows_2008_r2_byol.wf.json",
		"windows-2012":         "windows/translate_windows_2012.wf.json",
		"windows-2012-byol":    "windows/translate_windows_2012_byol.wf.json",
		"windows-2012r2":       "windows/translate_windows_2012_r2.wf.json",
		"windows-2012r2-byol":  "windows/translate_windows_2012_r2_byol.wf.json",
		"windows-2016":         "windows/translate_windows_2016.wf.json",
		"windows-2016-byol":    "windows/translate_windows_2016_byol.wf.json",
		"windows-2019":         "windows/translate_windows_2019.wf.json",
		"windows-2019-byol":    "windows/translate_windows_2019_byol.wf.json",
		"windows-7-byol":       "windows/translate_windows_7_byol.wf.json",
		"windows-8-1-x64-byol": "windows/translate_windows_8-1_x64_byol.wf.json",
		"windows-10-byol":      "windows/translate_windows_10_byol.wf.json",
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
		errMsg := fmt.Sprintf("os `%v` is invalid. Allowed values: %v", osID, reflect.ValueOf(osChoices).MapKeys())
		return daisy.Errf(errMsg)
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
	updateAllInstanceAccessConfig(workflow, func() []*compute.AccessConfig {
		return []*compute.AccessConfig{}
	})
}

func updateAllInstanceAccessConfig(workflow *daisy.Workflow, accessConfigProvider func() []*compute.AccessConfig) {
	for _, step := range workflow.Steps {
		if step.IncludeWorkflow != nil {
			//recurse into included workflow
			updateAllInstanceAccessConfig(step.IncludeWorkflow.Workflow, accessConfigProvider)
		}
		if step.CreateInstances != nil {
			for _, instance := range *step.CreateInstances {
				if instance.Instance.NetworkInterfaces == nil {
					return
				}
				for _, networkInterface := range instance.Instance.NetworkInterfaces {
					networkInterface.AccessConfigs = accessConfigProvider()
				}
			}
		}
	}
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
	message = privacyRegex.ReplaceAllString(message, privacyInfoReplacement)

	return message
}

// RemovePrivacyLogTag removes privacy log tag.
func RemovePrivacyLogTag(message string) string {
	// All import/export bash scripts enclose privacy info inside a pair of tag "[Privacy->XXX<-Privacy]".
	// Let's remove the tag to make it readable.
	message = privacyTagRegex.ReplaceAllString(message, "")

	return message
}
