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

// Package attributes posts data to Guest Attributes
package attributes

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// PostAttribute posts data to Guest Attributes
func PostAttribute(url string, value io.Reader) error {
	req, err := http.NewRequest("PUT", url, value)
	if err != nil {
		return err
	}
	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(`received status code %q for request "%s %s"`, resp.Status, req.Method, req.URL.String())
	}
	return nil
}

// PostAttributeCompressed compresses and posts data to Guest Attributes
func PostAttributeCompressed(url string, body interface{}) error {
	buf := &bytes.Buffer{}
	b := base64.NewEncoder(base64.StdEncoding, buf)
	zw := gzip.NewWriter(b)
	w := json.NewEncoder(zw)
	if err := w.Encode(body); err != nil {
		return err
	}

	if err := zw.Close(); err != nil {
		return err
	}
	if err := b.Close(); err != nil {
		return err
	}

	return PostAttribute(url, buf)
}
