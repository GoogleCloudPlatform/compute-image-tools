//  Copyright 2020 Google Inc. All Rights Reserved.
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

package importer

import (
	"bytes"
	"os"
	"path"
	"testing"

	daisy_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
)

var opensuse15workflow string

func init() {
	settings, err := daisy_utils.GetTranslationSettings("opensuse-15")
	if err != nil {
		panic(err)
	}
	opensuse15workflow = path.Join("../../../../daisy_workflows/image_import", settings.WorkflowPath)
}

func TestBootableDiskProcessor_Process_WritesSourceDiskVar(t *testing.T) {
	args := ImportArguments{
		OS: "opensuse-15",
	}
	p, err := newBootableDiskProcessor(args, opensuse15workflow)
	assert.NoError(t, err)
	_, err = p.process(persistentDisk{uri: "uri"}, service.NewSingleImageImportLoggableBuilder())
	assert.Equal(t, "uri", p.(*bootableDiskProcessor).workflow.Vars["source_disk"].Value)
}

// gcloud expects log lines to start with the substring "[import". Daisy
// constructs the log prefix using the workflow's name.
func TestBootableDiskProcessor_SetsWorkflowNameToGcloudPrefix(t *testing.T) {
	args := defaultImportArgs()
	processor, e := newBootableDiskProcessor(args, opensuse15workflow)
	assert.NoError(t, e)
	assert.Equal(t, (processor.(*bootableDiskProcessor)).workflow.Name, "import-image")
}

func TestBootableDiskProcessor_PopulatesWorkflowVarsUsingArgs(t *testing.T) {
	imageSpec := defaultImportArgs()
	imageSpec.Description = "Fedora 12 customized"
	imageSpec.Family = "fedora"
	imageSpec.ImageName = "fedora-12-imported"
	imageSpec.Network = "network-copied-verbatum"
	imageSpec.Subnet = "subnet-copied-verbatum"
	imageSpec.NoGuestEnvironment = true
	imageSpec.Region = "us-central"
	imageSpec.SysprepWindows = true

	actual := asMap(createAndRunPrePostFunctions(t, imageSpec).workflow.Vars)
	assert.Equal(t, map[string]string{
		"source_disk":          "", // source_disk is written in process, since a previous processor may create a new disk
		"description":          "Fedora 12 customized",
		"family":               "fedora",
		"image_name":           "fedora-12-imported",
		"import_network":       "network-copied-verbatum",
		"import_subnet":        "subnet-copied-verbatum",
		"install_gce_packages": "false",
		"sysprep":              "true"},
		actual)
}

func TestBootableDiskProcessor_SupportsWorkflowDefaultVars(t *testing.T) {
	actual := asMap(createAndRunPrePostFunctions(t, defaultImportArgs()).workflow.Vars)
	assert.Equal(t, map[string]string{
		"source_disk":          "",
		"description":          "",
		"family":               "",
		"image_name":           "",
		"import_network":       "",
		"import_subnet":        "",
		"install_gce_packages": "true",
		"sysprep":              "false"}, actual)
}

func TestBootableDiskProcessor_SetsWorkerDiskTrackingValues(t *testing.T) {
	userLabels := map[string]string{
		"user-key": "user-val",
	}
	// https://cloud.google.com/cloud-build/docs/configuring-builds/substitute-variable-values
	os.Setenv("BUILD_ID", "build-id")
	imageSpec := defaultImportArgs()
	imageSpec.Labels = userLabels
	actual := createAndRunPrePostFunctions(t, imageSpec)
	disk := getFirstCreatedDisk(t, actual.workflow)

	assert.Equal(t, map[string]string{
		"gce-image-import-build-id": "build-id",
		"gce-image-import-tmp":      "true",
		"user-key":                  "user-val"}, disk.Labels)
}

func TestBootableDiskProcessor_SetsWorkerTrackingValues(t *testing.T) {
	userLabels := map[string]string{
		"user-key": "user-val",
	}
	// https://cloud.google.com/cloud-build/docs/configuring-builds/substitute-variable-values
	os.Setenv("BUILD_ID", "build-id")
	imageSpec := defaultImportArgs()
	imageSpec.Labels = userLabels
	actual := createAndRunPrePostFunctions(t, imageSpec)
	worker := getWorkerInstance(t, actual.workflow)

	assert.Equal(t, map[string]string{
		"gce-image-import-build-id": "build-id",
		"gce-image-import-tmp":      "true",
		"user-key":                  "user-val"}, worker.Labels)
}

