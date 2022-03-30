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
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/stretchr/testify/assert"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

func Test_GetTranslationSettings_ResolveSameWorkflowPathAsOldMap(t *testing.T) {
	// This map was used by the old logic for resolving workflows from an osID.
	// By iterating over each, we verify that the new system returns the same
	// values as the old system.
	osChoices := map[string]string{
		"debian-8":            "debian/translate_debian_8.wf.json",
		"debian-9":            "debian/translate_debian_9.wf.json",
		"centos-7":            "enterprise_linux/translate_centos_7.wf.json",
		"centos-8":            "enterprise_linux/translate_centos_8.wf.json",
		"opensuse-15":         "suse/translate_opensuse_15.wf.json",
		"rhel-6":              "enterprise_linux/translate_rhel_6_licensed.wf.json",
		"rhel-7":              "enterprise_linux/translate_rhel_7_licensed.wf.json",
		"rhel-8":              "enterprise_linux/translate_rhel_8_licensed.wf.json",
		"rhel-6-byol":         "enterprise_linux/translate_rhel_6_byol.wf.json",
		"rhel-7-byol":         "enterprise_linux/translate_rhel_7_byol.wf.json",
		"rhel-8-byol":         "enterprise_linux/translate_rhel_8_byol.wf.json",
		"sles-12":             "suse/translate_sles_12.wf.json",
		"sles-12-byol":        "suse/translate_sles_12_byol.wf.json",
		"sles-sap-12":         "suse/translate_sles_sap_12.wf.json",
		"sles-sap-12-byol":    "suse/translate_sles_sap_12_byol.wf.json",
		"sles-15":             "suse/translate_sles_15.wf.json",
		"sles-15-byol":        "suse/translate_sles_15_byol.wf.json",
		"sles-sap-15":         "suse/translate_sles_sap_15.wf.json",
		"sles-sap-15-byol":    "suse/translate_sles_sap_15_byol.wf.json",
		"ubuntu-1404":         "ubuntu/translate_ubuntu_1404.wf.json",
		"ubuntu-1604":         "ubuntu/translate_ubuntu_1604.wf.json",
		"ubuntu-1804":         "ubuntu/translate_ubuntu_1804.wf.json",
		"ubuntu-2004":         "ubuntu/translate_ubuntu_2004.wf.json",
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
		"windows-2022":        "windows/translate_windows_2022.wf.json",
		"windows-2022-byol":   "windows/translate_windows_2022_byol.wf.json",
		"windows-7-x64-byol":  "windows/translate_windows_7_x64_byol.wf.json",
		"windows-7-x86-byol":  "windows/translate_windows_7_x86_byol.wf.json",
		"windows-8-x64-byol":  "windows/translate_windows_8_x64_byol.wf.json",
		"windows-8-x86-byol":  "windows/translate_windows_8_x86_byol.wf.json",
		"windows-10-x64-byol": "windows/translate_windows_10_x64_byol.wf.json",
		"windows-10-x86-byol": "windows/translate_windows_10_x86_byol.wf.json",

		// Legacy:
		"windows-7-byol":       "windows/translate_windows_7_x64_byol.wf.json",
		"windows-8-1-x64-byol": "windows/translate_windows_8_x64_byol.wf.json",
		"windows-10-byol":      "windows/translate_windows_10_x64_byol.wf.json",
	}

	for osID, workflowPath := range osChoices {
		t.Run(osID, func(t *testing.T) {
			settings, err := GetTranslationSettings(osID)
			assert.NoError(t, err)
			assert.Equal(t, workflowPath, settings.WorkflowPath)
		})
	}
}

