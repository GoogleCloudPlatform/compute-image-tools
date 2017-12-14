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

// instance_inventory pretty prints out an instances inventory data.
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/packages"
	"google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

var (
	project  = flag.String("project", "", "GCE Project")
	zone     = flag.String("zone", "", "GCE Zone")
	instance = flag.String("instance", "", "GCE Instance")
)

func getAttribute(service *compute.Service, p, z, i, at string) (string, error) {
	ga, err := service.Instances.GetGuestAttributes(p, z, i).VariableKey(at).Do()
	if err != nil {
		return "", err
	}
	return ga.VariableValue, nil
}

func main() {
	flag.Parse()
	if *project == "" {
		log.Fatal("The flag -project must be provided")
	}
	if *zone == "" {
		log.Fatal("The flag -zone must be provided")
	}
	if *instance == "" {
		log.Fatal("The flag -instance must be provided")
	}

	ctx := context.Background()

	hc, _, err := transport.NewHTTPClient(ctx, []option.ClientOption{option.WithScopes(compute.ComputeScope)}...)
	if err != nil {
		log.Fatalf("error creating HTTP client: %v", err)
	}
	service, err := compute.New(hc)
	if err != nil {
		log.Fatalf("error creating compute client: %v", err)
	}

	ats := []string{"Timestamp", "Hostname", "LongName", "ShortName", "Version", "KernelVersion", "InstalledPackages", "PackageUpdates", "Errors"}

	//ii := map[string]string{}
	var ii sync.Map
	var wg sync.WaitGroup
	for _, at := range ats {
		wg.Add(1)
		go func(at string, wg *sync.WaitGroup) {
			defer wg.Done()
			value, err := getAttribute(service, *project, *zone, *instance, at)
			if err != nil {
				log.Printf("error getting attribute %q: %v", at, err)
				return
			}
			if at == "InstalledPackages" || at == "PackageUpdates" {
				decoded, err := base64.StdEncoding.DecodeString(value)
				if err != nil {
					log.Print(err)
					return
				}

				zr, err := gzip.NewReader(bytes.NewReader(decoded))
				if err != nil {
					log.Print(err)
					return
				}

				var buf bytes.Buffer
				if _, err := io.Copy(&buf, zr); err != nil {
					log.Print(err)
					return
				}
				zr.Close()

				// Unmarshal so we can then re-marshal to pretty print.
				var pi map[string][]packages.PkgInfo
				if err := json.Unmarshal(buf.Bytes(), &pi); err != nil {
					log.Print(err)
					return
				}

				out, err := json.MarshalIndent(pi, "", "  ")
				if err != nil {
					log.Print(err)
					return
				}

				ii.Store(at, string(out))
			} else {
				ii.Store(at, value)
			}
		}(at, &wg)
	}
	wg.Wait()

	for _, at := range ats {
		v, ok := ii.Load(at)
		if !ok {
			continue
		}
		fmt.Printf("%s: %s\n", at, v.(string))
	}
}
