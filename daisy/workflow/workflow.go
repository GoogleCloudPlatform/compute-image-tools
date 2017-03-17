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

// Package workflow describes a daisy workflow.
package workflow

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/option"
)

type step interface {
	run(w *Workflow) error
	validate(w *Workflow) error
}

// Step is a single daisy workflow step.
type Step struct {
	name string
	// Time to wait for this step to complete (default 10m).
	// Must be parsable by https://golang.org/pkg/time/#ParseDuration.
	Timeout string
	timeout time.Duration
	// Only one of the below fields should exist for each instance of Step.
	AttachDisks             *AttachDisks             `json:"attach_disks"`
	CreateDisks             *CreateDisks             `json:"create_disks"`
	CreateImages            *CreateImages            `json:"create_images"`
	CreateInstances         *CreateInstances         `json:"create_instances"`
	DeleteResources         *DeleteResources         `json:"delete_resources"`
	ExportImages            *ExportImages            `json:"export_images"`
	ImportDisks             *ImportDisks             `json:"import_disks"`
	RunTests                *RunTests                `json:"run_tests"`
	SubWorkflow             *SubWorkflow             `json:"sub_workflow"`
	WaitForInstancesSignal  *WaitForInstancesSignal  `json:"wait_for_instances_signal"`
	WaitForInstancesStopped *WaitForInstancesStopped `json:"wait_for_instances_stopped"`
	testType                step
}

func (s *Step) realStep() (step, error) {
	var result step
	matchCount := 0
	if s.AttachDisks != nil {
		matchCount++
		result = s.AttachDisks
	}
	if s.CreateDisks != nil {
		matchCount++
		result = s.CreateDisks
	}
	if s.CreateImages != nil {
		matchCount++
		result = s.CreateImages
	}
	if s.CreateInstances != nil {
		matchCount++
		result = s.CreateInstances
	}
	if s.DeleteResources != nil {
		matchCount++
		result = s.DeleteResources
	}
	if s.ExportImages != nil {
		matchCount++
		result = s.ExportImages
	}
	if s.ImportDisks != nil {
		matchCount++
		result = s.ImportDisks
	}
	if s.RunTests != nil {
		matchCount++
		result = s.RunTests
	}
	if s.SubWorkflow != nil {
		matchCount++
		result = s.SubWorkflow
	}
	if s.WaitForInstancesSignal != nil {
		matchCount++
		result = s.WaitForInstancesSignal
	}
	if s.WaitForInstancesStopped != nil {
		matchCount++
		result = s.WaitForInstancesStopped
	}
	if s.testType != nil {
		matchCount++
		result = s.testType
	}

	if matchCount == 0 {
		return nil, errors.New("no step type defined")
	}
	if matchCount > 1 {
		return nil, errors.New("multiple step types defined")
	}
	return result, nil
}

func (s *Step) run(w *Workflow) error {
	realStep, err := s.realStep()
	if err != nil {
		return s.wrapRunError(err)
	}
	if err = realStep.run(w); err != nil {
		return s.wrapRunError(err)
	}
	return nil
}

func (s *Step) validate(w *Workflow) error {
	realStep, err := s.realStep()
	if err != nil {
		return s.wrapValidateError(err)
	}
	if err = realStep.validate(w); err != nil {
		return s.wrapValidateError(err)
	}
	return nil
}

func (s *Step) wrapRunError(e error) error {
	return fmt.Errorf("%q run error: %s", s.name, e)
}

func (s *Step) wrapValidateError(e error) error {
	return fmt.Errorf("%q validation error: %s", s.name, e)
}