func Test_ComputeServiceAccountVar_SupportedByAllOSes(t *testing.T) {
	workflowDir := "../../../../daisy_workflows/image_import"
	for _, o := range supportedOS {
		t.Run(o.GcloudOsFlag, func(t *testing.T) {
			workflowPath := path.Join(workflowDir, o.WorkflowPath)
			if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
				t.Fatal("Can't find", workflowPath)
			}

			wf, err := daisy.NewFromFile(workflowPath)
			assert.NoError(t, err)
			_, ok := wf.Vars["compute_service_account"]
			assert.True(t, ok, "compute_service_account not supported by %s", o.WorkflowPath)
		})
	}
}

func Test_GetTranslationSettings_ReturnsSameLicenseAsContainedInJSON(t *testing.T) {
	// Originally, the JSON workflows in daisy_workflows/image_import were the source of truth
	// for licensing info. This test verifies that the license returned by GetTranslationSettings
	// is the same as the JSON workflow.

	workflowDir := "../../../../daisy_workflows/image_import"

	if _, err := os.Stat(workflowDir); os.IsNotExist(err) {
		t.Fatal("Can't find", workflowDir)
	}
	for _, osID := range GetSortedOSIDs() {
		t.Run(osID, func(t *testing.T) {
			settings, err := GetTranslationSettings(osID)
			assert.NoError(t, err)
			assert.NotEmpty(t, settings.WorkflowPath)
			assert.Contains(t, settings.LicenseURI, "licenses/")

			workflowPath := path.Join(workflowDir, settings.WorkflowPath)
			if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
				t.Fatal("Can't find", workflowPath)
			}

			// Ensure that the license from TranslationSettings is specified in the
			// JSON workflow.
			var licensesInWorkflow []string
			wf, err := daisy.NewFromFile(workflowPath)
			assert.NoError(t, err)

			// SLES workflows put the license in a variable. All others
			// put the license directly in the CreateImage step.
			if strings.Contains(osID, "sles") {
				licensesInWorkflow = []string{wf.Vars["license"].Value}
			} else {
				for _, step := range wf.Steps {
					if step.CreateImages != nil {
						for _, image := range step.CreateImages.Images {
							licensesInWorkflow = image.Licenses
						}
					}
				}
			}
			assert.Contains(t, licensesInWorkflow, settings.LicenseURI)
		})
	}
}

func TestValidateOsValid(t *testing.T) {
	err := ValidateOS("ubuntu-1604")
	if err != nil {
		t.Errorf("expected nil error, got `%v`", err)
	}
}

func TestValidateOsInvalid(t *testing.T) {
	err := ValidateOS("not-an-OS")
	if err == nil {
		t.Errorf("expected non-nil error")
	}
}

func TestGetSortedOSIDs(t *testing.T) {
	actual := GetSortedOSIDs()
	assert.Len(t, actual, len(supportedOS))
	assert.True(t, sort.StringsAreSorted(actual))
	for _, choice := range supportedOS {
		assert.Contains(t, actual, choice.GcloudOsFlag)
	}
}

func TestRemovePrivacyLogInfoNoPrivacyInfo(t *testing.T) {
	testRemovePrivacyLogInfo(t,
		"No privacy info",
		"No privacy info",
		"Regular message should stay the same")

	testRemovePrivacyLogInfo(t,
		"[Privacy-> info",
		"[Privacy-> info",
		"Incomplete privacy info (left bracket) should stay the same")

	testRemovePrivacyLogInfo(t,
		"info <-Privacy]",
		"info <-Privacy]",
		"Incomplete privacy info (right bracket) should stay the same")
}

func TestRemovePrivacyLogInfoTranslateFailed(t *testing.T) {
	testRemovePrivacyLogInfo(t,
		"[DaisyLog] TranslateFailed: OS not detected\nFailed for project my-project",
		"TranslateFailed",
		"TranslateFailed error details should be hidden")
}

func TestRemovePrivacyLogInfoSingle(t *testing.T) {
	testRemovePrivacyLogInfo(t,
		"[Privacy-> info 1 <-Privacy] info 2",
		" info 2",
		"Privacy info (on the head) should be hidden")

	testRemovePrivacyLogInfo(t,
		"info 0 [Privacy-> info 1 <-Privacy]",
		"info 0 ",
		"Privacy info (on the tail) should be hidden")
}

