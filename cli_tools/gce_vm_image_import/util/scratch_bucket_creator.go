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

package gcevmimageimportutil

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/domain"
	"google.golang.org/api/iterator"
	"log"
	"strings"
)

// Creates scratch bucket
type ScratchBucketCreator struct {
	StorageClient         domain.StorageClientInterface
	Ctx                   context.Context
	BucketIteratorCreator domain.BucketIteratorCreatorInterface
}

func NewScratchBucketCreator(storageClient *storage.Client, ctx context.Context) *ScratchBucketCreator {
	return &ScratchBucketCreator{NewStorageClient(storageClient, ctx), ctx, &BucketIteratorCreator{}}
}

// Creates scratch bucket in the same region as sourceFileFlag. Returns (bucket_name, region, error)
func (c *ScratchBucketCreator) CreateScratchBucket(
	sourceFileFlag string, projectFlag string) (string, string, error) {
	if sourceFileFlag != "" {
		// source file provided, create bucket in the same region for cost/performance reasons
		return c.createBucketMatchFileRegion(sourceFileFlag, projectFlag)
	}

	// we don't care about scratch bucket region if no source file
	return "", "", nil
}

func (c *ScratchBucketCreator) createBucketMatchFileRegion(fileGcsPath string, project string) (string, string, error) {
	fileBucket, _, err := SplitGCSPath(fileGcsPath)
	if err != nil || fileBucket == "" {
		return "", "", fmt.Errorf("file GCS path `%v` is invalid: %v", fileGcsPath, err)
	}

	fileBucketAttrs, err := c.StorageClient.GetBucketAttrs(fileBucket)
	if err != nil || fileBucketAttrs == nil {
		return "", "", fmt.Errorf("couldn't determine region for bucket `%v` : %v", fileBucket, err)
	}

	bucket := strings.ToLower(strings.Replace(project, ":", "-", -1) +
		"-daisy-bkt-" + fileBucketAttrs.Location)
	bucketAttrs := createBucketAttrsWithLocationStorageType(bucket, fileBucketAttrs)

	it := c.BucketIteratorCreator.CreateBucketIterator(c.Ctx, c.StorageClient, project)
	for itBucketAttrs, err := it.Next(); err != iterator.Done; itBucketAttrs, err = it.Next() {
		if err != nil {
			return "", "", err
		}
		if itBucketAttrs.Name == bucket {
			return bucket, itBucketAttrs.Location, nil
		}
	}

	log.Printf("Creating scratch bucket `%v` in %v region", bucket, fileBucketAttrs.Location)
	if err := c.StorageClient.CreateBucket(bucket, c.Ctx, project, bucketAttrs); err != nil {
		return "", "", err
	}
	return bucket, fileBucketAttrs.Location, nil
}

func createBucketAttrsWithLocationStorageType(name string,
	bucketAttrs *storage.BucketAttrs) *storage.BucketAttrs {
	return &storage.BucketAttrs{
		Name:         name,
		Location:     bucketAttrs.Location,
		StorageClass: bucketAttrs.StorageClass,
	}
}
