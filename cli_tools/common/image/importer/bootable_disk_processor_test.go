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
	"errors"
	"fmt"
	"path"
	"testing"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/distro"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

var opensuse15workflow string
var ubuntu1804workflow string
var windows2019workflow string

func init() {
	settings, err := daisyutils.GetTranslationSettings("opensuse-15")
	if err != nil {
		panic(err)
	}
	opensuse15workflow = path.Join("../../../../daisy_workflows/image_import", settings.WorkflowPath)
	ubuntu1804workflow = "../../../../daisy_workflows/image_import/ubuntu/translate_ubuntu_1804.wf.json"
	windows2019workflow = "../../../../daisy_workflows/image_import/windows/translate_windows_2019.wf.json"
}

func TestBootableDiskProcessor_Process_WritesSourceDiskVar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	daisyWorker := mocks.NewMockDaisyWorker(ctrl)
	daisyWorker.EXPECT().Run(map[string]string{"source_disk": "uri"})
	processor := &bootableDiskProcessor{
		worker: daisyWorker,
		vars:   map[string]string{},
		logger: logging.NewToolLogger("test"),
	}
	_, err := processor.process(persistentDisk{uri: "uri"})
	assert.NoError(t, err)
}

func TestBootableDiskProcessor_Process_PropagatesErrorFromDaisyFailure(t *testing.T) {
	expectedError := errors.New("failed to process disk")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	daisyWorker := mocks.NewMockDaisyWorker(ctrl)
	daisyWorker.EXPECT().Run(map[string]string{"source_disk": "uri"}).Return(expectedError)
	processor := &bootableDiskProcessor{
		worker: daisyWorker,
		vars:   map[string]string{},
		logger: logging.NewToolLogger("test"),
	}
	_, err := processor.process(persistentDisk{uri: "uri"})
	assert.Equal(t, expectedError, err)
}

func TestBootableDiskProcessor_CreatesExternalIPOnWorker_ByDefault(t *testing.T) {
	args := defaultImportArgs()
	args.NoExternalIP = false
	realProcessor := createProcessor(t, args)
	daisyutils.CheckEnvironment(realProcessor.worker, func(env daisyutils.EnvironmentSettings) {
		assert.False(t, env.NoExternalIP)
	})
}

func TestBootableDiskProcessor_SupportsNoExternalIPForWorker(t *testing.T) {
	args := defaultImportArgs()
	args.NoExternalIP = true
	realProcessor := createProcessor(t, args)
	daisyutils.CheckEnvironment(realProcessor.worker, func(env daisyutils.EnvironmentSettings) {
		assert.True(t, env.NoExternalIP)
	})
}

// gcloud expects log lines to start with the substring "[import". Daisy
// constructs the log prefix using the workflow's name.
func TestBootableDiskProcessor_SetsWorkflowNameToGcloudPrefix(t *testing.T) {
	args := defaultImportArgs()
	args.DaisyLogLinePrefix = "disk-1"
	processor := newBootableDiskProcessor(args, opensuse15workflow, logging.NewToolLogger(t.Name()),
		distro.FromGcloudOSArgumentMustParse("windows-2008r2"))

	daisyutils.CheckEnvironment((processor.(*bootableDiskProcessor)).worker, func(env daisyutils.EnvironmentSettings) {
		assert.Equal(t, "disk-1-translate", env.DaisyLogLinePrefix)
	})
}

func TestBootableDiskProcessor_PopulatesWorkflowVarsUsingArgs(t *testing.T) {
	imageSpec := defaultImportArgs()
	imageSpec.Description = "Fedora 12 customized"
	imageSpec.Family = "fedora"
	imageSpec.ImageName = "fedora-12-imported"
	imageSpec.Network = "network-copied-verbatum"
	imageSpec.Subnet = "subnet-copied-verbatum"
	imageSpec.NoGuestEnvironment = true
	imageSpec.SysprepWindows = true
	imageSpec.ComputeServiceAccount = "csa@email.com"
	diskProcessor := createProcessor(t, imageSpec)
	daisyutils.CheckWorkflow(diskProcessor.worker, func(wf *daisy.Workflow, err error) {
		actual := asMap(wf.Vars)
		assert.Equal(t, map[string]string{
			"source_disk":             "", // source_disk is written in process, since a previous processor may create a new disk
			"description":             "Fedora 12 customized",
			"family":                  "fedora",
			"image_name":              "fedora-12-imported",
			"import_network":          "network-copied-verbatum",
			"import_subnet":           "subnet-copied-verbatum",
			"install_gce_packages":    "false",
			"sysprep":                 "true",
			"compute_service_account": "csa@email.com"},
			actual)
	})
}

