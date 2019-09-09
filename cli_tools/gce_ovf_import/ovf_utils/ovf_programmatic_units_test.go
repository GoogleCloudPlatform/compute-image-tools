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
	"errors"
	"reflect"
	"testing"
)

type getCapacityInGBTest struct {
	expected        int
	capacity        string
	allocationUnits string
	expectedErr     error
}

var getCapacityInGBTests = []getCapacityInGBTest{
	// GB
	{20, "20", "byte * 2^30", nil},
	{20, "20", "byte * 2^30", nil},
	{10, "10", "byte * 2^30", nil},
	{1, "1", "byte * 2^30", nil},
	{1024, "1024", "byte * 2^30", nil},
	{5242880, "5242880", "byte * 2^30", nil},

	// MB
	{1, "1", "byte * 2^20", nil},
	{1, "1024", "byte * 2^20", nil},
	{5 * 1024, "5242880", "byte * 2^20", nil},

	// TB
	{1024, "1", "byte * 2^40", nil},
	{5242880 * 1024, "5242880", "byte * 2^40", nil},

	// Parse error due to allocation units not being recognized.
	{0, "1024", "megabytes",
		errors.New("can't parse `megabytes` disk allocation units")},
	{0, "1024", "mb",
		errors.New("can't parse `mb` disk allocation units")},
}

func TestGetCapacityInGB(t *testing.T) {
	for _, test := range getCapacityInGBTests {
		capacityInGB, err := getCapacityInGB(test.capacity, test.allocationUnits)
		if capacityInGB != test.expected || !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("getCapacityInGB(%v, %v) = (%v, %v) want (%v, %v)",
				test.capacity, test.allocationUnits, capacityInGB, err, test.expected, test.expectedErr)
		}
	}
}
