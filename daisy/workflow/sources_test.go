package workflow

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUploadSources(t *testing.T) {
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
	if err := w.populate(); err != nil {
		t.Fatal(err)
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
		{"dne local path", map[string]string{"local": "./this/file/dne"}, "stat ./this/file/dne: no such file or directory", nil},
		{"dne GCS path", map[string]string{"gcs": "gs://gcs/path/dne"}, `source "gs://gcs/path/dne" is not a GCS bucket or object`, nil},
		{"GCS path, no object", map[string]string{"gcs": "gs://folder"}, "", []string{w.sourcesPath + "/gcs/object", w.sourcesPath + "/gcs/folder/object"}},
	}

	for _, tt := range tests {
		w.Sources = tt.sources
		testGCSObjs = nil
		err = w.uploadSources()
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
}
