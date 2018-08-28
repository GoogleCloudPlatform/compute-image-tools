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
	"fmt"
	"log"

	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

var (
	oauth    = flag.String("oauth", "", "path to oauth json file")
	resource = flag.String("resource", "", "projects/*/zones/*/instances/*")
	basePath = flag.String("base_path", "https://staging-osconfig.sandbox.googleapis.com/v1alpha1/", "")
)

const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

func main() {
	flag.Parse()
	ctx := context.Background()

	hc, _, err := transport.NewHTTPClient(ctx, option.WithScopes(cloudPlatformScope), option.WithCredentialsFile(*oauth))
	if err != nil {
		log.Fatal(err)
	}

	res, err := lookupConfigs(hc, *basePath, *resource)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", res)
	runPackageConfig(res)

	//runUpdates()

}
