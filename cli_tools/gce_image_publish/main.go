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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

const gcsImageObj = "root.tar.gz"

var (
	oauth          = flag.String("oauth", "", "path to oauth json file")
	sourceVersion  = flag.String("source_version", "v"+time.Now().UTC().Format("20060102"), "version on source image")
	publishVersion = flag.String("publish_version", *sourceVersion, "version for published image if different from source")
	skipDup        = flag.Bool("skip_duplicates", false, "skip publishing an image that already exists")
	rollback       = flag.Bool("rollback", false, "rollback image publish")
	noConfirm      = flag.Bool("skip_confirmation", false, "don't ask for confirmation")
)

type publish struct {
	// Name for this publish template.
	Name string
	// Project to perform the work in.
	WorkProject string
	// Project to source images from, should not be used with SourceGCSPath.
	SourceProject string
	// GCS path to source images from, should not be used with SourceProject.
	SourceGCSPath string
	// Project to publish images to.
	PublishProject string
	// Optional compute endpoint override
	ComputeEndpoint string
	// Images to publish.
	Images []*image

	// Populated from the source_version flag, added to the image prefix to
	// lookup source image.
	sourceVersion string
	// Populated from the publish_version flag, added to the image prefix to
	// create the publish name.
	publishVersion string
}

type image struct {
	// Prefix for the image, image naming format is '${ImagePrefix}-v${ImageVersion}'.
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

func publishImage(p *publish, img *image, pubImgs []*compute.Image, skipDuplicates bool) (*daisy.CreateImages, *daisy.DeprecateImages, error) {
	publishName := fmt.Sprintf("%s-%s", img.Prefix, p.publishVersion)
	sourceName := fmt.Sprintf("%s-%s", img.Prefix, p.sourceVersion)
	var gosf []*compute.GuestOsFeature
	for _, f := range img.GuestOsFeatures {
		gosf = append(gosf, &compute.GuestOsFeature{Type: f})
	}

	// Replace text in Description for the print out, let daisy replace other fields.
	replacer := strings.NewReplacer("${source_version}", p.sourceVersion, "${publish_version}", p.publishVersion)
	ci := daisy.CreateImage{
		Image: compute.Image{
			Name:            publishName,
			Description:     replacer.Replace(img.Description),
			Licenses:        img.Licenses,
			GuestOsFeatures: gosf,
			Family:          img.Family,
		},
		NoCleanup: true,
		Project:   p.PublishProject,
		RealName:  publishName,
	}

	var source string
	if p.SourceProject != "" && p.SourceGCSPath != "" {
		return nil, nil, errors.New("only one of SourceProject or SourceGCSPath should be set")
	}
	if p.SourceProject != "" {
		source = fmt.Sprintf("projects/%s/global/images/%s", p.SourceProject, sourceName)
		ci.Image.SourceImage = source
	} else if p.SourceGCSPath != "" {
		source = fmt.Sprintf("%s/%s/%s", p.SourceGCSPath, sourceName, gcsImageObj)
		ci.Image.RawDisk = &compute.ImageRawDisk{Source: source}
	} else {
		return nil, nil, errors.New("neither SourceProject or SourceGCSPath was set")
	}
	cis := &daisy.CreateImages{&ci}

	dis := &daisy.DeprecateImages{}
	for _, pubImg := range pubImgs {
		if pubImg.Name == publishName {
			msg := fmt.Sprintf("image %q already exists in project %q", publishName, p.PublishProject)
			if !skipDuplicates {
				return nil, nil, errors.New(msg)
			}
			log.Printf("%s, skipping image creation", msg)
			cis = nil
			continue
		}

		// Deprecate all images in the same family.
		if pubImg.Family == img.Family && (pubImg.Deprecated == nil || pubImg.Deprecated.State == "") {
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

	return cis, dis, nil
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
		log.Printf("%q does not exist in %q, not rolling back", publishName, p.PublishProject)
		return nil, nil
	}

	for _, pubImg := range pubImgs {
		// Un-deprecate the first deprecated image in the family based on insertion time.
		if pubImg.Family == img.Family && pubImg.Deprecated != nil {
			*dis = append(*dis, &daisy.DeprecateImage{
				Image:   pubImg.Name,
				Project: p.PublishProject,
				// Not setting a DeprecationStatus (leaving it blank) un-deprecates the image.
				//DeprecationStatus: compute.DeprecationStatus{
				//ForceSendFields: []string{"State"},
				//	State: "",
				//},
			})
			break
		}
	}
	return dr, dis
}

func printList(list []string) {
	for _, i := range list {
		fmt.Printf(" - [ %s ]\n", i)
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

	// Un-deprecate before delete on rollback.
	if deleteStep != nil && deprecateStep != nil {
		w.AddDependency(deleteStep, deprecateStep)
	}

	return nil
}

func createPrintOut(createImages *daisy.CreateImages) []string {
	if createImages == nil {
		return nil
	}
	var toCreate []string
	for _, ci := range *createImages {
		toCreate = append(toCreate, fmt.Sprintf("%s: (%s)", ci.Name, ci.Description))
	}
	return toCreate
}

func deletePrintOut(deleteResources *daisy.DeleteResources) []string {
	if deleteResources == nil {
		return nil
	}

	var toDelete []string
	for _, img := range deleteResources.Images {
		toDelete = append(toDelete, path.Base(img))
	}
	return toDelete
}

func deprecatePrintOut(deprecateImages *daisy.DeprecateImages) ([]string, []string, []string) {
	if deprecateImages == nil {
		return nil, nil, nil
	}

	var toDeprecate []string
	var toObsolete []string
	var toUndeprecate []string
	for _, di := range *deprecateImages {
		image := path.Base(di.Image)
		switch di.DeprecationStatus.State {
		case "DEPRECATED":
			toDeprecate = append(toDeprecate, image)
		case "OBSOLETE":
			toObsolete = append(toObsolete, image)
		case "":
			toUndeprecate = append(toUndeprecate, image)
		}
	}
	return toDeprecate, toObsolete, toUndeprecate
}

func populateWorkflow(ctx context.Context, w *daisy.Workflow, p *publish, pubImgs []*compute.Image, rb, sd bool) error {
	var err error

	w.Name = p.Name
	w.Project = p.WorkProject
	w.AddVar("source_version", p.sourceVersion)
	w.AddVar("publish_version", p.publishVersion)

	var toCreate []string
	var toDelete []string
	var toDeprecate []string
	var toObsolete []string
	var toUndeprecate []string

	for _, img := range p.Images {
		var createImages *daisy.CreateImages
		var deprecateImages *daisy.DeprecateImages
		var deleteResources *daisy.DeleteResources
		if rb {
			deleteResources, deprecateImages = rollbackImage(p, img, pubImgs)
		} else {
			createImages, deprecateImages, err = publishImage(p, img, pubImgs, sd)
			if err != nil {
				return err
			}
		}

		if err := populateSteps(w, img.Prefix, createImages, deprecateImages, deleteResources); err != nil {
			return err
		}

		toCreate = append(toCreate, createPrintOut(createImages)...)
		toDelete = append(toDelete, deletePrintOut(deleteResources)...)
		td, to, tu := deprecatePrintOut(deprecateImages)
		toDeprecate = append(toDeprecate, td...)
		toObsolete = append(toObsolete, to...)
		toUndeprecate = append(toUndeprecate, tu...)
	}

	if len(toCreate) > 0 {
		fmt.Printf("The following images will be created in %q:\n", p.PublishProject)
		printList(toCreate)
	}

	if len(toDeprecate) > 0 {
		fmt.Printf("\nThe following images will be deprecated in %q:\n", p.PublishProject)
		printList(toDeprecate)
	}

	if len(toObsolete) > 0 {
		fmt.Printf("\nThe following images will be obsoleted in %q:\n", p.PublishProject)
		printList(toObsolete)
	}

	if len(toUndeprecate) > 0 {
		fmt.Printf("\nThe following images will be un-deprecated in %q:\n", p.PublishProject)
		printList(toUndeprecate)
	}

	if len(toDelete) > 0 {
		fmt.Printf("The following images will be deleted in %q:\n", p.PublishProject)
		printList(toDelete)
	}

	return nil
}

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("Not enough args, first arg needs to be the path to a publish template.")
	}
	ctx := context.Background()

	f := flag.Arg(0)
	data, err := ioutil.ReadFile(f)
	if err != nil {
		log.Fatal(err)
	}

	var p publish
	if err := json.Unmarshal(data, &p); err != nil {
		log.Fatal(daisy.JSONError(f, data, err))
	}
	p.sourceVersion = *sourceVersion
	p.publishVersion = *publishVersion

	w := daisy.New()
	if p.ComputeEndpoint != "" {
		w.ComputeClient, err = daisyCompute.NewClient(ctx, option.WithEndpoint(p.ComputeEndpoint))
		if err != nil {
			log.Fatal(err)
		}
	}
	pubImgs, err := w.ComputeClient.ListImages(p.PublishProject, daisyCompute.OrderBy("creationTimestamp desc"))
	if err != nil {
		log.Fatal(err)
	}
	if err := populateWorkflow(ctx, w, &p, pubImgs, *rollback, *skipDup); err != nil {
		log.Fatal(err)
	}

	if len(w.Steps) == 0 {
		fmt.Println("Nothing to do.")
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

	if err := w.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
