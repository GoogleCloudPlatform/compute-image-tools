//  Copyright 2019 Google Inc. All Rights Reserved.
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

package ovfimporter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const (
	// InstanceNameFlagKey is key for instance name CLI flag
	InstanceNameFlagKey = "instance-names"

	// MachineImageNameFlagKey is key for machine image name CLI flag
	MachineImageNameFlagKey = "machine-image-name"

	// MachineImageStorageLocationFlagKey is key for machine image storage location CLI flag
	MachineImageStorageLocationFlagKey = "machine-image-storage-location"

	// ClientIDFlagKey is key for client ID CLI flag
	ClientIDFlagKey = "client-id"

	// OvfGcsPathFlagKey is key for OVF/OVA GCS path CLI flag
	OvfGcsPathFlagKey = "ovf-gcs-path"

	// HostnameFlagKey is key for hostname CLI flag
	HostnameFlagKey = "hostname"

	// Prefix for valid instance access config scopes
	instanceAccessScopePrefix = "https://www.googleapis.com/auth/"
)

// ParamValidatorAndPopulator validates parameters and infers missing values.
type ParamValidatorAndPopulator struct {
	metadataClient        domain.MetadataGCEInterface
	zoneValidator         domain.ZoneValidatorInterface
	bucketIteratorCreator domain.BucketIteratorCreatorInterface
	storageClient         domain.StorageClientInterface
	NetworkResolver       param.NetworkResolver
	logger                logging.Logger
}

