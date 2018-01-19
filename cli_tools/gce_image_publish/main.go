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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

const gcsImageObj = "root.tar.gz"

var (
	oauth          = flag.String("oauth", "", "path to oauth json file")
	workProject    = flag.String("work_project", "", "project to perform the work in, passed to Daisy as workflow project, will override WorkProject in template")
	sourceVersion  = flag.String("source_version", "v"+time.Now().UTC().Format("20060102"), "version on source image")
	sourceGCS      = flag.String("source_gcs_path", "", "GCS path to source images from, should not be used with source_project, will override SourceGCSPath in template")
	sourceProject  = flag.String("source_project", "", "project to source images from, should not be used with source_gcs_path, will override SourceProject in template")
	publishVersion = flag.String("publish_version", "", "version for published image if different from source")
	publishProject = flag.String("publish_project", "", "project to publish images to, will override PublishProject in template")
	skipDup        = flag.Bool("skip_duplicates", false, "skip publishing any images that already exist, should not be used along with -replace")
	replace        = flag.Bool("replace", false, "replace any images that already exist, should not be used along with -skip_duplicates")
	rollback       = flag.Bool("rollback", false, "rollback image publish")
	print          = flag.Bool("print", false, "print out the parsed workflow for debugging")
	validate       = flag.Bool("validate", false, "validate the workflow and exit")
	noConfirm      = flag.Bool("skip_confirmation", false, "don't ask for confirmation")
	ce             = flag.String("compute_endpoint_override", "", "API endpoint to override default, will override ComputeEndpoint in template")
	filter         = flag.String("filter", "", "regular expression to filter images to publish by prefixes")
)

type publish struct {
	// Name for this publish workflow, passed to Daisy as workflow name.
	Name string
	// Project to perform the work in, passed to Daisy as workflow project.
	WorkProject string
	// Project to source images from, should not be used with SourceGCSPath.
	SourceProject string
	// GCS path to source images from, should not be used with SourceProject.
	SourceGCSPath string
	// Project to publish images to.
	PublishProject string
	// Optional compute endpoint override
	ComputeEndpoint string
	// Optional period of time to keep images, any images with an create time
	// older than this period will be deleted.
	// Format consists of 2 sections, the first must parsable by
	// https://golang.org/pkg/time/#ParseDuration, the second is a multiplier
	// separated by '*'.
	// 24h = 1 day
	// 24h*7 = 1 week
	// 24h*7*4 = ~1 month
	// 24h*365 = ~1 year
	DeleteAfter string
	expiryDate  *time.Time
	// Images to publish.
	Images []*image

	// Populated from the source_version flag, added to the image prefix to
	// lookup source image.
	sourceVersion string
	// Populated from the publish_version flag, added to the image prefix to
	// create the publish name.
	publishVersion string

	toCreate      []string
	toDelete      []string
	toDeprecate   []string
	toObsolete    []string
	toUndeprecate []string
}

type image struct {
	// Prefix for the image, image naming format is '${ImagePrefix}-${ImageVersion}'.
	// This prefix is used for source image lookup and publish image name.
	Prefix string
	// Image family to set for the image.
	Family string
	// Image description to set for the image.
	Description string
	// Licenses to add to the image.
	Licenses []string
	// GuestOsFeatures to add to the image.
	GuestOsFeatures []string
}

