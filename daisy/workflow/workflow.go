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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"bytes"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/option"
	"io"
	"log"
	"os"
)

type gcsLogger struct {
	client         *storage.Client
	bucket, object string
	buf            *bytes.Buffer
	ctx            context.Context
}

func (l *gcsLogger) Write(b []byte) (int, error) {
	if l.buf == nil {
		l.buf = new(bytes.Buffer)
	}
	l.buf.Write(b)
	wc := l.client.Bucket(l.bucket).Object(l.object).NewWriter(l.ctx)
	wc.ContentType = "text/plain"
	n, err := wc.Write(l.buf.Bytes())
	if err := wc.Close(); err != nil {
		return 0, err
	}
	return n, err
}

type refMap struct {
	m  map[string]*resource
	mx sync.Mutex
}

func (rm *refMap) add(name string, r *resource) {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	if rm.m == nil {
		rm.m = map[string]*resource{}
	}
	rm.m[name] = r
}

func (rm *refMap) del(name string) {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	if rm.m != nil {
		delete(rm.m, name)
	}
}

func (rm *refMap) get(name string) (*resource, bool) {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	if rm.m == nil {
		return nil, false
	}
	r, ok := rm.m[name]
	return r, ok
}

type resource struct {
	name, real, link string
	noCleanup        bool
}

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
	// Used for unit tests.
	testType step
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
	var st string
	if t := reflect.TypeOf(realStep); t.Kind() == reflect.Ptr {
		st = t.Elem().Name()
	} else {
		st = t.Name()
	}
	w.logger.Printf("Running step %q (%s)", s.name, st)
	if err = realStep.run(w); err != nil {
		return s.wrapRunError(err)
	}
	return nil
}

func (s *Step) validate(w *Workflow) error {
	w.logger.Printf("Validating step %q", s.name)
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
	return fmt.Errorf("step %q run error: %s", s.name, e)
}

func (s *Step) wrapValidateError(e error) error {
	return fmt.Errorf("step %q validation error: %s", s.name, e)
}

// Workflow is a single Daisy workflow workflow.
type Workflow struct {
	// Populated on New() construction.
	Ctx    context.Context    `json:"-"`
	Cancel context.CancelFunc `json:"-"`

	// Workflow template fields.
	// Workflow name.
	Name string
	// Project to run in.
	Project string
	// Zone to run in.
	Zone string
	// GCS Path to use for scratch data and write logs/results to.
	GCSPath string `json:"gcs_path"`
	// Path to OAuth credentials file.
	OAuthPath string `json:"oauth_path"`
	// Sources used by this workflow, map of destination to source.
	Sources map[string]string
	// Vars defines workflow variables, substitution is done at Workflow run time.
	Vars  map[string]string
	Steps map[string]*Step
	// Map of steps to their dependencies.
	Dependencies map[string][]string

	// Working fields.
	diskRefs      *refMap
	instanceRefs  *refMap
	imageRefs     *refMap
	parent        *Workflow
	bucket        string
	scratchPath   string
	sourcesPath   string
	logsPath      string
	outsPath      string
	ComputeClient *compute.Client `json:"-"`
	StorageClient *storage.Client `json:"-"`
	id            string
	logger        *log.Logger
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
		step.SubWorkflow.workflow = sw
		sw.parent = w
	}

	return nil
}

// Run runs a workflow.
func (w *Workflow) Run() error {
	if err := w.populate(); err != nil {
		w.Cancel()
		return err
	}

	w.logger.Print("Validating workflow")
	if err := w.validate(); err != nil {
		w.logger.Printf("Error validating workflow: %v", err)
		w.Cancel()
		return err
	}
	w.logger.Print("Uploading sources")
	if err := w.uploadSources(); err != nil {
		w.logger.Printf("Error uploading sources: %v", err)
		w.Cancel()
		return err
	}
	w.logger.Print("Running workflow")
	if err := w.run(); err != nil {
		w.logger.Printf("Error running workflow: %v", err)
		w.Cancel()
		return err
	}
	return nil
}

func (w *Workflow) String() string {
	f := "{Name:%q Project:%q Zone:%q Bucket:%q OAuthPath:%q Sources:%s Vars:%s Steps:%s Dependencies:%s id:%q}"
	return fmt.Sprintf(f, w.Name, w.Project, w.Zone, w.bucket, w.OAuthPath, w.Sources, w.Vars, w.Steps, w.Dependencies, w.id)
}

func (w *Workflow) cleanup() {
	w.logger.Print("Cleaning ephemeral resources")
	w.cleanupHelper(w.imageRefs, w.deleteImage)
	w.cleanupHelper(w.instanceRefs, w.deleteInstance)
	w.cleanupHelper(w.diskRefs, w.deleteDisk)
}

func (w *Workflow) cleanupHelper(rm *refMap, deleteFn func(*resource) error) {
	var wg sync.WaitGroup
	toDel := map[string]*resource{}
	for ref, res := range rm.m {
		// Delete only non-persistent resources.
		if !res.noCleanup {
			toDel[ref] = res
		}
	}
	for ref, res := range toDel {
		wg.Add(1)
		go func(ref string, r *resource) {
			defer wg.Done()
			if err := deleteFn(r); err != nil {
				fmt.Println(err)
			}
		}(ref, res)
	}
	wg.Wait()
}

