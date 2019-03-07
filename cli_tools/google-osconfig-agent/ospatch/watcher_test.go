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

// Package config stores and retrieves configuration settings for the OS Config agent.
package ospatch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWatcher(t *testing.T) {
	var c chan struct{}

	var i int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close channel at second request.
		if i == 1 {
			close(c)
		}
		q := r.URL.Query()
		if q.Get("name") == "cancel" {
			close(c)
		}
		fmt.Fprintln(w, fmt.Sprintf(`{"osconfig-patch-notify":"%s"}`, q.Get("name")))

		i++
	}))
	defer ts.Close()

	var ran bool
	action := func(_ context.Context, _ string) {
		ran = true
	}

	ctx := context.Background()

	var tests = []struct {
		desc    string
		url     string
		name    string
		wantRun bool
	}{
		{
			"normal case",
			ts.URL + "?name=foo",
			"",
			true,
		},
		{
			"no metadata",
			ts.URL,
			"",
			false,
		},
		{
			"same patch run name",
			ts.URL + "?name=foo",
			"foo",
			false,
		},
		{
			"canceled case",
			ts.URL + "?name=cancel",
			"",
			false,
		},
	}
	for _, tt := range tests {
		metadataURL = tt.url
		c = make(chan struct{})
		i = 0
		ran = false
		watcher(ctx, tt.name, c, action)
		if ran != tt.wantRun {
			t.Errorf("%s: wantRun=%t, got=%t", tt.desc, tt.wantRun, ran)
		}

	}
}
