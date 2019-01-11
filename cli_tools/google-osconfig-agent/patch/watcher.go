//  Copyright 2019 Google Inc. All Rights Reserved.
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
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
)

const defaultEtag = "NONE"

var (
	metadataURL = "http://metadata.google.internal/computeMetadata/v1/instance/attributes/osconfig-patch-notify?wait_for_change=true&last_etag="
	etag        = defaultEtag
)

func updateEtag(resp *http.Response) bool {
	oldEtag := etag
	etag = resp.Header.Get("etag")
	if etag == "" {
		etag = defaultEtag
	}
	return etag != oldEtag
}

func watchMetadata() (string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", metadataURL+etag, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	updateEtag(resp)

	md, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}

	return string(md), nil
}

func watcher(ctx context.Context) {
	webError := 0
	for {
		md, err := watchMetadata()
		if err != nil {
			// Only log the second web error to avoid transient errors and
			// not to spam the log on network failures.
			if webError == 1 {
				if urlErr, ok := err.(*url.Error); ok {
					if _, ok := urlErr.Err.(*net.DNSError); ok {
						logger.Errorf("DNS error when requesting metadata, check DNS settings and ensure metadata.internal.google is setup in your hosts file.")
					}
					if _, ok := urlErr.Err.(*net.OpError); ok {
						logger.Errorf("Network error when requesting metadata, make sure your instance has an active network and can reach the metadata server.")
					}
				}
				logger.Errorf(err.Error())
			}
			webError++
			time.Sleep(5 * time.Second)
			continue
		}
		id := strings.Split(md, ",")[0]
		if id != "" {
			ackPatch(ctx, id)
		}
		webError = 0
	}
}
