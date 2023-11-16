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

package pbtesting

import (
	"fmt"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

type protoMatcher struct {
	expected proto.Message
}

func (m protoMatcher) Got(got interface{}) string {
	message, ok := got.(proto.Message)
	if !ok {
		return fmt.Sprintf("%v", ok)
	}
	return proto.MarshalTextString(message)
}

func (m protoMatcher) Matches(actual interface{}) bool {
	return cmp.Diff(m.expected, actual, protocmp.Transform()) == ""
}

func (m protoMatcher) String() string {
	return proto.MarshalTextString(m.expected)
}

// ProtoEquals returns a protoMatcher for the proto.Message
func ProtoEquals(expected proto.Message) gomock.Matcher {
	return protoMatcher{expected}
}
