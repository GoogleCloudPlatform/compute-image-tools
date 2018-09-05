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

	osconfig "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	"google.golang.org/api/option"
)

var (
	oauth    = flag.String("oauth", "", "path to oauth json file")
	resource = flag.String("resource", "", "projects/*/zones/*/instances/*")
	endpoint = flag.String("endpoint", "osconfig.googleapis.com:443", "osconfig endpoint override")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	client, err := osconfig.NewClient(ctx, option.WithEndpoint(*endpoint), option.WithCredentialsFile(*oauth))
	if err != nil {
		log.Fatal(err)
	}

	res, err := lookupConfigs(ctx, client, *resource)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", res)
	fmt.Printf("%+v\n", res.Apt)
	fmt.Printf("%+v\n", res.Goo)
	fmt.Printf("%+v\n", res.Yum)
	fmt.Printf("%+v\n", res.WindowsUpdate)
	//fmt.Printf("%+v\n", res.PatchPolicies)
	runPackageConfig(res)

	//runUpdates()

}
