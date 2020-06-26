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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyValueSetReturnsMap(t *testing.T) {
	input := "KEY1=AB,KEY2=CD"
	expected := map[string]string{"KEY1": "AB", "KEY2": "CD"}
	var s KeyValueString = nil
	err := s.Set(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, map[string]string(s))
}

func TestKeyValueSetReturnsEmptyMap(t *testing.T) {
	input := ""
	expected := map[string]string{}
	var s KeyValueString = nil
	err := s.Set(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, map[string]string(s))
}

func TestKeyValueSetReturnsErrorIfNotNil(t *testing.T) {
	input := "KEY1=AB,KEY2=CD"
	var s KeyValueString = map[string]string{}
	err := s.Set(input)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "only one instance of this flag is allowed")
}

func TestKeyValueSetReturnsErrorIfWrongStringFormat(t *testing.T) {
	input := "KEY1->AB,KEY2->CD"
	var s KeyValueString = nil
	err := s.Set(input)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to parse key-value pair. "+
		"key-value should be in the following format: KEY=VALUE")
}

func TestKeyValueStringReturnsFormattedString(t *testing.T) {
	var s KeyValueString = map[string]string{"KEY1": "AB", "KEY2": "CD"}
	output := s.String()
	// Both strings can be valid output, as order is not maintained in map.
	validOutput1 := "KEY1=AB,KEY2=CD"
	validOutput2 := "KEY2=CD,KEY1=AB"
	expected := []string{validOutput1, validOutput2}
	assert.Contains(t, expected, output)
}