func (w *Workflow) deleteDisk(r *resource) error {
	if err := w.ComputeClient.DeleteDisk(w.Project, w.Zone, r.real); err != nil {
		return err
	}
	w.diskRefs.del(r.name)
	return nil
}

func (w *Workflow) deleteImage(r *resource) error {
	if err := w.ComputeClient.DeleteImage(w.Project, r.real); err != nil {
		return err
	}
	w.imageRefs.del(r.name)
	return nil
}

func (w *Workflow) deleteInstance(r *resource) error {
	if err := w.ComputeClient.DeleteInstance(w.Project, w.Zone, r.real); err != nil {
		return err
	}
	w.instanceRefs.del(r.name)
	return nil
}

func (w *Workflow) genName(n string) string {
	prefix := fmt.Sprintf("%s-%s", n, w.Name)
	if len(prefix) > 57 {
		prefix = prefix[0:56]
	}
	result := fmt.Sprintf("%s-%s", prefix, w.id)
	if len(result) > 64 {
		result = result[0:63]
	}
	return result
}

func (w *Workflow) nameToDiskLink(n string) string {
	return fmt.Sprintf("projects/%s/zones/%s/disks/%s", w.Project, w.Zone, n)
}

func (w *Workflow) nameToImageLink(n string) string {
	return fmt.Sprintf("projects/%s/global/images/%s", w.Project, n)
}

func (w *Workflow) nameToInstanceLink(n string) string {
	return fmt.Sprintf("projects/%s/zones/%s/instances/%s", w.Project, w.Zone, n)
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

	// Recurse on subworkflows.
	if step.SubWorkflow == nil {
		return nil
	}
	step.SubWorkflow.workflow.GCSPath = w.GCSPath
	step.SubWorkflow.workflow.Project = w.Project
	step.SubWorkflow.workflow.Zone = w.Zone
	step.SubWorkflow.workflow.OAuthPath = w.OAuthPath
	step.SubWorkflow.workflow.ComputeClient = w.ComputeClient
	step.SubWorkflow.workflow.StorageClient = w.StorageClient
	return step.SubWorkflow.workflow.populate()
}

func (w *Workflow) populate() error {
	w.id = randString(5)

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

	w.diskRefs = &refMap{m: map[string]*resource{}}
	w.imageRefs = &refMap{m: map[string]*resource{}}
	w.instanceRefs = &refMap{m: map[string]*resource{}}

	bkt, p, err := splitGCSPath(w.GCSPath)
	if err != nil {
		return err
	}
	w.bucket = bkt
	w.scratchPath = path.Join(p, fmt.Sprintf("daisy-%s-%s", w.Name, w.id))
	w.sourcesPath = path.Join(w.scratchPath, "sources")
	w.logsPath = path.Join(w.scratchPath, "logs")
	w.outsPath = path.Join(w.scratchPath, "outs")

	if w.logger == nil {
		gcs := &gcsLogger{client: w.StorageClient, bucket: w.bucket, object: path.Join(w.logsPath, "daisy.log"), ctx: w.Ctx}
		name := w.Name
		for parent := w.parent; parent != nil; parent = w.parent.parent {
			name = parent.Name + "." + name
		}
		prefix := fmt.Sprintf("[%s]: ", name)
		flags := log.Ldate | log.Ltime
		log.New(os.Stdout, prefix, flags).Println("Logs will be streamed to", "gs://"+path.Join(w.bucket, w.logsPath, "daisy.log"))
		w.logger = log.New(io.MultiWriter(os.Stdout, gcs), prefix, flags)
	}

	for name, s := range w.Steps {
		s.name = name
		if err := w.populateStep(s); err != nil {
			return err
		}
	}
	return nil
}

func (w *Workflow) run() error {
	defer w.cleanup()
	return w.traverseDAG(func(s step) error {
		return w.runStep(s.(*Step))
	})
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
	for p, origPath := range w.Sources {
		dst := w.StorageClient.Bucket(w.bucket).Object(path.Join(w.sourcesPath, p))
		if b, o, err := splitGCSPath(origPath); err == nil {
			// GCS to GCS.
			src := w.StorageClient.Bucket(b).Object(o)
			if _, err := dst.CopierFrom(src).Run(w.Ctx); err != nil {
				return err
			}
		} else {
			// Local to GCS.
			gcs := dst.NewWriter(w.Ctx)
			f, err := os.Open(origPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(gcs, f); err != nil {
				return err
			}
			if err := gcs.Close(); err != nil {
				return err
			}
		}
	}
	for _, step := range w.Steps {
		if step.SubWorkflow != nil {
			step.SubWorkflow.workflow.uploadSources()
		}
	}
	return nil
}

// New instantiates a new workflow.
func New(ctx context.Context) *Workflow {
	var w Workflow
	w.Ctx, w.Cancel = context.WithCancel(ctx)
	return &w
}

func resolveLink(name string, rm *refMap) string {
	if isLink(name) {
		return name
	} else if r, ok := rm.get(name); ok {
		return r.link
	}
	return ""
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
