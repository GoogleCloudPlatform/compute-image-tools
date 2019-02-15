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
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
	"io/ioutil"
	"net/http"
)

type StorageClientInterface interface {
	CreateBucket(bucketName string, ctx context.Context, project string, attrs *storage.BucketAttrs) error
	Buckets(ctx context.Context, projectID string) *storage.BucketIterator
	GetBucketAttrs(bucket string) (*storage.BucketAttrs, error)
}

type StorageClient struct {
	client *storage.Client
	ctx    context.Context
}

func NewStorageClient(client *storage.Client, ctx context.Context) *StorageClient {
	return &StorageClient{client, ctx}
}

func (sc *StorageClient) CreateBucket(bucketName string, ctx context.Context,
	project string, attrs *storage.BucketAttrs) error {
	return sc.client.Bucket(bucketName).Create(ctx, project, attrs)
}

func (sc *StorageClient) Buckets(ctx context.Context, projectID string) *storage.BucketIterator {
	return sc.client.Buckets(ctx, projectID)
}

func (sc *StorageClient) GetBucketAttrs(bucket string) (*storage.BucketAttrs, error) {
	o := []option.ClientOption{
		option.WithScopes("https://www.googleapis.com/auth/devstorage.full_control"),
	}
	hc, _, err := htransport.NewClient(sc.ctx, o...)
	if err != nil {
		return nil, err
	}
	resp, err := hc.Get("https://www.googleapis.com/storage/v1/b/" + bucket + "?fields=location%2CstorageClass")
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
