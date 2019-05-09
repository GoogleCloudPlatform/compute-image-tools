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

//+build !test

package argutils

import (
	"cloud.google.com/go/storage"
	"fmt"
	"context"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"google.golang.org/api/iterator"
)

// Get project id from either current project or flag
func GetProjectID(mgce commondomain.MetadataGCEInterface, projectFlag *string) (*string, error) {
	if *projectFlag == "" {
		if !mgce.OnGCE() {
			return nil, fmt.Errorf("project cannot be determined because build is not running on GCE")
		}
		aProject, err := mgce.ProjectID()
		if err != nil || aProject == "" {
			return nil, fmt.Errorf("project cannot be determined %v", err)
		}
		return &aProject, nil
	}
	return projectFlag, nil
}

func createScratchBucket(ctx context.Context, bucketIteratorCreator commondomain.BucketIteratorCreatorInterface,
		storageClient commondomain.StorageClientInterface, bucket string, project string, region string) (string, error) {
	it := bucketIteratorCreator.CreateBucketIterator(ctx, storageClient, project)
	for itBucketAttrs, err := it.Next(); err != iterator.Done; itBucketAttrs, err = it.Next() {
		if err != nil {
			return "", err
		}
		if itBucketAttrs.Name == bucket {
			scratchBucketGcsPath := fmt.Sprintf("gs://%v/", bucket)
			return scratchBucketGcsPath, nil
		}
	}

	oi.logger.Log(fmt.Sprintf("Creating scratch bucket `%v` in %v region", bucket, region))
	if err := storageClient.CreateBucket(
		bucket, project,
		&storage.BucketAttrs{Name: bucket, Location: region}); err != nil {
		return err
	}
	*scratchBucketGcsPath = fmt.Sprintf("gs://%v/", bucket)
	return nil
}