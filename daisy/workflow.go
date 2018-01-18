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

// Package daisy describes a daisy workflow.
package daisy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const defaultTimeout = "10m"

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
	if err != nil {
		return 0, err
	}
	if err := wc.Close(); err != nil {
		return 0, err
	}
	return n, err
}

type syncedWriter struct {
	buf *bufio.Writer
	mx  sync.Mutex
}

func (l *syncedWriter) Write(b []byte) (int, error) {
	l.mx.Lock()
	defer l.mx.Unlock()
	return l.buf.Write(b)
}

func (l *syncedWriter) Flush() error {
	l.mx.Lock()
	defer l.mx.Unlock()
	return l.buf.Flush()
}

func daisyBkt(ctx context.Context, client *storage.Client, project string) (string, dErr) {
	dBkt := strings.Replace(project, ":", "-", -1) + "-daisy-bkt"
	it := client.Buckets(ctx, project)
	for bucketAttrs, err := it.Next(); err != iterator.Done; bucketAttrs, err = it.Next() {
		if err != nil {
			return "", typedErr(apiError, err)
		}
		if bucketAttrs.Name == dBkt {
			return dBkt, nil
		}
	}

	if err := client.Bucket(dBkt).Create(ctx, project, nil); err != nil {
		return "", typedErr(apiError, err)
	}
	return dBkt, nil
}

// Var is a type with a flexible JSON representation. A Var can be represented
// by either a string, or by this struct definition. A Var that is represented
// by a string will unmarshal into the struct: {Value: <string>, Required: false, Description: ""}.
type Var struct {
	Value       string
	Required    bool
	Description string
}

// UnmarshalJSON unmarshals a Var.
func (v *Var) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		v.Value = s
		return nil
	}

	// We can't unmarshal into Var directly as it would create an infinite loop.
	type aVar Var
	return json.Unmarshal(b, &struct{ *aVar }{aVar: (*aVar)(v)})
}

// Workflow is a single Daisy workflow workflow.
type Workflow struct {
	// Populated on New() construction.
	Cancel chan struct{} `json:"-"`

	// Workflow template fields.
	// Workflow name.
	Name string
	// Project to run in.
	Project string
	// Zone to run in.
	Zone string
	// GCS Path to use for scratch data and write logs/results to.
	GCSPath string
	// Path to OAuth credentials file.
	OAuthPath string `json:",omitempty"`
	// Sources used by this workflow, map of destination to source.
	Sources map[string]string `json:",omitempty"`
	// Vars defines workflow variables, substitution is done at Workflow run time.
	Vars  map[string]Var `json:",omitempty"`
	Steps map[string]*Step
	// Map of steps to their dependencies.
	Dependencies map[string][]string

	// Working fields.
	autovars     map[string]string
	workflowDir  string
	parent       *Workflow
	bucket       string
	scratchPath  string
	sourcesPath  string
	logsPath     string
	outsPath     string
	username     string
	gcsLogging   bool
	gcsLogWriter *syncedWriter
	// Optional compute endpoint override.
	ComputeEndpoint string
	ComputeClient   compute.Client  `json:"-"`
	StorageClient   *storage.Client `json:"-"`
	id              string
	Logger          *log.Logger `json:"-"`
	cleanupHooks    []func() dErr
	cleanupHooksMx  sync.Mutex

	// Resource registries.
	disks     *diskRegistry
	images    *imageRegistry
	instances *instanceRegistry
	networks  *networkRegistry
}

// AddVar adds a variable set to the Workflow.
func (w *Workflow) AddVar(k, v string) {
	if w.Vars == nil {
		w.Vars = map[string]Var{}
	}
	w.Vars[k] = Var{Value: v}
}

func (w *Workflow) addCleanupHook(hook func() dErr) {
	w.cleanupHooksMx.Lock()
	w.cleanupHooks = append(w.cleanupHooks, hook)
	w.cleanupHooksMx.Unlock()
}

