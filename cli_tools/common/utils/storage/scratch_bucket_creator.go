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

package storage

import (
	"context"
	"log"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/iterator"
)

const (
	defaultRegion        = "US"
	defaultStorageClass  = "MULTI_REGIONAL"
	regionalStorageClass = "REGIONAL"
)

// ScratchBucketCreator is responsible for creating Daisy scratch bucketets
type ScratchBucketCreator struct {
	StorageClient         domain.StorageClientInterface
	Ctx                   context.Context
	BucketIteratorCreator domain.BucketIteratorCreatorInterface
}

// NewScratchBucketCreator creates a ScratchBucketCreator
func NewScratchBucketCreator(ctx context.Context, storageClient domain.StorageClientInterface) *ScratchBucketCreator {
	return &ScratchBucketCreator{storageClient, ctx, &BucketIteratorCreator{}}
}

// CreateScratchBucket creates scratch bucket in the same region as sourceFileFlag. If failed to
// determine region by source file, we will try to determine region by fallbackZone.
// Returns (bucket_name, region, error)
func (c *ScratchBucketCreator) CreateScratchBucket(sourceFileFlag string, project string,
	fallbackZone string) (string, string, error) {

	bucketAttrs, err := c.getBucketAttrs(sourceFileFlag, project, fallbackZone)
	if err != nil {
		return "", "", err
	}

	region, err := c.createBucketIfNotExisting(project, bucketAttrs)
	if err != nil {
		return "", "", err
	}
	return bucketAttrs.Name, region, nil
}

func (c *ScratchBucketCreator) getBucketAttrs(fileGcsPath string, project string,
	fallbackZone string) (*storage.BucketAttrs, error) {

	if project == "" {
		return nil, daisy.Errf("can't get bucket attributes if project not specified")
	}

	if fileGcsPath != "" {
		// file provided, create bucket in the same region for cost/performance reasons
		return c.getBucketAttrsFromInputFile(fileGcsPath, project, fallbackZone)
	}

	// if file is not provided, fallback to input / default zone.
	return c.getBucketAttrsOnFallbackZone(project, fallbackZone)
}

func (c *ScratchBucketCreator) getBucketAttrsFromInputFile(fileGcsPath string, project string,
	fallbackZone string) (*storage.BucketAttrs, error) {

	fileBucket, err := GetBucketNameFromGCSPath(fileGcsPath)
	if err != nil || fileBucket == "" {
		return nil, daisy.Errf("file GCS path `%v` is invalid: %v", fileGcsPath, err)
	}

	fileBucketAttrs, err := c.StorageClient.GetBucketAttrs(fileBucket)
	if err != nil || fileBucketAttrs == nil {
		// if region can't be determined by bucket (which usually means bucket doesn't exist), then
		// fallback to input / default zone
		return c.getBucketAttrsOnFallbackZone(project, fallbackZone)
	}

	bucket := c.formatScratchBucketName(project, fileBucketAttrs.Location)
	return &storage.BucketAttrs{
		Name:         bucket,
		Location:     fileBucketAttrs.Location,
		StorageClass: fileBucketAttrs.StorageClass,
	}, nil
}

func (c *ScratchBucketCreator) getBucketAttrsOnFallbackZone(project string, fallbackZone string) (*storage.BucketAttrs, error) {
	fallbackRegion := defaultRegion
	storageClass := defaultStorageClass
	var err error
	if fallbackZone != "" {
		if fallbackRegion, err = paramhelper.GetRegion(fallbackZone); err != nil {
			return nil, err
		}
		storageClass = regionalStorageClass
	}
	bucket := c.formatScratchBucketName(project, fallbackRegion)
	return &storage.BucketAttrs{Name: bucket, Location: fallbackRegion, StorageClass: storageClass}, nil
}

func (c *ScratchBucketCreator) createBucketIfNotExisting(project string,
	bucketAttrs *storage.BucketAttrs) (string, error) {

	foundBucketAttrs, err := c.getBucketAttrsIfInProject(project, bucketAttrs.Name)
	if err != nil {
		return "", err
	}
	if foundBucketAttrs != nil {
		return foundBucketAttrs.Location, nil
	}
	log.Printf("Creating scratch bucket `%v` in %v region", bucketAttrs.Name, bucketAttrs.Location)
	if err := c.StorageClient.CreateBucket(bucketAttrs.Name, project, bucketAttrs); err != nil {
		return "", err
	}
	return bucketAttrs.Location, nil
}

// IsBucketInProject checks if bucket belongs to a project
func (c *ScratchBucketCreator) IsBucketInProject(project string, bucketName string) bool {
	foundBucketAttrs, _ := c.getBucketAttrsIfInProject(project, bucketName)
	return foundBucketAttrs != nil
}

func (c *ScratchBucketCreator) getBucketAttrsIfInProject(project string, bucketName string) (*storage.BucketAttrs, error) {
	it := c.BucketIteratorCreator.CreateBucketIterator(c.Ctx, c.StorageClient, project)
	for itBucketAttrs, err := it.Next(); err != iterator.Done; itBucketAttrs, err = it.Next() {
		if err != nil {
			return nil, err
		}
		if itBucketAttrs.Name == bucketName {
			return itBucketAttrs, nil
		}
	}
	return nil, nil
}

func (c *ScratchBucketCreator) formatScratchBucketName(project string, location string) string {
	bucket := strings.Replace(strings.Replace(project, "google.com", "google_com", -1), ":", "-", -1) + "-daisy-bkt"
	if location != "" {
		bucket = bucket + "-" + location
	}
	return strings.ToLower(bucket)
}
