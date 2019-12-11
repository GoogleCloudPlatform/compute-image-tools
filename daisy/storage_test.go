//  Copyright 2017 Google Inc. All Rights Reserved.
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

package daisy

import (
	"testing"
)

func TestGetGCSAPIPath(t *testing.T) {
	got, err := getGCSAPIPath("gs://foo/bar")
	want := "https://storage.cloud.google.com/foo/bar"
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if got != want {
		t.Errorf("unexpected result: got: %q, want: %q", got, want)
	}
}

func TestSplitGCSPath(t *testing.T) {
	tests := []struct {
		input     string
		bucket    string
		object    string
		shouldErr bool
	}{
		{"gs://foo", "foo", "", false},
		{"gs://foo/bar", "foo", "bar", false},
		{"http://foo.storage.googleapis.com/bar", "foo", "bar", false},
		{"https://foo.storage.googleapis.com/bar", "foo", "bar", false},
		{"http://storage.cloud.google.com/foo/bar", "foo", "bar", false},
		{"https://storage.cloud.google.com/foo/bar/bar", "foo", "bar/bar", false},
		{"http://storage.googleapis.com/foo/bar", "foo", "bar", false},
		{"https://storage.googleapis.com/foo/bar", "foo", "bar", false},
		{"http://commondatastorage.googleapis.com/foo/bar", "foo", "bar", false},
		{"https://commondatastorage.googleapis.com/foo/bar", "foo", "bar", false},
		{"/local/path", "", "", true},
	}

	for _, tt := range tests {
		b, o, err := splitGCSPath(tt.input)
		if tt.shouldErr && err == nil {
			t.Errorf("splitGCSPath(%q) should have thrown an error", tt.input)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("splitGCSPath(%q) should not have thrown an error", tt.input)
		}
		if b != tt.bucket || o != tt.object {
			t.Errorf("splitGCSPath(%q) returned incorrect values -- want bucket=%q, object=%q; got bucket=%q, object=%q", tt.input, tt.bucket, tt.object, b, o)
		}
	}
}
