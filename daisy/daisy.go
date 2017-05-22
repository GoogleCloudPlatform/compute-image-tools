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

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/workflow"
	"google.golang.org/api/option"
)

var (
	oauth     = flag.String("oauth", "", "path to oauth json file, overrides what is set in workflow")
	project   = flag.String("project", "", "project to run in, overrides what is set in workflow")
	gcsPath   = flag.String("gcs_path", "", "GCS bucket to use, overrides what is set in workflow")
	zone      = flag.String("zone", "", "zone to run in, overrides what is set in workflow")
	variables = flag.String("variables", "", "comma separated list of variables, in the form 'key=value'")
	print     = flag.Bool("print", false, "print out the parsed workflow for debugging")
	validate  = flag.Bool("validate", false, "validate the workflow and exit")
	ce        = flag.String("compute_endpoint_override", "", "API endpoint to override default")
	se        = flag.String("storage_endpoint_override", "", "API endpoint to override default")
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

func parseWorkflows(paths []string, varMap map[string]string, project, zone, gcsPath, oauth, cEndpoint, sEndpoint string) ([]*workflow.Workflow, error) {
	var ws []*workflow.Workflow
	for _, path := range paths {
		w, err := workflow.NewFromFile(context.Background(), path)
		if err != nil {
			return nil, err
		}
		for k, v := range varMap {
			w.AddVar(k, v)
		}
		if project != "" {
			w.Project = project
		}
		if zone != "" {
			w.Zone = zone
		}
		if gcsPath != "" {
			w.GCSPath = gcsPath
		}
		if oauth != "" {
			w.OAuthPath = oauth
		}

		if cEndpoint != "" {
			w.ComputeClient, err = compute.NewClient(w.Ctx, option.WithEndpoint(cEndpoint), option.WithServiceAccountFile(w.OAuthPath))
			if err != nil {
				return nil, err
			}
		}

		if sEndpoint != "" {
			w.StorageClient, err = storage.NewClient(w.Ctx, option.WithEndpoint(sEndpoint), option.WithServiceAccountFile(w.OAuthPath))
			if err != nil {
				return nil, err
			}
		}

		ws = append(ws, w)
	}
	return ws, nil
}

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("Not enough args, first arg needs to be the path to a workflow.")
	}

	varMap := splitVariables(*variables)
	ws, err := parseWorkflows(flag.Args(), varMap, *project, *zone, *gcsPath, *oauth, *ce, *se)
	if err != nil {
		log.Fatal(err)
	}

	errors := make(chan error, len(ws))
	var wg sync.WaitGroup
	for _, w := range ws {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			select {
			case <-c:
				fmt.Printf("\nCtrl-C caught, sending cancel signal to %q...\n", w.Name)
				close(w.Cancel)
				errors <- fmt.Errorf("workflow %q was canceled", w.Name)
			case <-w.Cancel:
			}
		}()
		if *print {
			fmt.Printf("[Daisy] Printing workflow %q\n", w.Name)
			w.Print()
			continue
		}
		if *validate {
			fmt.Printf("[Daisy] Validating workflow %q\n", w.Name)
			w.Validate()
			continue
		}
		wg.Add(1)
		go func(wf *workflow.Workflow) {
			defer wg.Done()
			fmt.Printf("[Daisy] Running workflow %q\n", wf.Name)
			if err := wf.Run(); err != nil {
				errors <- err
				return
			}
			fmt.Printf("[Daisy] Workflow %q finished\n", wf.Name)
		}(w)
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
		if !*print && !*validate {
			fmt.Println("[Daisy] All workflows completed successfully.")
		}
	}
}
