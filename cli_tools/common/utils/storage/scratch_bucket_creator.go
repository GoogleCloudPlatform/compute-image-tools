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
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
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

// CreateScratchBucket creates scratch bucket in the same region as sourceFileFlag.
// Returns (bucket_name, region, error)
func (c *ScratchBucketCreator) CreateScratchBucket(
	sourceFileFlag string, project string, fallBackZone string) (string, string, error) {

	if project == "" {
		return "", "", fmt.Errorf("can't create scratch bucket if project not specified")
	}

	if sourceFileFlag != "" {
		// source file provided, create bucket in the same region for cost/performance reasons
		return c.createBucketMatchFileRegion(sourceFileFlag, project, fallBackZone)
	}

	// if source file is not provided, fallback to input / default region.
	return c.createBucketOnFallbackZone(project, fallBackZone)
}

func (c *ScratchBucketCreator) createBucketMatchFileRegion(fileGcsPath string, project string, zone string) (string, string, error) {
	fileBucket, _, err := SplitGCSPath(fileGcsPath)
	if err != nil || fileBucket == "" {
		return "", "", fmt.Errorf("file GCS path `%v` is invalid: %v", fileGcsPath, err)
	}

	fileBucketAttrs, err := c.StorageClient.GetBucketAttrs(fileBucket)
	if err != nil || fileBucketAttrs == nil {
		// if region can't be determined by bucket (which usually means bucket doesn't exist), then
		// fallback to input / default region
		return c.createBucketOnFallbackZone(project, zone)
	}

	bucket := c.formatScratchBucketName(project, fileBucketAttrs.Location)
	bucketAttrs := c.createBucketAttrsWithLocationStorageType(bucket, fileBucketAttrs)

	location, err := c.createBucketIfNotExisting(bucket, project, bucketAttrs)
	if err != nil {
		return "", "", err
	}
	return bucket, location, nil
}

func (c *ScratchBucketCreator) createBucketOnFallbackZone(project string, fallbackZone string) (string, string, error) {
	fallbackRegion := defaultRegion
	storageClass := defaultStorageClass
	var err error
	if fallbackZone != "" {
		if fallbackRegion, err = getRegion(fallbackZone); err != nil {
			return "", "", err
		}
		storageClass = regionalStorageClass
	}
	bucket := c.formatScratchBucketName(project, fallbackRegion)
	region, err := c.createBucketIfNotExisting(bucket, project, &storage.BucketAttrs{Name: bucket, Location: fallbackRegion, StorageClass: storageClass})
	if err != nil {
		return "", "", err
	}
	return bucket, region, nil
}

func (c *ScratchBucketCreator) createBucketIfNotExisting(bucket string, project string,
	bucketAttrs *storage.BucketAttrs) (string, error) {
	it := c.BucketIteratorCreator.CreateBucketIterator(c.Ctx, c.StorageClient, project)
	for itBucketAttrs, err := it.Next(); err != iterator.Done; itBucketAttrs, err = it.Next() {
		if err != nil {
			return "", err
		}
		if itBucketAttrs.Name == bucket {
			return itBucketAttrs.Location, nil
		}
	}

	log.Printf("Creating scratch bucket `%v` in %v region", bucket, bucketAttrs.Location)
	if err := c.StorageClient.CreateBucket(bucket, project, bucketAttrs); err != nil {
		return "", err
	}
	return bucketAttrs.Location, nil
}

func (c *ScratchBucketCreator) createBucketAttrsWithLocationStorageType(name string,
	bucketAttrs *storage.BucketAttrs) *storage.BucketAttrs {
	return &storage.BucketAttrs{
		Name:         name,
		Location:     bucketAttrs.Location,
		StorageClass: bucketAttrs.StorageClass,
	}
}

func (c *ScratchBucketCreator) formatScratchBucketName(project string, location string) string {
	bucket := strings.Replace(project, ":", "-", -1) + "-daisy-bkt"
	if location != "" {
		bucket = bucket + "-" + location
	}
	return strings.ToLower(bucket)
}

func getRegion(zone string) (string, error) {
	if zone == "" {
		return "", fmt.Errorf("zone is empty. Can't determine region")
	}
	zoneStrs := strings.Split(zone, "-")
	if len(zoneStrs) < 2 {
		return "", fmt.Errorf("%v is not a valid zone", zone)
	}
	return strings.Join(zoneStrs[:len(zoneStrs)-1], "-"), nil
}
