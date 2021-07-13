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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const defaultTimeout = "10m"

func daisyBkt(ctx context.Context, client *storage.Client, project string) (string, DError) {
	dBkt := strings.Replace(project, ":", "-", -1) + "-daisy-bkt"
	it := client.Buckets(ctx, project)
	for bucketAttrs, err := it.Next(); err != iterator.Done; bucketAttrs, err = it.Next() {
		if err != nil {
			return "", typedErr(apiError, "failed to iterate buckets", err)
		}
		if bucketAttrs.Name == dBkt {
			return dBkt, nil
		}
	}

	if err := client.Bucket(dBkt).Create(ctx, project, nil); err != nil {
		return "", typedErr(apiError, "failed to create bucket", err)
	}
	return dBkt, nil
}

// TimeRecord is a type with info of a step execution time
type TimeRecord struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
}

// Var is a type with a flexible JSON representation. A Var can be represented
// by either a string, or by this struct definition. A Var that is represented
// by a string will unmarshal into the struct: {Value: <string>, Required: false, Description: ""}.
type Var struct {
	Value       string
	Required    bool   `json:",omitempty"`
	Description string `json:",omitempty"`
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
	Cancel     chan struct{} `json:"-"`
	isCanceled bool
	cancelMx   sync.Mutex

	// Workflow template fields.
	// Workflow name.
	Name string `json:",omitempty"`
	// Project to run in.
	Project string `json:",omitempty"`
	// Zone to run in.
	Zone string `json:",omitempty"`
	// GCS Path to use for scratch data and write logs/results to.
	GCSPath string `json:",omitempty"`
	// Path to OAuth credentials file.
	OAuthPath string `json:",omitempty"`
	// Sources used by this workflow, map of destination to source.
	Sources map[string]string `json:",omitempty"`
	// Vars defines workflow variables, substitution is done at Workflow run time.
	Vars  map[string]Var   `json:",omitempty"`
	Steps map[string]*Step `json:",omitempty"`
	// Map of steps to their dependencies.
	Dependencies map[string][]string `json:",omitempty"`
	// Default timout for each step, defaults to 10m.
	// Must be parsable by https://golang.org/pkg/time/#ParseDuration.
	DefaultTimeout string `json:",omitempty"`
	defaultTimeout time.Duration

	// Working fields.
	autovars              map[string]string
	workflowDir           string
	parent                *Workflow
	bucket                string
	scratchPath           string
	sourcesPath           string
	logsPath              string
	outsPath              string
	username              string
	externalLogging       bool
	gcsLoggingDisabled    bool
	cloudLoggingDisabled  bool
	stdoutLoggingDisabled bool
	id                    string
	Logger                Logger `json:"-"`
	cleanupHooks          []func() DError
	cleanupHooksMx        sync.Mutex
	recordTimeMx          sync.Mutex
	stepWait              sync.WaitGroup
	logProcessHook        func(string) string

	// Optional compute endpoint override.stepWait
	ComputeEndpoint    string          `json:",omitempty"`
	ComputeClient      compute.Client  `json:"-"`
	StorageClient      *storage.Client `json:"-"`
	cloudLoggingClient *logging.Client

	// Resource registries.
	disks           *diskRegistry
	forwardingRules *forwardingRuleRegistry
	firewallRules   *firewallRuleRegistry
	images          *imageRegistry
	machineImages   *machineImageRegistry
	instances       *instanceRegistry
	networks        *networkRegistry
	subnetworks     *subnetworkRegistry
	targetInstances *targetInstanceRegistry
	objects         *objectRegistry
	snapshots       *snapshotRegistry

	// Cache of resources
	machineTypeCache    twoDResourceCache
	instanceCache       twoDResourceCache
	diskCache           twoDResourceCache
	subnetworkCache     twoDResourceCache
	targetInstanceCache twoDResourceCache
	forwardingRuleCache twoDResourceCache
	imageCache          oneDResourceCache
	imageFamilyCache    oneDResourceCache
	machineImageCache   oneDResourceCache
	networkCache        oneDResourceCache
	firewallRuleCache   oneDResourceCache
	zonesCache          oneDResourceCache
	regionsCache        oneDResourceCache
	licenseCache        oneDResourceCache
	snapshotCache       oneDResourceCache

	stepTimeRecords             []TimeRecord
	serialControlOutputValues   map[string]string
	serialControlOutputValuesMx sync.Mutex
	//Forces cleanup on error of all resources, including those marked with NoCleanup
	ForceCleanupOnError bool
	// forceCleanup is set to true when resources should be forced clean, even when NoCleanup is set to true
	forceCleanup bool
	// cancelReason provides custom reason when workflow is canceled. f
	cancelReason string
}