// Workflow is a single Daisy workflow workflow.
type Workflow struct {
	// Populated on New() construction.
	id     string
	Ctx    context.Context    `json:"-"`
	Cancel context.CancelFunc `json:"-"`

	// Workflow template fields.
	// Workflow name.
	Name         string
	// Project to run in.
	Project      string
	// Zone to run in.
	Zone         string
	// GCS Bucket to use for scratch data and write logs/results to.
	Bucket       string
	// Path to OAuth credentials file.
	OAuthPath    string `json:"oauth_path"`
	// Sources used by this workflow, map of destination to source.
	Sources      map[string]string
	// Vars defines workflow variables, substitution is done at Workflow run time.
	Vars         map[string]string
	Steps        map[string]*Step
	// Map of steps to their dependencies.
	Dependencies map[string][]string

	// Working fields.
	createdDisks       map[string]string
	createdDisksMx     sync.Mutex
	createdInstances   []string
	createdInstancesMx sync.Mutex
	createdImages      map[string]string
	createdImagesMx    sync.Mutex
	scratchPath        string
	sourcesPath        string
	logsPath           string
	outsPath           string
	ComputeClient      *compute.Client    `json:"-"`
	StorageClient      *storage.Client    `json:"-"`
}

// FromFile reads and unmarshals a workflow file.
// Recursively reads subworkflow steps as well.
func (w *Workflow) FromFile(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}

	// We need to unmarshal any SubWorkflows.
	for name, step := range w.Steps {
		step.name = name

		if step.SubWorkflow == nil {
			continue
		}

		sw := New(w.Ctx)
		if err := sw.FromFile(step.SubWorkflow.Path); err != nil {
			return err
		}

		step.SubWorkflow.Workflow = sw
	}

	return nil
}

// Run runs a workflow.
func (w *Workflow) Run() error {
	w.prerun()
	return w.run()
}

func (w *Workflow) String() string {
	f := "{Name:%q Project:%q Zone:%q Bucket:%q OAuthPath:%q Sources:%s Vars:%s Steps:%s Dependencies:%s id:%q}"
	return fmt.Sprintf(f, w.Name, w.Project, w.Zone, w.Bucket, w.OAuthPath, w.Sources, w.Vars, w.Steps, w.Dependencies, w.id)
}

func (w *Workflow) addCreatedDisk(name, link string) {
	w.createdDisksMx.Lock()
	defer w.createdDisksMx.Unlock()
	if w.createdDisks == nil {
		w.createdDisks = map[string]string{}
	}
	w.createdDisks[name] = link
}

func (w *Workflow) addCreatedImage(name, link string) {
	w.createdImagesMx.Lock()
	defer w.createdImagesMx.Unlock()
	if w.createdImages == nil {
		w.createdImages = map[string]string{}
	}
	w.createdImages[name] = link
}

func (w *Workflow) addCreatedInstance(name string) {
	w.createdInstancesMx.Lock()
	defer w.createdInstancesMx.Unlock()
	w.createdInstances = append(w.createdInstances, name)
}

func (w *Workflow) cleanup() {
	var wg sync.WaitGroup
	for _, i := range w.createdInstances {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			if err := w.deleteInstance(i); err != nil {
				fmt.Println(err)
			}
		}(i)
	}
	wg.Wait()

	for d := range w.createdDisks {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			if err := w.deleteDisk(d); err != nil {
				fmt.Println(err)
			}
		}(d)
	}
	wg.Wait()
}

func (w *Workflow) delCreatedDisk(name string) {
	w.createdDisksMx.Lock()
	defer w.createdDisksMx.Unlock()
	delete(w.createdDisks, name)
}

func (w *Workflow) delCreatedImage(name string) {
	w.createdImagesMx.Lock()
	defer w.createdImagesMx.Unlock()
	delete(w.createdImages, name)
}

func (w *Workflow) delCreatedInstance(name string) {
	w.createdInstancesMx.Lock()
	defer w.createdInstancesMx.Unlock()
	for i, n := range w.createdInstances {
		if n == name {
			w.createdInstances = append(w.createdInstances[:i], w.createdInstances[i+1:]...)
		}
	}
}