func TestBootableDiskProcessor_SupportsWorkflowDefaultVars(t *testing.T) {
	diskProcessor := createProcessor(t, defaultImportArgs())
	daisyutils.CheckWorkflow(diskProcessor.worker, func(wf *daisy.Workflow, err error) {
		actual := asMap(wf.Vars)
		assert.Equal(t, map[string]string{
			"source_disk":             "",
			"description":             "",
			"family":                  "",
			"image_name":              "",
			"import_network":          "",
			"import_subnet":           "",
			"install_gce_packages":    "true",
			"sysprep":                 "false",
			"compute_service_account": "default"}, actual)
	})
}

func TestBootableDiskProcessor_SetsWorkerDiskTrackingValues(t *testing.T) {
	userLabels := map[string]string{
		"user-key": "user-val",
	}
	imageSpec := defaultImportArgs()
	imageSpec.ExecutionID = "build-id"
	imageSpec.Labels = userLabels
	actual := createProcessor(t, imageSpec)

	var wf *daisy.Workflow
	daisyutils.CheckWorkflow(actual.worker, func(w *daisy.Workflow, err error) {
		wf = w
	})
	daisyutils.CheckResourceLabeler(actual.worker, func(rl *daisyutils.ResourceLabeler) {
		assert.NoError(t, rl.PreRunHook(wf))
		disk := getFirstCreatedDisk(t, wf)

		assert.Equal(t, map[string]string{
			"gce-image-import-build-id": "build-id",
			"gce-image-import-tmp":      "true",
			"user-key":                  "user-val"}, disk.Labels)
	})
}

func TestBootableDiskProcessor_SetsWorkerTrackingValues(t *testing.T) {
	userLabels := map[string]string{
		"user-key": "user-val",
	}
	imageSpec := defaultImportArgs()
	imageSpec.ExecutionID = "build-id"
	imageSpec.Labels = userLabels
	actual := createProcessor(t, imageSpec)

	var wf *daisy.Workflow
	daisyutils.CheckWorkflow(actual.worker, func(w *daisy.Workflow, err error) {
		wf = w
	})
	daisyutils.CheckResourceLabeler(actual.worker, func(rl *daisyutils.ResourceLabeler) {
		assert.NoError(t, rl.PreRunHook(wf))
		worker := getWorkerInstance(t, wf)

		assert.Equal(t, map[string]string{
			"gce-image-import-build-id": "build-id",
			"gce-image-import-tmp":      "true",
			"user-key":                  "user-val"}, worker.Labels)
	})
}

func TestBootableDiskProcessor_SetsImageTrackingValues(t *testing.T) {
	userLabels := map[string]string{
		"user-key": "user-val",
	}
	imageSpec := defaultImportArgs()
	imageSpec.ExecutionID = "build-id"
	imageSpec.Labels = userLabels
	actual := createProcessor(t, imageSpec)

	var wf *daisy.Workflow
	daisyutils.CheckWorkflow(actual.worker, func(w *daisy.Workflow, err error) {
		wf = w
	})
	daisyutils.CheckResourceLabeler(actual.worker, func(rl *daisyutils.ResourceLabeler) {
		assert.NoError(t, rl.PreRunHook(wf))
		image := getImage(t, wf)

		assert.Equal(t, map[string]string{
			"gce-image-import":          "true",
			"gce-image-import-build-id": "build-id",
			"user-key":                  "user-val"}, image.Labels)
	})
}

func TestBootableDiskProcessor_SupportsStorageLocation(t *testing.T) {
	imageSpec := defaultImportArgs()
	imageSpec.StorageLocation = "north-america"
	actual := createProcessor(t, imageSpec)
	daisyutils.CheckResourceLabeler(actual.worker, func(rl *daisyutils.ResourceLabeler) {
		assert.Equal(t, "north-america", rl.ImageLocation)
	})
}

func TestBootableDiskProcessor_PermitsUnsetStorageLocation(t *testing.T) {
	actual := createProcessor(t, defaultImportArgs())
	daisyutils.CheckWorkflow(actual.worker, func(wf *daisy.Workflow, err error) {
		image := getImage(t, wf)
		assert.Empty(t, image.StorageLocations)
	})
}

func TestBootableDiskProcessor_SupportsCancel(t *testing.T) {
	args := defaultImportArgs()
	processor := newBootableDiskProcessor(args, opensuse15workflow, logging.NewToolLogger(t.Name()),
		distro.FromGcloudOSArgumentMustParse("windows-2008r2"))

	realProcessor := processor.(*bootableDiskProcessor)
	realProcessor.cancel("timed-out")
	err := realProcessor.worker.Run(map[string]string{})
	assert.EqualError(t, err, "workflow canceled: timed-out")
}

