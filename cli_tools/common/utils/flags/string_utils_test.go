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

const expected = "test string"

func TestTrimmedStringSetTrimsSpace(t *testing.T) {
	input := "     test string        "
	var s = TrimmedString("")
	err := s.Set(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, string(s))
}

func TestTrimmedStringStringReturnsString(t *testing.T) {
	var s = TrimmedString(expected)
	output := s.String()
	assert.Equal(t, expected, output)
}

func TestLowerTrimmedStringSetTrimsSpace(t *testing.T) {
	input := "     TeSt StRiNg        "
	var s = LowerTrimmedString("")
	err := s.Set(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, string(s))
}

func TestLowerTrimmedStringStringReturnsString(t *testing.T) {
	var s = LowerTrimmedString(expected)
	output := s.String()
	assert.Equal(t, expected, output)
}