func (w *Workflow) deleteDisk(name string) error {
	if err := w.ComputeClient.DeleteDisk(w.Project, w.Zone, name); err != nil {
		return err
	}
	w.delCreatedDisk(name)
	return nil
}

func (w *Workflow) deleteImage(name string) error {
	if err := w.ComputeClient.DeleteImage(w.Project, name); err != nil {
		return err
	}
	w.delCreatedImage(name)
	return nil
}

func (w *Workflow) deleteInstance(name string) error {
	if err := w.ComputeClient.DeleteInstance(w.Project, w.Zone, name); err != nil {
		return err
	}
	w.delCreatedInstance(name)
	return nil
}

func (w *Workflow) getCreatedDisk(name string) string {
	w.createdDisksMx.Lock()
	defer w.createdDisksMx.Unlock()
	if v, ok := w.createdDisks[name]; ok {
		return v
	}
	return ""
}

func (w *Workflow) getCreatedImage(name string) string {
	w.createdImagesMx.Lock()
	defer w.createdImagesMx.Unlock()
	if v, ok := w.createdImages[name]; ok {
		return v
	}
	return ""
}

func (w *Workflow) hasCreatedInstance(name string) bool {
	w.createdInstancesMx.Lock()
	defer w.createdInstancesMx.Unlock()
	return containsString(name, w.createdInstances)
}

func (w *Workflow) populateStep(step *Step) error {
	if step.Timeout == "" {
		step.Timeout = defaultTimeout
	}
	timeout, err := time.ParseDuration(step.Timeout)
	if err != nil {
		return err
	}
	step.timeout = timeout

	return nil
}

func (w *Workflow) populate() error {
	var oldnew []string
	for k, v := range w.Vars {
		oldnew = append(oldnew, fmt.Sprintf("${%s}", k), v)
	}
	substitute(reflect.ValueOf(w).Elem(), strings.NewReplacer(oldnew...))

	var err error
	if w.ComputeClient == nil {
		w.ComputeClient, err = compute.NewClient(w.Ctx, option.WithServiceAccountFile(w.OAuthPath))
		if err != nil {
			return err
		}
	}

	if w.StorageClient == nil {
		w.StorageClient, err = storage.NewClient(w.Ctx, option.WithServiceAccountFile(w.OAuthPath))
		if err != nil {
			return err
		}
	}

	w.scratchPath = fmt.Sprintf("gs://%s/daisy-%s-%s", w.Bucket, w.Name, w.id)
	w.sourcesPath = fmt.Sprintf("%s/sources", w.scratchPath)
	w.logsPath = fmt.Sprintf("%s/logs", w.scratchPath)
	w.outsPath = fmt.Sprintf("%s/outs", w.scratchPath)

	for name, step := range w.Steps {
		step.name = name
		if err := w.populateStep(step); err != nil {
			return err
		}
	}
	return nil
}

func (w *Workflow) prerun() error {
	if err := w.populate(); err != nil {
		return err
	}
	if err := w.validate(); err != nil {
		return err
	}
	if err := w.uploadSources(); err != nil {
		return err
	}

	// Modify subworkflows' GCE and GCS locations to match this workflow, then run their own preruns.
	for _, step := range w.Steps {
		if step.SubWorkflow == nil {
			continue
		}
		step.SubWorkflow.Workflow.Bucket = w.Bucket
		step.SubWorkflow.Workflow.Project = w.Project
		step.SubWorkflow.Workflow.Zone = w.Zone
		step.SubWorkflow.Workflow.OAuthPath = w.OAuthPath
		step.SubWorkflow.Workflow.ComputeClient = w.ComputeClient
		step.SubWorkflow.Workflow.StorageClient = w.StorageClient
		if err := step.SubWorkflow.Workflow.prerun(); err != nil {
			return err
		}
	}
	return nil
}

func (w *Workflow) run() error {
	return w.traverseDAG(func(s step) error { return w.runStep(s.(*Step)) })
}