// Validate runs validation on the workflow.
func (w *Workflow) Validate(ctx context.Context) error {
	if err := w.PopulateClients(ctx); err != nil {
		close(w.Cancel)
		return errf("error populating workflow: %v", err)
	}

	if err := w.validateRequiredFields(); err != nil {
		close(w.Cancel)
		return errf("error validating workflow: %v", err)
	}

	if err := w.populate(ctx); err != nil {
		close(w.Cancel)
		return errf("error populating workflow: %v", err)
	}

	w.Logger.Print("Validating workflow")
	if err := w.validate(ctx); err != nil {
		w.Logger.Printf("Error validating workflow: %v", err)
		close(w.Cancel)
		return err
	}
	w.Logger.Print("Validation Complete")
	return nil
}

// Run runs a workflow.
func (w *Workflow) Run(ctx context.Context) error {
	w.gcsLogging = true
	if err := w.Validate(ctx); err != nil {
		return err
	}
	defer w.cleanup()
	w.Logger.Println("Using the GCS path", "gs://"+path.Join(w.bucket, w.scratchPath))

	w.Logger.Print("Uploading sources")
	if err := w.uploadSources(ctx); err != nil {
		w.Logger.Printf("Error uploading sources: %v", err)
		close(w.Cancel)
		return err
	}
	w.Logger.Print("Running workflow")
	if err := w.run(ctx); err != nil {
		w.Logger.Printf("Error running workflow: %v", err)
		select {
		case <-w.Cancel:
		default:
			close(w.Cancel)
		}
		return err
	}
	return nil
}

func (w *Workflow) cleanup() {
	w.Logger.Printf("Workflow %q cleaning up (this may take up to 2 minutes).", w.Name)
	for _, hook := range w.cleanupHooks {
		if err := hook(); err != nil {
			w.Logger.Printf("Error returned from cleanup hook: %s", err)
		}
	}
	if w.gcsLogWriter != nil {
		w.gcsLogWriter.Flush()
	}
}

func (w *Workflow) genName(n string) string {
	name := w.Name
	for parent := w.parent; parent != nil; parent = parent.parent {
		name = parent.Name + "-" + name
	}
	prefix := fmt.Sprintf("%s-%s", n, name)
	if len(prefix) > 57 {
		prefix = prefix[0:56]
	}
	result := fmt.Sprintf("%s-%s", prefix, w.id)
	if len(result) > 64 {
		result = result[0:63]
	}
	return strings.ToLower(result)
}

func (w *Workflow) getSourceGCSAPIPath(s string) string {
	return fmt.Sprintf("%s/%s", gcsAPIBase, path.Join(w.bucket, w.sourcesPath, s))
}