func TestRemovePrivacyLogInfoMultiple(t *testing.T) {
	testRemovePrivacyLogInfo(t,
		"info 0 [Privacy-> info 1 <-Privacy] info 2 [Privacy-> info 3 <-Privacy] info 4",
		"info 0  info 2  info 4",
		"Multiple privacy info should be hidden")
}

func testRemovePrivacyLogInfo(t *testing.T, originalMessage string, expectedMessage string, onFailure string) {
	m := RemovePrivacyLogInfo(originalMessage)
	if m != expectedMessage {
		t.Errorf("%v. Expect: `%v`, actual: `%v`", onFailure, expectedMessage, m)
	}
}

func TestRemovePrivacyTagSingle(t *testing.T) {
	testRemovePrivacyTag(t,
		"[Privacy-> info 1 <-Privacy]",
		" info 1 ",
		"Single privacy tag should be removed")
}

func TestRemovePrivacyTagMultiple(t *testing.T) {
	testRemovePrivacyTag(t,
		"Error: [Privacy->abc <-Privacy] [Privacy-> xyz<-Privacy] and <-Privacy]",
		"Error: abc   xyz and ",
		"Multiple privacy tag should be removed")
}

func testRemovePrivacyTag(t *testing.T, originalMessage string, expectedMessage string, onFailure string) {
	m := RemovePrivacyLogTag(originalMessage)
	if m != expectedMessage {
		t.Errorf("%v. Expect: `%v`, actual: `%v`", onFailure, expectedMessage, m)
	}
}

func TestUpdateToUEFICompatible(t *testing.T) {
	w := createWorkflowWithCreateDiskImageAndIncludeWorkflow()

	UpdateToUEFICompatible(w)

	assert.Equal(t, 1, len((*w.Steps["cd"].CreateDisks)[0].GuestOsFeatures))
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["cd"].CreateDisks)[0].GuestOsFeatures[0].Type)

	assert.Equal(t, 0, len((*w.Steps["cd"].CreateDisks)[1].GuestOsFeatures))

	assert.Equal(t, 1, len((*w.Steps["iw"].IncludeWorkflow.Workflow.Steps["iw-cd"].CreateDisks)[0].GuestOsFeatures))
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["iw"].IncludeWorkflow.Workflow.Steps["iw-cd"].CreateDisks)[0].GuestOsFeatures[0].Type)

	assert.Equal(t, 1, len((*w.Steps["cimg"].CreateImages).Images[0].GuestOsFeatures))
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["cimg"].CreateImages).Images[0].GuestOsFeatures[0])
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["cimg"].CreateImages).Images[0].Image.GuestOsFeatures[0].Type)

	assert.Equal(t, 1, len((*w.Steps["cimg"].CreateImages).ImagesBeta[0].GuestOsFeatures))
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["cimg"].CreateImages).ImagesBeta[0].GuestOsFeatures[0])
	assert.Equal(t, "UEFI_COMPATIBLE", (*w.Steps["cimg"].CreateImages).ImagesBeta[0].Image.GuestOsFeatures[0].Type)
}