func (w *Workflow) runStep(s *Step) error {
	timeout := make(chan struct{})
	go func() {
		time.Sleep(s.timeout)
		close(timeout)
	}()

	e := make(chan error)
	go func() {
		e <- s.run(w)
	}()

	select {
	case err := <-e:
		return err
	case <-timeout:
		return fmt.Errorf("step %q did not stop in specified timeout of %s", s.name, s.timeout)
	}
}

func (w *Workflow) stepDepends(consumer, consumed string) bool {
	q := w.Dependencies[consumer]
	seen := map[string]bool{}

	for i := 0; i < len(q); i++ {
		name := q[i]
		if seen[name] {
			continue
		}
		seen[name] = true
		if name == consumed {
			return true
		}
		for _, dep := range w.Dependencies[name] {
			q = append(q, dep)
		}
	}

	return false
}

// Concurrently traverse the DAG, running func f on each step.
func (w *Workflow) traverseDAG(f func(step) error) error {
	notify := map[string]chan error{}
	done := map[string]chan error{}

	// Setup: channels, copy dependencies.
	waiting := map[string][]string{}
	for name := range w.Steps {
		waiting[name] = w.Dependencies[name]
		notify[name] = make(chan error)
		done[name] = make(chan error)
	}
	// Setup: goroutine for each step. Each waits to be notified to start.
	for name, s := range w.Steps {
		go func(name string, s *Step) {
			// Wait for signal, then run the function. Return any errs.
			if err := <-notify[name]; err != nil {
				done[name] <- err
			} else if err := f(s); err != nil {
				done[name] <- err
			}
			close(done[name])
		}(name, s)
	}

	// Main signaling logic.
	var running []string
	for len(waiting) != 0 || len(running) != 0 {
		select {
		case <-w.Ctx.Done():
			waiting = map[string][]string{}
		default:
		}
		// Kick off all steps that aren't waiting for anything.
		for name, deps := range waiting {
			if len(deps) == 0 {
				delete(waiting, name)
				running = append(running, name)
				close(notify[name])
			}
		}

		if len(running) == 0 {
			continue
		}
		// Get next finished step.
		finished, err := stepsListen(running, done)
		if err != nil {
			return err
		}
		for name, deps := range waiting {
			waiting[name] = filter(deps, finished)
		}
		running = filter(running, finished)
	}
	return nil
}

func (w *Workflow) uploadSources() error {
	dstB := w.StorageClient.Bucket(w.Bucket)
	for path, origPath := range w.Sources {
		dstOPath := strings.TrimLeft(fmt.Sprintf("%s/%s", w.sourcesPath, path), "/")
		dstO := dstB.Object(dstOPath)
		if b, o, err := splitGCSPath(origPath); err == nil {
			// GCS to GCS.
			srcB := w.StorageClient.Bucket(b)
			srcO := srcB.Object(o)
			if _, err := dstO.CopierFrom(srcO).Run(w.Ctx); err != nil {
				return err
			}
		} else {
			// Local to GCS.
			writer := dstO.NewWriter(w.Ctx)
			bs, err := ioutil.ReadFile(origPath)
			if err != nil {
				return err
			}
			buf := bufio.NewWriterSize(writer, len(bs))
			if _, err = buf.Write(bs); err != nil {
				return err
			}
			if err = buf.Flush(); err != nil {
				return err
			}
		}
	}
	return nil
}

// New instantiates a new workflow.
func New(ctx context.Context) *Workflow {
	var w Workflow
	w.Ctx, w.Cancel = context.WithCancel(ctx)
	w.id = randString(5)
	return &w
}

// stepsListen returns the first step that finishes/errs.
func stepsListen(names []string, chans map[string]chan error) (string, error) {
	cases := make([]reflect.SelectCase, len(names))
	for i, name := range names {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(chans[name])}
	}
	caseIndex, value, recvOk := reflect.Select(cases)
	name := names[caseIndex]
	if recvOk {
		return name, value.Interface().(error)
	}
	return name, nil
}
