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

package main

import (
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
	"os"
	"testing"
)

func TestGetRegion(t *testing.T) {
	tests := []struct {
		input string
		want string
		err error
	}{
		{"us-central1-c","us-central1", nil},
		{"europe-north1-a", "europe-north1", nil},
		{"europe", "", fmt.Errorf("%v is not a valid zone", "europe")},
		{"", "", fmt.Errorf("zone is empty. Can't determine region")},
	}

	oldZone := zone
	for _, test := range tests {
		zone = &test.input
		got, err := getRegion()
		if test.want != got {
			t.Errorf("%v != %v", test.want, got)
		} else if err != test.err && test.err.Error() != err.Error() {
			t.Errorf("%v != %v", test.err, err)
		}
	}
	zone = oldZone
}

func TestPopulateRegion(t *testing.T) {
	tests := []struct {
		input string
		want string
		err error
	}{
		{"us-central1-c","us-central1", nil},
		{"europe", "", fmt.Errorf("%v is not a valid zone", "europe")},
		{"", "", fmt.Errorf("zone is empty. Can't determine region")},
	}

	oldZone := zone
	for _, test := range tests {
		zone = &test.input
		region = nil
		err := populateRegion()
		fmt.Printf("got err `%v`, expected err `%v`\n", err, test.err)
		if err != test.err && test.err.Error() != err.Error() {
			t.Errorf("%v != %v", test.err, err)
		} else if region!=nil && test.want != *region {
			t.Errorf("%v != %v", test.want, *region)
		}
	}
	zone = oldZone
}

func TestGetWorkflowPathsFromImage(t *testing.T) {
	defer setStringP(&sourceImage, "image-1")()
	workflow, translate := getWorkflowPaths()
	if workflow != importFromImageWorkflow && translate != "" {
		t.Errorf("%v != %v and/or translate not empty", workflow, importFromImageWorkflow)
	}
}

func TestGetWorkflowPathsDataDisk(t *testing.T) {
	defer setBoolP(&dataDisk, true)()
	workflow, translate := getWorkflowPaths()
	if workflow != importWorkflow && translate != "" {
		t.Errorf("%v != %v and/or translate not empty", workflow, importWorkflow)
	}
}

func TestGetWorkflowPathsFromFile(t *testing.T) {
	defer setBoolP(&dataDisk, false)()
	defer setStringP(&sourceImage, "image-1")()
	defer setStringP(&osId, "ubuntu-1404")()

	workflow, translate := getWorkflowPaths()

	if workflow != "ubuntu/translate_ubuntu_1404.wf.json" && translate != "" {
		t.Errorf("%v != %v and/or translate not empty", workflow, "ubuntu/translate_ubuntu_1404.wf.json")
	}
}

func TestFlagsImageNameNotProvided(t *testing.T) {
	err := validateFlags()
	expected := fmt.Errorf("The flag -image_name must be provided")
	if err != expected && err.Error() != expected.Error() {
		t.Errorf("%v != %v", err, expected)
	}
}

