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

// Daisy is a GCE workflow tool.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/workflow"
)

var (
	oauth     = flag.String("oauth", "", "path to oauth json file, overrides what is set in workflow")
	project   = flag.String("project", "", "project to run in, overrides what is set in workflow")
	gcsPath   = flag.String("gcs_path", "", "GCS bucket to use, overrides what is set in workflow")
	zone      = flag.String("zone", "", "zone to run in, overrides what is set in workflow")
	variables = flag.String("variables", "", "comma separated list of variables, in the form 'key=value'")
	// TODO(ajackura): Implement the endpoint overrides.
	//ce      = flag.String("compute_endpoint_override", "", "API endpoint to override default")
	//se      = flag.String("storage_endpoint_override", "", "API endpoint to override default")
)

func splitVariables(input string) map[string]string {
	varMap := map[string]string{}
	if input == "" {
		return varMap
	}
	for _, v := range strings.Split(input, ",") {
		i := strings.Index(v, "=")
		if i == -1 {
			continue
		}
		varMap[v[:i]] = v[i+1:]
	}
	return varMap
}

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("Not enough args, first arg needs to be the path to a workflow.")
	}

	varMap := splitVariables(*variables)

	var wfs []*workflow.Workflow
	for _, path := range flag.Args() {
		wf, err := workflow.NewFromFile(context.Background(), path)
		if err != nil {
			log.Fatal(err)
		}
		for k, v := range varMap {
			wf.Vars[k] = v
		}
		if *project != "" {
			wf.Project = *project
		}
		if *zone != "" {
			wf.Zone = *zone
		}
		if *gcsPath != "" {
			wf.GCSPath = *gcsPath
		}
		if *oauth != "" {
			wf.OAuthPath = *oauth
		}

		wfs = append(wfs, wf)
	}

	errors := make(chan error, len(wfs))
	var wg sync.WaitGroup
	for _, wf := range wfs {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			select {
			case <-c:
				fmt.Printf("\nCtrl-C caught, stopping and cleaning up workflow %q, this may take a second...\n", wf.Name)
				close(wf.Cancel)
				errors <- fmt.Errorf("workflow %q was canceled", wf.Name)
			case <-wf.Cancel:
			}
		}()
		wg.Add(1)
		go func(wf *workflow.Workflow) {
			defer wg.Done()
			fmt.Printf("[Daisy] Running workflow %q\n", wf.Name)
			if err := wf.Run(); err != nil {
				errors <- err
				return
			}
			fmt.Printf("[Daisy] Workflow %q finished\n", wf.Name)
		}(wf)
	}
	wg.Wait()

	select {
	case err := <-errors:
		fmt.Fprintln(os.Stderr, "\n[Daisy] Errors in one or more workflows:")
		fmt.Fprintln(os.Stderr, " ", err)
		for {
			select {
			case err := <-errors:
				fmt.Fprintln(os.Stderr, " ", err)
				continue
			default:
				os.Exit(1)
			}
		}
	default:
		fmt.Println("[Daisy] All workflows completed successfully.")
	}
}
