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

package upgrader

import (
	"fmt"
	"regexp"
	"strings"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-daisy/compute"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
)

const (
	rfc1035       = "[a-z]([-a-z0-9]*[a-z0-9])?"
	projectRgxStr = "[a-z]([-.:a-z0-9]*[a-z0-9])?"
)

var (
	instanceURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[2]s)/instances/(?P<instance>%[2]s)$`, projectRgxStr, rfc1035))

	computeClient daisyCompute.Client
	mgce          domain.MetadataGCEInterface
)

// validateAndDeriveParams validates input params, and infers derived params
// from input params. For example, project and zone can be derived from the
// instance URI.
func (u *upgrader) validateAndDeriveParams() error {
	if u.validateAndDeriveParamsFn != nil {
		return u.validateAndDeriveParamsFn()
	}

	if u.derivedVars == nil {
		u.derivedVars = &derivedVars{}
	}

	if err := validateOSVersion(u.SourceOS, u.TargetOS); err != nil {
		return err
	}
	if err := validateAndDeriveInstanceURI(u.Instance, u.ProjectPtr, u.Zone, u.derivedVars); err != nil {
		return err
	}
	if err := validateAndDeriveInstance(u.derivedVars, u.SourceOS, u.TargetOS); err != nil {
		return err
	}

	if u.Timeout == "" {
		u.Timeout = DefaultTimeout
	}

	// Prepare resource names with a random suffix
	suffix := path.RandString(8)
	u.machineImageBackupName = fmt.Sprintf("windows-upgrade-backup-%v", suffix)
	u.osDiskSnapshotName = fmt.Sprintf("windows-upgrade-backup-os-%v", suffix)
	u.newOSDiskName = fmt.Sprintf("windows-upgraded-os-%v", suffix)
	u.installMediaDiskName = fmt.Sprintf("windows-install-media-%v", suffix)

	// Update '-project' flag value for logging purpose.
	// Since '-project' may not be input by user explicitly, we need to populate it
	// when it's referred in order to track usage by project number.
	*u.ProjectPtr = u.instanceProject

	return nil
}

func validateOSVersion(sourceOS, targetOS string) error {
	if sourceOS == "" {
		return daisy.Errf("Flag -source-os must be provided. Please choose a supported version from {%v}.", strings.Join(SupportedVersions, ", "))
	}
	if !isSupportedOSVersion(sourceOS) {
		return daisy.Errf("Flag -source-os value '%v' unsupported. Please choose a supported version from {%v}.", sourceOS, strings.Join(SupportedVersions, ", "))
	}
	if targetOS == "" {
		return daisy.Errf("Flag -target-os must be provided. Please choose a supported version from {%v}.", strings.Join(SupportedVersions, ", "))
	}
	if !isSupportedOSVersion(targetOS) {
		return daisy.Errf("Flag -target-os value '%v' unsupported. Please choose a supported version from {%v}.", targetOS, strings.Join(SupportedVersions, ", "))
	}
	if !isSupportedUpgradePath(sourceOS, targetOS) {
		return daisy.Errf("Can't upgrade from %v to %v. Supported upgrade paths are: %v.", sourceOS, targetOS, strings.Join(getAllUpgradePaths(), ", "))
	}
	return nil
}

func getAllUpgradePaths() []string {
	paths := []string{}
	for sourceOS, targets := range upgradePaths {
		for targetOS, upgradePath := range targets {
			if upgradePath.enabled {
				paths = append(paths, fmt.Sprintf("%v => %v", sourceOS, targetOS))
			}
		}
	}
	return paths
}

func validateAndDeriveInstanceURI(instance string, projectPtr *string, inputZone string, derivedVars *derivedVars) error {
	if instance == "" {
		return daisy.Errf("Flag -instance must be provided")
	}
	derivedVars.instanceURI = instance
	if !strings.Contains(instance, "/") {
		if err := param.PopulateProjectIfMissing(mgce, projectPtr); err != nil {
			return err
		}
		if inputZone == "" {
			return daisy.Errf("--zone must be provided when --instance is not a URI with zone info.")
		}
		derivedVars.instanceURI = daisyutils.GetInstanceURI(*projectPtr, inputZone, instance)
	}

	m := daisy.NamedSubexp(instanceURLRgx, derivedVars.instanceURI)
	if m == nil {
		return daisy.Errf("Please provide the instance flag either with the name of the instance or in the form of 'projects/<project>/zones/<zone>/instances/<instance>', not %s", instance)
	}
	derivedVars.instanceProject = m["project"]
	derivedVars.instanceZone = m["zone"]
	derivedVars.instanceName = m["instance"]
	return nil
}

func validateAndDeriveInstance(derivedVars *derivedVars, sourceOS, targetOS string) error {
	inst, err := computeClient.GetInstance(derivedVars.instanceProject, derivedVars.instanceZone, derivedVars.instanceName)
	if err != nil {
		return daisy.Errf("Failed to get instance: %v", err)
	}

	if len(inst.Disks) == 0 {
		return daisy.Errf("No disks attached to the instance.")
	}
	// Boot disk is always with index=0: https://cloud.google.com/compute/docs/reference/rest/v1/instances/attachDisk
	// "0 is reserved for the boot disk"
	bootDisk := inst.Disks[0]
	if err := validateAndDeriveOSDisk(bootDisk, derivedVars); err != nil {
		return err
	}
	if err := validateLicense(bootDisk, sourceOS, targetOS); err != nil {
		return err
	}

	// We need to launch upgrade by a startup script, whose URL is set by a metadata
	// 'windows-startup-script-url'.
	// If that metadata key has been used by the customer before the upgrade, we need
	// to backup it and restore after the upgrade finished. We backup it to metadata
	// 'windows-startup-script-url-backup'.
	// There are 3 possible scenarios:
	// 1. 'windows-startup-script-url' doesn't exist originally. Which means, the customer
	//    doesn't set it. In that case, we don't need to backup anything.
	// 2. 'windows-startup-script-url' exists. Which means, the customer set it for
	//    their purposes. We should backup it in order to restore from it when cleanup
	//    or rollback.
	// 3. 'windows-startup-script-url' exists, but 'windows-startup-script-url-backup'
	//    also exists. That means the customer tried to run upgrade before but got
	//    interrupted for some reason. In that case, 'windows-startup-script-url'
	//    must have been modified, so we should backup 'windows-startup-script-url-backup'
	//    instead.
	if inst.Metadata != nil && inst.Metadata.Items != nil {
		originalURL := getMetadataValue(inst.Metadata.Items, metadataWindowsStartupScriptURLBackup)
		if originalURL == nil {
			originalURL = getMetadataValue(inst.Metadata.Items, metadataWindowsStartupScriptURL)
		}
		derivedVars.originalWindowsStartupScriptURL = originalURL
	}

	return nil
}

func getMetadataValue(items []*compute.MetadataItems, key string) *string {
	for _, metadataItem := range items {
		if metadataItem.Key == key && metadataItem.Value != nil && *metadataItem.Value != "" {
			return metadataItem.Value
		}
	}
	return nil
}

func validateAndDeriveOSDisk(osDisk *compute.AttachedDisk, derivedVars *derivedVars) error {
	if osDisk.Boot == false {
		return daisy.Errf("The instance has no boot disk.")
	}
	osDiskName := daisyutils.GetResourceID(osDisk.Source)
	d, err := computeClient.GetDisk(derivedVars.instanceProject, derivedVars.instanceZone, osDiskName)
	if err != nil {
		return daisy.Errf("Failed to get boot disk info: %v", err)
	}

	derivedVars.osDiskURI = param.GetZonalResourcePath(derivedVars.instanceZone, "disks", osDisk.Source)
	derivedVars.osDiskDeviceName = osDisk.DeviceName
	derivedVars.osDiskAutoDelete = osDisk.AutoDelete
	derivedVars.osDiskType = daisyutils.GetResourceID(d.Type)
	return nil
}

func validateLicense(osDisk *compute.AttachedDisk, sourceOS, targetOS string) error {
	matchSourceOSVersion := false
	upgraded := false
	for _, lic := range osDisk.Licenses {
		for _, expectedLic := range upgradePaths[sourceOS][targetOS].expectedCurrentLicense {
			if strings.HasSuffix(lic, expectedLic) {
				matchSourceOSVersion = true
			} else if strings.HasSuffix(lic, upgradePaths[sourceOS][targetOS].licenseToAdd) {
				upgraded = true
			}
		}
	}
	if !matchSourceOSVersion {
		return daisy.Errf(fmt.Sprintf("No valid Windows Server PayG license can be found. Any of the following licenses are required: %v", upgradePaths[sourceOS][targetOS].expectedCurrentLicense))
	}
	if upgraded {
		return daisy.Errf(fmt.Sprintf("The GCE instance has the %v license attached. This likely means the instance either has been upgraded or has started an upgrade in the past.", upgradePaths[sourceOS][targetOS].licenseToAdd))
	}
	return nil
}
