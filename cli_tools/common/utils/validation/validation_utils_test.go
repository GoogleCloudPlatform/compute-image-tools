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
	"testing"
)

func TestValidateFqdnValidValue(t *testing.T) {
	err := ValidateFqdn("host.domain", "hostname")
	if err != nil {
		t.Errorf("error should be nil, but got %v", err)
	}
}

func TestValidateFqdnSingleLabel(t *testing.T) {
	err := ValidateFqdn("host", "hostname")
	if err == nil {
		t.Error("error expected")
	}
}

func TestValidateFqdnEmptyString(t *testing.T) {
	err := ValidateFqdn("", "hostname")
	if err == nil {
		t.Error("error expected")
	}
}

func TestValidateFqdnInvalidChars(t *testing.T) {
	err := ValidateFqdn("host|name.domain", "hostname")
	if err == nil {
		t.Error("error expected")
	}
}

func TestValidateFqdnTooLong(t *testing.T) {
	err := ValidateFqdn("host.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain", "hostname")
	if err == nil {
		t.Error("error expected")
	}
}
