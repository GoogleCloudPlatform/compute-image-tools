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

package domain

import (
	"cloud.google.com/go/storage"
	"context"
	"google.golang.org/api/compute/v1"
	"net/http"
)

// MetadataGCEInterface represents GCE metadata
type MetadataGCEInterface interface {
	OnGCE() bool
	Zone() (string, error)
	ProjectID() (string, error)
}

// BucketIteratorCreatorInterface represents GCS bucket creator
type BucketIteratorCreatorInterface interface {
	CreateBucketIterator(ctx context.Context, storageClient StorageClientInterface, projectID string) BucketIteratorInterface
}

// ScratchBucketCreatorInterface represents Daisy scratch (temporary) bucket creator
type ScratchBucketCreatorInterface interface {
	CreateScratchBucket(sourceFileFlag string, projectFlag string) (string, string, error)
}

// StorageClientInterface represents GCS storage client
type StorageClientInterface interface {
	CreateBucket(ctx context.Context, bucketName string, project string, attrs *storage.BucketAttrs) error
	Buckets(ctx context.Context, projectID string) *storage.BucketIterator
	GetBucketAttrs(bucket string) (*storage.BucketAttrs, error)
}

//BucketIteratorInterface represents GCS bucket iterator
type BucketIteratorInterface interface {
	Next() (*storage.BucketAttrs, error)
}

// ZoneRetrieverInterface represents Daisy GCE zone retriever
type ZoneRetrieverInterface interface {
	GetZone(storageRegion string, project string) (string, error)
}

// ComputeServiceInterface represents GCE compute service
type ComputeServiceInterface interface {
	GetZones(project string) ([]*compute.Zone, error)
}

// HTTPClientInterface represents HTTP client
type HTTPClientInterface interface {
	Get(url string) (resp *http.Response, err error)
}
