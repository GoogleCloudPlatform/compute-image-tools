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
	"log"
	"path/filepath"
	"reflect"
	"runtime"
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
	sw.logger = log.New(ioutil.Discard, "", 0)
	w.Steps = map[string]*Step{
		"sub": {w: w, SubWorkflow: &SubWorkflow{w: sw}},
	}
	if err := w.populate(ctx); err != nil {
		t.Fatal(err)
	}

	localDNEErr := "stat this/file/dne: no such file or directory"
	if runtime.GOOS == "windows" {
		localDNEErr = "GetFileAttributesEx this\\file\\dne: The system cannot find the path specified."
	}
	tests := []struct {
		desc    string
		sources map[string]string
		err     string
		gcs     []string
	}{
		{"normal local file to GCS", map[string]string{"local": testPath}, "", []string{w.sourcesPath + "/local"}},
		{"normal local folder to GCS", map[string]string{"local": dir}, "", []string{w.sourcesPath + "/local/test"}},
		{"normal GCS obj to GCS", map[string]string{"gcs": "gs://gcs/file"}, "", []string{w.sourcesPath + "/gcs"}},
		{"normal GCS bkt to GCS", map[string]string{"gcs": "gs://gcs/folder/"}, "", []string{w.sourcesPath + "/gcs/object", w.sourcesPath + "/gcs/folder/object"}},
		{"dne local path", map[string]string{"local": "./this/file/dne"}, localDNEErr, nil},
		{"dne GCS path", map[string]string{"gcs": "gs://gcs/path/dne"}, `error copying from file gs://gcs/path/dne: googleapi: got HTTP response code 404 with body: storage: object doesn't exist`, nil},
		{"GCS path, no object", map[string]string{"gcs": "gs://folder"}, "", []string{w.sourcesPath + "/gcs/object", w.sourcesPath + "/gcs/folder/object"}},
	}

	for _, tt := range tests {
		w.Sources = tt.sources
		testGCSObjs = nil
		err = w.uploadSources(ctx)
		if tt.err != "" && err == nil {
			t.Errorf("should have returned error, test case: %q; input: %s", tt.desc, tt.sources)
		} else if tt.err != "" && err != nil && err.Error() != tt.err {
			t.Errorf("unexpected error, test case: %q; input: %s; want error: %s, got error: %s", tt.desc, tt.sources, tt.err, err)
		} else if tt.err == "" && err != nil {
			t.Errorf("unexpected error, test case: %q; input: %s; error result: %s", tt.desc, tt.sources, err)
		}
		if !reflect.DeepEqual(tt.gcs, testGCSObjs) {
			t.Errorf("expected GCS objects list does not match, test case: %q; input: %s; want: %q, got: %q", tt.desc, tt.sources, tt.gcs, testGCSObjs)
		}
	}

	// Check that subworkflows report errors as well.
	w.Sources = map[string]string{}
	for _, tt := range tests {
		sw.Sources = tt.sources
		testGCSObjs = nil
		err = w.uploadSources(ctx)
		if tt.err != "" && err == nil {
			t.Errorf("should have returned error, test case: %q; input: %s", tt.desc, tt.sources)
		} else if tt.err != "" && err != nil && err.Error() != tt.err {
			t.Errorf("unexpected error, test case: %q; input: %s; want error: %s, got error: %s", tt.desc, tt.sources, tt.err, err)
		} else if tt.err == "" && err != nil {
			t.Errorf("unexpected error, test case: %q; input: %s; error result: %s", tt.desc, tt.sources, err)
		}
		// Test cases were built for the parent workflow, not the subworkflow.
		// Modify the expected GCS paths to match the subworkflow.
		for i, s := range tt.gcs {
			tt.gcs[i] = strings.TrimPrefix(s, w.sourcesPath)
			tt.gcs[i] = sw.sourcesPath + tt.gcs[i]
		}
		if !reflect.DeepEqual(tt.gcs, testGCSObjs) {
			t.Errorf("expected GCS objects list does not match, test case: %q; input: %s; want: %q, got: %q", tt.desc, tt.sources, tt.gcs, testGCSObjs)
		}
	}
}