func TestBootableDiskProcessor_AttachDataDisksWithLinux(t *testing.T) {
	args := defaultImportArgs()
	args.CreatedDataDisks = []domain.Disk{}

	for i := 0; i < 3; i++ {
		diskName := fmt.Sprintf("disk-%d", i+1)
		disk, err := disk.NewDisk(args.Project, args.Zone, diskName)
		assert.NoError(t, err)
		args.CreatedDataDisks = append(args.CreatedDataDisks, disk)
	}

	processor := newBootableDiskProcessor(args, ubuntu1804workflow, logging.NewToolLogger(t.Name()),
		distro.FromGcloudOSArgumentMustParse("ubuntu-1804"))

	realProcessor := processor.(*bootableDiskProcessor)

	daisyutils.CheckWorkflow(realProcessor.worker, func(wf *daisy.Workflow, err error) {
		disks := wf.Steps["translate-disk"].IncludeWorkflow.Workflow.Steps["translate-disk-inst"].CreateInstances.Instances[0].Disks
		assert.Equal(t, len(disks), 5)
		for i, disk := range disks[2:] {
			assert.Equal(t, disk.Source, args.CreatedDataDisks[i].GetURI())
		}
	})
}

func TestBootableDiskProcessor_AttachDataDisksWithWindows(t *testing.T) {
	args := defaultImportArgs()
	args.CreatedDataDisks = []domain.Disk{}

	for i := 0; i < 3; i++ {
		diskName := fmt.Sprintf("disk-%d", i+1)
		disk, err := disk.NewDisk(args.Project, args.Zone, diskName)
		assert.NoError(t, err)
		args.CreatedDataDisks = append(args.CreatedDataDisks, disk)
	}

	processor := newBootableDiskProcessor(args, windows2019workflow, logging.NewToolLogger(t.Name()),
		distro.FromGcloudOSArgumentMustParse("windows-2019"))

	realProcessor := processor.(*bootableDiskProcessor)

	daisyutils.CheckWorkflow(realProcessor.worker, func(wf *daisy.Workflow, err error) {
		disks := wf.Steps["import"].IncludeWorkflow.Workflow.Steps["bootstrap"].CreateInstances.Instances[0].Disks
		assert.Equal(t, len(disks), 2)
	})
}

func TestBootableDiskProcessor_AttachDataDisksWithoutInternalWorkflow(t *testing.T) {
	args := defaultImportArgs()

	args.CreatedDataDisks = []domain.Disk{}

	for i := 0; i < 3; i++ {
		diskName := fmt.Sprintf("disk-%d", i+1)
		disk, err := disk.NewDisk(args.Project, args.Zone, diskName)
		assert.NoError(t, err)
		args.CreatedDataDisks = append(args.CreatedDataDisks, disk)
	}

	processor := newBootableDiskProcessor(args, opensuse15workflow, logging.NewToolLogger(t.Name()),
		distro.FromGcloudOSArgumentMustParse("opensuse-15"))

	realProcessor := processor.(*bootableDiskProcessor)

	daisyutils.CheckWorkflow(realProcessor.worker, func(wf *daisy.Workflow, err error) {
		disks := wf.Steps["translate-disk-inst"].CreateInstances.Instances[0].Disks
		assert.Equal(t, len(disks), 5)
		for i, disk := range disks[2:] {
			assert.Equal(t, disk.Source, args.CreatedDataDisks[i].GetURI())
		}
	})
}

func createProcessor(t *testing.T, request ImageImportRequest) *bootableDiskProcessor {
	processor := newBootableDiskProcessor(request, opensuse15workflow, logging.NewToolLogger(t.Name()),
		distro.FromGcloudOSArgumentMustParse("windows-2008r2"))
	realTranslator := processor.(*bootableDiskProcessor)
	// A concrete logger is required since the import/export logging framework writes a log entry
	// when the workflow starts. Without this there's a panic.
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

func defaultImportArgs() ImageImportRequest {
	return ImageImportRequest{
		OS:      "opensuse-15",
		Project: "project-15",
		Zone:    "zone-1",
	}
}

func asMap(vars map[string]daisy.Var) map[string]string {
	m := map[string]string{}
	for k, v := range vars {
		m[k] = v.Value
	}
	return m
}

type daisyLogger struct {
	serials    []string
	logEntries []*daisy.LogEntry
}

func (d *daisyLogger) WriteLogEntry(e *daisy.LogEntry) {
	d.logEntries = append(d.logEntries, e)
}

func (d *daisyLogger) WriteSerialPortLogs(w *daisy.Workflow, instance string, buf bytes.Buffer) {
	panic("unexpected call")
}

func (d *daisyLogger) ReadSerialPortLogs() []string {
	return d.serials
}

func (d *daisyLogger) Flush() {
	panic("unexpected call")
}
