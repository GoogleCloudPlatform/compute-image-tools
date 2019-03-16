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

package computeutils

import (
	"fmt"

	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

// ZoneValidator is responsible for validating zone name corresponds to a valid zone in a given
// project
type ZoneValidator struct {
	ComputeClient daisycompute.Client
}

// ZoneValid validates zone is valid in the given project
func (zv *ZoneValidator) ZoneValid(project string, zone string) error {
	zl, err := zv.ComputeClient.ListZones(project)
	if err != nil {
		return err
	}
	for _, z := range zl {
		if z.Name == zone {
			return nil
		}
	}
	return fmt.Errorf("%v is not a valid zone", zone)
}
