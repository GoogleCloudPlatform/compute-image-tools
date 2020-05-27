//  Copyright 2020 Google Inc. All Rights Reserved.
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

package importer

import (
	"context"

	"google.golang.org/api/compute/v1"
)

type dataDiskFinisher struct {
	client  imageClient
	project string
	request compute.Image
}

func newDataDiskFinisher(pd pd, client imageClient, project string,
	userLabels map[string]string, userStorageLocation string,
	description string, family string, imageName string) finisher {
	labels := map[string]string{"gce-image-import": "true"}
	for k, v := range userLabels {
		labels[k] = v
	}
	var storageLocation []string
	if userStorageLocation != "" {
		storageLocation = []string{userStorageLocation}
	}

	return dataDiskFinisher{
		client:  client,
		project: project,
		request: compute.Image{
			Description:      description,
			Family:           family,
			Labels:           labels,
			Name:             imageName,
			SourceDisk:       pd.uri,
			StorageLocations: storageLocation,
			Licenses:         []string{"projects/compute-image-tools/global/licenses/virtual-disk-import"},
		},
	}
}

func (d dataDiskFinisher) serials() []string {
	return []string{}
}

func (d dataDiskFinisher) run(ctx context.Context) (err error) {
	return d.client.CreateImage(d.project, &d.request)
}

type imageClient interface {
	CreateImage(project string, i *compute.Image) error
}
