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

package testconfig

// Project is details of test Project.
type Project struct {
	TestProjectID        string
	ServiceAccountEmail  string
	TestZone             string
	ServiceAccountScopes []string
}

// GetProject ...
func GetProject(projectID, testZone string) *Project {
	return &Project{
		TestProjectID:       projectID,
		TestZone:            testZone,
		ServiceAccountEmail: "default",
		ServiceAccountScopes: []string{
			"https://www.googleapis.com/auth/cloud-platform",
		},
	}
}