//DisableCloudLogging disables logging to Cloud Logging for this workflow.
func (w *Workflow) DisableCloudLogging() {
	w.cloudLoggingDisabled = true
}

//DisableGCSLogging disables logging to GCS for this workflow.
func (w *Workflow) DisableGCSLogging() {
	w.gcsLoggingDisabled = true
}

//DisableStdoutLogging disables logging to stdout for this workflow.
func (w *Workflow) DisableStdoutLogging() {
	w.stdoutLoggingDisabled = true
}

// AddVar adds a variable set to the Workflow.
func (w *Workflow) AddVar(k, v string) {
	if w.Vars == nil {
		w.Vars = map[string]Var{}
	}
	w.Vars[k] = Var{Value: v}
}

// AddSerialConsoleOutputValue adds an serial-output key-value pair to the Workflow.
func (w *Workflow) AddSerialConsoleOutputValue(k, v string) {
	w.serialControlOutputValuesMx.Lock()
	if w.serialControlOutputValues == nil {
		w.serialControlOutputValues = map[string]string{}
	}
	w.serialControlOutputValues[k] = v
	w.serialControlOutputValuesMx.Unlock()
}

// GetSerialConsoleOutputValue gets an serial-output value by key.
func (w *Workflow) GetSerialConsoleOutputValue(k string) string {
	return w.serialControlOutputValues[k]
}

func (w *Workflow) addCleanupHook(hook func() DError) {
	w.cleanupHooksMx.Lock()
	w.cleanupHooks = append(w.cleanupHooks, hook)
	w.cleanupHooksMx.Unlock()
}

// SetLogProcessHook sets a hook function to process log string
func (w *Workflow) SetLogProcessHook(hook func(string) string) {
	w.logProcessHook = hook
}

// Validate runs validation on the workflow.
func (w *Workflow) Validate(ctx context.Context) DError {
	if err := w.PopulateClients(ctx); err != nil {
		w.CancelWorkflow()
		return Errf("error populating workflow: %v", err)
	}

	if err := w.validateRequiredFields(); err != nil {
		w.CancelWorkflow()
		return Errf("error validating workflow: %v", err)
	}

	if err := w.populate(ctx); err != nil {
		w.CancelWorkflow()
		return Errf("error populating workflow: %v", err)
	}

	w.LogWorkflowInfo("Validating workflow")
	if err := w.validate(ctx); err != nil {
		w.LogWorkflowInfo("Error validating workflow: %v", err)
		w.CancelWorkflow()
		return err
	}
	w.LogWorkflowInfo("Validation Complete")
	return nil
}

// WorkflowModifier is a function type for functions that can modify a Workflow object.
type WorkflowModifier func(*Workflow)

// Run runs a workflow.
func (w *Workflow) Run(ctx context.Context) error {
	return w.RunWithModifiers(ctx, nil, nil)
}

// RunWithModifiers runs a workflow with the ability to modify it before and/or after validation.
func (w *Workflow) RunWithModifiers(
	ctx context.Context,
	preValidateWorkflowModifier WorkflowModifier,
	postValidateWorkflowModifier WorkflowModifier) (err DError) {

	w.externalLogging = true
	if preValidateWorkflowModifier != nil {
		preValidateWorkflowModifier(w)
	}
	if err = w.Validate(ctx); err != nil {
		return err
	}

	if postValidateWorkflowModifier != nil {
		postValidateWorkflowModifier(w)
	}
	defer w.cleanup()
	defer func() {
		if err != nil {
			w.forceCleanup = w.ForceCleanupOnError
		}
	}()

	w.LogWorkflowInfo("Workflow Project: %s", w.Project)
	w.LogWorkflowInfo("Workflow Zone: %s", w.Zone)
	w.LogWorkflowInfo("Workflow GCSPath: %s", w.GCSPath)
	w.LogWorkflowInfo("Daisy scratch path: https://console.cloud.google.com/storage/browser/%s", path.Join(w.bucket, w.scratchPath))

	w.LogWorkflowInfo("Uploading sources")
	if err = w.uploadSources(ctx); err != nil {
		w.LogWorkflowInfo("Error uploading sources: %v", err)
		w.CancelWorkflow()
		return err
	}
	w.LogWorkflowInfo("Running workflow")
	defer func() {
		for k, v := range w.serialControlOutputValues {
			w.LogWorkflowInfo("Serial-output value -> %v:%v", k, v)
		}
	}()
	if err = w.run(ctx); err != nil {
		w.LogWorkflowInfo("Error running workflow: %v", err)
		return err
	}

	return nil
}

