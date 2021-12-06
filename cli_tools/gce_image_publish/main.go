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

// gce_image_publish is a tool for publishing GCE images.
package main

import (
	"context"

	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_image_publish/publish"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
)

var (
	oauth          = flag.String("oauth", "", "path to oauth json file")
	workProject    = flag.String("work_project", "", "project to perform the work in, passed to Daisy as workflow project, will override WorkProject in template")
	sourceVersion  = flag.String("source_version", "v"+time.Now().UTC().Format("20060102"), "version on source image")
	sourceGCS      = flag.String("source_gcs_path", "", "GCS path to source images from, should not be used with source_project, will override SourceGCSPath in template")
	sourceProject  = flag.String("source_project", "", "project to source images from, should not be used with source_gcs_path, will override SourceProject in template")
	publishVersion = flag.String("publish_version", "", "version for published image if different from source")
	publishProject = flag.String("publish_project", "", "project to publish images to, will override PublishProject in template")
	skipDup        = flag.Bool("skip_duplicates", false, "skip publishing any images that already exist, should not be used along with -replace")
	noRoot         = flag.Bool("no_root", false, "with -source_gcs_path, append .tar.gz instead of /root.tar.gz")
	replace        = flag.Bool("replace", false, "replace any images that already exist, should not be used along with -skip_duplicates")
	rollback       = flag.Bool("rollback", false, "rollback image publish")
	print          = flag.Bool("print", false, "print out the parsed workflow for debugging")
	validate       = flag.Bool("validate", false, "validate the workflow and exit")
	noConfirm      = flag.Bool("skip_confirmation", false, "don't ask for confirmation")
	ce             = flag.String("compute_endpoint_override", "", "API endpoint to override default, will override ComputeEndpoint in template")
	filter         = flag.String("filter", "", "regular expression to filter images to publish by prefixes")
	rolloutRate    = flag.Int("rollout_rate", 60, "The number of minutes between the image rolling out between zones. 0 minutes will not use a rollout policy.")
)

const (
	flgDefValue   = "flag generated for workflow variable"
	varFlagPrefix = "var:"
)

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

func checkError(errors chan error) {
	select {
	case err := <-errors:
		fmt.Fprintln(os.Stderr, "\n[Publish] Errors in one or more workflows:")
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
		return
	}
}

func main() {
	addFlags(os.Args[1:])
	flag.Parse()

	varMap := map[string]string{}
	flag.Visit(func(flg *flag.Flag) {
		if strings.HasPrefix(flg.Name, varFlagPrefix) {
			varMap[strings.TrimPrefix(flg.Name, varFlagPrefix)] = flg.Value.String()
		}
	})

	if *rolloutRate < 0 {
		fmt.Println("-rollout_rate cannot be less than 0.")
		os.Exit(1)
	}

	if *skipDup && *replace {
		fmt.Println("Cannot set both -skip_duplicates and -replace")
		os.Exit(1)
	}

	if len(flag.Args()) == 0 {
		fmt.Println("Not enough args, first arg needs to be the path to a publish template.")
		os.Exit(1)
	}
	var regex *regexp.Regexp
	if *filter != "" {
		var err error
		regex, err = regexp.Compile(*filter)
		if err != nil {
			fmt.Println("-filter flag not valid:", err)
			os.Exit(1)
		}
	}

	ctx := context.Background()

	var errs []error
	var ws []*daisy.Workflow
	imagesCache := map[string][]*computeAlpha.Image{}
	for _, path := range flag.Args() {
		p, err := publish.CreatePublish(
			*sourceVersion, *publishVersion, *workProject, *publishProject, *sourceGCS, *sourceProject, *ce, path, "", varMap, imagesCache)
		if err != nil {
			loadErr := fmt.Errorf("Loading publish error %s from %q", err, path)
			fmt.Println(loadErr)
			errs = append(errs, loadErr)
			continue
		}
		w, err := p.CreateWorkflows(ctx, varMap, regex, *rollback, *skipDup, *replace, *noRoot, *oauth, time.Now(), *rolloutRate)
		if err != nil {
			createWorkflowErr := fmt.Errorf("Workflow creation error: %s", err)
			fmt.Println(createWorkflowErr)
			errs = append(errs, createWorkflowErr)
			continue
		}
		if w != nil {
			ws = append(ws, w...)
		}
	}

	errors := make(chan error, len(ws)+len(errs))
	for _, err := range errs {
		errors <- err
	}
	if len(ws) == 0 {
		checkError(errors)
		fmt.Println("[Publish] Nothing to do")
		return
	}

	if *print {
		for _, w := range ws {
			fmt.Printf("[Publish] Printing workflow %q\n", w.Name)
			w.Print(ctx)
		}
		checkError(errors)
		return
	}

	if *validate {
		for _, w := range ws {
			fmt.Printf("[Publish] Validating workflow %q\n", w.Name)
			if err := w.Validate(ctx); err != nil {
				errors <- fmt.Errorf("Error validating workflow %s: %v", w.Name, err)
			}
		}
		checkError(errors)
		return
	}

	if !*noConfirm {
		var c string
		fmt.Print("\nContinue with publish? (y/N): ")
		fmt.Scanln(&c)
		c = strings.ToLower(c)
		if c != "y" && c != "yes" {
			return
		}
	}

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

		wg.Add(1)
		go func(w *daisy.Workflow) {
			defer wg.Done()
			fmt.Printf("[Publish] Running workflow %q\n", w.Name)
			if err := w.Run(ctx); err != nil {
				errors <- fmt.Errorf("%s: %v", w.Name, err)
				return
			}
			fmt.Printf("[Publish] Workflow %q finished\n", w.Name)
		}(w)
	}
	wg.Wait()

	checkError(errors)
	fmt.Println("[Publish] Workflows completed successfully.")
}