// ValidateAndPopulate validates OVFImportParams, and populates values that are missing.
// It returns an error if params are invalid.
func (p *ParamValidatorAndPopulator) ValidateAndPopulate(params *ovfdomain.OVFImportParams) (err error) {
	params.BuildID = p.generateBuildIDIfMissing(params.BuildID)

	if params.Project, err = p.lookupProjectIfMissing(*params.Project); err != nil {
		return err
	}

	if params.Zone, err = p.lookupZoneIfMissing(*params.Project, params.Zone); err != nil {
		return err
	}

	if params.Region, err = p.getRegionFromZone(params.Zone); err != nil {
		return err
	}

	if params.Network, params.Subnet, err = p.NetworkResolver.Resolve(
		params.Network, params.Subnet, params.Region, *params.Project); err != nil {
		return err
	}

	if params.ReleaseTrack, err = p.resolveReleaseTrack(params.ReleaseTrack); err != nil {
		return err
	}

	if params.Deadline, err = p.calculateDeadlineFromTimeout(params.Timeout); err != nil {
		return err
	}

	if params.ScratchBucketGcsPath, err = p.createScratchBucketIfMissing(
		params.ScratchBucketGcsPath, *params.Project, params.Region, params.BuildID); err != nil {
		return err
	}

	params.InstanceNames = strings.ToLower(strings.TrimSpace(params.InstanceNames))
	params.MachineImageName = strings.ToLower(strings.TrimSpace(params.MachineImageName))
	params.Description = strings.TrimSpace(params.Description)
	params.PrivateNetworkIP = strings.TrimSpace(params.PrivateNetworkIP)
	params.NetworkTier = strings.TrimSpace(params.NetworkTier)
	params.ComputeServiceAccount = strings.TrimSpace(params.ComputeServiceAccount)
	params.InstanceServiceAccount = strings.TrimSpace(params.InstanceServiceAccount)
	params.InstanceAccessScopesFlag = strings.TrimSpace(params.InstanceAccessScopesFlag)
	if params.InstanceAccessScopesFlag != "" {
		params.InstanceAccessScopes = strings.Split(params.InstanceAccessScopesFlag, ",")
		for _, scope := range params.InstanceAccessScopes {
			if !strings.HasPrefix(scope, instanceAccessScopePrefix) {
				return daisy.Errf("Scope `%v` is invalid because it doesn't start with `%v`", scope, instanceAccessScopePrefix)
			}
		}
	} else {
		params.InstanceAccessScopes = []string{}
	}

	if params.InstanceNames == "" && params.MachineImageName == "" {
		return daisy.Errf("Either the flag -%v or -%v must be provided", InstanceNameFlagKey, MachineImageNameFlagKey)
	}

	if params.InstanceNames != "" && params.MachineImageName != "" {
		return daisy.Errf("-%v and -%v can't be provided at the same time", InstanceNameFlagKey, MachineImageNameFlagKey)
	}

	if params.IsInstanceImport() {
		// instance import specific validation
		instanceNameSplits := strings.Split(params.InstanceNames, ",")
		if len(instanceNameSplits) > 1 {
			return daisy.Errf("OVF import doesn't support multi instance import at this time")
		}

		if params.MachineImageStorageLocation != "" {
			return daisy.Errf("-%v can't be provided when importing an instance", MachineImageStorageLocationFlagKey)
		}
	}

	if err := validation.ValidateStringFlagNotEmpty(params.OvfOvaGcsPath, OvfGcsPathFlagKey); err != nil {
		return err
	}

	if _, err := storageutils.GetBucketNameFromGCSPath(params.OvfOvaGcsPath); err != nil {
		return daisy.Errf("%v should be a path to OVF or OVA package in Cloud Storage", OvfGcsPathFlagKey)
	}

	if params.Labels != "" {
		var err error
		params.UserLabels, err = param.ParseKeyValues(params.Labels)
		if err != nil {
			return err
		}
	}

	if params.Tags != "" {
		params.UserTags = strings.Split(params.Tags, ",")
		for _, tag := range params.UserTags {
			if tag == "" {
				return errors.New("tags cannot be empty")
			}
			if err := validation.ValidateRfc1035Label(tag); err != nil {
				return fmt.Errorf("tag `%v` is not RFC1035 compliant", tag)
			}
		}
	}

	if params.NodeAffinityLabelsFlag != nil {
		var err error
		params.NodeAffinities, params.NodeAffinitiesBeta, err = compute.ParseNodeAffinityLabels(params.NodeAffinityLabelsFlag)
		if err != nil {
			return err
		}
	}

	if params.Hostname != "" {
		err := validation.ValidateFqdn(params.Hostname, HostnameFlagKey)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *ParamValidatorAndPopulator) lookupProjectIfMissing(originalProject string) (*string, error) {
	project, err := param.GetProjectID(p.metadataClient, strings.TrimSpace(originalProject))
	return &project, err
}

func (p *ParamValidatorAndPopulator) getRegionFromZone(zone string) (string, error) {
	zoneSplits := strings.Split(zone, "-")
	if len(zoneSplits) < 2 {
		return "", daisy.Errf("%v is not a valid zone", zone)
	}
	return strings.Join(zoneSplits[:len(zoneSplits)-1], "-"), nil
}

func (p *ParamValidatorAndPopulator) lookupZoneIfMissing(project, originalZone string) (zone string, err error) {
	zone = strings.TrimSpace(originalZone)
	if zone != "" {
		if err := p.zoneValidator.ZoneValid(project, zone); err != nil {
			return "", err
		}
		return zone, nil
	}
	if !p.metadataClient.OnGCE() {
		return "", fmt.Errorf("zone cannot be determined because build is not running on Google Compute Engine")
	}
	// determine zone based on the zone Cloud Build is running in
	zone, err = p.metadataClient.Zone()
	if err != nil || zone == "" {
		return "", fmt.Errorf("can't infer zone: %v", err)
	}
	return zone, nil
}

func (p *ParamValidatorAndPopulator) createScratchBucketIfMissing(originalBucket, project, region, buildID string) (scratchBucket string, err error) {
	if originalBucket != "" {
		scratchBucket = originalBucket
	} else {
		safeProjectName := strings.Replace(project, "google", "elgoog", -1)
		safeProjectName = strings.Replace(safeProjectName, ":", "-", -1)
		if strings.HasPrefix(safeProjectName, "goog") {
			safeProjectName = strings.Replace(safeProjectName, "goog", "ggoo", 1)
		}
		bucket := strings.ToLower(safeProjectName + "-ovf-import-bkt-" + region)
		it := p.bucketIteratorCreator.CreateBucketIterator(context.Background(), p.storageClient, project)
		var projectHasBucket bool
		for itBucketAttrs, err := it.Next(); err != iterator.Done; itBucketAttrs, err = it.Next() {
			if err != nil {
				return "", err
			}
			if itBucketAttrs.Name == bucket {
				projectHasBucket = true
				break
			}
		}
		if !projectHasBucket {
			p.logger.User(fmt.Sprintf("Creating scratch bucket `%v` in %v region", bucket, region))
			if err := p.storageClient.CreateBucket(
				bucket, project,
				&storage.BucketAttrs{Name: bucket, Location: region}); err != nil {
				return "", err
			}
		}
		scratchBucket = fmt.Sprintf("gs://%v/", bucket)
	}

	return pathutils.JoinURL(scratchBucket, buildID), nil
}

func (p *ParamValidatorAndPopulator) resolveReleaseTrack(releaseTrack string) (string, error) {
	if releaseTrack == "" {
		releaseTrack = ovfdomain.GA
	}
	if releaseTrack != ovfdomain.Alpha && releaseTrack != ovfdomain.Beta && releaseTrack != ovfdomain.GA {
		return "", daisy.Errf("invalid value for release-track flag: %v", releaseTrack)
	}
	return releaseTrack, nil
}

func (p *ParamValidatorAndPopulator) calculateDeadlineFromTimeout(originalTimeout string) (time.Time, error) {
	timeout, err := time.ParseDuration(originalTimeout)
	if err != nil {
		return time.Time{}, daisy.Errf("Error parsing timeout `%v`", originalTimeout)
	}
	return time.Now().Add(timeout), nil
}

func (p *ParamValidatorAndPopulator) generateBuildIDIfMissing(originalBuildID string) string {
	if originalBuildID != "" {
		return originalBuildID
	}
	buildID := os.Getenv("BUILD_ID")
	if buildID == "" {
		buildID = pathutils.RandString(5)
	}
	return buildID
}
