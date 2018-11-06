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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/go/osinfo"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/service"
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
	InstalledPackages packages.Packages
	PackageUpdates    packages.Packages
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

	return postAttribute(url, buf)
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

// disabled checks if the inventory agent is disabled in either instance or
// project metadata.
// Instance metadata takes precedence.
func disabled(md *metadataJSON) bool {
	disabled, err := strconv.ParseBool(md.Instance.Attributes.DisableInventoryAgent)
	if err == nil {
		return disabled
	}
	disabled, err = strconv.ParseBool(md.Project.Attributes.DisableInventoryAgent)
	if err == nil {
		return disabled
	}
	return false
}

func getInventory() *instanceInventory {
	logger.Info("Gathering instance inventory.")

	hs := &instanceInventory{}

	hn, err := os.Hostname()
	if err != nil {
		hs.Errors = append(hs.Errors, err.Error())
	}

	hs.Hostname = hn

	di, err := osinfo.GetDistributionInfo()
	if err != nil {
		hs.Errors = append(hs.Errors, err.Error())
	}

	hs.LongName = di.LongName
	hs.ShortName = di.ShortName
	hs.Version = di.Version
	hs.KernelVersion = di.Kernel
	hs.Architecture = di.Architecture

	var errs []string
	hs.InstalledPackages, errs = packages.GetInstalledPackages(packages.Run)
	if len(errs) != 0 {
		hs.Errors = append(hs.Errors, errs...)
	}

	hs.PackageUpdates, errs = packages.GetPackageUpdates(packages.Run)
	if len(errs) != 0 {
		hs.Errors = append(hs.Errors, errs...)
	}

	return hs
}

func run(ctx context.Context) {
	agentDisabled := false

	ticker := time.NewTicker(30 * time.Minute)
	for {
		md, err := getMetadata(ctx)
		if err != nil {
			logger.Error(err)
			continue
		}

		if disabled(md) {
			if !agentDisabled {
				logger.Info("GCE inventory agent disabled by metadata")
			}
			agentDisabled = true
			continue
		}

		agentDisabled = false

		writeInventory(getInventory(), reportURL)

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	logger.Init("gce_inventory_agent", true, false, ioutil.Discard)
	ctx := context.Background()

	var action string
	if len(os.Args) > 1 {
		action = os.Args[1]
	}
	if action == "noservice" {
		writeInventory(getInventory(), reportURL)
		os.Exit(0)
	}
	if err := service.Register(ctx, "gce_inventory_agent", "GCE Inventory Agent", "", run, action); err != nil {
		logger.Fatal(err)
	}
}
