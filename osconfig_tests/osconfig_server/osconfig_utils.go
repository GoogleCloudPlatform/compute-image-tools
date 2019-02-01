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

// Package osconfigserver contains wrapper around osconfig service APIs and helper methods
package osconfigserver

import (
	"encoding/json"
	"log"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

// JsonToOsConfig creates an osconfig object from json string
func JsonToOsConfig(jsonString string, logger *log.Logger) (*osconfigpb.OsConfig, error) {

	var osconfig osconfigpb.OsConfig
	err := json.Unmarshal([]byte(jsonString), &osconfig)

	return &osconfig, err
}

// JsonToAssignment creates an assignment object from json string
func JsonToAssignment(jsonString string, logger *log.Logger) (*osconfigpb.Assignment, error) {

	var assignment osconfigpb.Assignment
	err := json.Unmarshal([]byte(jsonString), &assignment)
	logger.Printf("assignment from json: \n%s\n", assignment)
	return &assignment, err
}
