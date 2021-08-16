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
	"errors"
	"fmt"
	"path"
	"strings"

	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/distro"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

// processPlanner determines which actions need to be performed against a disk during processing
// to ensure a disk is bootable on GCE.
type processPlanner interface {
	plan(pd persistentDisk) (*processingPlan, error)
}

// newProcessPlanner returns a processPlanner that prioritizes information from ImageImportRequest,
// but falls back to disk.Inspector results when required.
func newProcessPlanner(request ImageImportRequest, diskInspector disk.Inspector, logger logging.Logger) processPlanner {
	return &defaultPlanner{request, diskInspector, logger}
}

// processingPlan describes the metadata and translation steps that need to be performed
// against a disk to make it bootable on GCE.
type processingPlan struct {
	requiredLicenses        []string
	requiredFeatures        []*compute.GuestOsFeature
	translationWorkflowPath string
	detectedOs              distro.Release
}

// metadataChangesRequired returns whether metadata needs to be updated on the
// GCE disk resource object.
func (plan *processingPlan) metadataChangesRequired() bool {
	return len(plan.requiredLicenses) > 0 || len(plan.requiredFeatures) > 0
}

type defaultPlanner struct {
	request       ImageImportRequest
	diskInspector disk.Inspector
	logger        logging.Logger
}

func (p *defaultPlanner) plan(pd persistentDisk) (*processingPlan, error) {
	// Don't run inspection if the user specified a custom workflow.
	if p.request.CustomWorkflow != "" {
		return &processingPlan{translationWorkflowPath: p.request.CustomWorkflow}, nil
	}

	inspectionResults, inspectionError := p.inspectDisk(pd.uri)
	var detectedOs distro.Release
	osID := p.request.OS
	requiresUEFI := p.request.UefiCompatible
	if inspectionError == nil && inspectionResults != nil {
		if inspectionResults.GetOsCount() == 1 && inspectionResults.GetOsRelease() != nil {
			detectedOs, _ = distro.FromGcloudOSArgument(inspectionResults.GetOsRelease().CliFormatted)
		}
		if osID == "" && inspectionResults.GetOsCount() == 1 && inspectionResults.GetOsRelease() != nil {
			osID = inspectionResults.GetOsRelease().CliFormatted
			if p.request.BYOL {
				osID += "-byol"
			}
		}

		if !requiresUEFI {
			hybridGPTBootable := inspectionResults.GetUefiBootable() && inspectionResults.GetBiosBootable()
			if hybridGPTBootable {
				p.logger.User("The boot disk can boot with either BIOS or a UEFI bootloader. The default setting for booting is BIOS. " +
					"If you want to boot using UEFI, please see https://cloud.google.com/compute/docs/import/importing-virtual-disks#importing_a_virtual_disk_with_uefi_bootloader'.")
			}
			requiresUEFI = inspectionResults.GetUefiBootable() && !hybridGPTBootable
		}
	}

	if osID == "" {
		return nil, errors.New("Could not detect operating system. Please re-import with the operating system specified. " +
			"For more information, see https://cloud.google.com/compute/docs/import/importing-virtual-disks#bootable")
	}

	settings, err := daisyutils.GetTranslationSettings(osID)
	if err != nil {
		return nil, err
	}

	var requiredGuestOSFeatures []*compute.GuestOsFeature
	if strings.Contains(osID, "windows") {
		requiredGuestOSFeatures = append(requiredGuestOSFeatures, &compute.GuestOsFeature{Type: "WINDOWS"})
	}
	if requiresUEFI {
		requiredGuestOSFeatures = append(requiredGuestOSFeatures, &compute.GuestOsFeature{Type: "UEFI_COMPATIBLE"})
	}

	return &processingPlan{
		requiredLicenses:        []string{settings.LicenseURI},
		requiredFeatures:        requiredGuestOSFeatures,
		translationWorkflowPath: path.Join(p.request.WorkflowDir, "image_import", settings.WorkflowPath),
		detectedOs:              detectedOs,
	}, nil
}

func (p *defaultPlanner) inspectDisk(uri string) (*pb.InspectionResults, error) {
	p.logger.User("Inspecting disk for OS and bootloader")
	ir, err := p.diskInspector.Inspect(uri)
	if err != nil {
		p.logger.User(fmt.Sprintf("Disk inspection error=%v", err))
		return ir, daisy.Errf("Disk inspection error: %v", err)
	}
	p.logger.User(fmt.Sprintf("Inspection result=%v", ir))
	return ir, nil
}
