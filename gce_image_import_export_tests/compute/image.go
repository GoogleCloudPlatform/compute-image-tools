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

// Package compute contains wrappers around the GCE compute API.
package compute

import (
	"context"

	computeApi "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	api "google.golang.org/api/compute/v1"
)

// Image is a compute image.
type Image struct {
	*api.Image
	Client  computeApi.Client
	Project string
}

// Cleanup deletes the image.
func (i *Image) Cleanup() error {
	return i.Client.DeleteImage(i.Project, i.Name)
}

// CreateImageObject creates an image object to be operated by API client
func CreateImageObject(ctx context.Context, project string, name string) (*Image, error) {
	client, err := computeApi.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	var apiImage *api.Image
	apiImage, err = client.GetImage(project, name)
	return &Image{apiImage, client, project}, err
}