func createWorkflowWithCreateDiskImageAndIncludeWorkflow() *daisy.Workflow {
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"cd": {
			CreateDisks: &daisy.CreateDisks{
				{
					Disk: compute.Disk{},
				},
				{
					Disk: compute.Disk{
						SourceImage: "projects/compute-image-tools/global/images/family/debian-9-worker",
					},
				},
			},
		},
		"cimg": {
			CreateImages: &daisy.CreateImages{
				Images: []*daisy.Image{
					{
						Image: compute.Image{
							Name: "final-image-1",
						},
					},
				},
				ImagesBeta: []*daisy.ImageBeta{
					{
						Image: computeBeta.Image{
							Name: "final-image-1",
						},
					},
				},
			},
		},
		"cins": {
			CreateInstances: &daisy.CreateInstances{
				Instances: []*daisy.Instance{
					{
						Instance: compute.Instance{
							Disks: []*compute.AttachedDisk{{Source: "key1"}},
						},
					},
				},
				InstancesBeta: []*daisy.InstanceBeta{
					{
						Instance: computeBeta.Instance{
							Disks: []*computeBeta.AttachedDisk{{Source: "key1"}},
						},
					},
				},
			},
		},
		"iw": {
			IncludeWorkflow: &daisy.IncludeWorkflow{
				Workflow: &daisy.Workflow{
					Steps: map[string]*daisy.Step{
						"iw-cd": {
							CreateDisks: &daisy.CreateDisks{
								{
									Disk: compute.Disk{},
								},
							},
						},
					},
				},
			},
		},
	}

	return w
}

func TestGetResourceID(t *testing.T) {
	type testCase struct {
		testName           string
		resourceName       string
		expectedResourceID string
	}

	tcs := []testCase{
		{"simple resource name", "resname", "resname"},
		{"URI", "path/resname", "resname"},
		{"longer URI", "https://resource/path/resname", "resname"},
	}

	for _, tc := range tcs {
		resourceID := GetResourceID(tc.resourceName)
		if resourceID != tc.expectedResourceID {
			t.Errorf("[%v]: Expected resource ID '%v' != actrual resource ID '%v'", tc.testName, tc.expectedResourceID, resourceID)
		}
	}
}

func TestGetDeviceURI(t *testing.T) {
	uri := GetDeviceURI("p", "z", "d")
	expectedURI := "projects/p/zones/z/devices/d"
	if uri != expectedURI {
		t.Errorf("URI '%v' doesn't match expected '%v'", uri, expectedURI)
	}
}

func TestGetDiskURI(t *testing.T) {
	uri := GetDiskURI("p", "z", "d")
	expectedURI := "projects/p/zones/z/disks/d"
	if uri != expectedURI {
		t.Errorf("URI '%v' doesn't match expected '%v'", uri, expectedURI)
	}
}

func TestGetInstanceURI(t *testing.T) {
	uri := GetInstanceURI("p", "z", "i")
	expectedURI := "projects/p/zones/z/instances/i"
	if uri != expectedURI {
		t.Errorf("URI '%v' doesn't match expected '%v'", uri, expectedURI)
	}
}

func Test_ApplyToWorkflow(t *testing.T) {
	for _, tt := range []struct {
		name               string
		env                EnvironmentSettings
		original, expected *daisy.Workflow
	}{
		{
			name: "always overwrite when source fields are non-empty",
			env: EnvironmentSettings{
				Project:         "lucky-lemur",
				Zone:            "us-west1-c",
				GCSPath:         "new-path",
				OAuth:           "new-oauth",
				Timeout:         "new-timeout",
				ComputeEndpoint: "new-endpoint",
			},
			original: &daisy.Workflow{
				Project:         "original-project",
				Zone:            "original-zone",
				GCSPath:         "original-path",
				OAuthPath:       "original-oauth",
				DefaultTimeout:  "original-timeout",
				ComputeEndpoint: "original-endpoint",
			},
			expected: &daisy.Workflow{
				Project:         "lucky-lemur",
				Zone:            "us-west1-c",
				GCSPath:         "new-path",
				OAuthPath:       "new-oauth",
				DefaultTimeout:  "new-timeout",
				ComputeEndpoint: "new-endpoint",
			},
		},
		{
			name: "project and zone overwrite when empty",
			env:  EnvironmentSettings{},
			original: &daisy.Workflow{
				Project:         "original-project",
				Zone:            "original-zone",
				GCSPath:         "original-path",
				OAuthPath:       "original-oauth",
				DefaultTimeout:  "original-timeout",
				ComputeEndpoint: "original-endpoint",
			},
			expected: &daisy.Workflow{
				GCSPath:         "original-path",
				OAuthPath:       "original-oauth",
				DefaultTimeout:  "original-timeout",
				ComputeEndpoint: "original-endpoint",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.env.ApplyToWorkflow(tt.original)
			assert.Equal(t, tt.original, tt.expected)
		})
	}
}

