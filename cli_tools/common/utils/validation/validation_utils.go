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

package validation

import (
	"fmt"
	"regexp"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

var (
	rfc1035LabelRegexpStr = "[A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9]"
	rfc1035LabelRegexp    = regexp.MustCompile(rfc1035LabelRegexpStr)
	fqdnRegexp            = regexp.MustCompile(fmt.Sprintf("^((%v)\\.)+(%v)$", rfc1035LabelRegexpStr, rfc1035LabelRegexpStr))
)

// ValidateStringFlagNotEmpty returns error with error message stating field must be provided if
// value is empty string. Returns nil otherwise.
func ValidateStringFlagNotEmpty(flagValue string, flagKey string) error {
	if flagValue == "" {
		return daisy.Errf(fmt.Sprintf("The flag -%v must be provided", flagKey))
	}
	return nil
}

// ValidateFqdn validates fully qualified domain name
func ValidateFqdn(flagValue string, flagKey string) error {
	flagValueLen := len(flagValue)
	if flagValueLen < 1 || flagValueLen > 253 || !fqdnRegexp.MatchString(flagValue) {
		return daisy.Errf(fmt.Sprintf("The flag `%v` must conform to RFC 1035 requirements for valid hostnames. "+
			"To meet this requirement, the value must contain a series of labels and each label is concatenated with a dot."+
			"Each label can be 1-63 characters long, and the entire sequence must not exceed 253 characters.", flagKey))
	}
	return nil
}

// ValidateRfc1035Label validates a single label per RFC 1035
func ValidateRfc1035Label(value string) error {
	if len(value) > 63 || !rfc1035LabelRegexp.MatchString(value) {
		return daisy.Errf(fmt.Sprintf("Value `%v` must conform to RFC 1035 requirements for valid labels.", value))
	}
	return nil
}
