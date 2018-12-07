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
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/apipoller"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/inventory"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/service"
)

func main() {
	flag.Parse()
	ctx := context.Background()

	action := flag.Arg(0)
	if action == "inventory" {
		inventory.RunInventory()
		os.Exit(0)
	}

	if err := service.Register(ctx, "google_osconfig_agent", "Google OSConfig Agent", "", apipoller.Poll, action); err != nil {
		logger.Fatalf("service.Register error: %v", err)
	}
}
