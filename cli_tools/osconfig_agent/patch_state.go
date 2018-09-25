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
	"encoding/json"
	"io/ioutil"
	"os"
)

const state = "osconfig_patch.state"

func loadState(state string) (*patchWindow, error) {
	d, err := ioutil.ReadFile(state)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var pw patchWindow
	return &pw, json.Unmarshal(d, &pw)
}

func saveState(state string, w *patchWindow) error {
	if w == nil {
		return ioutil.WriteFile(state, []byte("{}"), 0600)
	}

	w.mx.RLock()
	defer w.mx.RUnlock()

	d, err := json.Marshal(w)
	if err != nil {
		return err
	}

	// TODO: Once we are storing more state consider atomic state save
	// operations.
	return ioutil.WriteFile(state, d, 0600)
}
