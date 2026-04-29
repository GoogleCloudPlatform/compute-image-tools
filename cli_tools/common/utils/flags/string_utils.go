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
//  limitations under the License

package flags

import "strings"

// TrimmedString is an implementation of flag.Value that trims whitespace
// from the incoming argument prior to storing it.
type TrimmedString string

// String returns string representation of TrimmedString.
func (s TrimmedString) String() string { return (string)(s) }

// Set trims whitespace from input string and stores in TrimmedString.
func (s *TrimmedString) Set(input string) error {
	*s = TrimmedString(strings.TrimSpace(input))
	return nil
}

// LowerTrimmedString is an implementation of flag.Value that trims whitespace
// and converts to lowercase the incoming argument prior to storing it.
type LowerTrimmedString string

// String returns string representation of LowerTrimmedString.
func (s LowerTrimmedString) String() string { return (string)(s) }

// Set trims whitespace from input string and converts the string to lowercase,
// then stores in LowerTrimmedString.
func (s *LowerTrimmedString) Set(input string) error {
	*s = LowerTrimmedString(strings.ToLower(strings.TrimSpace(input)))
	return nil
}
