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

// osconfig_agent interacts with the osconfig api.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	osconfig "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	service "github.com/GoogleCloudPlatform/compute-image-tools/service_library"
	"github.com/GoogleCloudPlatform/compute-image-windows/logger"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/option"
)

var (
	oauth    = flag.String("oauth", "", "path to oauth json file")
	resource = flag.String("resource", "", "projects/*/zones/*/instances/*")
	endpoint = flag.String("endpoint", "osconfig.googleapis.com:443", "osconfig endpoint override")
)

var dump = &pretty.Config{IncludeUnexported: true}

const (
	// TODO: make this configurable.
	interval    = 10 * time.Minute
	metadataURL = "http://metadata.google.internal/computeMetadata/v1/instance/?recursive=true&alt=json"
)

type metadataJSON struct {
	ID   int
	Zone string
}

func getResource(r string) (string, error) {
	if r != "" {
		return r, nil
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", metadataURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Metadata-Flavor", "Google")

	var res *http.Response
	// Retry forever, increase sleep between retries (up to 5 times) in order
	// to wait for slow network initialization.
	var rt time.Duration
	for i := 1; ; i++ {
		res, err = client.Do(req)
		if err == nil {
			break
		}
		if i < 6 {
			rt = time.Duration(3*i) * time.Second
		}
		logger.Errorf("error connecting to metadata server, retrying in %s, error: %v", rt, err)
		time.Sleep(rt)
	}
	defer res.Body.Close()

	dec := json.NewDecoder(res.Body)
	var m metadataJSON
	for {
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s/instances/%d", m.Zone, m.ID), nil
}

func strIn(s string, ss []string) bool {
	for _, x := range ss {
		if s == x {
			return true
		}
	}
	return false
}

func run(ctx context.Context) {
	client, err := osconfig.NewClient(ctx, option.WithEndpoint(*endpoint), option.WithCredentialsFile(*oauth))
	if err != nil {
		log.Fatalln("NewClient Error:", err)
	}

	res, err := getResource(*resource)
	if err != nil {
		log.Fatalln("getResource error:", err)
	}

	patchInit()
	ticker := time.NewTicker(interval)
	for {
		res, err := lookupConfigs(ctx, client, res)
		if err != nil {
			log.Println("ERROR:", err)
		} else {
			patchManager(res.PatchPolicies)
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	flag.Parse()
	ctx := context.Background()

	if err := service.Register(ctx, "google_osconfig_agent", "Google OSConfig Agent", "", run, flag.Arg(0)); err != nil {
		log.Fatal(err)
	}
}
