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

package ovfutils

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Returns capacity in GB taking into account allocation units which should be in `bytes * 2^x`
// format. If capacity and allocationUnits are valid, returns at least 1 even if given capacity
// is less than 1GB
func getCapacityInGB(capacity string, allocationUnits string) (int, error) {
	capacityRaw, err := strconv.Atoi(capacity)
	if err != nil {
		return 0, err
	}
	allocationUnitPower, err := getAllocationUnitPowerOfTwo(allocationUnits)
	if err != nil {
		return 0, err
	}
	allocationUnitPowerToGB := float64(allocationUnitPower) - 30.0
	allocationUnitFactorToGB := math.Pow(2.0, allocationUnitPowerToGB)
	capacityInGB := float64(capacityRaw) * allocationUnitFactorToGB

	return int(math.Ceil(capacityInGB)), nil
}

func getAllocationUnitPowerOfTwo(allocationUnits string) (int, error) {
	allocationUnits = strings.ToLower(allocationUnits)
	if !strings.HasPrefix(allocationUnits, "byte * 2^") {
		return 0, fmt.Errorf("can't parse `%v` disk allocation units", allocationUnits)
	}
	return strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(allocationUnits, "byte * 2^")))
}