// PopulateClients populates the compute and storage clients for the workflow.
func (w *Workflow) PopulateClients(ctx context.Context) error {
	// API clients instantiation.
	var err error

	computeOptions := []option.ClientOption{option.WithCredentialsFile(w.OAuthPath)}
	if w.ComputeEndpoint != "" {
		computeOptions = append(computeOptions, option.WithEndpoint(w.ComputeEndpoint))
	}

	if w.ComputeClient == nil {
		w.ComputeClient, err = compute.NewClient(ctx, computeOptions...)
		if err != nil {
			return typedErr(apiError, err)
		}
	}

	storageOptions := []option.ClientOption{option.WithCredentialsFile(w.OAuthPath)}
	if w.StorageClient == nil {
		w.StorageClient, err = storage.NewClient(ctx, storageOptions...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Workflow) populateStep(ctx context.Context, s *Step) dErr {
	if s.Timeout == "" {
		s.Timeout = defaultTimeout
	}
	timeout, err := time.ParseDuration(s.Timeout)
	if err != nil {
		return newErr(err)
	}
	s.timeout = timeout

	var derr dErr
	var step stepImpl
	if step, derr = s.stepImpl(); derr != nil {
		return derr
	}
	return step.populate(ctx, s)
}

// populate does the following:
// - checks that all required Vars are set.
// - instantiates API clients, if needed.
// - sets generic autovars and do first round of var substitution.
// - sets GCS path information.
// - generates autovars from workflow fields (Name, Zone, etc) and run second round of var substitution.
// - sets up logger.
// - runs populate on each step.
func (w *Workflow) populate(ctx context.Context) dErr {
	for k, v := range w.Vars {
		if v.Required && v.Value == "" {
			return errf("cannot populate workflow, required var %q is unset", k)
		}
	}

	// Set some generic autovars and run first round of var substitution.
	w.id = randString(5)
	cwd, _ := os.Getwd()
	now := time.Now().UTC()
	w.username = getUser()

	w.autovars = map[string]string{
		"ID":        w.id,
		"DATE":      now.Format("20060102"),
		"DATETIME":  now.Format("20060102150405"),
		"TIMESTAMP": strconv.FormatInt(now.Unix(), 10),
		"USERNAME":  w.username,
		"WFDIR":     w.workflowDir,
		"CWD":       cwd,
	}

	var replacements []string
	for k, v := range w.autovars {
		replacements = append(replacements, fmt.Sprintf("${%s}", k), v)
	}
	for k, v := range w.Vars {
		replacements = append(replacements, fmt.Sprintf("${%s}", k), v.Value)
	}
	substitute(reflect.ValueOf(w).Elem(), strings.NewReplacer(replacements...))

	// Set up GCS paths.
	if w.GCSPath == "" {
		dBkt, err := daisyBkt(ctx, w.StorageClient, w.Project)
		if err != nil {
			return err
		}
		w.GCSPath = "gs://" + dBkt
	}
	bkt, p, err := splitGCSPath(w.GCSPath)
	if err != nil {
		return err
	}
	w.bucket = bkt
	w.scratchPath = path.Join(p, fmt.Sprintf("daisy-%s-%s-%s", w.Name, now.Format("20060102-15:04:05"), w.id))
	w.sourcesPath = path.Join(w.scratchPath, "sources")
	w.logsPath = path.Join(w.scratchPath, "logs")
	w.outsPath = path.Join(w.scratchPath, "outs")

	// Generate more autovars from workflow fields. Run second round of var substitution.
	w.autovars["NAME"] = w.Name
	w.autovars["ZONE"] = w.Zone
	w.autovars["PROJECT"] = w.Project
	w.autovars["GCSPATH"] = w.GCSPath
	w.autovars["SCRATCHPATH"] = fmt.Sprintf("gs://%s/%s", w.bucket, w.scratchPath)
	w.autovars["SOURCESPATH"] = fmt.Sprintf("gs://%s/%s", w.bucket, w.sourcesPath)
	w.autovars["LOGSPATH"] = fmt.Sprintf("gs://%s/%s", w.bucket, w.logsPath)
	w.autovars["OUTSPATH"] = fmt.Sprintf("gs://%s/%s", w.bucket, w.outsPath)

	replacements = []string{}
	for k, v := range w.autovars {
		replacements = append(replacements, fmt.Sprintf("${%s}", k), v)
	}
	substitute(reflect.ValueOf(w).Elem(), strings.NewReplacer(replacements...))

	w.populateLogger(ctx)

	// Run populate on each step.
	for name, s := range w.Steps {
		s.name = name
		s.w = w
		if err := w.populateStep(ctx, s); err != nil {
			return errf("error populating step %q: %v", name, err)
		}
	}
	return nil
}

func (w *Workflow) populateLogger(ctx context.Context) {
	if w.Logger != nil {
		return
	}
	name := w.Name
	for parent := w.parent; parent != nil; parent = parent.parent {
		name = parent.Name + "." + name
	}
	prefix := fmt.Sprintf("[%s]: ", name)
	flags := log.Ldate | log.Ltime
	if w.gcsLogWriter == nil {
		if !w.gcsLogging {
			w.gcsLogWriter = &syncedWriter{buf: bufio.NewWriter(ioutil.Discard)}
		}
		w.gcsLogWriter = &syncedWriter{buf: bufio.NewWriter(&gcsLogger{client: w.StorageClient, bucket: w.bucket, object: path.Join(w.logsPath, "daisy.log"), ctx: ctx})}
		go func() {
			for {
				time.Sleep(5 * time.Second)
				w.gcsLogWriter.Flush()
			}
		}()
	}
	w.Logger = log.New(io.MultiWriter(os.Stdout, w.gcsLogWriter), prefix, flags)
}

// AddDependency creates a dependency of dependent on each dependency. Returns an
// error if dependent or dependency are not steps in this workflow.
func (w *Workflow) AddDependency(dependent *Step, dependencies ...*Step) error {
	if _, ok := w.Steps[dependent.name]; !ok {
		return fmt.Errorf("can't create dependency: step %q does not exist", dependent.name)
	}
	if w.Dependencies == nil {
		w.Dependencies = map[string][]string{}
	}
	for _, dependency := range dependencies {
		if _, ok := w.Steps[dependency.name]; !ok {
			return fmt.Errorf("can't create dependency: step %q does not exist", dependency.name)
		}
		if !strIn(dependency.name, w.Dependencies[dependent.name]) { // Don't add if dependency already exists.
			w.Dependencies[dependent.name] = append(w.Dependencies[dependent.name], dependency.name)
		}
	}
	return nil
}

// NewIncludedWorkflow instantiates a new workflow with the same resources as the parent.
func (w *Workflow) NewIncludedWorkflow() *Workflow {
	iw := New()
	iw.Cancel = w.Cancel
	iw.parent = w
	iw.disks = w.disks
	iw.images = w.images
	iw.instances = w.instances
	iw.networks = w.networks
	return iw
}

// NewIncludedWorkflowFromFile reads and unmarshals a workflow with the same resources as the parent.
func (w *Workflow) NewIncludedWorkflowFromFile(file string) (*Workflow, error) {
	iw := w.NewIncludedWorkflow()
	if !filepath.IsAbs(file) {
		file = filepath.Join(w.workflowDir, file)
	}
	if err := readWorkflow(file, iw); err != nil {
		return nil, err
	}
	return iw, nil
}

// NewStep instantiates a new, typeless step for this workflow.
// The step type must be specified before running this workflow.
func (w *Workflow) NewStep(name string) (*Step, error) {
	if _, ok := w.Steps[name]; ok {
		return nil, fmt.Errorf("can't create step %q: a step already exists with that name", name)
	}
	s := &Step{name: name, w: w}
	if w.Steps == nil {
		w.Steps = map[string]*Step{}
	}
	w.Steps[name] = s
	return s, nil
}

// NewSubWorkflow instantiates a new workflow as a child to this workflow.
func (w *Workflow) NewSubWorkflow() *Workflow {
	sw := New()
	sw.Cancel = w.Cancel
	sw.parent = w
	return sw
}

// NewSubWorkflowFromFile reads and unmarshals a workflow as a child to this workflow.
func (w *Workflow) NewSubWorkflowFromFile(file string) (*Workflow, error) {
	sw := w.NewSubWorkflow()
	if !filepath.IsAbs(file) {
		file = filepath.Join(w.workflowDir, file)
	}
	if err := readWorkflow(file, sw); err != nil {
		return nil, err
	}
	return sw, nil
}

// Print populates then pretty prints the workflow.
func (w *Workflow) Print(ctx context.Context) {
	w.gcsLogging = false
	if err := w.PopulateClients(ctx); err != nil {
		fmt.Println("Error running PopulateClients:", err)
	}
	if err := w.populate(ctx); err != nil {
		fmt.Println("Error running populate:", err)
	}

	b, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling workflow for printing:", err)
	}
	fmt.Println(string(b))
}

func (w *Workflow) run(ctx context.Context) dErr {
	return w.traverseDAG(func(s *Step) dErr {
		return w.runStep(ctx, s)
	})
}

func (w *Workflow) runStep(ctx context.Context, s *Step) dErr {
	timeout := make(chan struct{})
	go func() {
		time.Sleep(s.timeout)
		close(timeout)
	}()

	e := make(chan dErr)
	go func() {
		e <- s.run(ctx)
	}()

	select {
	case err := <-e:
		return err
	case <-timeout:
		return errf("step %q did not stop in specified timeout of %s", s.name, s.timeout)
	}
}

// Concurrently traverse the DAG, running func f on each step.
// Return an error if f returns an error on any step.
func (w *Workflow) traverseDAG(f func(*Step) dErr) dErr {
	// waiting = steps and the dependencies they are waiting for.
	// running = the currently running steps.
	// start = map of steps' start channels/semaphores.
	// done = map of steps' done channels for signaling step completion.
	waiting := map[string][]string{}
	var running []string
	start := map[string]chan dErr{}
	done := map[string]chan dErr{}

	// Setup: channels, copy dependencies.
	for name := range w.Steps {
		waiting[name] = w.Dependencies[name]
		start[name] = make(chan dErr)
		done[name] = make(chan dErr)
	}
	// Setup: goroutine for each step. Each waits to be notified to start.
	for name, s := range w.Steps {
		go func(name string, s *Step) {
			// Wait for signal, then run the function. Return any errs.
			if err := <-start[name]; err != nil {
				done[name] <- err
			} else if err := f(s); err != nil {
				done[name] <- err
			}
			close(done[name])
		}(name, s)
	}

	// Main signaling logic.
	for len(waiting) != 0 || len(running) != 0 {
		// If we got a Cancel signal, kill all waiting steps.
		// Let running steps finish.
		select {
		case <-w.Cancel:
			waiting = map[string][]string{}
		default:
		}

		// Kick off all steps that aren't waiting for anything.
		for name, deps := range waiting {
			if len(deps) == 0 {
				delete(waiting, name)
				running = append(running, name)
				close(start[name])
			}
		}

		// Sanity check. There should be at least one running step,
		// but loop back through if there isn't.
		if len(running) == 0 {
			continue
		}

		// Get next finished step. Return the step error if it erred.
		finished, err := stepsListen(running, done)
		if err != nil {
			return err
		}

		// Remove finished step from other steps' waiting lists.
		for name, deps := range waiting {
			waiting[name] = filter(deps, finished)
		}

		// Remove finished from currently running list.
		running = filter(running, finished)
	}
	return nil
}

// New instantiates a new workflow.
func New() *Workflow {
	// We can't use context.WithCancel as we use the context even after cancel for cleanup.
	w := &Workflow{Cancel: make(chan struct{})}
	// Init nil'ed fields
	w.Sources = map[string]string{}
	w.Vars = map[string]Var{}
	w.Steps = map[string]*Step{}
	w.Dependencies = map[string][]string{}
	w.autovars = map[string]string{}

	// Resource registries and cleanup.
	w.disks = newDiskRegistry(w)
	w.images = newImageRegistry(w)
	w.instances = newInstanceRegistry(w)
	w.networks = newNetworkRegistry(w)
	w.addCleanupHook(func() dErr {
		w.disks.cleanup()
		w.images.cleanup()
		w.instances.cleanup()
		w.networks.cleanup()
		return nil
	})
	return w
}

// NewFromFile reads and unmarshals a workflow file.
// Recursively reads subworkflow steps as well.
func NewFromFile(file string) (*Workflow, error) {
	w := New()
	if err := readWorkflow(file, w); err != nil {
		return nil, err
	}
	return w, nil
}

// JSONError turns an error from json.Unmarshal and returns a more user
// friendly error.
func JSONError(file string, data []byte, err error) error {
	// If this is a syntax error return a useful error.
	sErr, ok := err.(*json.SyntaxError)
	if !ok {
		return err
	}

	// Byte number where the error line starts.
	start := bytes.LastIndex(data[:sErr.Offset], []byte("\n")) + 1
	// Assume end byte of error line is EOF unless this isn't the last line.
	end := len(data)
	if i := bytes.Index(data[start:], []byte("\n")); i >= 0 {
		end = start + i
	}

	// Line number of error.
	line := bytes.Count(data[:start], []byte("\n")) + 1
	// Position of error in line (where to place the '^').
	pos := int(sErr.Offset) - start
	if pos != 0 {
		pos = pos - 1
	}

	return fmt.Errorf("%s: JSON syntax error in line %d: %s \n%s\n%s^", file, line, err, data[start:end], strings.Repeat(" ", pos))
}

func readWorkflow(file string, w *Workflow) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	w.workflowDir, err = filepath.Abs(filepath.Dir(file))
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &w); err != nil {
		return JSONError(file, data, err)
	}

	if w.OAuthPath != "" && !filepath.IsAbs(w.OAuthPath) {
		w.OAuthPath = filepath.Join(w.workflowDir, w.OAuthPath)
	}

	for name, s := range w.Steps {
		s.name = name
		s.w = w
	}

	return nil
}

// stepsListen returns the first step that finishes/errs.
func stepsListen(names []string, chans map[string]chan dErr) (string, dErr) {
	cases := make([]reflect.SelectCase, len(names))
	for i, name := range names {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(chans[name])}
	}
	caseIndex, value, recvOk := reflect.Select(cases)
	name := names[caseIndex]
	if recvOk {
		// recvOk -> a step failed, return the error.
		return name, value.Interface().(dErr)
	}
	return name, nil
}
