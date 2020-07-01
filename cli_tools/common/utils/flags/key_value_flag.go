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

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
)

// KeyValueString is an implementation of flag.Value that creates a map
// from the user's argument prior to storing it.
type KeyValueString map[string]string

// String returns string representation of KeyValueString.
// The format of the return value is "KEY1=AB,KEY2=CD"
func (s KeyValueString) String() string {
	var parts []string
	for k, v := range s {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}

// Set creates a key-value map of the input string.
// The input string must be in the format of KEY1=AB,KEY2=CD
func (s *KeyValueString) Set(input string) error {
	if *s != nil {
		return fmt.Errorf("only one instance of this flag is allowed")
	}

	*s = make(map[string]string)
	if input != "" {
		var err error
		*s, err = param.ParseKeyValues(input)
		if err != nil {
			return err
		}
	}
	return nil
}
