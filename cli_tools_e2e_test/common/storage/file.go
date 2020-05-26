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

// Package storage contains wrappers around the GCE storage API.
package storage

import (
	"context"

	storageApi "cloud.google.com/go/storage"
	"google.golang.org/api/option"
	api "google.golang.org/api/storage/v1"
)

// File is a gcs file.
type File struct {
	*api.Object
	Client     storageApi.Client
	FileObject *storageApi.ObjectHandle
	ctx        context.Context
}

// Cleanup deletes the file.
func (f *File) Cleanup() error {
	return f.FileObject.Delete(f.ctx)
}

// CreateFileObject creates an file object to be operated by API client
func CreateFileObject(ctx context.Context, bucketName string, objectName string) (*File, error) {
	storageOptions := []option.ClientOption{}
	client, err := storageApi.NewClient(ctx, storageOptions...)
	if err != nil {
		return nil, err
	}

	fileObj := client.Bucket(bucketName).Object(objectName)
	apiObj := &api.Object{}
	_, err = fileObj.NewReader(ctx)
	return &File{apiObj, *client, fileObj, ctx}, err
}