func (w *Workflow) recordStepTime(stepName string, startTime time.Time, endTime time.Time) {
	if w.parent == nil {
		w.recordTimeMx.Lock()
		w.stepTimeRecords = append(w.stepTimeRecords, TimeRecord{stepName, startTime, endTime})
		w.recordTimeMx.Unlock()
	} else {
		w.parent.recordStepTime(fmt.Sprintf("%s.%s", w.Name, stepName), startTime, endTime)
	}
}

// GetStepTimeRecords returns time records of each steps
func (w *Workflow) GetStepTimeRecords() []TimeRecord {
	return w.stepTimeRecords
}

func (w *Workflow) cleanup() {
	startTime := time.Now()
	w.LogWorkflowInfo("Workflow %q cleaning up (this may take up to 2 minutes).", w.Name)

	select {
	case <-w.Cancel:
	default:
		w.CancelWorkflow()
	}

	// Allow goroutines that are watching w.Cancel an opportunity
	// to detect that the workflow was cancelled and to cleanup.
	c := make(chan struct{})
	go func() {
		w.stepWait.Wait()
		close(c)
	}()
	select {
	case <-c:
	case <-time.After(4 * time.Second):
	}

	for _, hook := range w.cleanupHooks {
		if err := hook(); err != nil {
			w.LogWorkflowInfo("Error returned from cleanup hook: %s", err)
		}
	}
	w.LogWorkflowInfo("Workflow %q finished cleanup.", w.Name)
	w.recordStepTime("workflow cleanup", startTime, time.Now())
}

