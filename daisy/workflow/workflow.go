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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
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

type vars struct {
	Value       string
	Required    bool
	Description string
}

// Workflow is a single Daisy workflow workflow.
type Workflow struct {
	// Populated on New() construction.
	Ctx    context.Context `json:"-"`
	Cancel chan struct{}   `json:"-"`

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
	Vars  map[string]json.RawMessage `json:",omitempty"`
	Steps map[string]*Step
	// Map of steps to their dependencies.
	Dependencies map[string][]string

	// Working fields.
	autovars       map[string]string
	vars           map[string]vars
	workflowDir    string
	parent         *Workflow
	bucket         string
	scratchPath    string
	sourcesPath    string
	logsPath       string
	outsPath       string
	username       string
	gcsLogging     bool
	gcsLogWriter   io.Writer
	ComputeClient  compute.Client  `json:"-"`
	StorageClient  *storage.Client `json:"-"`
	id             string
	logger         *log.Logger
	cleanupHooks   []func() error
	cleanupHooksMx sync.Mutex
}

func (w *Workflow) AddVar(k, v string) {
	if w.vars == nil {
		w.vars = map[string]vars{}
	}
	w.vars[k] = vars{Value: v}
}

func (w *Workflow) addCleanupHook(hook func() error) {
	w.cleanupHooksMx.Lock()
	w.cleanupHooks = append(w.cleanupHooks, hook)
	w.cleanupHooksMx.Unlock()
}

// Validate runs validation on the workflow.
func (w *Workflow) Validate() error {
	w.gcsLogWriter = ioutil.Discard
	if err := w.validateRequiredFields(); err != nil {
		close(w.Cancel)
		return fmt.Errorf("error validating workflow: %v", err)
	}

	if err := w.populate(); err != nil {
		close(w.Cancel)
		return fmt.Errorf("error populating workflow: %v", err)
	}

	w.logger.Print("Validating workflow")
	if err := w.validate(); err != nil {
		w.logger.Printf("Error validating workflow: %v", err)
		close(w.Cancel)
		return err
	}
	w.logger.Print("Validation Complete")
	return nil
}

// Run runs a workflow.
func (w *Workflow) Run() error {
	w.gcsLogging = true
	if err := w.Validate(); err != nil {
		return err
	}
	defer w.cleanup()
	w.logger.Print("Uploading sources")
	if err := w.uploadSources(); err != nil {
		w.logger.Printf("Error uploading sources: %v", err)
		close(w.Cancel)
		return err
	}
	w.logger.Print("Running workflow")
	if err := w.run(); err != nil {
		w.logger.Printf("Error running workflow: %v", err)
		select {
		case <-w.Cancel:
		default:
			close(w.Cancel)
		}
		return err
	}
	return nil
}

func (w *Workflow) String() string {
	f := "{Name:%q Project:%q Zone:%q Bucket:%q OAuthPath:%q Sources:%s Vars:%s Steps:%s Dependencies:%s id:%q}"
	return fmt.Sprintf(f, w.Name, w.Project, w.Zone, w.bucket, w.OAuthPath, w.Sources, w.Vars, w.Steps, w.Dependencies, w.id)
}

func (w *Workflow) cleanup() {
	w.logger.Printf("Workflow %q cleaning up (this may take up to 2 minutes.", w.Name)
	for _, hook := range w.cleanupHooks {
		if err := hook(); err != nil {
			w.logger.Printf("Error returned from cleanup hook: %s", err)
		}
	}
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
	return strings.ToLower(result)
}

func (w *Workflow) getSourceGCSAPIPath(s string) string {
	return fmt.Sprintf("%s/%s", gcsAPIBase, path.Join(w.bucket, w.sourcesPath, s))
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

	if step.WaitForInstancesSignal != nil {
		for i, s := range *step.WaitForInstancesSignal {
			if s.Interval == "" {
				s.Interval = defaultInterval
			}
			interval, err := time.ParseDuration(s.Interval)
			if err != nil {
				return err
			}
			(*step.WaitForInstancesSignal)[i].interval = interval
		}
	}

	// Recurse on subworkflows.
	if step.SubWorkflow != nil {
		return step.SubWorkflow.populate(step)
	}

	if step.IncludeWorkflow != nil {
		return step.IncludeWorkflow.populate(step)
	}
	return nil
}

func (w *Workflow) populateVars() error {
	if w.vars == nil {
		w.vars = map[string]vars{}
	}
	for k, v := range w.Vars {
		// Don't overwrite existing vars (applies to subworkflows).
		if _, ok := w.vars[k]; ok {
			continue
		}
		var sv string
		if err := json.Unmarshal(v, &sv); err == nil {
			w.vars[k] = vars{Value: sv}
			continue
		}
		var vv vars
		if err := json.Unmarshal(v, &vv); err == nil {
			if vv.Required && vv.Value == "" {
				msg := fmt.Sprintf("required vars cannot be blank, var: %q", k)
				if vv.Description != "" {
					msg = fmt.Sprintf("%s, description: %q", msg, vv.Description)
				}
				return errors.New(msg)
			}
			w.vars[k] = vv
			continue
		}
		return fmt.Errorf("cannot unmarshal Var %q, value: %s", k, v)
	}
	return nil
}

