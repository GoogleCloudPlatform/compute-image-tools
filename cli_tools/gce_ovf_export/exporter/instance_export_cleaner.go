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

package ovfexporter

import (
	"fmt"
	"strings"

	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

type instanceExportCleanerImpl struct {
	wf               *daisy.Workflow
	attachDiskWfs    []*daisy.Workflow
	startInstanceWf  *daisy.Workflow
	logger           logging.Logger
	wfPreRunCallback wfCallback
}

// NewInstanceExportCleaner creates a new instance export cleaner which is
// responsible for bringing the exported VM back to its pre-export state
func NewInstanceExportCleaner(logger logging.Logger) ovfexportdomain.InstanceExportCleaner {
	return &instanceExportCleanerImpl{logger: logger}
}

func (iec *instanceExportCleanerImpl) init(instance *compute.Instance, params *ovfexportdomain.OVFExportArgs) error {
	// don't use default timeout as it might not be long enough for cleanup,
	// e.g. if it's very short (e.g. 10s)
	wfTimeout := "10m"
	attachedDisks := instance.Disks

	for _, attachedDisk := range attachedDisks {
		attachDiskWf, err := generateWorkflowWithSteps(fmt.Sprintf("ovf-export-clean-attach-disk-%v", attachedDisk.DeviceName),
			wfTimeout,
			func(w *daisy.Workflow) error {
				iec.addAttachDiskStepStep(w, instance, params, attachedDisk)
				return nil
			})
		if err != nil {
			return err
		}
		iec.attachDiskWfs = append(iec.attachDiskWfs, attachDiskWf)

	}

	wasInstanceRunningBeforeExport := isInstanceRunning(instance)
	if wasInstanceRunningBeforeExport {
		var err error
		iec.startInstanceWf, err = generateWorkflowWithSteps("ovf-export-clean-start-instance", wfTimeout,
			func(w *daisy.Workflow) error {
				if wasInstanceRunningBeforeExport {
					iec.addStartInstanceStep(w, instance, params)
				}
				return nil
			})
		if err != nil {
			return err
		}
	}

	return nil
}

func (iec *instanceExportCleanerImpl) addStartInstanceStep(w *daisy.Workflow,
	instance *compute.Instance, params *ovfexportdomain.OVFExportArgs) {
	stepName := "start-instance"
	startInstanceStep := daisy.NewStepDefaultTimeout(stepName, w)
	startInstanceStep.StartInstances = &daisy.StartInstances{Instances: []string{getInstancePath(instance, params.Project)}}
	w.Steps[stepName] = startInstanceStep
}

func (iec *instanceExportCleanerImpl) addAttachDiskStepStep(w *daisy.Workflow,
	instance *compute.Instance, params *ovfexportdomain.OVFExportArgs, attachedDisk *compute.AttachedDisk) {
	diskPath := attachedDisk.Source[strings.Index(attachedDisk.Source, "projects/"):]
	attachDiskStepName := "attach-disk"
	attachDiskStep := daisy.NewStepDefaultTimeout(attachDiskStepName, w)
	attachDiskStep.AttachDisks = &daisy.AttachDisks{
		{
			Instance: getInstancePath(instance, params.Project),
			AttachedDisk: compute.AttachedDisk{
				Mode:       attachedDisk.Mode,
				Source:     diskPath,
				Boot:       attachedDisk.Boot,
				DeviceName: attachedDisk.DeviceName,
			},
		},
	}
	w.Steps[attachDiskStepName] = attachDiskStep
}

func (iec *instanceExportCleanerImpl) Clean(instance *compute.Instance, params *ovfexportdomain.OVFExportArgs) error {
	err := iec.init(instance, params)
	if err != nil {
		return err
	}
	for _, attachDiskWf := range iec.attachDiskWfs {
		if iec.wfPreRunCallback != nil {
			iec.wfPreRunCallback(attachDiskWf)
		}
		// ignore errors as these will be due to instance being already started or disks already attached
		_ = daisyutils.NewDaisyWorker(func() (*daisy.Workflow, error) {
			return attachDiskWf, nil
		}, params.EnvironmentSettings(attachDiskWf.Name), iec.logger).Run(map[string]string{})
	}
	if iec.startInstanceWf != nil {
		if iec.wfPreRunCallback != nil {
			iec.wfPreRunCallback(iec.startInstanceWf)
		}
		_ = daisyutils.NewDaisyWorker(func() (*daisy.Workflow, error) {
			return iec.startInstanceWf, nil
		}, params.EnvironmentSettings(iec.startInstanceWf.Name), iec.logger).Run(map[string]string{})
	}
	return err
}

func (iec *instanceExportCleanerImpl) Cancel(reason string) bool {
	// cleaner is not cancelable
	return false
}
