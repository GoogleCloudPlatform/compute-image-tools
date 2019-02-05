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

package patch

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/attributes"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
)

const state = "osconfig_patch.state"

func loadState(state string) (*patchRun, error) {
	d, err := ioutil.ReadFile(state)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var pw patchRun
	return &pw, json.Unmarshal(d, &pw)
}

func saveState(state string, w *patchRun) error {
	if w == nil {
		return ioutil.WriteFile(state, []byte("{}"), 0600)
	}

	d, err := json.Marshal(w)
	if err != nil {
		return err
	}

	// This sends state to guest attributes and isn't required for patching to work.
	if err := attributes.PostAttribute(config.ReportURL+"/osConfig/patchRunner", bytes.NewReader(d)); err != nil {
		logger.Debugf("postAttribute error: %v", err)
	}

	// TODO: Once we are storing more state consider atomic state save
	// operations.
	return ioutil.WriteFile(state, d, 0600)
}
