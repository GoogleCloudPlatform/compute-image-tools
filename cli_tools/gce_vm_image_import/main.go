//  Copyright 2018 Google Inc. All Rights Reserved.
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

package main

import (
	"os"
	"path"
	"path/filepath"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/cli"
)

// logPrefix is a string that conforms to gcloud's output filter.
// To ensure that a line is shown by gcloud, emit a line to stdout
// using this string surrounded in brackets.
const logPrefix = "[import-image]"

func main() {
	// Directory where workflows are located; in this case, the value indicates that
	// that the `daisy_workflows` directory is located in the same directory as the current binary.
	workflowDir := path.Join(filepath.Dir(os.Args[0]), "daisy_workflows")
	toolLogger := logging.NewToolLogger(logPrefix)
	if err := cli.Main(os.Args[1:], toolLogger, workflowDir); err != nil {
		// Main is responsible for logging the failure.
		os.Exit(1)
	}
}
