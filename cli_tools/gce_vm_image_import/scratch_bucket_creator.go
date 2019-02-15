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

package main

import (
	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"google.golang.org/api/iterator"
	"log"
	"strings"
)

type ScratchBucketCreatorInterface interface {
	CreateScratchBucketIfNotSet(scratchBucketGcsPathFlag *string, sourceFileFlag *string) (string, string, error)
}

// Creates scratch bucket
type ScratchBucketCreator struct {
	metadata      MetadataGCE
	storageClient StorageClientInterface
	ctx           context.Context
}

func NewScratchBucketCreator(metadata MetadataGCE, storageClient *storage.Client,
	ctx context.Context) *ScratchBucketCreator {
	return &ScratchBucketCreator{metadata, NewStorageClient(storageClient, ctx), ctx}
}

func (c *ScratchBucketCreator) CreateScratchBucketIfNotSet(scratchBucketGcsPathFlag *string,
	sourceFileFlag *string) (string, string, error) {
	if scratchBucketGcsPathFlag != nil && *scratchBucketGcsPathFlag != "" {
		// scratch bucket was provided as a flag, use that
		return *scratchBucketGcsPathFlag, "", nil
	}

	if sourceFileFlag != nil && *sourceFileFlag != "" {
		// source file provided, create bucket in the same region for cost/performance reasons
		bucket, region, err := c.createBucketMatchFileRegion(*sourceFileFlag)
		if err != nil {
			return "", "", err
		}
		return bucket, region, nil
	}

	// we don't care about scratch bucket region, let Daisy create it
	return *scratchBucketGcsPathFlag, "", nil
}

func (c *ScratchBucketCreator) createBucketMatchFileRegion(fileGcsPath string) (string, string, error) {
	aProject := *project
	if aProject == "" {
		var err error
		aProject, err = metadata.ProjectID()
		if err != nil || aProject == "" {
			return "", "", fmt.Errorf("project cannot be determined %v", err)
		}
	}

	fileBucket, _, err := splitGCSPath(fileGcsPath)
	if err != nil || fileBucket == "" {
		return "", "", fmt.Errorf("file GCS path `%v` is invalid: %v", fileGcsPath, err)
	}

	fileBucketAttrs, err := c.storageClient.GetBucketAttrs(fileBucket)
	if err != nil {
		return "", "", fmt.Errorf("couldn't determine region for bucket `%v` : %v", fileBucket, err)
	}

	bucket := strings.ToLower(strings.Replace(aProject, ":", "-", -1) +
		"-daisy-bkt-" + fileBucketAttrs.Location)
	bucketAttrs := createBucketAttrsWithLocationStorageType(bucket, fileBucketAttrs)

	it := c.storageClient.Buckets(c.ctx, aProject)
	for bucketAttrs, err := it.Next(); err != iterator.Done; bucketAttrs, err = it.Next() {
		if err != nil {
			return "", "", err
		}
		if bucketAttrs.Name == bucket {
			return bucket, "", nil
		}
	}

	log.Printf("Creating scratch bucket `%v` in %v region", bucket, fileBucketAttrs.Location)
	if err := c.storageClient.CreateBucket(bucket, c.ctx, aProject, bucketAttrs); err != nil {
		return "", "", err
	}
	return bucket, fileBucketAttrs.Location, nil
}

func (c *ScratchBucketCreator) getBucketRegion(bucket string) (string, error) {
	return "", nil
}

func createBucketAttrsWithLocationStorageType(name string,
	bucketAttrs *storage.BucketAttrs) *storage.BucketAttrs {
	return &storage.BucketAttrs{
		Name:         name,
		Location:     bucketAttrs.Location,
		StorageClass: bucketAttrs.StorageClass,
	}
}
