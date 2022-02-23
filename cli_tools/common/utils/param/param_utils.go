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

package param

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-daisy/compute"
	"google.golang.org/api/option"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
)

// GetProjectID gets project id from flag if exists; otherwise, try to retrieve from GCE metadata.
func GetProjectID(mgce domain.MetadataGCEInterface, projectFlag string) (string, error) {
	if projectFlag == "" {
		if !mgce.OnGCE() {
			return "", daisy.Errf("project cannot be determined because build is not running on GCE")
		}
		aProject, err := mgce.ProjectID()
		if err != nil || aProject == "" {
			return "", daisy.Errf("project cannot be determined %v", err)
		}
		return aProject, nil
	}
	return projectFlag, nil
}

// populateScratchBucketGcsPath validates the scratch bucket, creating a new one if not
// provided, and returns the region of the scratch bucket. If the scratch bucket is
// already populated, and the owning project doesn't match `project`, then an error is returned.
// In that case, if `file` resides in the non-owned scratch bucket and `removeFileWhenScratchNotOwned`
// is specified, then `file` is deleted from GCS.
func populateScratchBucketGcsPath(scratchBucketGcsPath *string, zone string, mgce domain.MetadataGCEInterface,
	scratchBucketCreator domain.ScratchBucketCreatorInterface, file string, project *string,
	storageClient domain.StorageClientInterface, removeFileWhenScratchNotOwned bool) (string, error) {

	scratchBucketRegion := ""
	if *scratchBucketGcsPath == "" {
		fallbackZone := zone
		if fallbackZone == "" && mgce.OnGCE() {
			var err error
			if fallbackZone, err = mgce.Zone(); err != nil {
				// reset fallback zone if failed to get zone from running GCE
				fallbackZone = ""
			}
		}

		scratchBucketName, sbr, err := scratchBucketCreator.CreateScratchBucket(file, *project, fallbackZone)
		scratchBucketRegion = sbr
		if err != nil {
			return "", daisy.Errf("failed to create scratch bucket: %v", err)
		}

		*scratchBucketGcsPath = fmt.Sprintf("gs://%v/", scratchBucketName)
	} else {
		scratchBucketName, err := storage.GetBucketNameFromGCSPath(*scratchBucketGcsPath)
		if err != nil {
			return "", daisy.Errf("invalid scratch bucket GCS path %v", scratchBucketGcsPath)
		}

		if !scratchBucketCreator.IsBucketInProject(*project, scratchBucketName) {
			anonymizedErrorMessage := "Scratch bucket %q is not in project %q"

			substitutions := []interface{}{scratchBucketName, *project}

			if removeFileWhenScratchNotOwned && strings.HasPrefix(file, fmt.Sprintf("gs://%s/", scratchBucketName)) {
				err := storageClient.DeleteObject(file)
				if err == nil {
					anonymizedErrorMessage += ". Deleted %q"
					substitutions = append(substitutions, file)
				} else {
					anonymizedErrorMessage += ". Failed to delete %q: %v. " +
						"Check with the owner of gs://%q for more information"
					substitutions = append(substitutions, file, err, scratchBucketName)
				}
			}

			return "", daisy.Errf(anonymizedErrorMessage, substitutions...)
		}

		scratchBucketAttrs, err := storageClient.GetBucketAttrs(scratchBucketName)
		if err == nil {
			scratchBucketRegion = scratchBucketAttrs.Location
		}
	}
	return scratchBucketRegion, nil
}

// PopulateProjectIfMissing populates project id for cli tools
func PopulateProjectIfMissing(mgce domain.MetadataGCEInterface, projectFlag *string) error {
	var err error
	*projectFlag, err = GetProjectID(mgce, *projectFlag)
	return err
}

// PopulateRegion populates region based on the value extracted from zone param
func PopulateRegion(region *string, zone string) error {
	aRegion, err := paramhelper.GetRegion(zone)
	if err != nil {
		return err
	}
	*region = aRegion
	return nil
}

// CreateComputeClient creates a new compute client
func CreateComputeClient(ctx *context.Context, oauth string, ce string) (daisyCompute.Client, error) {
	computeOptions := []option.ClientOption{option.WithCredentialsFile(oauth)}
	if ce != "" {
		computeOptions = append(computeOptions, option.WithEndpoint(ce))
	}

	computeClient, err := daisyCompute.NewClient(*ctx, computeOptions...)
	if err != nil {
		return nil, daisy.Errf("failed to create compute client: %v", err)
	}
	return computeClient, nil
}

var fullResourceURLPrefix = "https://www.googleapis.com/compute/[^/]*/"
var fullResourceURLRegex = regexp.MustCompile(fmt.Sprintf("^(%s)", fullResourceURLPrefix))

func getResourcePath(scope string, resourceType string, resourceName string) string {
	// handle full URL: transform to relative URL
	if prefix := fullResourceURLRegex.FindString(resourceName); prefix != "" {
		return strings.TrimPrefix(resourceName, prefix)
	}

	// handle relative (partial) URL: use it as-is
	if strings.Contains(resourceName, "/") {
		return resourceName
	}

	// handle pure name: treat it as current project
	return fmt.Sprintf("%v/%v/%v", scope, resourceType, resourceName)
}

// GetImageResourcePath gets the resource path for an image. It will panic if either
// projectID or imageName is invalid. To avoid panic, pre-validate using the
// functions in the `validation` package.
func GetImageResourcePath(projectID, imageName string) string {
	if err := validation.ValidateImageName(imageName); err != nil {
		panic(fmt.Sprintf("Invalid image name %q: %v", imageName, err))
	}
	if err := validation.ValidateProjectID(projectID); err != nil {
		panic(fmt.Sprintf("Invalid projectID %q: %v", projectID, err))
	}
	return fmt.Sprintf("projects/%s/global/images/%s", projectID, imageName)
}

// GetGlobalResourcePath gets global resource path based on either a local resource name or a path
func GetGlobalResourcePath(resourceType string, resourceName string) string {
	return getResourcePath("global", resourceType, resourceName)
}

// GetRegionalResourcePath gets regional resource path based on either a local resource name or a path
func GetRegionalResourcePath(region string, resourceType string, resourceName string) string {
	return getResourcePath(fmt.Sprintf("regions/%v", region), resourceType, resourceName)
}

// GetZonalResourcePath gets zonal resource path based on either a local resource name or a path
func GetZonalResourcePath(zone string, resourceType string, resourceName string) string {
	return getResourcePath(fmt.Sprintf("zones/%v", zone), resourceType, resourceName)
}
