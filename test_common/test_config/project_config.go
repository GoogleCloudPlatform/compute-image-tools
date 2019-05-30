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

import (
	"fmt"
	"math/rand"
	"sync"
)

// Project is details of test Project.
type Project struct {
	TestProjectID        string
	ServiceAccountEmail  string
	ServiceAccountScopes []string
	testZones            map[string]int
	zoneIndices          []string
	mux                  sync.Mutex
}

// GetProject ...
func GetProject(projectID string, testZones map[string]int) *Project {
	var zoneIndices []string

	for z := range testZones {
		zoneIndices = append(zoneIndices, z)
	}

	return &Project{
		TestProjectID:       projectID,
		testZones:           testZones,
		zoneIndices:         zoneIndices,
		ServiceAccountEmail: "default",
		ServiceAccountScopes: []string{
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/devstorage.full_control",
		},
	}
}

// GetZone gets a random zone that still has capacity.
func (p *Project) GetZone() string {
	p.mux.Lock()
	defer p.mux.Unlock()

	zc := len(p.zoneIndices)
	if zc == 0 {
		// TODO: return an error instead of stopping the process.
		msg := "Not enough zone quota sepcified. Specify additional quota in `test_zones`."
		fmt.Println(msg)
		return msg
	}

	zi := rand.Intn(zc)
	z := p.zoneIndices[zi]

	p.testZones[z]--
	if p.testZones[z] == 0 {
		p.zoneIndices = append(p.zoneIndices[:zi], p.zoneIndices[zi+1:]...)
	}

	return z
}
