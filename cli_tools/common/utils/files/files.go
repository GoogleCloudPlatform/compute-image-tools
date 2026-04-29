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
	"fmt"
	"os"
	"path/filepath"
)

// DirectoryExists returns whether dir references an existing directory.
func DirectoryExists(dir string) bool {
	stat, err := os.Stat(dir)
	return !os.IsNotExist(err) && stat.IsDir()
}

// Exists returns whether path references an existing file or directory.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// MakeAbsolute converts path to absolute, relative to the process's current working directory.
// Panics if path isn't found.
func MakeAbsolute(path string) string {
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		path = filepath.Join(wd, path)
	}
	if !Exists(path) {
		panic(fmt.Sprintf("%s: File not found", path))
	}
	return path
}
