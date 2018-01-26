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

// gce_inventory_agent gathers and writes instance inventory to guest attributes.
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/osinfo"
	"github.com/GoogleCloudPlatform/compute-image-tools/packages"
	"github.com/google/logger"
)

const (
	reportURL = "http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/guestInventory"
)

type instanceInventory struct {
	Hostname          string
	LongName          string
	ShortName         string
	Version           string
	Architecture      string
	KernelVersion     string
	InstalledPackages map[string][]packages.PkgInfo
	PackageUpdates    map[string][]packages.PkgInfo
	Errors            []string
}

func postAttribute(url string, value io.Reader) error {
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

func postAttributeCompressed(url string, body interface{}) error {
	msg, err := json.Marshal(body)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	if _, err := zw.Write(msg); err != nil {
		return err
	}

	if err := zw.Close(); err != nil {
		return err
	}

	return postAttribute(url, strings.NewReader(base64.StdEncoding.EncodeToString(buf.Bytes())))
}

func writeInventory(state *instanceInventory, url string) {
	logger.Info("Writing instance inventory.")

	if err := postAttribute(url+"/Timestamp", strings.NewReader(time.Now().UTC().Format(time.RFC3339))); err != nil {
		state.Errors = append(state.Errors, err.Error())
		logger.Error(err)
	}

	e := reflect.ValueOf(state).Elem()
	t := e.Type()
	for i := 0; i < e.NumField(); i++ {
		f := e.Field(i)
		u := fmt.Sprintf("%s/%s", url, t.Field(i).Name)
		switch f.Kind() {
		case reflect.String:
			if err := postAttribute(u, strings.NewReader(f.String())); err != nil {
				state.Errors = append(state.Errors, err.Error())
				logger.Error(err)
			}
		case reflect.Map:
			if err := postAttributeCompressed(u, f.Interface()); err != nil {
				state.Errors = append(state.Errors, err.Error())
				logger.Error(err)
			}
		}
	}
	if err := postAttribute(url+"/Errors", strings.NewReader(fmt.Sprintf("%q", state.Errors))); err != nil {
		logger.Error(err)
	}
}

func main() {
	logger.Init("inventory", true, false, ioutil.Discard)
	logger.Info("Gathering instance inventory.")

	hn, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}

	di, err := osinfo.GetDistributionInfo()
	if err != nil {
		logger.Fatal(err)
	}

	hs := &instanceInventory{
		Hostname:      hn,
		LongName:      di.LongName,
		ShortName:     di.ShortName,
		Version:       di.Version,
		KernelVersion: di.Kernel,
		Architecture:  di.Architecture,
	}

	var errs []string
	hs.InstalledPackages, errs = packages.GetInstalledPackages()
	if len(errs) != 0 {
		hs.Errors = append(hs.Errors, errs...)
	}

	hs.PackageUpdates, errs = packages.GetPackageUpdates()
	if len(errs) != 0 {
		hs.Errors = append(hs.Errors, errs...)
	}

	writeInventory(hs, reportURL)
}
