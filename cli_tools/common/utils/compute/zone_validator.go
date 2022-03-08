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
//  limitations under the License

package compute

import (
	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-daisy/compute"
	"google.golang.org/api/googleapi"
)

// ZoneValidator is responsible for validating zone name corresponds to a valid zone in a given
// project
type ZoneValidator struct {
	ComputeClient daisyCompute.Client
}

// ZoneValid validates zone is valid in the given project
func (zv *ZoneValidator) ZoneValid(project string, zone string) error {
	zl, err := zv.ComputeClient.ListZones(project)
	if err != nil {
		// Check for Compute Engine API disabled
		gAPIErr, isGAPIErr := err.(*googleapi.Error)
		if isGAPIErr && gAPIErr.Code == 403 && len(gAPIErr.Errors) > 0 && gAPIErr.Errors[0].Reason == "accessNotConfigured" {
			return daisy.Errf("Compute Engine API not configured: %v", gAPIErr)
		}
		return daisy.Errf("Couldn't validate zone `%v`: %v", zone, gAPIErr)
	}
	for _, z := range zl {
		if z.Name == zone {
			return nil
		}
	}
	return daisy.Errf("%v is not a valid zone", zone)
}
