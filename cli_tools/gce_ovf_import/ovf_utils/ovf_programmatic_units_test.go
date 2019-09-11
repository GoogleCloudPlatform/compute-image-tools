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
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseTest struct {
	expectedGB      int
	capacity        int64
	allocationUnits string
	expectedErr     string
}

var parseTests = []parseTest{
	// Don't allow zero or negative capacity.
	{-1, -1, "GB", "expected a positive value for capacity. Given: `-1`"},
	{-1, 0, "GB", "expected a positive value for capacity. Given: `0`"},

	// Always round up to the nearest GB.
	{1, 1, "byte", ""},
	{1, 1, "KB", ""},
	{1, 1, "MB", ""},
	{2, 1 + 1<<30, "byte", ""},
	{2, 1 + 1<<20, "KB", ""},
	{2, 1 + 1<<10, "MB", ""},

	// Support the largest GCP disk, which is currently 64TB:
	//   https://cloud.google.com/persistent-disk/
	{64 * 1024, 64, "TB", ""},

	// Parse error due to allocation units not being recognized.
	{-1, 1024, "rhino",
		"invalid allocation unit: rhino"},

	// Test all forms of unit names (full name, abbreviation, and scientific):
	{1, 10, "byte", ""},
	{1, 10, "bytes", ""},
	{1, 10, "kilobyte", ""},
	{1, 10, "kilobytes", ""},
	{1, 10, "kb", ""},
	{1, 10, "byte * 2^10", ""},
	{1, 10, "megabyte", ""},
	{1, 10, "megabytes", ""},
	{1, 10, "mb", ""},
	{1, 10, "byte * 2^20", ""},
	{10, 10, "gigabyte", ""},
	{10, 10, "gigabytes", ""},
	{10, 10, "gb", ""},
	{10, 10, "byte * 2^30", ""},
	{10 * 1024, 10, "terabyte", ""},
	{10 * 1024, 10, "terabytes", ""},
	{10 * 1024, 10, "tb", ""},
	{10 * 1024, 10, "byte * 2^40", ""},

	// Ensure that we match the unit regardless of whitespace and casing.
	{3, 3000, "mb", ""},
	{3, 3000, "MegaByte", ""},
	{3, 3000, "Megabytes", ""},
	{3, 3000, "byte\t*\t2\t^\t20", ""},
	{3, 3000, "   byte* 2 ^ 20  ", ""},
}

func TestParse(t *testing.T) {
	for _, test := range parseTests {
		capacity, err := Parse(test.capacity, test.allocationUnits)
		caseDescription := fmt.Sprintf("getCapacityInGB(%v, %v) = (%v, %v) want (%v, %v)",
			test.capacity, test.allocationUnits, capacity, err, test.expectedGB, test.expectedErr)

		if test.expectedErr == "" {
			assert.Nil(t, err, caseDescription)
			assert.Equal(t, test.expectedGB, capacity.ToGB(), caseDescription)
		} else {
			assert.EqualError(t, err, test.expectedErr, caseDescription)
		}
	}
}

func TestConversionToMB(t *testing.T) {
	var mb int64 = 1 << 20

	// Panic if trying to convert from a negative capacity.
	assert.PanicsWithValue(t,
		"Unexpected non-positive value for bytes: -100",
		func() { (&ByteCapacity{-100}).ToMB() })
	assert.PanicsWithValue(t,
		"Unexpected non-positive value for bytes: -1",
		func() { (&ByteCapacity{-1}).ToMB() })
	assert.PanicsWithValue(t,
		"Unexpected non-positive value for bytes: 0",
		func() { (&ByteCapacity{0}).ToMB() })

	// Round up to the nearest megabyte.
	assert.Equal(t, 1, (&ByteCapacity{1}).ToMB())
	assert.Equal(t, 1, (&ByteCapacity{mb - 1}).ToMB())
	assert.Equal(t, 1, (&ByteCapacity{mb}).ToMB())
	assert.Equal(t, 2, (&ByteCapacity{mb + 1}).ToMB())
	assert.Equal(t, 2, (&ByteCapacity{2*mb - 1}).ToMB())
	assert.Equal(t, 2, (&ByteCapacity{2 * mb}).ToMB())
	assert.Equal(t, 3, (&ByteCapacity{2*mb + 1}).ToMB())
}

func TestConversionToGB(t *testing.T) {
	var gb int64 = 1 << 30

	// Panic if trying to convert from a negative capacity.
	assert.PanicsWithValue(t,
		"Unexpected non-positive value for bytes: -100",
		func() { (&ByteCapacity{-100}).ToGB() })
	assert.PanicsWithValue(t,
		"Unexpected non-positive value for bytes: -1",
		func() { (&ByteCapacity{-1}).ToGB() })
	assert.PanicsWithValue(t,
		"Unexpected non-positive value for bytes: 0",
		func() { (&ByteCapacity{0}).ToGB() })

	// Round up to the nearest gigabyte.
	assert.Equal(t, 1, (&ByteCapacity{1}).ToGB())
	assert.Equal(t, 1, (&ByteCapacity{gb - 1}).ToGB())
	assert.Equal(t, 1, (&ByteCapacity{gb}).ToGB())
	assert.Equal(t, 2, (&ByteCapacity{gb + 1}).ToGB())
	assert.Equal(t, 2, (&ByteCapacity{2*gb - 1}).ToGB())
	assert.Equal(t, 2, (&ByteCapacity{2 * gb}).ToGB())
	assert.Equal(t, 3, (&ByteCapacity{2*gb + 1}).ToGB())
}
