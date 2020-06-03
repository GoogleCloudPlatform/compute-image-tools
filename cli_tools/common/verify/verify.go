//  Copyright 2020 Google Inc. All Rights Reserved.
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

package verify

import "fmt"

// GreaterThanOrEqualTo verifies that value is greater than or equal to limit,
// and panics if the verification fails.
func GreaterThanOrEqualTo(value int, limit int) {
	if value < limit {
		panic(fmt.Sprintf("Expected %d >= %d", value, limit))
	}
}

// Contains verifies that element is a member of arr, and panics if the verification fails.
func Contains(element string, arr []string) {
	for _, e := range arr {
		if e == element {
			return

		}
	}
	panic(fmt.Sprintf("%s is not a member of %v", element, arr))
}
