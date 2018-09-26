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
	"flag"
	"io/ioutil"
	"log"
	"os"
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

// TODO: make this configurable.
const interval = 10 * time.Minute

func strIn(s string, ss []string) bool {
	for _, x := range ss {
		if s == x {
			return true
		}
	}
	return false
}

func main() {
	flag.Parse()
	patchInit()

	ctx := context.Background()

	client, err := osconfig.NewClient(ctx, option.WithEndpoint(*endpoint), option.WithCredentialsFile(*oauth))
	if err != nil {
		log.Fatal(err)
	}

	res, err := lookupConfigs(ctx, client, *resource)
	if err != nil {
		log.Fatal(err)
	}
	patchManager(res.PatchPolicies)

	//runUpdates()
}

func run(ctx context.Context) {
	agentDisabled := false

	ticker := time.NewTicker(interval)
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
