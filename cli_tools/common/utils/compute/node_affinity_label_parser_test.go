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

package computeutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNodeAffinities(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"env,IN,prod,test,staging", "os,NOT_IN,windows-2012,windows-2016"})

	assert.NotNil(t, affinities)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(affinities))

	assert.Equal(t, "env", affinities[0].Key)
	assert.Equal(t, "IN", affinities[0].Operator)
	assert.Equal(t, 3, len(affinities[0].Values))
	assert.Equal(t, "prod", affinities[0].Values[0])
	assert.Equal(t, "test", affinities[0].Values[1])
	assert.Equal(t, "staging", affinities[0].Values[2])

	assert.Equal(t, "os", affinities[1].Key)
	assert.Equal(t, "NOT_IN", affinities[1].Operator)
	assert.Equal(t, 2, len(affinities[1].Values))
	assert.Equal(t, "windows-2012", affinities[1].Values[0])
	assert.Equal(t, "windows-2016", affinities[1].Values[1])
}

func TestParseNodeAffinitiesSingleValue(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"compute.googleapis.com/node-group-name,IN,zoran-playground-node-group"})

	assert.NotNil(t, affinities)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(affinities))

	assert.Equal(t, "compute.googleapis.com/node-group-name", affinities[0].Key)
	assert.Equal(t, "IN", affinities[0].Operator)
	assert.Equal(t, 1, len(affinities[0].Values))
	assert.Equal(t, "zoran-playground-node-group", affinities[0].Values[0])
}

func TestParseNodeAffinitiesExtraSpaces(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"   env , IN   , prod,test,   staging    "})

	assert.NotNil(t, affinities)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(affinities))

	assert.Equal(t, "env", affinities[0].Key)
	assert.Equal(t, "IN", affinities[0].Operator)
	assert.Equal(t, 3, len(affinities[0].Values))
	assert.Equal(t, "prod", affinities[0].Values[0])
	assert.Equal(t, "test", affinities[0].Values[1])
	assert.Equal(t, "staging", affinities[0].Values[2])
}

func TestParseNodeAffinitiesNoValue(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"env,IN"})

	assert.Nil(t, affinities)
	assert.NotNil(t, err)
}

func TestParseNodeAffinitiesEmptyValue(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"env,IN,"})

	assert.Nil(t, affinities)
	assert.NotNil(t, err)
}

func TestParseNodeAffinitiesWhiteSpaceValue(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"env,IN,  "})

	assert.Nil(t, affinities)
	assert.NotNil(t, err)
}

func TestParseNodeAffinitiesNoOperator(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"env"})

	assert.Nil(t, affinities)
	assert.NotNil(t, err)
}

func TestParseNodeAffinitiesEmptyOperator(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"env,,test"})

	assert.Nil(t, affinities)
	assert.NotNil(t, err)
}

func TestParseNodeAffinitiesWhiteSpaceOperator(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"env,  ,test"})

	assert.Nil(t, affinities)
	assert.NotNil(t, err)
}

func TestParseNodeAffinitiesEmptyOperatorNoValue(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"env,"})

	assert.Nil(t, affinities)
	assert.NotNil(t, err)
}

func TestParseNodeAffinitiesInvalidOperator(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"env,CONTAINS,test"})

	assert.Nil(t, affinities)
	assert.NotNil(t, err)
}

func TestParseNodeAffinitiesEmptyKey(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{",IN,test"})

	assert.Nil(t, affinities)
	assert.NotNil(t, err)
}

func TestParseNodeAffinitiesWhiteSpacesKey(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{"   ,IN,test"})

	assert.Nil(t, affinities)
	assert.NotNil(t, err)
}

func TestParseNodeAffinitiesEmptyString(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels([]string{})

	assert.Equal(t, 0, len(affinities))
	assert.Nil(t, err)
}

func TestParseNodeAffinitiesNilLabels(t *testing.T) {
	affinities, err := ParseNodeAffinityLabels(nil)

	assert.Equal(t, 0, len(affinities))
	assert.Nil(t, err)
}
