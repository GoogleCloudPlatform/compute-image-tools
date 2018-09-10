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
	"time"

	osconfig "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/type/timeofday"
)

var (
	oauth    = flag.String("oauth", "", "path to oauth json file")
	resource = flag.String("resource", "", "projects/*/zones/*/instances/*")
	endpoint = flag.String("endpoint", "osconfig.googleapis.com:443", "osconfig endpoint override")
)

var dump = &pretty.Config{IncludeUnexported: true}

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
	ctx := context.Background()

	fmt.Println("-- register foo, should not run")
	fmt.Println("-- register bar, should complete then deregister")
	fmt.Println("-- register baz, should run then timeout")
	fmt.Println("-- register flipyflappy, should request reboot and exit and run imediately next time")

	now := time.Now().UTC()
	time.Sleep(5 * time.Second)
	patchManager(
		[]*osconfigpb.PatchPolicy{
			&osconfigpb.PatchPolicy{
				Name: "foo",
				PatchWindow: &osconfigpb.PatchWindow{
					Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
					StartTime: &timeofday.TimeOfDay{Hours: int32(now.Hour()), Minutes: int32(now.Minute())},
					Duration:  &duration.Duration{Seconds: 10},
				},
			},
			/*&osconfigpb.PatchPolicy{
				Name: "bar",
				PatchWindow: &osconfigpb.PatchWindow{
					Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
					StartTime: &timeofday.TimeOfDay{Hours: int32(now.Hour()), Minutes: int32(now.Minute()), Seconds: int32(now.Second())},
					Duration:  &duration.Duration{Seconds: 10},
				},
			},
			&osconfigpb.PatchPolicy{
				Name: "baz",
				PatchWindow: &osconfigpb.PatchWindow{
					Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
					StartTime: &timeofday.TimeOfDay{Hours: int32(now.Hour()), Minutes: int32(now.Minute()), Seconds: int32(now.Add(5 * time.Second).Second())},
					Duration:  &duration.Duration{Seconds: 4},
				},
			},
			&osconfigpb.PatchPolicy{
				Name: "flipyflappy",
				PatchWindow: &osconfigpb.PatchWindow{
					Frequency: &osconfigpb.PatchWindow_Daily_{Daily: &osconfigpb.PatchWindow_Daily{}},
					StartTime: &timeofday.TimeOfDay{Hours: int32(now.Hour()), Minutes: int32(now.Minute()), Seconds: int32(now.Add(6 * time.Second).Second())},
					Duration:  &duration.Duration{Seconds: 60},
				},
			},*/
		},
	)

	//fmt.Println("----sleep----")
	//time.Sleep(30 * time.Second)

//	return

	client, err := osconfig.NewClient(ctx, option.WithEndpoint(*endpoint), option.WithCredentialsFile(*oauth))
	if err != nil {
		log.Fatal(err)
	}

	res, err := lookupConfigs(ctx, client, *resource)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("DEBUG: LookupConfigs response:\n%s\n\n", dump.Sprint(res))
	//fmt.Printf("%+v\n", res.PatchPolicies)

	//runPackageConfig(res)
	//runUpdates()

}
