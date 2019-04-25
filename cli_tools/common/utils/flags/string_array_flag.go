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
//  limitations under the License

package flags

import "strings"

// StringArrayFlag represents a CLI flag with multiple string values
type StringArrayFlag []string

// Set adds a string value to StringArrayFlag
func (i *StringArrayFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// String returns string representation of StringArrayFlag
func (i *StringArrayFlag) String() string {
	if i == nil {
		return ""
	}
	return strings.Join(*i, ",")
}
