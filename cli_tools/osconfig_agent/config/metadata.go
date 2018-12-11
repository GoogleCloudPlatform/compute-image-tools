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

package config

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/logger"
)

const (
	metadataRecursive = instanceMetadata + "/?recursive=true&alt=json"
)

type instanceMetadataJSON struct {
	ID   int
	Zone string
}

func getInstanceMetadata() (*instanceMetadataJSON, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", metadataRecursive, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Metadata-Flavor", "Google")

	var res *http.Response
	// Retry forever, increase sleep between retries (up to 20s) in order
	// to wait for slow network initialization.
	for i := 1; ; i++ {
		res, err = client.Do(req)
		if err == nil {
			break
		}
		rt := time.Duration(3*i) * time.Second
		maxRetryDelay := MaxMetadataRetryDelay()
		if rt > maxRetryDelay {
			rt = maxRetryDelay
		}
		logger.Errorf("Error connecting to metadata server (error number: %d), retrying in %s, error: %v\n", i, rt, err)
		time.Sleep(rt)
	}
	defer res.Body.Close()

	dec := json.NewDecoder(res.Body)
	var m instanceMetadataJSON
	for {
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}
	return &m, nil
}
