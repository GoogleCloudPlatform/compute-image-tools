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

package packages

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
)

var pkgs = []string{"pkg1", "pkg2"}

func getMockRun(content []byte, err error) func(cmd *exec.Cmd) ([]byte, error) {
	return func(cmd *exec.Cmd) ([]byte, error) {
		return content, err
	}
}

// TODO: move this to a common helper package
func helperLoadBytes(name string) ([]byte, error) {
	path := filepath.Join("testdata", name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
