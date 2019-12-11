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
	"context"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestUploadSources(t *testing.T) {
	ctx := context.Background()

	// Set up a local test file.
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("error when setting up test file: %s", err)
	}
	testPath := filepath.Join(dir, "test")
	ioutil.WriteFile(testPath, []byte("Hello world"), 0600)
	if err != nil {
		t.Fatalf("error when setting up test file: %s", err)
	}

	w := testWorkflow()
	sw := w.NewSubWorkflow()
	sw.Name = "test-sw"
	sw.Logger = &MockLogger{}
	w.Steps = map[string]*Step{
		"sub": {w: w, SubWorkflow: &SubWorkflow{Workflow: sw}},
	}
	if err := w.populate(ctx); err != nil {
		t.Fatal(err)
	}

	const NOERR = "NOERR"
	tests := []struct {
		desc        string
		sources     map[string]string
		wantErrType string
		gcs         []string
	}{
		{"normal local file to GCS", map[string]string{"local": testPath}, NOERR, []string{w.sourcesPath + "/local"}},
		{"normal local folder to GCS", map[string]string{"local": dir}, NOERR, []string{w.sourcesPath + "/local/test"}},
		{"normal GCS obj to GCS", map[string]string{"gcs": "gs://gcs/file"}, NOERR, []string{w.sourcesPath + "/gcs"}},
		{"normal GCS bkt to GCS", map[string]string{"gcs": "gs://gcs/folder/"}, NOERR, []string{w.sourcesPath + "/gcs/object", w.sourcesPath + "/gcs/folder/object"}},
		{"dne local path", map[string]string{"local": "./this/file/dne"}, fileIOError, nil},
		{"dne GCS path", map[string]string{"gcs": "gs://gcs/path/dne"}, resourceDNEError, nil},
		//{"GCS path, no object", map[string]string{"gcs": "gs://folder"}, NOERR, []string{w.sourcesPath + "/gcs/object", w.sourcesPath + "/gcs/folder/object"}},
	}

	for _, tt := range tests {
		w.Sources = tt.sources
		// Subworkflow sources should not show up
		sw.Sources = tt.sources
		testGCSObjs = nil
		derr := w.uploadSources(ctx)

		if tt.wantErrType == NOERR && derr != nil {
			t.Errorf("unexpected error, test case: %q; i: %s; error result: %s", tt.desc, tt.sources, derr)
		} else if tt.wantErrType != NOERR {
			if derr == nil {
				t.Errorf("should have returned error, test case: %q; i: %s", tt.desc, tt.sources)
			} else if derr.etype() != tt.wantErrType {
				t.Errorf("unexpected error, test case: %q; i: %s; want error type: %q, got error type: %q", tt.desc, tt.sources, tt.wantErrType, derr.etype())
			}
		}

		if !reflect.DeepEqual(tt.gcs, testGCSObjs) {
			t.Errorf("expected GCS objects list does not match, test case: %q; i: %s; want: %q, got: %q", tt.desc, tt.sources, tt.gcs, testGCSObjs)
		}
	}

	// Check that subworkflows report errors as well.
	w.Sources = map[string]string{}
	for _, tt := range tests {
		// Parent sources should not show up
		w.Sources = tt.sources
		sw.Sources = tt.sources
		testGCSObjs = nil

		derr := sw.uploadSources(ctx)
		if tt.wantErrType == NOERR && derr != nil {
			t.Errorf("unexpected error, test case: %q; i: %s; error result: %s", tt.desc, tt.sources, derr)
		} else if tt.wantErrType != NOERR {
			if derr == nil {
				t.Errorf("should have returned error, test case: %q; i: %s", tt.desc, tt.sources)
			} else if derr.etype() != tt.wantErrType {
				t.Errorf("unexpected error, test case: %q; i: %s; want error type: %q, got error type: %q", tt.desc, tt.sources, tt.wantErrType, derr.etype())
			}
		}

		// Test cases were built for the parent workflow, not the subworkflow.
		// Modify the expected GCS paths to match the subworkflow.
		for i, s := range tt.gcs {
			tt.gcs[i] = strings.TrimPrefix(s, w.sourcesPath)
			tt.gcs[i] = sw.sourcesPath + tt.gcs[i]
		}
		if !reflect.DeepEqual(tt.gcs, testGCSObjs) {
			t.Errorf("expected GCS objects list does not match, test case: %q; i: %s; want: %q, got: %q", tt.desc, tt.sources, tt.gcs, testGCSObjs)
		}
	}
}
