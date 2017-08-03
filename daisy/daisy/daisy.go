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

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
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

func parseWorkflow(ctx context.Context, path string, varMap map[string]string, project, zone, gcsPath, oauth, cEndpoint, sEndpoint string) (*daisy.Workflow, error) {
	w, err := daisy.NewFromFile(path)
	if err != nil {
		return nil, err
	}
	for k, v := range varMap {
		w.AddVar(k, v)
	}

	if project != "" {
		w.Project = project
	} else if w.Project == "" && metadata.OnGCE() {
		w.Project, err = metadata.ProjectID()
		if err != nil {
			return nil, err
		}
	}
	if zone != "" {
		w.Zone = zone
	} else if w.Zone == "" && metadata.OnGCE() {
		w.Zone, err = metadata.Zone()
		if err != nil {
			return nil, err
		}
	}
	if gcsPath != "" {
		w.GCSPath = gcsPath
	}
	if oauth != "" {
		w.OAuthPath = oauth
	}

	if cEndpoint != "" {
		w.ComputeClient, err = compute.NewClient(ctx, option.WithEndpoint(cEndpoint), option.WithCredentialsFile(w.OAuthPath))
		if err != nil {
			return nil, err
		}
	}

	if sEndpoint != "" {
		w.StorageClient, err = storage.NewClient(ctx, option.WithEndpoint(sEndpoint), option.WithCredentialsFile(w.OAuthPath))
		if err != nil {
			return nil, err
		}
	}

	return w, nil
}

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("Not enough args, first arg needs to be the path to a workflow.")
	}
	ctx := context.Background()

	var ws []*daisy.Workflow
	varMap := splitVariables(*variables)
	for _, path := range flag.Args() {
		w, err := parseWorkflow(ctx, path, varMap, *project, *zone, *gcsPath, *oauth, *ce, *se)
		if err != nil {
			log.Fatalf("error parsing workflow %q: %v", path, err)
		}
		ws = append(ws, w)
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
			w.Print(ctx)
			continue
		}
		if *validate {
			fmt.Printf("[Daisy] Validating workflow %q\n", w.Name)
			if err := w.Validate(ctx); err != nil {
				fmt.Fprintln(os.Stderr, "[Daisy] Error validating workflow:", err)
			}
			continue
		}
		wg.Add(1)
		go func(wf *daisy.Workflow) {
			defer wg.Done()
			fmt.Printf("[Daisy] Running workflow %q\n", wf.Name)
			if err := wf.Run(ctx); err != nil {
				errors <- fmt.Errorf("%s: %v", wf.Name, err)
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
