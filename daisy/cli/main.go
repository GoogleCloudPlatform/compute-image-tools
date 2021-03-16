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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

var (
	oauth              = flag.String("oauth", "", "path to oauth json file, overrides what is set in workflow")
	project            = flag.String("project", "", "project to run in, overrides what is set in workflow")
	gcsPath            = flag.String("gcs_path", "", "GCS bucket to use, overrides what is set in workflow")
	zone               = flag.String("zone", "", "zone to run in, overrides what is set in workflow")
	variables          = flag.String("variables", "", "comma separated list of variables, in the form 'key=value'")
	print              = flag.Bool("print", false, "print out the parsed workflow for debugging")
	printPerf          = flag.Bool("print_perf", false, "print out the performance profile")
	validate           = flag.Bool("validate", false, "validate the workflow and exit")
	format             = flag.Bool("format_workflow", false, "format the workflow file(s) and exit")
	defaultTimeout     = flag.String("default_timeout", "", "sets the default timeout for the workflow")
	ce                 = flag.String("compute_endpoint_override", "", "API endpoint to override default")
	gcsLogsDisabled    = flag.Bool("disable_gcs_logging", false, "do not stream logs to GCS")
	cloudLogsDisabled  = flag.Bool("disable_cloud_logging", false, "do not stream logs to Cloud Logging")
	stdoutLogsDisabled = flag.Bool("disable_stdout_logging", false, "do not display individual workflow logs on stdout")
)

const (
	flgDefValue   = "flag generated for workflow variable"
	varFlagPrefix = "var:"
)

func populateVars(input string) map[string]string {
	varMap := map[string]string{}
	if input != "" {
		for _, v := range strings.Split(input, ",") {
			i := strings.Index(v, "=")
			if i == -1 {
				continue
			}
			varMap[v[:i]] = v[i+1:]
		}
	}

	flag.Visit(func(flg *flag.Flag) {
		if strings.HasPrefix(flg.Name, varFlagPrefix) {
			varMap[strings.TrimPrefix(flg.Name, varFlagPrefix)] = flg.Value.String()
		}
	})

	return varMap
}

func parseWorkflow(ctx context.Context, path string, varMap map[string]string, project, zone, gcsPath, oauth, dTimeout, cEndpoint string, disableGCSLogs, diableCloudLogs, disableStdoutLogs bool) (*daisy.Workflow, error) {
	w, err := daisy.NewFromFile(path)
	if err != nil {
		return nil, err
	}
Loop:
	for k, v := range varMap {
		for wv := range w.Vars {
			if k == wv {
				w.AddVar(k, v)
				continue Loop
			}
		}
		return nil, fmt.Errorf("unknown workflow Var %q passed to Workflow %q", k, w.Name)
	}

	if project != "" {
		w.Project = project
	} else if w.Project == "" && metadata.OnGCE() {
		w.Project, err = metadata.ProjectID()
		if err != nil {
			return nil, fmt.Errorf("Failed to get GCE project id from metadata: %v", err)
		}
	}
	if zone != "" {
		w.Zone = zone
	} else if w.Zone == "" && metadata.OnGCE() {
		w.Zone, err = metadata.Zone()
		if err != nil {
			return nil, fmt.Errorf("Failed to get GCE zone from metadata: %v", err)
		}
	}
	if gcsPath != "" {
		w.GCSPath = gcsPath
	}
	if oauth != "" {
		w.OAuthPath = oauth
	}
	if dTimeout != "" {
		w.DefaultTimeout = dTimeout
	}

	if cEndpoint != "" {
		w.ComputeEndpoint = cEndpoint
	}

	if disableGCSLogs {
		w.DisableGCSLogging()
	}
	if diableCloudLogs {
		w.DisableCloudLogging()
	}
	if disableStdoutLogs {
		w.DisableStdoutLogging()
	}

	return w, nil
}

func addFlags(args []string) {
	for _, arg := range args {
		if len(arg) <= 1 || arg[0] != '-' {
			continue
		}

		name := arg[1:]
		if name[0] == '-' {
			name = name[1:]
		}

		if !strings.HasPrefix(name, varFlagPrefix) {
			continue
		}

		name = strings.SplitN(name, "=", 2)[0]

		if flag.Lookup(name) != nil {
			continue
		}

		flag.String(name, "", flgDefValue)
	}
}

func fmtWorkflow(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	var w *daisy.Workflow
	if err := json.Unmarshal(data, &w); err != nil {
		return daisy.JSONError(path, data, err)
	}

	newData, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}

	if err := f.Truncate(0); err != nil {
		return err
	}

	if _, err := f.WriteAt(newData, 0); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return nil
}

func printPerfProfile(workflow *daisy.Workflow) {
	timeRecords := workflow.GetStepTimeRecords()
	if len(timeRecords) == 0 {
		return
	}

	wfStartTime := time.Now()
	wfEndTime := time.Time{}
	fmt.Println("\nPerf Profile:")
	for _, r := range timeRecords {
		if wfStartTime.After(r.StartTime) {
			wfStartTime = r.StartTime
		}
		if wfEndTime.Before(r.EndTime) {
			wfEndTime = r.EndTime
		}
		fmt.Printf("- %v: %v\n", r.Name, formatDuration(r.EndTime.Sub(r.StartTime)))
	}
	fmt.Printf("Total time: %v\n\n", formatDuration(wfEndTime.Sub(wfStartTime)))
}

func formatDuration(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("[hh:mm:ss] %v:%v:%v", s/3600, s/60%60, s%60)
}

func main() {
	addFlags(os.Args[1:])
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("Not enough args, first arg needs to be the path to a workflow.")
	}

	if *format {
		for _, path := range flag.Args() {
			fmt.Printf("[Daisy] Formating workflow file %q\n", path)
			if err := fmtWorkflow(path); err != nil {
				fmt.Print(err)
			}
		}
		return
	}

	ctx := context.Background()

	var ws []*daisy.Workflow
	varMap := populateVars(*variables)

	for _, path := range flag.Args() {
		w, err := parseWorkflow(ctx, path, varMap, *project, *zone, *gcsPath, *oauth, *defaultTimeout, *ce, *gcsLogsDisabled, *cloudLogsDisabled, *stdoutLogsDisabled)
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
		go func(w *daisy.Workflow) {
			select {
			case <-c:
				fmt.Printf("\nCtrl-C caught, sending cancel signal to %q...\n", w.Name)
				w.CancelWorkflow()
				errors <- fmt.Errorf("workflow %q was canceled", w.Name)
			case <-w.Cancel:
			}
		}(w)
		if *print {
			fmt.Printf("[Daisy] Printing workflow %q\n", w.Name)
			w.Print(ctx)
			continue
		}
		if *validate {
			fmt.Printf("[Daisy] Validating workflow %q\n", w.Name)
			if err := w.Validate(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "[Daisy] Error validating workflow %q: %v\n", w.Name, err)
			}
			continue
		}
		wg.Add(1)
		go func(w *daisy.Workflow) {
			defer wg.Done()
			if *printPerf {
				defer printPerfProfile(w)
			}
			fmt.Printf("[Daisy] Running workflow %q (id=%s)\n", w.Name, w.ID())
			if err := w.Run(ctx); err != nil {
				errors <- fmt.Errorf("%s: %v", w.Name, err)
				return
			}
			fmt.Printf("[Daisy] Workflow %q finished\n", w.Name)
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
