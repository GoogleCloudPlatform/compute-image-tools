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
	"encoding/json"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/domain"
	"github.com/prometheus/common/log"
	"google.golang.org/api/option"
	storagev1 "google.golang.org/api/storage/v1"
	htransport "google.golang.org/api/transport/http"
	"io/ioutil"
	"net/http"
)

// StorageClient implements domain.StorageClientInterface. It implements main Storage functions
// used by image import features.
type StorageClient struct {
	storageClient *storage.Client
	httpClient    domain.HTTPClientInterface
	ctx           context.Context
}

// HTTPClient implements domain.HTTPClientInterface which abstracts HTTP functionality used by
// image import features.
type HTTPClient struct {
	httpClient *http.Client
}

// Get executes HTTP GET request for given URL
func (hc *HTTPClient) Get(url string) (resp *http.Response, err error) {
	return hc.httpClient.Get(url)
}

// NewStorageClient creates a StorageClient
func NewStorageClient(ctx context.Context, client *storage.Client) *StorageClient {
	o := []option.ClientOption{option.WithScopes(storagev1.DevstorageReadOnlyScope)}
	hc, _, err := htransport.NewClient(ctx, o...)
	if err != nil {
		log.Fatalf("Cannot create storage HTTP client %v", err.Error())
		return nil
	}
	return &StorageClient{client, &HTTPClient{hc}, ctx}
}

// CreateBucket creates a GCS bucket
func (sc *StorageClient) CreateBucket(ctx context.Context, bucketName string,
	project string, attrs *storage.BucketAttrs) error {
	return sc.storageClient.Bucket(bucketName).Create(ctx, project, attrs)
}

// Buckets returns a bucket iterator for all buckets within a project
func (sc *StorageClient) Buckets(ctx context.Context, projectID string) *storage.BucketIterator {
	return sc.storageClient.Buckets(ctx, projectID)
}

// GetBucketAttrs returns bucket attributes for given bucket
func (sc *StorageClient) GetBucketAttrs(bucket string) (*storage.BucketAttrs, error) {
	resp, err := sc.httpClient.Get("https://www.googleapis.com/storage/v1/b/" + bucket + "?fields=location%2CstorageClass")
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		bucketAttrs := &storage.BucketAttrs{}

		if err = json.Unmarshal(body, &bucketAttrs); err != nil {
			return nil, err
		}
		return bucketAttrs, nil
	}
	return nil, fmt.Errorf("error while retrieving `%v` bucket attributes: Error %v, %v", bucket, resp.StatusCode, resp.Status)
}
