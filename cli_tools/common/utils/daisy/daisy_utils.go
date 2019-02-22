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
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"reflect"
)

var (
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
)

func ValidateOs(osId string) error {
	if _, osValid := osChoices[osId]; !osValid {
		return fmt.Errorf("os %v is invalid. Allowed values: %v", osId, reflect.ValueOf(osChoices).MapKeys())
	}
	return nil
}

func GetTranslateWorkflowPath(os *string) string {
	return osChoices[*os]
}

// ParseWorkflow parses Daisy workflow file and returns Daisy workflow object or error in case of failure
func ParseWorkflow(metadataGCE commondomain.MetadataGCEInterface, path string,
	varMap map[string]string, project, zone, gcsPath, oauth, dTimeout, cEndpoint string,
	disableGCSLogs, diableCloudLogs, disableStdoutLogs bool) (*daisy.Workflow, error) {
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
		return nil, fmt.Errorf("unknown workflow Var %q passed to Workflow %q", k, w.Name)
	}

	if project != "" {
		w.Project = project
	} else if w.Project == "" && metadataGCE.OnGCE() {
		w.Project, err = metadataGCE.ProjectID()
		if err != nil {
			return nil, err
		}
	}
	if zone != "" {
		w.Zone = zone
	} else if w.Zone == "" && metadataGCE.OnGCE() {
		w.Zone, err = metadataGCE.Zone()
		if err != nil {
			return nil, err
		}
	}
	if gcsPath != "" {
		w.GCSPath = gcsPath
	}
	if oauth != "" {
		w.OAuthPath = oauth
	}
	if dTimeout != "" {
		w.DefaultTimeout = dTimeout
	}

	if cEndpoint != "" {
		w.ComputeEndpoint = cEndpoint
	}

	if disableGCSLogs {
		w.DisableGCSLogging()
	}
	if diableCloudLogs {
		w.DisableCloudLogging()
	}
	if disableStdoutLogs {
		w.DisableStdoutLogging()
	}

	return w, nil
}
