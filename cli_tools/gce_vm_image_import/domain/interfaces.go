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

type MetadataGCEInterface interface {
	OnGCE() bool
	Zone() (string, error)
	ProjectID() (string, error)
}

type BucketIteratorCreatorInterface interface {
	CreateBucketIterator(ctx context.Context, storageClient StorageClientInterface, projectId string) BucketIteratorInterface
}

type ScratchBucketCreatorInterface interface {
	CreateScratchBucket(sourceFileFlag string, projectFlag string) (string, string, error)
}

type StorageClientInterface interface {
	CreateBucket(bucketName string, ctx context.Context, project string, attrs *storage.BucketAttrs) error
	Buckets(ctx context.Context, projectID string) *storage.BucketIterator
	GetBucketAttrs(bucket string) (*storage.BucketAttrs, error)
}

type BucketIteratorInterface interface {
	Next() (*storage.BucketAttrs, error)
}

type ZoneRetrieverInterface interface {
	GetZone(storageRegion string, project string) (string, error)
}

type ComputeServiceInterface interface {
	GetZones(project string) ([]*compute.Zone, error)
}

type HttpClientInterface interface {
	Get(url string) (resp *http.Response, err error)
}