func TestFlagsSourceFile(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgs(cliArgs)
	t.Logf("osArgs %v", os.Args)

	if err := validateFlags(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsInvalidSourceFile(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	cliArgs["source_file"] = "invalidSourceFile"
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgs(cliArgs)
	t.Logf("osArgs %v", os.Args)

	if err := validateFlags(); err == nil {
		t.Errorf("Expected error")
	}
}

func TestFlagsSourceImage(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "source_file", &sourceFile)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgs(cliArgs)
	t.Logf("osArgs %v", os.Args)

	if err := validateFlags(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsDataDisk(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	defer clearStringFlag(cliArgs, "os", &osId)()
	buildOsArgs(cliArgs)
	t.Logf("osArgs %v", os.Args)

	if err := validateFlags(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsInvalidOS(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	cliArgs["os"] = "invalidOs"
	buildOsArgs(cliArgs)
	t.Logf("osArgs %v", os.Args)

	if err := validateFlags(); err == nil {
		t.Errorf("Expected error")
	}
}

func TestUpdateWorkflowInstancesLabelled(t *testing.T) {
	defer setBoolP(&noExternalIP, false)()
	buildId = "abc"

	w := daisy.New()
	extraLabels := map[string]string{"labelKey": "labelValue"}
	w.Steps =  map[string]*daisy.Step{
		"ci": {
			CreateInstances: &daisy.CreateInstances{
				{
					Instance: compute.Instance{
						Disks: []*compute.AttachedDisk{{Source: "key1"}},
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Instance: compute.Instance{
						Disks: []*compute.AttachedDisk{{Source: "key2"}},
					},
				},
			},
		},
	}

	updateWorkflow(w)
	validateLabels(&(*w.Steps["ci"].CreateInstances)[0].Instance.Labels,"gce-image-import-tmp", t, &extraLabels)
	validateLabels(&(*w.Steps["ci"].CreateInstances)[1].Instance.Labels,"gce-image-import-tmp", t)
}

func TestUpdateWorkflowDisksLabelled(t *testing.T) {
	defer setBoolP(&noExternalIP, false)()
	buildId = "abc"

	w := daisy.New()
	extraLabels := map[string]string{"labelKey": "labelValue"}
	w.Steps =  map[string]*daisy.Step{
		"cd": {
			CreateDisks: &daisy.CreateDisks{
				{
					Disk: compute.Disk{
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Disk: compute.Disk{},
				},
			},
		},
	}

	updateWorkflow(w)
	validateLabels(&(*w.Steps["cd"].CreateDisks)[0].Disk.Labels,"gce-image-import-tmp", t, &extraLabels)
	validateLabels(&(*w.Steps["cd"].CreateDisks)[1].Disk.Labels,"gce-image-import-tmp", t)
}

func TestUpdateWorkflowImagesLabelled(t *testing.T) {
	defer setBoolP(&noExternalIP, false)()
	buildId = "abc"

	w := daisy.New()
	extraLabels := map[string]string{"labelKey": "labelValue"}
	w.Steps =  map[string]*daisy.Step{
		"cimg": {
			CreateImages: &daisy.CreateImages{
				{
					Image: compute.Image{
						Name: "final-image-1",
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Image: compute.Image{
						Name: "final-image-2",
					},
				},
				{
					Image: compute.Image{
						Name: "untranslated-image-1",
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Image: compute.Image{
						Name: "untranslated-image-2",
					},
				},

			},
		},
	}

	updateWorkflow(w)

	validateLabels(&(*w.Steps["cimg"].CreateImages)[0].Image.Labels, "gce-image-import", t, &extraLabels)
	validateLabels(&(*w.Steps["cimg"].CreateImages)[1].Image.Labels, "gce-image-import", t)

	validateLabels(&(*w.Steps["cimg"].CreateImages)[2].Image.Labels, "gce-image-import-tmp", t, &extraLabels)
	validateLabels(&(*w.Steps["cimg"].CreateImages)[3].Image.Labels, "gce-image-import-tmp", t)
}

func TestUpdateWorkflowInstancesConfiguredForNoExternalIP(t *testing.T) {
	defer setBoolP(&noExternalIP, true)()

	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	updateWorkflow(w)

	if len((*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces[0].AccessConfigs) != 0 {
		t.Errorf("Instance AccessConfigs not empty")
	}
}

func TestUpdateWorkflowInstancesNotModifiedIfExternalIPAllowed(t *testing.T) {
	defer setBoolP(&noExternalIP, false)()

	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	updateWorkflow(w)

	if len((*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces[0].AccessConfigs) != 1 {
		t.Errorf("Instance AccessConfigs doesn't have exactly one instance")
	}
}

func TestBuildDaisyVars(t *testing.T) {
	defer setStringP(&imageName, "image-a")()
	defer setBoolP(&noGuestEnvironment, true)()
	defer setStringP(&sourceFile, "source-file-path")()
	defer setStringP(&family, "a-family")()
	defer setStringP(&description, "a-description")()
	defer setStringP(&network, "a-network")()
	defer setStringP(&subnet, "a-subnet")()
	defer setStringP(&region, "a-region")()

	got := buildDaisyVars("translate/workflow/path")

	assertEqual(got["image_name"], "image-a", t)
	assertEqual(got["translate_workflow"], "translate/workflow/path", t)
	assertEqual(got["install_gce_packages"], "false", t)
	assertEqual(got["source_disk_file"], "source-file-path", t)
	assertEqual(got["family"], "a-family", t)
	assertEqual(got["description"], "a-description", t)
	assertEqual(got["import_network"], "global/networks/a-network", t)
	assertEqual(got["import_subnet"], "regions/a-region/subnetworks/a-subnet", t)
}

func assertEqual(i1 interface{}, i2 interface{}, t *testing.T) {
	if i1 != i2 {
		t.Errorf("%v != %v", i1, i2)
	}
}

func createWorkflowWithCreateInstanceNetworkAccessConfig() *daisy.Workflow {
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"ci": {
			CreateInstances: &daisy.CreateInstances{
				{
					Instance: compute.Instance{
						Disks: []*compute.AttachedDisk{{Source: "key1"}},
						NetworkInterfaces: []*compute.NetworkInterface{
							{
								Network: "n",
								AccessConfigs: []*compute.AccessConfig{
									{Type: "ONE_TO_ONE_NAT"},
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

func validateLabels(labels *map[string]string, typeLabel string, t *testing.T, extraLabelsArr ...*map[string]string) {
	var extraLabels *map[string]string
	if len(extraLabelsArr) > 0 {
		extraLabels = extraLabelsArr[0]
	}
	extraLabelCount := 0
	if extraLabels != nil {
		extraLabelCount = len(*extraLabels)
	}
	if len(*labels) != extraLabelCount + 2 {
		t.Errorf("Labels %v should have only 2 elements", labels)
	}
	got := (*labels)["gce-image-import-build-id"]
	if buildId != got {
		t.Errorf("%v != %v", buildId, got)
	}
	got = (*labels)[typeLabel]
	if "true" != got {
		t.Errorf("%v not true, %v instead", typeLabel, got)
	}

	if extraLabels != nil {
		for extraKey, extraValue := range *extraLabels {
			if value, ok := (*labels)[extraKey]; !ok || value != extraValue {
				t.Errorf("Key %v from labels missing or value %v not matching", extraKey, value)
			}
		}
	}
}

func backupOsArgs() func() {
	oldArgs := os.Args
	return func() { os.Args = oldArgs }
}

func buildOsArgs(cliArgs map[string]interface{}) {
	os.Args = make([]string, len(cliArgs)+1)
	i := 0
	os.Args[i] = "cmd"
	i++
	for key, value := range cliArgs {
		if value != nil {
			os.Args[i] = formatCliArg(key, value)
			i++
		}
	}
}

func formatCliArg(argKey, argValue interface{}) string {
	if argValue == true {
		return fmt.Sprintf("-%v", argKey)
	}
	if argValue != false {
		return fmt.Sprintf("-%v=%v", argKey, argValue)
	}
	return ""
}

func getAllCliArgs() map[string]interface{} {
	return map[string]interface{} {
		imageNameFlagKey: "img",
		clientIdFlagKey: "aClient",
		"data_disk": true,
		"os": "ubuntu-1404",
		"source_file": "gs://source_bucket/source_file",
		"source_image": "anImage",
		"no_guest_environment": true,
		"family": "aFamily",
		"description": "aDescription",
		"network": "aNetwork",
		"subnet": "aSubnet",
		"timeout": "2h",
		"zone": "us-central1-c",
		"project": "aProject",
		"scratch_bucket_gcs_path": "gs://bucket/folder",
		"oauth": "oAuthFilePath",
		"compute_endpoint_override": "us-east1-c",
		"disable_gcs_logging": true,
		"disable_cloud_logging": true,
		"disable_stdout_logging": true,
		"kms_key": "aKmsKey",
		"kms_keyring": "aKmsKeyRing",
		"kms_location": "aKmsLocation",
		"kms_project": "aKmsProject",
	}
}

func setStringP(p **string, value string) func() {
	oldValue := *p
	*p = &value
	return func() {
		*p = oldValue
	}
}

func setBoolP(p **bool, value bool) func() {
	oldValue := *p
	*p = &value
	return func() {*p = oldValue}
}

func clearStringFlag(cliArgs map[string]interface{}, flagKey string, flag **string) func() {
	delete(cliArgs, flagKey)
	return setStringP(flag, "")
}

func clearBoolFlag(cliArgs map[string]interface{}, flagKey string, flag **bool) func() {
	delete(cliArgs, flagKey)
	return setBoolP(flag, false)
}