func TestBootableDiskProcessor_SetsImageTrackingValues(t *testing.T) {
	userLabels := map[string]string{
		"user-key": "user-val",
	}
	// https://cloud.google.com/cloud-build/docs/configuring-builds/substitute-variable-values
	os.Setenv("BUILD_ID", "build-id")
	imageSpec := defaultImportArgs()
	imageSpec.Labels = userLabels
	actual := createAndRunPrePostFunctions(t, imageSpec)
	image := getImage(t, actual.workflow)

	assert.Equal(t, map[string]string{
		"gce-image-import":          "true",
		"gce-image-import-build-id": "build-id",
		"user-key":                  "user-val"}, image.Labels)
}

func TestBootableDiskProcessor_SupportsStorageLocation(t *testing.T) {
	imageSpec := defaultImportArgs()
	imageSpec.StorageLocation = "north-america"
	actual := createAndRunPrePostFunctions(t, imageSpec)
	image := getImage(t, actual.workflow)

	assert.Equal(t, []string{"north-america"}, image.StorageLocations)
}

func TestBootableDiskProcessor_PermitsUnsetStorageLocation(t *testing.T) {
	actual := createAndRunPrePostFunctions(t, defaultImportArgs())
	image := getImage(t, actual.workflow)

	assert.Empty(t, image.StorageLocations)
}

func TestBootableDiskProcessor_SupportsSerialLogs(t *testing.T) {
	expected := []string{"serials"}
	args := defaultImportArgs()
	translator, e := newBootableDiskProcessor(args, opensuse15workflow)
	realTranslator := translator.(*bootableDiskProcessor)
	realTranslator.workflow.Logger = daisyLogger{
		serials: expected,
	}
	assert.NoError(t, e)
	assert.Equal(t, expected, translator.traceLogs())
}

func TestBootableDiskProcessor_SupportsCancel(t *testing.T) {
	args := defaultImportArgs()
	processor, e := newBootableDiskProcessor(args, opensuse15workflow)
	assert.NoError(t, e)

	realProcessor := processor.(*bootableDiskProcessor)
	realProcessor.cancel("timed-out")
	_, channelOpen := <-realProcessor.workflow.Cancel
	assert.False(t, channelOpen, "realProcessor.workflow.Cancel should be closed on timeout")
}

func createAndRunPrePostFunctions(t *testing.T, args ImportArguments) *bootableDiskProcessor {
	translator, e := newBootableDiskProcessor(args, opensuse15workflow)
	assert.NoError(t, e)
	realTranslator := translator.(*bootableDiskProcessor)
	// A concrete logger is required since the import/export logging framework writes a log entry
	// when the workflow starts. Without this there's a panic.
	realTranslator.workflow.Logger = daisyLogger{}
	realTranslator.preValidateFunc()(realTranslator.workflow)
	realTranslator.postValidateFunc()(realTranslator.workflow)
	return realTranslator
}

func getFirstCreatedDisk(t *testing.T, workflow *daisy.Workflow) daisy.Disk {
	for _, step := range workflow.Steps {
		if step.CreateDisks != nil {
			disks := *step.CreateDisks
			assert.Len(t, disks, 1)
			return *disks[0]
		}
	}
	panic("expected create disks step")
}

func getWorkerInstance(t *testing.T, workflow *daisy.Workflow) daisy.Instance {
	for _, step := range workflow.Steps {
		if step.CreateInstances != nil {
			instances := step.CreateInstances.Instances
			assert.Len(t, instances, 1)
			return *instances[0]
		}
	}
	panic("expected create instance step")
}

func getImage(t *testing.T, workflow *daisy.Workflow) daisy.Image {
	for _, step := range workflow.Steps {
		if step.CreateImages != nil {
			images := step.CreateImages.Images
			assert.Len(t, images, 1)
			return *images[0]
		}
	}
	panic("expected create image step")
}

func defaultImportArgs() ImportArguments {
	return ImportArguments{OS: "opensuse-15"}
}

func asMap(vars map[string]daisy.Var) map[string]string {
	m := map[string]string{}
	for k, v := range vars {
		m[k] = v.Value
	}
	return m
}

type daisyLogger struct {
	serials []string
}

func (d daisyLogger) WriteLogEntry(e *daisy.LogEntry) {

}

func (d daisyLogger) WriteSerialPortLogs(w *daisy.Workflow, instance string, buf bytes.Buffer) {
	panic("unexpected call")
}

func (d daisyLogger) ReadSerialPortLogs() []string {
	return d.serials
}

func (d daisyLogger) Flush() {
	panic("unexpected call")
}
