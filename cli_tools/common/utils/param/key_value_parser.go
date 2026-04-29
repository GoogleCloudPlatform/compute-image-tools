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

package param

import (
	"strings"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
)

// ParseKeyValues parses a comma-separated list of [key=value] pairs.
func ParseKeyValues(keyValues string) (map[string]string, error) {
	labelsMap := make(map[string]string)
	splits := strings.Split(keyValues, ",")
	for _, split := range splits {
		if len(split) == 0 {
			continue
		}
		key, value, err := parseKeyValue(split)
		if err != nil {
			return nil, err
		}
		labelsMap[key] = value
	}
	return labelsMap, nil
}

func parseKeyValue(keyValueSplit string) (string, string, daisy.DError) {
	splits := strings.Split(keyValueSplit, "=")
	if len(splits) != 2 {
		return "", "", daisy.Errf("failed to parse key-value pair. key-value should be in the following format: KEY=VALUE, but it's %v", keyValueSplit)
	}
	key := strings.TrimSpace(splits[0])
	value := strings.TrimSpace(splits[1])
	if len(key) == 0 {
		return "", "", daisy.Errf("failed to parse key-value pair. key is empty string: %v", keyValueSplit)
	}
	if len(value) == 0 {
		return "", "", daisy.Errf("failed to parse key-value pair. value is empty string: %v", keyValueSplit)
	}
	return key, value, nil
}
