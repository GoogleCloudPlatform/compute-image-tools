//  Copyright 2019 Google Inc. All Rights Reserved.
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

// Package publish defines the publish object and utilities to create daisy workflows
// from a publish object.
package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

// Publish holds info to create a daisy workflow for gce_image_publish
type Publish struct {
	// Name for this publish workflow, passed to Daisy as workflow name.
	Name string `json:",omitempty"`
	// Project to perform the work in, passed to Daisy as workflow project.
	WorkProject string `json:",omitempty"`
	// Project to source images from, should not be used with SourceGCSPath.
	SourceProject string `json:",omitempty"`
	// GCS path to source images from, should not be used with SourceProject.
	SourceGCSPath string `json:",omitempty"`
	// Project to publish images to.
	PublishProject string `json:",omitempty"`
	// Optional compute endpoint override
	ComputeEndpoint string `json:",omitempty"`
	// Optional period of time to keep images, any images with an create time
	// older than this period will be deleted.
	// Format consists of 2 sections, the first must parsable by
	// https://golang.org/pkg/time/#ParseDuration, the second is a multiplier
	// separated by '*'.
	// 24h = 1 day
	// 24h*7 = 1 week
	// 24h*7*4 = ~1 month
	// 24h*365 = ~1 year
	DeleteAfter string `json:",omitempty"`
	expiryDate  *time.Time
	// Images to
	Images []*Image `json:",omitempty"`

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

// Image is a metadata holder for the image to be published/rollback
type Image struct {
	// Prefix for the image, image naming format is '${ImagePrefix}-${ImageVersion}'.
	// This prefix is used for source image lookup and publish image name.
	Prefix string `json:",omitempty"`
	// Image family to set for the image.
	Family string `json:",omitempty"`
	// Image description to set for the image.
	Description string `json:",omitempty"`
	// Licenses to add to the image.
	Licenses []string `json:",omitempty"`
	// GuestOsFeatures to add to the image.
	GuestOsFeatures []string `json:",omitempty"`
}

var (
	funcMap = template.FuncMap{
		"trim":       strings.Trim,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
	}
	publishTemplate = template.New("publishTemplate").Option("missingkey=zero").Funcs(funcMap)
)

// CreatePublish creates a publish object
func CreatePublish(sourceVersion, publishVersion, workProject, publishProject, sourceGCS, sourceProject, ce, path string, varMap map[string]string) (*Publish, error) {
	p := Publish{
		sourceVersion:  sourceVersion,
		publishVersion: publishVersion,
	}
	if p.publishVersion == "" {
		p.publishVersion = sourceVersion
	}
	varMap["source_version"] = p.sourceVersion
	varMap["publish_version"] = p.publishVersion

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}

	tmpl, err := publishTemplate.Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, varMap); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}

	if err := json.Unmarshal(buf.Bytes(), &p); err != nil {
		return nil, daisy.JSONError(path, buf.Bytes(), err)
	}
	p.expiryDate, err = calculateExpiryDate(p.DeleteAfter)
	if err != nil {
		return nil, fmt.Errorf("%s: error parsing DeleteAfter: %v", path, err)
	}

	if workProject != "" {
		p.WorkProject = workProject
	}
	if publishProject != "" {
		p.PublishProject = publishProject
	}
	if sourceGCS != "" {
		p.SourceGCSPath = sourceGCS
	}
	if sourceProject != "" {
		p.SourceProject = sourceProject
	}
	if ce != "" {
		p.ComputeEndpoint = ce
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

	fmt.Printf("[%q] Created a publish object successfully from %s\n", p.Name, path)
	return &p, nil
}

// CreateWorkflows creates a list of daisy workflows from the publish object
func (p *Publish) CreateWorkflows(ctx context.Context, varMap map[string]string, regex *regexp.Regexp, rollback, skipDup, replace bool, oauth string) ([]*daisy.Workflow, error) {
	fmt.Printf("[%q] Preparing workflows from template\n", p.Name)

	var ws []*daisy.Workflow
	for _, img := range p.Images {
		if regex != nil && !regex.MatchString(img.Prefix) {
			continue
		}
		w, err := p.createWorkflow(ctx, img, varMap, rollback, skipDup, replace, oauth)
		if err != nil {
			return nil, err
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

// ------------------ private methods -------------------------

const gcsImageObj = "root.tar.gz"

func publishImage(p *Publish, img *Image, pubImgs []*compute.Image, skipDuplicates, rep bool) (*daisy.CreateImages, *daisy.DeprecateImages, *daisy.DeleteResources, error) {
	if skipDuplicates && rep {
		return nil, nil, nil, errors.New("cannot set both skipDuplicates and replace")
	}
	publishName := fmt.Sprintf("%s-%s", img.Prefix, p.publishVersion)
	sourceName := fmt.Sprintf("%s-%s", img.Prefix, p.sourceVersion)

	ci := daisy.Image{
		Image: compute.Image{
			Name:        publishName,
			Description: img.Description,
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

func rollbackImage(p *Publish, img *Image, pubImgs []*compute.Image) (*daisy.DeleteResources, *daisy.DeprecateImages) {
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
		// TODO(ajackura): Maybe add a timeout field override to the template?
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

	// Create before deprecate on
	if deprecateStep != nil && createStep != nil {
		w.AddDependency(deprecateStep, createStep)
	}

	// Create before delete on
	if deleteStep != nil && createStep != nil {
		w.AddDependency(deleteStep, createStep)
	}

	// Create before delete on
	if deleteStep != nil && createStep != nil {
		w.AddDependency(deleteStep, createStep)
	}

	// Un-deprecate before delete on rollback.
	if deleteStep != nil && deprecateStep != nil {
		w.AddDependency(deleteStep, deprecateStep)
	}

	return nil
}

func (p *Publish) createPrintOut(createImages *daisy.CreateImages) {
	if createImages == nil {
		return
	}
	for _, ci := range *createImages {
		p.toCreate = append(p.toCreate, fmt.Sprintf("%s: (%s)", ci.Name, ci.Description))
	}
	return
}

func (p *Publish) deletePrintOut(deleteResources *daisy.DeleteResources) {
	if deleteResources == nil {
		return
	}

	for _, img := range deleteResources.Images {
		p.toDelete = append(p.toDelete, path.Base(img))
	}
}

func (p *Publish) deprecatePrintOut(deprecateImages *daisy.DeprecateImages) {
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

func (p *Publish) populateWorkflow(ctx context.Context, w *daisy.Workflow, pubImgs []*compute.Image, img *Image, rb, sd, rep bool) error {
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

var imagesCache map[string][]*compute.Image

func (p *Publish) createWorkflow(ctx context.Context, img *Image, varMap map[string]string, rb, sd, rep bool, oauth string) (*daisy.Workflow, error) {
	fmt.Printf("  - Creating publish workflow for %q\n", img.Prefix)
	w := daisy.New()
	for k, v := range varMap {
		w.AddVar(k, v)
	}

	if oauth != "" {
		w.OAuthPath = oauth
	}

	if p.ComputeEndpoint != "" {
		w.ComputeEndpoint = p.ComputeEndpoint
	}

	if err := w.PopulateClients(ctx); err != nil {
		return nil, err
	}

	w.Name = img.Prefix
	w.Project = p.WorkProject

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

func printList(list []string) {
	for _, i := range list {
		fmt.Printf("   - [ %s ]\n", i)
	}
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
