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

package ospatch

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
)

const (
	defaultEtag = ""
)

var (
	metadataURL         = "http://metadata.google.internal/computeMetadata/v1/instance/attributes/?recursive=true&wait_for_change=true&last_etag="
	currentPatchJobName = ""
)

type watchMetadataRet struct {
	attr *attributesJSON
	err  error
}

type attributesJSON struct {
	PatchNotify string `json:"osconfig-patch-notify"`
}

func watchMetadata(c chan watchMetadataRet, etag *string) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", metadataURL+*etag, nil)
	if err != nil {
		c <- watchMetadataRet{
			attr: nil,
			err:  err,
		}
		return
	}

	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		c <- watchMetadataRet{
			attr: nil,
			err:  err,
		}
		return
	}
	*etag = resp.Header.Get("etag")

	md, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		c <- watchMetadataRet{
			attr: nil,
			err:  err,
		}
		return
	}

	var metadata attributesJSON
	err = json.Unmarshal(md, &metadata)

	c <- watchMetadataRet{
		attr: &metadata,
		err:  err,
	}
}

func formatError(err error) string {
	if urlErr, ok := err.(*url.Error); ok {
		if _, ok := urlErr.Err.(*net.DNSError); ok {
			return fmt.Sprintf("DNS error when requesting metadata, check DNS settings and ensure metadata.internal.google is setup in your hosts file.")
		}
		if _, ok := urlErr.Err.(*net.OpError); ok {
			return fmt.Sprintf("Network error when requesting metadata, make sure your instance has an active network and can reach the metadata server.")
		}
	}
	return err.Error()
}

func watcher(ctx context.Context, cancel <-chan struct{}, action func(context.Context, string)) {
	webError := 0
	// We use a pointer so that each loops goroutine can update this.
	// If this was a global var we would have a data race, and puting
	// locks around it is more work than necessary.
	etag := func() *string { e := defaultEtag; return &e }()
	for {
		c := make(chan watchMetadataRet)
		go func(c chan watchMetadataRet, etag *string) {
			watchMetadata(c, etag)
		}(c, etag)

		select {
		case <-cancel:
			return
		case ret := <-c:
			if ret.err != nil {
				// Only log the second web error to avoid transient errors and
				// not to spam the log on network failures.
				if webError == 1 {
					logger.Errorf(formatError(ret.err))
				}
				webError++
				time.Sleep(5 * time.Second)
				continue
			}

			patchJobName := strings.Split(ret.attr.PatchNotify, ",")[0]
			if patchJobName == "" {
				continue
			}
			action(ctx, patchJobName)

			webError = 0
		}
	}
}
