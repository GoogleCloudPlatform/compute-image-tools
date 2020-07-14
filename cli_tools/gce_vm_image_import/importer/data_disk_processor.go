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
	"log"

	"google.golang.org/api/compute/v1"
)

type dataDiskProcessor struct {
	client  createImageClient
	project string
	request compute.Image
}

func newDataDiskProcessor(pd persistentDisk, client createImageClient, project string,
	userLabels map[string]string, userStorageLocation string,
	description string, family string, imageName string) processor {
	labels := map[string]string{"gce-image-import": "true"}
	for k, v := range userLabels {
		labels[k] = v
	}
	var storageLocation []string
	if userStorageLocation != "" {
		storageLocation = []string{userStorageLocation}
	}

	return &dataDiskProcessor{
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

func (d dataDiskProcessor) traceLogs() []string {
	return []string{}
}

func (d dataDiskProcessor) process() (err error) {
	log.Printf("Creating image \"%v\"", d.request.Name)
	return d.client.CreateImage(d.project, &d.request)
}

func (d dataDiskProcessor) cancel(reason string) bool {
	//indicate cancel was not performed
	return false
}

// createImageClient is the subset of the GCP compute API that is used by dataDiskProcessor.
type createImageClient interface {
	CreateImage(project string, i *compute.Image) error
}
