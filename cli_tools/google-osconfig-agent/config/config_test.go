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
package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestSetConfig(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"project":{"numericProjectID":12345,"projectId":"projectId","attributes":{"os-config-endpoint":"bad!!1","os-inventory-enabled":"false"}},"instance":{"id":12345,"name":"name","zone":"zone","attributes":{"os-config-endpoint":"SvcEndpoint","os-inventory-enabled":"1","os-config-debug-enabled":"true","os-config-enabled-prerelease-features":"ospackage,ospatch", "os-config-poll-interval":"3"}}}`)
	}))
	defer ts.Close()

	if err := os.Setenv("GCE_METADATA_HOST", strings.Trim(ts.URL, "http://")); err != nil {
		t.Fatalf("Error running os.Setenv: %v", err)
	}

	if err := SetConfig(); err != nil {
		t.Fatalf("Error running SetConfig: %v", err)
	}

	testsString := []struct {
		desc string
		op   func() string
		want string
	}{
		{"SvcEndpoint", SvcEndpoint, "SvcEndpoint"},
		{"Instance", Instance, "zone/instances/name"},
		{"ID", ID, "12345"},
		{"ProjectID", ProjectID, "projectId"},
		{"Zone", Zone, "zone"},
		{"Name", Name, "name"},
	}
	for _, tt := range testsString {
		if tt.op() != tt.want {
			t.Errorf("%q: got(%q) != want(%q)", tt.desc, tt.op(), tt.want)
		}
	}

	testsBool := []struct {
		desc string
		op   func() bool
		want bool
	}{
		{"osinventory should be enabled (proj disabled, inst enabled)", OSInventoryEnabled, true},
		{"ospatch should be enabled (inst enabled)", OSPatchEnabled, true},
		{"ospackage should be enabled (proj enabled)", OSPackageEnabled, true},
		{"debugenabled should be true (proj disabled, inst enabled)", Debug, true},
	}
	for _, tt := range testsBool {
		if tt.op() != tt.want {
			t.Errorf("%q: got(%t) != want(%t)", tt.desc, tt.op(), tt.want)
		}
	}

	if SvcPollInterval().Minutes() != float64(3) {
		t.Errorf("Default poll interval: got(%f) != want(%d)", SvcPollInterval().Minutes(), 3)
	}
	if NumericProjectID() != 12345 {
		t.Errorf("NumericProjectID: got(%v) != want(%d)", NumericProjectID(), 12345)
	}
}

func TestSetConfigDefaultValues(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{}`)
	}))
	defer ts.Close()

	if err := os.Setenv("GCE_METADATA_HOST", strings.Trim(ts.URL, "http://")); err != nil {
		t.Fatalf("Error running os.Setenv: %v", err)
	}

	if err := SetConfig(); err != nil {
		t.Fatalf("Error running SetConfig: %v", err)
	}

	testsString := []struct {
		op   func() string
		want string
	}{
		{AptRepoFilePath, aptRepoFilePath},
		{YumRepoFilePath, yumRepoFilePath},
		{ZypperRepoFilePath, zypperRepoFilePath},
		{GooGetRepoFilePath, googetRepoFilePath},
	}
	for _, tt := range testsString {
		if tt.op() != tt.want {
			f := filepath.Base(runtime.FuncForPC(reflect.ValueOf(tt.op).Pointer()).Name())
			t.Errorf("%q: got(%q) != want(%q)", f, tt.op(), tt.want)
		}
	}

	testsBool := []struct {
		op   func() bool
		want bool
	}{
		{OSInventoryEnabled, osInventoryEnabledDefault},
		{OSPatchEnabled, osPatchEnabledDefault},
		{OSPackageEnabled, osPackageEnabledDefault},
		{Debug, debugEnabledDefault},
	}
	for _, tt := range testsBool {
		if tt.op() != tt.want {
			f := filepath.Base(runtime.FuncForPC(reflect.ValueOf(tt.op).Pointer()).Name())
			t.Errorf("%q: got(%t) != want(%t)", f, tt.op(), tt.want)
		}
	}

	if SvcPollInterval().Minutes() != float64(osConfigPollIntervalDefault) {
		t.Errorf("Default poll interval: got(%f) != want(%d)", SvcPollInterval().Minutes(), osConfigPollIntervalDefault)
	}

	if SvcEndpoint() != prodEndpoint {
		t.Errorf("Default endpoint: got(%s) != want(%s)", SvcEndpoint(), prodEndpoint)
	}
}

func TestVersion(t *testing.T) {
	if Version() != "" {
		t.Errorf("Unexpected version %q, want \"\"", Version())
	}
	var v = "1"
	SetVersion(v)
	if Version() != v {
		t.Errorf("Unexpected version %q, want %q", Version(), v)
	}
}