func (w *Workflow) genName(n string) string {
	name := w.Name
	for parent := w.parent; parent != nil; parent = parent.parent {
		name = parent.Name + "-" + name
	}
	prefix := name
	if n != "" {
		prefix = fmt.Sprintf("%s-%s", n, name)
	}
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
			return typedErr(apiError, "failed to create compute client", err)
		}
	}

	storageOptions := []option.ClientOption{option.WithCredentialsFile(w.OAuthPath)}
	if w.StorageClient == nil {
		w.StorageClient, err = storage.NewClient(ctx, storageOptions...)
		if err != nil {
			return err
		}
	}

	loggingOptions := []option.ClientOption{option.WithCredentialsFile(w.OAuthPath)}
	if w.externalLogging && w.cloudLoggingClient == nil {
		w.cloudLoggingClient, err = logging.NewClient(ctx, w.Project, loggingOptions...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Workflow) populateStep(ctx context.Context, s *Step) DError {
	if s.Timeout == "" {
		s.Timeout = w.DefaultTimeout
	}
	timeout, err := time.ParseDuration(s.Timeout)
	if err != nil {
		return newErr(fmt.Sprintf("failed to parse duration for workflow %v, step %v", w.Name, s.name), err)
	}
	s.timeout = timeout

	var derr DError
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
func (w *Workflow) populate(ctx context.Context) DError {
	for k, v := range w.Vars {
		if v.Required && v.Value == "" {
			return Errf("cannot populate workflow, required var %q is unset", k)
		}
	}

	// Set some generic autovars and run first round of var substitution.
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

	// Parse timeout.
	timeout, err := time.ParseDuration(w.DefaultTimeout)
	if err != nil {
		return Errf("failed to parse timeout for workflow: %v", err)
	}
	w.defaultTimeout = timeout

	// Set up GCS paths.
	if w.GCSPath == "" {
		dBkt, err := daisyBkt(ctx, w.StorageClient, w.Project)
		if err != nil {
			return err
		}
		w.GCSPath = "gs://" + dBkt
	}
	bkt, p, derr := splitGCSPath(w.GCSPath)
	if derr != nil {
		return derr
	}
	w.bucket = bkt
	w.scratchPath = path.Join(p, fmt.Sprintf("daisy-%s-%s-%s", w.Name, now.Format("20060102-15:04:05"), w.id))
	w.sourcesPath = path.Join(w.scratchPath, "sources")
	w.logsPath = path.Join(w.scratchPath, "logs")
	w.outsPath = path.Join(w.scratchPath, "outs")

	// Generate more autovars from workflow fields. Run second round of var substitution.
	w.autovars["NAME"] = w.Name
	w.autovars["FULLNAME"] = w.genName("")
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

	// We do this here, and not in validate, as embedded startup scripts could
	// have what we think are daisy variables.
	if err := w.validateVarsSubbed(); err != nil {
		return err
	}

	if err := w.substituteSourceVars(ctx, reflect.ValueOf(w).Elem()); err != nil {
		return err
	}

	if w.Logger == nil {
		w.createLogger(ctx)
	}

	// Run populate on each step.
	for name, s := range w.Steps {
		s.name = name
		s.w = w
		if err := w.populateStep(ctx, s); err != nil {
			return Errf("error populating step %q: %v", name, err)
		}
	}
	return nil
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

func (w *Workflow) includeWorkflow(iw *Workflow) {
	iw.Cancel = w.Cancel
	iw.parent = w
	iw.disks = w.disks
	iw.forwardingRules = w.forwardingRules
	iw.firewallRules = w.firewallRules
	iw.images = w.images
	iw.machineImages = w.machineImages
	iw.instances = w.instances
	iw.networks = w.networks
	iw.subnetworks = w.subnetworks
	iw.targetInstances = w.targetInstances
	iw.snapshots = w.snapshots
	iw.objects = w.objects
}

// ID is the unique identifyier for this Workflow.
func (w *Workflow) ID() string {
	return w.id
}

// NewIncludedWorkflowFromFile reads and unmarshals a workflow with the same resources as the parent.
func (w *Workflow) NewIncludedWorkflowFromFile(file string) (*Workflow, error) {
	iw := New()
	w.includeWorkflow(iw)
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
	w.externalLogging = false
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

func (w *Workflow) run(ctx context.Context) DError {
	return w.traverseDAG(func(s *Step) DError {
		return w.runStep(ctx, s)
	})
}

func (w *Workflow) runStep(ctx context.Context, s *Step) DError {
	timeout := make(chan struct{})
	go func() {
		time.Sleep(s.timeout)
		close(timeout)
	}()

	e := make(chan DError)
	go func() {
		e <- s.run(ctx)
	}()

	select {
	case err := <-e:
		return err
	case <-timeout:
		return s.getTimeoutError()
	}
}

// Concurrently traverse the DAG, running func f on each step.
// Return an error if f returns an error on any step.
func (w *Workflow) traverseDAG(f func(*Step) DError) DError {
	// waiting = steps and the dependencies they are waiting for.
	// running = the currently running steps.
	// start = map of steps' start channels/semaphores.
	// done = map of steps' done channels for signaling step completion.
	waiting := map[string][]string{}
	var running []string
	start := map[string]chan DError{}
	done := map[string]chan DError{}

	// Setup: channels, copy dependencies.
	for name := range w.Steps {
		waiting[name] = w.Dependencies[name]
		start[name] = make(chan DError)
		done[name] = make(chan DError)
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
	w.DefaultTimeout = defaultTimeout
	w.autovars = map[string]string{}

	// Resource registries and cleanup.
	w.disks = newDiskRegistry(w)
	w.forwardingRules = newForwardingRuleRegistry(w)
	w.firewallRules = newFirewallRuleRegistry(w)
	w.images = newImageRegistry(w)
	w.machineImages = newMachineImageRegistry(w)
	w.instances = newInstanceRegistry(w)
	w.networks = newNetworkRegistry(w)
	w.subnetworks = newSubnetworkRegistry(w)
	w.objects = newObjectRegistry(w)
	w.targetInstances = newTargetInstanceRegistry(w)
	w.snapshots = newSnapshotRegistry(w)
	w.addCleanupHook(func() DError {
		w.instances.cleanup() // instances need to be done before disks/networks
		w.images.cleanup()
		w.machineImages.cleanup()
		w.disks.cleanup()
		w.forwardingRules.cleanup()
		w.targetInstances.cleanup()
		w.firewallRules.cleanup()
		w.subnetworks.cleanup()
		w.networks.cleanup()
		w.snapshots.cleanup()
		return nil
	})

	w.id = randString(5)
	return w
}

// NewFromFile reads and unmarshals a workflow file.
// Recursively reads sub and included steps as well.
func NewFromFile(file string) (w *Workflow, err error) {
	w = New()
	if err := readWorkflow(file, w); err != nil {
		return nil, err
	}
	for _, step := range w.Steps {
		if step.SubWorkflow != nil && step.SubWorkflow.Path != "" {
			step.SubWorkflow.Workflow, err = w.NewSubWorkflowFromFile(step.SubWorkflow.Path)
		} else if step.IncludeWorkflow != nil && step.IncludeWorkflow.Path != "" {
			step.IncludeWorkflow.Workflow, err = w.NewIncludedWorkflowFromFile(step.IncludeWorkflow.Path)
		} else {
			continue
		}
		if err != nil {
			return nil, err
		}
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

func readWorkflow(file string, w *Workflow) DError {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return newErr("failed to read workflow file", err)
	}

	w.workflowDir, err = filepath.Abs(filepath.Dir(file))
	if err != nil {
		return newErr("failed to get absolute path of workflow file", err)
	}

	if err := json.Unmarshal(data, &w); err != nil {
		return newErr("failed to unmarshal workflow file", JSONError(file, data, err))
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
func stepsListen(names []string, chans map[string]chan DError) (string, DError) {
	cases := make([]reflect.SelectCase, len(names))
	for i, name := range names {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(chans[name])}
	}
	caseIndex, value, recvOk := reflect.Select(cases)
	name := names[caseIndex]
	if recvOk {
		// recvOk -> a step failed, return the error.
		return name, value.Interface().(DError)
	}
	return name, nil
}

// IterateWorkflowSteps iterates over all workflow steps, including included
// workflow steps, and calls cb callback function
func (w *Workflow) IterateWorkflowSteps(cb func(step *Step)) {
	for _, step := range w.Steps {
		if step.IncludeWorkflow != nil {
			//recurse into included workflow
			step.IncludeWorkflow.Workflow.IterateWorkflowSteps(cb)
		}
		cb(step)
	}
}

// CancelWithReason cancels workflow with a specific reason.
// The specific reason replaces "is canceled" in the default error message.
// Multiple invocations will not cause an error, but only the first reason
// will be retained.
func (w *Workflow) CancelWithReason(reason string) {
	w.cancelMx.Lock()
	if w.cancelReason == "" {
		w.cancelReason = reason
	}
	w.cancelMx.Unlock()
	w.CancelWorkflow()
}

// CancelWorkflow cancels the workflow. Safe to call multiple times.
// Prefer this to closing the w.Cancel channel,
// which will panic if it has already been closed.
func (w *Workflow) CancelWorkflow() {
	w.cancelMx.Lock()
	defer w.cancelMx.Unlock()

	if !w.isCanceled {
		w.isCanceled = true
		// Extra guard in case something manually closed the channel.
		defer func() { recover() }()
		close(w.Cancel)
	}
}

func (w *Workflow) getCancelReason() string {
	cancelReason := w.cancelReason
	for wi := w; cancelReason == "" && wi != nil; wi = wi.parent {
		cancelReason = wi.cancelReason
	}
	return cancelReason
}

func (w *Workflow) onStepCancel(s *Step, stepClass string) DError {
	if s == nil {
		return nil
	}
	cancelReason := w.getCancelReason()
	if cancelReason == "" {
		cancelReason = "is canceled"
	}
	errorMessageFormat := "Step %q (%s) " + cancelReason + "."

	s.w.LogWorkflowInfo(errorMessageFormat, s.name, stepClass)
	return Errf(errorMessageFormat, s.name, stepClass)
}