func Test_ApplyToWorkflow_PropagatesLogging(t *testing.T) {
	original := &daisy.Workflow{}

	// ApplyToWorkflow calls methods to disable logging, which in turn updates private
	// fields on daisy.Workflow. This test inspects private fields directly
	// to validate that logging is disabled.
	privateLoggingFields := []string{"gcsLoggingDisabled", "stdoutLoggingDisabled", "cloudLoggingDisabled"}
	for _, fieldName := range privateLoggingFields {
		realValue := reflect.ValueOf(original).Elem().FieldByName(fieldName)
		assert.False(t, realValue.Bool(), "field: %s", fieldName)
	}

	EnvironmentSettings{
		DisableGCSLogs:    true,
		DisableCloudLogs:  true,
		DisableStdoutLogs: true,
	}.ApplyToWorkflow(original)

	for _, fieldName := range privateLoggingFields {
		realValue := reflect.ValueOf(original).Elem().FieldByName(fieldName)
		assert.True(t, realValue.Bool(), "field: %s", fieldName)
	}
}

func Test_ParseWorkflow_HappyCase(t *testing.T) {
	path := "test_data/test.wf.json"
	varMap := map[string]string{"bootstrap_instance_name": "bootstrap-${NAME}", "key1": "var1", "key2": "var2", "machine_type": "n1-standard-1", "network": "", "subnet": ""}
	project := "project"
	zone := "zone"
	gcsPath := "gcspath"
	oauth := "oauthpath"
	dTimeout := "10m"
	endpoint := "endpoint"
	w, err := ParseWorkflow(path, varMap, project, zone, gcsPath, oauth, dTimeout, endpoint, true,
		true, true)
	if err != nil {
		t.Fatal(err)
	}

	assertWorkflow(t, w, project, zone, gcsPath, oauth, dTimeout, endpoint, varMap)
}

func Test_ParseWorkflow_RaisesErrorWhenInvalidPath(t *testing.T) {
	varMap := map[string]string{"key1": "var1", "key2": "var2", "network": "", "subnet": ""}
	project := "project"
	zone := "zone"
	gcsPath := "gcspath"
	oauth := "oauthpath"
	dTimeout := "10m"
	endpoint := "endpoint"
	w, err := ParseWorkflow("/file/not/found", varMap, project, zone, gcsPath, oauth, dTimeout, endpoint,
		true, true, true)
	assert.Nil(t, w)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "/file/not/found: no such file or directory")
}

func assertWorkflow(t *testing.T, w *daisy.Workflow, project string, zone string, gcsPath string,
	oauth string, dTimeout string, endpoint string, varMap map[string]string) {
	t.Helper()
	tests := []struct {
		want, got interface{}
	}{
		{w.Project, project},
		{w.Zone, zone},
		{w.GCSPath, gcsPath},
		{w.OAuthPath, oauth},
		{w.DefaultTimeout, dTimeout},
		{w.ComputeEndpoint, endpoint},
	}
	for _, tt := range tests {
		if tt.want != tt.got {
			t.Errorf("%v != %v", tt.want, tt.got)
		}
	}
	assertEqualWorkflowVars(t, w, varMap)
}

func assertEqualWorkflowVars(t *testing.T, wf *daisy.Workflow, expectedVars map[string]string) {
	t.Helper()
	actualVars := map[string]string{}
	for k, v := range wf.Vars {
		actualVars[k] = v.Value
	}
	assert.Equal(t, expectedVars, actualVars)
}
