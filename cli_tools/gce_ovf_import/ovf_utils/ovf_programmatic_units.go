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
	"regexp"
	"strings"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
)

const (
	kilo int64 = 1 << 10
	mega int64 = 1 << 20
	giga int64 = 1 << 30
	tera int64 = 1 << 40
)

var (
	whitespace = regexp.MustCompile("\\s+")
)

// ByteCapacity assists with interpreting and converting
// quantities related to bytes.
type ByteCapacity struct {
	bytes int64
}

// Parse interprets a human-friendly description of a byte quantity.
// Units can be represented as a name, abbreviation, or with scientific
// notation. All of the following are valid ways to represent the gigabyte unit:
//
//   - GB
//   - gigabyte
//   - gigabytes
//   - byte * 2^30
//
// Values are calculated in binary: 1 GB is 2^30 bytes.
//
// For more info on programmatic units, see:
// https://www.dmtf.org/sites/default/files/standards/documents/DSP0004_2.7.pdf
func Parse(quantity int64, unit string) (*ByteCapacity, error) {
	if quantity <= 0 {
		return nil, daisy.Errf("expected a positive value for capacity. Given: `%d`", quantity)
	}

	// In the wild, we've seen examples such as 'MegaBytes' and 'byte* 2 ^ 10 '.
	// To account for this, we remove whitepsace and convert to lowercase.
	standardizeUnit := strings.ToLower(whitespace.ReplaceAllString(unit, ""))
	var bytes int64
	switch standardizeUnit {
	case "byte", "bytes":
		bytes = quantity
	case "kb", "kilobyte", "kilobytes", "byte*2^10":
		bytes = kilo * quantity
	case "mb", "megabyte", "megabytes", "byte*2^20":
		bytes = mega * quantity
	case "gb", "gigabyte", "gigabytes", "byte*2^30":
		bytes = giga * quantity
	case "tb", "terabyte", "terabytes", "byte*2^40":
		bytes = tera * quantity
	default:
		return nil, daisy.Errf("invalid allocation unit: " + unit)
	}

	return &ByteCapacity{bytes}, nil
}

// By rounding up, we're ensuring that we always allocate enough
// capacity for the customer's request.
func divideAndRoundUp(numerator int64, denominator int64) int {
	if numerator%denominator == 0 {
		return int(numerator / denominator)
	}
	return 1 + int(numerator/denominator)
}

// ToMB converts to megabytes by dividing by 1024^2.
// Remainders are always rounded up.
func (b *ByteCapacity) ToMB() int {
	if b.bytes < 1 {
		panic(fmt.Sprintf("Unexpected non-positive value for bytes: %d", b.bytes))
	} else {
		return divideAndRoundUp(b.bytes, mega)
	}
}

// ToGB converts to gigabytes by dividing by 1024^3.
// Remainders are always rounded up.
func (b *ByteCapacity) ToGB() int {
	if b.bytes < 1 {
		panic(fmt.Sprintf("Unexpected non-positive value for bytes: %d", b.bytes))
	} else {
		return divideAndRoundUp(b.bytes, giga)
	}
}