func publishImage(p *publish, img *image, pubImgs []*compute.Image, skipDuplicates, rep bool) (*daisy.CreateImages, *daisy.DeprecateImages, *daisy.DeleteResources, error) {
	if skipDuplicates && rep {
		return nil, nil, nil, errors.New("cannot set both skipDuplicates and replace")
	}
	publishName := fmt.Sprintf("%s-%s", img.Prefix, p.publishVersion)
	sourceName := fmt.Sprintf("%s-%s", img.Prefix, p.sourceVersion)

	// Replace text in Description for the print out, let daisy replace other fields.
	replacer := strings.NewReplacer("${source_version}", p.sourceVersion, "${publish_version}", p.publishVersion)
	ci := daisy.Image{
		Image: compute.Image{
			Name:        publishName,
			Description: replacer.Replace(img.Description),
			Licenses:    img.Licenses,
			Family:      img.Family,
		},
		GuestOsFeatures: img.GuestOsFeatures,
		Resource: daisy.Resource{
			NoCleanup: true,
			Project:   p.PublishProject,
			RealName:  publishName,
		},
	}

	var source string
	if p.SourceProject != "" && p.SourceGCSPath != "" {
		return nil, nil, nil, errors.New("only one of SourceProject or SourceGCSPath should be set")
	}
	if p.SourceProject != "" {
		source = fmt.Sprintf("projects/%s/global/images/%s", p.SourceProject, sourceName)
		ci.Image.SourceImage = source
	} else if p.SourceGCSPath != "" {
		source = fmt.Sprintf("%s/%s/%s", p.SourceGCSPath, sourceName, gcsImageObj)
		ci.Image.RawDisk = &compute.ImageRawDisk{Source: source}
	} else {
		return nil, nil, nil, errors.New("neither SourceProject or SourceGCSPath was set")
	}
	cis := &daisy.CreateImages{&ci}

	dis := &daisy.DeprecateImages{}
	drs := &daisy.DeleteResources{}
	for _, pubImg := range pubImgs {
		if pubImg.Name == publishName {
			msg := fmt.Sprintf("%q already exists in project %q", publishName, p.PublishProject)
			if skipDuplicates {
				fmt.Printf("    Image %s, skipping image creation\n", msg)
				cis = nil
				continue
			} else if rep {
				fmt.Printf("    Image %s, replacing\n", msg)
				(*cis)[0].OverWrite = true
				continue
			}
			return nil, nil, nil, errors.New(msg)
		}

		if pubImg.Family != img.Family {
			continue
		}

		// Delete all images in the same family with insert date older than p.expiryDate.
		if p.expiryDate != nil {
			createTime, err := time.Parse(time.RFC3339, pubImg.CreationTimestamp)
			if err != nil {
				continue
			}
			if createTime.Before(*p.expiryDate) {
				drs.Images = append(drs.Images, fmt.Sprintf("projects/%s/global/images/%s", p.PublishProject, pubImg.Name))
				continue
			}
		}

		// Deprecate all images in the same family.
		if pubImg.Deprecated == nil || pubImg.Deprecated.State == "" {
			*dis = append(*dis, &daisy.DeprecateImage{
				Image:   pubImg.Name,
				Project: p.PublishProject,
				DeprecationStatus: compute.DeprecationStatus{
					State:       "DEPRECATED",
					Replacement: fmt.Sprintf(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/images/%s", p.PublishProject, publishName)),
				},
			})
		}
	}
	if len(*dis) == 0 {
		dis = nil
	}
	if len(drs.Images) == 0 {
		drs = nil
	}

	return cis, dis, drs, nil
}

func rollbackImage(p *publish, img *image, pubImgs []*compute.Image) (*daisy.DeleteResources, *daisy.DeprecateImages) {
	publishName := fmt.Sprintf("%s-%s", img.Prefix, p.publishVersion)
	dr := &daisy.DeleteResources{}
	dis := &daisy.DeprecateImages{}
	for _, pubImg := range pubImgs {
		if pubImg.Name != publishName || pubImg.Deprecated != nil {
			continue
		}
		dr.Images = []string{fmt.Sprintf("projects/%s/global/images/%s", p.PublishProject, publishName)}
	}

	if len(dr.Images) == 0 {
		fmt.Printf("   %q does not exist in %q, not rolling back\n", publishName, p.PublishProject)
		return nil, nil
	}

	for _, pubImg := range pubImgs {
		// Un-deprecate the first deprecated image in the family based on insertion time.
		if pubImg.Family == img.Family && pubImg.Deprecated != nil {
			*dis = append(*dis, &daisy.DeprecateImage{
				Image:   pubImg.Name,
				Project: p.PublishProject,
			})
			break
		}
	}
	return dr, dis
}

func printList(list []string) {
	for _, i := range list {
		fmt.Printf("   - [ %s ]\n", i)
	}
}

func populateSteps(w *daisy.Workflow, prefix string, createImages *daisy.CreateImages, deprecateImages *daisy.DeprecateImages, deleteResources *daisy.DeleteResources) error {
	var createStep *daisy.Step
	var deprecateStep *daisy.Step
	var deleteStep *daisy.Step
	var err error
	if createImages != nil {
		createStep, err = w.NewStep("publish-" + prefix)
		if err != nil {
			return err
		}
		createStep.CreateImages = createImages
		// The default of 10m is a bit low, 1h is excessive for most use cases.
		// TODO(ajackura): Maybe add a timout field override to the template?
		createStep.Timeout = "1h"
	}

	if deprecateImages != nil {
		deprecateStep, err = w.NewStep("deprecate-" + prefix)
		if err != nil {
			return err
		}
		deprecateStep.DeprecateImages = deprecateImages
	}

	if deleteResources != nil {
		deleteStep, err = w.NewStep("delete-" + prefix)
		if err != nil {
			return err
		}
		deleteStep.DeleteResources = deleteResources
	}

	// Create before deprecate on publish.
	if deprecateStep != nil && createStep != nil {
		w.AddDependency(deprecateStep, createStep)
	}

	// Create before delete on publish.
	if deleteStep != nil && createStep != nil {
		w.AddDependency(deleteStep, createStep)
	}

	// Create before delete on publish.
	if deleteStep != nil && createStep != nil {
		w.AddDependency(deleteStep, createStep)
	}

	// Un-deprecate before delete on rollback.
	if deleteStep != nil && deprecateStep != nil {
		w.AddDependency(deleteStep, deprecateStep)
	}

	return nil
}

func (p *publish) createPrintOut(createImages *daisy.CreateImages) {
	if createImages == nil {
		return
	}
	for _, ci := range *createImages {
		p.toCreate = append(p.toCreate, fmt.Sprintf("%s: (%s)", ci.Name, ci.Description))
	}
	return
}

func (p *publish) deletePrintOut(deleteResources *daisy.DeleteResources) {
	if deleteResources == nil {
		return
	}

	for _, img := range deleteResources.Images {
		p.toDelete = append(p.toDelete, path.Base(img))
	}
}

func (p *publish) deprecatePrintOut(deprecateImages *daisy.DeprecateImages) {
	if deprecateImages == nil {
		return
	}

	for _, di := range *deprecateImages {
		image := path.Base(di.Image)
		switch di.DeprecationStatus.State {
		case "DEPRECATED":
			p.toDeprecate = append(p.toDeprecate, image)
		case "OBSOLETE":
			p.toObsolete = append(p.toObsolete, image)
		case "":
			p.toUndeprecate = append(p.toUndeprecate, image)
		}
	}
}

func (p *publish) populateWorkflow(ctx context.Context, w *daisy.Workflow, pubImgs []*compute.Image, img *image, rb, sd, rep bool) error {
	var err error
	var createImages *daisy.CreateImages
	var deprecateImages *daisy.DeprecateImages
	var deleteResources *daisy.DeleteResources
	if rb {
		deleteResources, deprecateImages = rollbackImage(p, img, pubImgs)
	} else {
		createImages, deprecateImages, deleteResources, err = publishImage(p, img, pubImgs, sd, rep)
		if err != nil {
			return err
		}
	}

	if err := populateSteps(w, img.Prefix, createImages, deprecateImages, deleteResources); err != nil {
		return err
	}

	p.createPrintOut(createImages)
	p.deletePrintOut(deleteResources)
	p.deprecatePrintOut(deprecateImages)

	return nil
}

func calculateExpiryDate(deleteAfter string) (*time.Time, error) {
	if deleteAfter == "" {
		return nil, nil
	}
	split := strings.Split(deleteAfter, "*")
	base, err := time.ParseDuration(split[0])
	if err != nil {
		return nil, err
	}
	m := 1
	for i, s := range split {
		if i == 0 {
			continue
		}
		nm, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		m = m * nm
	}
	deleteTime := base * time.Duration(m)
	expiryDate := time.Now().UTC().Add(-deleteTime)

	return &expiryDate, nil
}

var imagesCache map[string][]*compute.Image

func (p *publish) createWorkflow(ctx context.Context, img *image, varMap map[string]string, rb, sd, rep bool) (*daisy.Workflow, error) {
	fmt.Printf("  - Creating publish workflow for %q\n", img.Prefix)
	w := daisy.New()
	for k, v := range varMap {
		w.AddVar(k, v)
	}

	if *oauth != "" {
		w.OAuthPath = *oauth
	}

	if p.ComputeEndpoint != "" {
		w.ComputeEndpoint = p.ComputeEndpoint
	}

	if err := w.PopulateClients(ctx); err != nil {
		return nil, err
	}

	w.Name = img.Prefix
	w.Project = p.WorkProject
	w.AddVar("source_version", p.sourceVersion)
	w.AddVar("publish_version", p.publishVersion)

	cacheKey := w.ComputeClient.BasePath() + p.PublishProject
	pubImgs, ok := imagesCache[cacheKey]
	if !ok {
		var err error
		pubImgs, err = w.ComputeClient.ListImages(p.PublishProject, daisyCompute.OrderBy("creationTimestamp desc"))
		if err != nil {
			return nil, err
		}
		if imagesCache == nil {
			imagesCache = map[string][]*compute.Image{}
		}
		imagesCache[cacheKey] = pubImgs
	}

	if err := p.populateWorkflow(ctx, w, pubImgs, img, rb, sd, rep); err != nil {
		return nil, err
	}
	if len(w.Steps) == 0 {
		return nil, nil
	}
	return w, nil
}

func createWorkflows(ctx context.Context, path string, varMap map[string]string, regex *regexp.Regexp) ([]*daisy.Workflow, error) {
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Option("missingkey=zero").Execute(&buf, varMap); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}

	var p publish
	if err := json.Unmarshal(buf.Bytes(), &p); err != nil {
		return nil, daisy.JSONError(path, buf.Bytes(), err)
	}
	p.expiryDate, err = calculateExpiryDate(p.DeleteAfter)
	if err != nil {
		return nil, fmt.Errorf("%s: error parsing DeleteAfter: %v", path, err)
	}

	p.sourceVersion = *sourceVersion
	p.publishVersion = *publishVersion
	if *publishVersion == "" {
		p.publishVersion = *sourceVersion
	}
	if *workProject != "" {
		p.WorkProject = *workProject
	}
	if *publishProject != "" {
		p.PublishProject = *publishProject
	}
	if *sourceGCS != "" {
		p.SourceGCSPath = *sourceGCS
	}
	if *sourceProject != "" {
		p.SourceProject = *sourceProject
	}
	if *ce != "" {
		p.ComputeEndpoint = *ce
	}
	if p.WorkProject == "" {
		if metadata.OnGCE() {
			p.WorkProject, err = metadata.ProjectID()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("%s: WorkProject unspecified", path)
		}
	}

	fmt.Printf("[%q] Preparing workflows from template\n", p.Name)

	var ws []*daisy.Workflow
	for _, img := range p.Images {
		if regex != nil && !regex.MatchString(img.Prefix) {
			continue
		}
		w, err := p.createWorkflow(ctx, img, varMap, *rollback, *skipDup, *replace)
		if err != nil {
			fmt.Printf("    Workflow creation error: %v\n", err)
			continue
		}
		if w == nil {
			continue
		}
		ws = append(ws, w)
	}
	if len(ws) == 0 {
		fmt.Println("  Nothing to do.")
		return nil, nil
	}

	if len(p.toCreate) > 0 {
		fmt.Printf("  The following images will be created in %q:\n", p.PublishProject)
		printList(p.toCreate)
	}

	if len(p.toDeprecate) > 0 {
		fmt.Printf("  The following images will be deprecated in %q:\n", p.PublishProject)
		printList(p.toDeprecate)
	}

	if len(p.toObsolete) > 0 {
		fmt.Printf("  The following images will be obsoleted in %q:\n", p.PublishProject)
		printList(p.toObsolete)
	}

	if len(p.toUndeprecate) > 0 {
		fmt.Printf("  The following images will be un-deprecated in %q:\n", p.PublishProject)
		printList(p.toUndeprecate)
	}

	if len(p.toDelete) > 0 {
		fmt.Printf("  The following images will be deleted in %q:\n", p.PublishProject)
		printList(p.toDelete)
	}

	return ws, nil
}

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
		if !*print && !*validate {
			fmt.Println("[Publish] Workflows completed successfully.")
		}
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

	var errorSeen bool
	var ws []*daisy.Workflow
	for _, path := range flag.Args() {
		w, err := createWorkflows(ctx, path, varMap, regex)
		if err != nil {
			fmt.Println("   Error:", err)
			errorSeen = true
			continue
		}
		if w != nil {
			ws = append(ws, w...)
		}
	}

	if len(ws) == 0 {
		fmt.Println("[Publish] Nothing to do")
		if errorSeen {
			os.Exit(1)
		}
		return
	}

	if *print {
		for _, w := range ws {
			fmt.Printf("[Publish] Printing workflow %q\n", w.Name)
			w.Print(ctx)
		}
		return
	}

	errors := make(chan error, len(ws))
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
				close(w.Cancel)
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
}