func (w *Workflow) populate() error {
	w.id = randString(5)
	now := time.Now().UTC()
	cu, err := user.Current()
	if err != nil {
		w.username = "unknown"
	} else {
		w.username = cu.Username
	}

	if err := w.populateVars(); err != nil {
		return err
	}

	cwd, _ := os.Getwd()

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
	for k, v := range w.vars {
		replacements = append(replacements, fmt.Sprintf("${%s}", k), v.Value)
	}
	substitute(reflect.ValueOf(w).Elem(), strings.NewReplacer(replacements...))

	// Set up GCS paths.
	bkt, p, err := splitGCSPath(w.GCSPath)
	if err != nil {
		return err
	}
	w.bucket = bkt
	w.scratchPath = path.Join(p, fmt.Sprintf("daisy-%s-%s-%s", w.Name, now.Format("20060102-15:04:05"), w.id))
	w.sourcesPath = path.Join(w.scratchPath, "sources")
	w.logsPath = path.Join(w.scratchPath, "logs")
	w.outsPath = path.Join(w.scratchPath, "outs")

	// Do replacement for autovars. Autovars pull from workflow fields,
	// so Vars replacement must run before this to resolve the final
	// value for those fields.
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

	if w.logger == nil {
		name := w.Name
		for parent := w.parent; parent != nil; parent = w.parent.parent {
			name = parent.Name + "." + name
		}
		prefix := fmt.Sprintf("[%s]: ", name)
		flags := log.Ldate | log.Ltime
		if w.gcsLogging {
			w.gcsLogWriter = &gcsLogger{client: w.StorageClient, bucket: w.bucket, object: path.Join(w.logsPath, "daisy.log"), ctx: w.Ctx}
			log.New(os.Stdout, prefix, flags).Println("Logs will be streamed to", "gs://"+path.Join(w.bucket, w.logsPath, "daisy.log"))
		}
		w.logger = log.New(io.MultiWriter(os.Stdout, w.gcsLogWriter), prefix, flags)
	}

	for name, s := range w.Steps {
		s.name = name
		if err := w.populateStep(s); err != nil {
			return err
		}
	}
	return nil
}

// NewIncludedWorkflow instantiates a new workflow with the same resources as the parent.
func (w *Workflow) NewIncludedWorkflow() *Workflow {
	iw := &Workflow{Ctx: w.Ctx, Cancel: w.Cancel, parent: w}
	shareWorkflowResources(w, iw)
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

// NewSubWorkflow instantiates a new workflow with this workflow as the parent.
func (w *Workflow) NewSubWorkflow() *Workflow {
	return &Workflow{Ctx: w.Ctx, Cancel: w.Cancel, parent: w}
}

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
func (w *Workflow) Print() {
	w.gcsLogWriter = ioutil.Discard
	if err := w.populate(); err != nil {
		fmt.Println("Error running populate:", err)
	}

	b, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling workflow for printing:", err)
	}
	fmt.Println(string(b))
}

func (w *Workflow) run() error {
	return w.traverseDAG(func(s *Step) error {
		return w.runStep(s)
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
		e <- s.run()
	}()

	select {
	case err := <-e:
		return err
	case <-timeout:
		return fmt.Errorf("step %q did not stop in specified timeout of %s", s.name, s.timeout)
	}
}

// Concurrently traverse the DAG, running func f on each step.
// Return an error if f returns an error on any step.
func (w *Workflow) traverseDAG(f func(*Step) error) error {
	// waiting = steps and the dependencies they are waiting for.
	// running = the currently running steps.
	// start = map of steps' start channels/semaphores.
	// done = map of steps' done channels for signaling step completion.
	waiting := map[string][]string{}
	var running []string
	start := map[string]chan error{}
	done := map[string]chan error{}

	// Setup: channels, copy dependencies.
	for name := range w.Steps {
		waiting[name] = w.Dependencies[name]
		start[name] = make(chan error)
		done[name] = make(chan error)
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
func New(ctx context.Context) *Workflow {
	// We can't use context.WithCancel as we use the context even after cancel for cleanup.
	w := &Workflow{Ctx: ctx, Cancel: make(chan struct{})}
	initWorkflowResources(w)
	return w
}

// NewFromFile reads and unmarshals a workflow file.
// Recursively reads subworkflow steps as well.
func NewFromFile(ctx context.Context, file string) (*Workflow, error) {
	w := New(ctx)
	if err := readWorkflow(file, w); err != nil {
		return nil, err
	}
	return w, nil
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

	if w.OAuthPath != "" && !filepath.IsAbs(w.OAuthPath) {
		w.OAuthPath = filepath.Join(w.workflowDir, w.OAuthPath)
	}

	for name, s := range w.Steps {
		s.name = name
		s.w = w
		var err error

		if s.SubWorkflow != nil {
			if s.SubWorkflow.w, err = w.NewSubWorkflowFromFile(s.SubWorkflow.Path); err != nil {
				return err
			}
		}

		if s.IncludeWorkflow != nil {
			if s.IncludeWorkflow.w, err = w.NewIncludedWorkflowFromFile(s.IncludeWorkflow.Path); err != nil {
				return err
			}
		}
	}
	return nil
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
		// recvOk -> a step failed, return the error.
		return name, value.Interface().(error)
	}
	return name, nil
}
