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

package string

import (
	"sort"
	"strconv"
)

// CombineStringSlices merges two slices of strings and
// returns a new slice instance. Duplicates are removed.
func CombineStringSlices(s1 []string, s2 ...string) []string {

	values := map[string]bool{}
	for _, value := range s2 {
		values[value] = true
	}
	for _, value := range s1 {
		values[value] = true
	}
	ret := make([]string, 0)
	for value := range values {
		ret = append(ret, value)
	}
	// Sort elements by type, lexically. This ensures
	// stability of output ordering for tests.
	sort.Slice(ret, func(i, j int) bool {
		return ret[i] < ret[j]
	})
	return ret
}

// SafeStringToInt returns the base-10 integer represented by s, or zero if
// there is a parse failure.
func SafeStringToInt(s string) int64 {
	i, e := strconv.Atoi(s)
	if e != nil {
		return 0
	}
	return int64(i)
}
