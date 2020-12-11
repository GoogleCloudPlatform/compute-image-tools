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
	"log"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	daisy_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
	"google.golang.org/api/compute/v1"
)

// processPlanner determines which actions need to be performed against a disk during processing
// to ensure a disk is bootable on GCE.
type processPlanner interface {
	plan(pd persistentDisk) (*processingPlan, error)
}

// newProcessPlanner returns a processPlanner that prioritizes information from ImportArguments,
// but falls back to disk.Inspector results when required.
func newProcessPlanner(args ImportArguments, diskInspector disk.Inspector) processPlanner {
	return &defaultPlanner{args, diskInspector}
}

// processingPlan describes the metadata and translation steps that need to be performed
// against a disk to make it bootable on GCE.
type processingPlan struct {
	requiredLicenses        []string
	requiredFeatures        []*compute.GuestOsFeature
	translationWorkflowPath string
}

// metadataChangesRequired returns whether metadata needs to be updated on the
// GCE disk resource object.
func (plan *processingPlan) metadataChangesRequired() bool {
	return len(plan.requiredLicenses) > 0 || len(plan.requiredFeatures) > 0
}

type defaultPlanner struct {
	args          ImportArguments
	diskInspector disk.Inspector
}

func (p *defaultPlanner) plan(pd persistentDisk) (*processingPlan, error) {
	// Don't run inspection if the user specified a custom workflow.
	if p.args.CustomWorkflow != "" {
		return &processingPlan{translationWorkflowPath: p.args.CustomWorkflow}, nil
	}

	inspectionResults, inspectionError := p.inspectDisk(pd.uri)

	osID := p.args.OS
	requiresUEFI := p.args.UefiCompatible
	if inspectionError == nil && inspectionResults != nil {
		if osID == "" && inspectionResults.GetOsCount() == 1 && inspectionResults.GetOsRelease() != nil {
			osID = inspectionResults.GetOsRelease().CliFormatted
			if p.args.BYOL {
				osID += "-byol"
			}
		}

		if !requiresUEFI {
			hybridGPTBootable := inspectionResults.GetUefiBootable() && inspectionResults.GetBiosBootable()
			if hybridGPTBootable {
				log.Printf("This disk can boot with either BIOS or a UEFI bootloader. The default setting for booting is BIOS. " +
					"If you want to boot using UEFI, please see https://cloud.google.com/compute/docs/import/importing-virtual-disks#importing_a_virtual_disk_with_uefi_bootloader'.")
			}
			requiresUEFI = inspectionResults.GetUefiBootable() && !hybridGPTBootable
		}
	}

	if osID == "" {
		return nil, errors.New("Could not detect operating system. Please re-import with the operating system specified. " +
			"For more information, see https://cloud.google.com/compute/docs/import/importing-virtual-disks#bootable")
	}

	settings, err := daisy_utils.GetTranslationSettings(osID)
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
		translationWorkflowPath: path.Join(p.args.WorkflowDir, "image_import", settings.WorkflowPath),
	}, nil
}

func (p *defaultPlanner) inspectDisk(uri string) (*pb.InspectionResults, error) {
	log.Printf("Running disk inspections on %v.", uri)
	ir, err := p.diskInspector.Inspect(uri, p.args.Inspect)
	if err != nil {
		log.Printf("Disk inspection error=%v", err)
		return ir, daisy.Errf("Disk inspection error: %v", err)
	}

	log.Printf("Disk inspection result=%v", ir)
	return ir, nil
}
