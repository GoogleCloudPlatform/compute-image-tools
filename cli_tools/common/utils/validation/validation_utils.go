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

package validationutils

import (
	"fmt"
)

// ValidateStringFlagNotEmpty returns error with error message stating field must be provided if
// value is empty string. Returns nil otherwise.
func ValidateStringFlagNotEmpty(flagValue string, flagKey string) error {
	return ValidateStringNotEmpty(flagValue, flagKey, "The flag -%v must be provided")
}

// ValidateStringNotEmpty returns error with specific error message if string is empty. Returns nil otherwise.
func ValidateStringNotEmpty(value string, key string, errorMessage string) error {
	if value == "" {
		return fmt.Errorf(errorMessage, key)
	}
	return nil
}
