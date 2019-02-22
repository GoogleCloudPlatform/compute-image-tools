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

package parseutils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseKeyValuesReturnsEmptyMap(t *testing.T) {
	keyValueCsv := ""
	keyValueMap, err := ParseKeyValues(&keyValueCsv)
	assert.Nil(t, err)
	assert.NotNil(t, keyValueMap)
	if len(*keyValueMap) > 0 {
		t.Errorf("key-values map %v should be empty, but it's size is %v.", keyValueMap, len(*keyValueMap))
	}
}

func TestParseKeyValues(t *testing.T) {
	doTestParseKeyValues("userkey1=uservalue1,userkey2=uservalue2", t)
}

func TestParseKeyValuesWithWhiteChars(t *testing.T) {
	doTestParseKeyValues("	 userkey1 = uservalue1	,	 userkey2	 =	 uservalue2	 ", t)
}

func doTestParseKeyValues(keyValueStr string, t *testing.T) {
	keyValuesMap, err := ParseKeyValues(&keyValueStr)
	assert.Nil(t, err)
	assert.NotNil(t, keyValuesMap)
	if len(*keyValuesMap) != 2 {
		t.Errorf("key-value map %v should be have size 2, but it's %v.", keyValuesMap, len(*keyValuesMap))
	}
	assert.Equal(t, "uservalue1", (*keyValuesMap)["userkey1"])
	assert.Equal(t, "uservalue2", (*keyValuesMap)["userkey2"])
}

func TestParseKeyValuesNoEqualsSignSingleValue(t *testing.T) {
	doTestParseKeyValuesNoEqualsSign("userkey1", t)
}

func TestParseKeyValuesNoEqualsSignLast(t *testing.T) {
	doTestParseKeyValuesNoEqualsSign("userkey2=uservalue2,userkey1", t)
}

func TestParseKeyValuesNoEqualsSignFirst(t *testing.T) {
	doTestParseKeyValuesNoEqualsSign("userkey1,userkey2=uservalue2", t)
}

func TestParseKeyValuesNoEqualsSignMiddle(t *testing.T) {
	doTestParseKeyValuesNoEqualsSign("userkey3=uservalue3,userkey1,userkey2=uservalue2", t)
}

func TestParseKeyValuesNoEqualsSignMultiple(t *testing.T) {
	doTestParseKeyValuesNoEqualsSign("userkey1,userkey2", t)
}

func TestParseKeyValuesWhiteSpacesOnlyInKey(t *testing.T) {
	doTestParseKeyValuesError(" 	=uservalue1", "key-values map %v should be nil for %v", t)
}

func TestParseKeyValuesWhiteSpacesOnlyInValue(t *testing.T) {
	doTestParseKeyValuesError("userkey= 	", "key-values map %v should be nil for %v", t)
}

func TestParseKeyValuesWhiteSpacesOnlyInKeyAndValue(t *testing.T) {
	doTestParseKeyValuesError(" 	= 	", "key-values map %v should be nil for %v", t)
}

func doTestParseKeyValuesNoEqualsSign(keyValueStr string, t *testing.T) {
	doTestParseKeyValuesError(keyValueStr, "key-values map %v should be nil for v%", t)
}

func doTestParseKeyValuesError(keyValueStr string, errorMsg string, t *testing.T) {
	valuesMap, err := ParseKeyValues(&keyValueStr)
	if valuesMap != nil {
		t.Errorf(errorMsg, valuesMap, keyValueStr)
	}
	assert.NotNil(t, err)
}
