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

package files

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestExists(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Return true when directory exists",
			path:     makeTempDir(t),
			expected: true,
		},
		{
			name:     "Return true when file exists",
			path:     makeTempFile(t),
			expected: true,
		},
		{
			name: "Return false when file not found",
			path: makeNotExistantFile(t),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Exists(tt.path))
		})
	}
}

func TestDirectoryExists(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Return true when directory exists",
			path:     makeTempDir(t),
			expected: true,
		},
		{
			name: "Return true when path is a file",
			path: makeTempFile(t),
		},
		{
			name: "Return false when path not found",
			path: makeNotExistantFile(t),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, DirectoryExists(tt.path))
		})
	}
}

func TestMakeAbsolute_HappyCase(t *testing.T) {
	// Return to the test directory after running the test.
	curr, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(curr)

	// Change to a temporary directory, and write a file.
	tmpDir := makeTempDir(t)
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Create("tmp.txt")
	if err != nil {
		t.Fatal(err)
	}

	expected := path.Join(tmpDir, "tmp.txt")
	actual := MakeAbsolute("tmp.txt")
	assert.Equal(t, expected, actual)
}

func TestMakeAbsolute_PanicWhenTargetDoesntExist(t *testing.T) {
	assert.Panics(t, func() {
		MakeAbsolute(makeNotExistantFile(t))
	})
}

func TestMakeAbsolute_WhenParamIsAbsolute_DontModify(t *testing.T) {
	absoluteDir := makeTempDir(t)
	assert.True(t, filepath.IsAbs(absoluteDir))
	assert.Equal(t, absoluteDir, MakeAbsolute(absoluteDir))
}

// makeNotExistantFile returns a filesystem path that is guaranteed to *not* point to a file.
func makeNotExistantFile(t *testing.T) string {
	notAFile := uuid.New().String()
	_, err := os.Stat(notAFile)
	if !os.IsNotExist(err) {
		t.Fatalf("Expected %s to not exist", notAFile)
	}
	return notAFile
}

// makeTempFile returns the path to a new file in a temporary directory.
func makeTempFile(t *testing.T) string {
	tmpFileObj, err := os.CreateTemp("", "*.txt")
	assert.NoError(t, err)
	tmpFile := tmpFileObj.Name()
	return tmpFile
}

// makeTempDir returns the path to a new temporary directory.
func makeTempDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	return tmpDir
}
