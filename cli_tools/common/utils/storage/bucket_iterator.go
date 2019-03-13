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

package storageutils

import (
	"cloud.google.com/go/storage"

	"context"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
)

// BucketIteratorCreator is responsible for creating GCS bucket iterator
type BucketIteratorCreator struct {
}

// CreateBucketIterator creates GCS bucket iterator
func (bic *BucketIteratorCreator) CreateBucketIterator(ctx context.Context,
	storageClient commondomain.StorageClientInterface, projectID string) commondomain.BucketIteratorInterface {
	return &BucketIterator{storageClient.Buckets(projectID)}
}

// BucketIterator is a wrapper around storage.BucketIterator. Implements BucketIteratorInterface.
type BucketIterator struct {
	it *storage.BucketIterator
}

// Next returns the next result. Its second return value is iterator.Done if
// there are no more results. Once Next returns iterator.Done, all subsequent
// calls will return iterator.Done.
func (bi *BucketIterator) Next() (*storage.BucketAttrs, error) {
	return bi.it.Next()
}
