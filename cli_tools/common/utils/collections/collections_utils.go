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

package collections

// ReverseMap reverses keys and their values.
// The 2nd returned value indicates whether the operation succeeded. A map with
// duplicate values can't be reversed. In that case, a 'false' is returned.
func ReverseMap(m map[string]string) (map[string]string, bool) {
	newMap := make(map[string]string, len(m))
	for k, v := range m {
		if _, ok := newMap[v]; ok {
			return nil, false
		}
		newMap[v] = k
	}
	return newMap, true
}

// GetKeys gets all keys of the map.
func GetKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
